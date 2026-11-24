package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPresenceRankStillNeedsAFetch(t *testing.T) {
	c := &riotClient{rankCache: map[string]rankHit{}}
	id := "11111111-1111-1111-1111-111111111111"
	c.rememberRank(id, 15, 20, 0)
	players := []player{{PUUID: id, Tier: 15}}
	if !c.ranksMissing(players) {
		t.Fatal("a presence rank without a fetch must still be loaded")
	}
	c.mu.Lock()
	hit := c.rankCache[id]
	c.mu.Unlock()
	if hit.fetched || rankDone(hit, true, time.Now()) {
		t.Fatal("presence rank was treated as finished")
	}
}

func TestFailedRankRetriesQuickly(t *testing.T) {
	c := &riotClient{rankCache: map[string]rankHit{
		"22222222-2222-2222-2222-222222222222": {at: time.Now().Add(-9 * time.Second)},
	}}
	players := []player{{PUUID: "22222222-2222-2222-2222-222222222222"}}
	if !c.ranksMissing(players) {
		t.Fatal("a failed rank was hidden")
	}
	now := time.Now()
	c.mu.Lock()
	hit := c.rankCache[players[0].PUUID]
	c.mu.Unlock()
	if ok := true; ok && !hit.fetched && !hit.known && now.Sub(hit.at) < 8*time.Second {
		t.Fatal("retry window is still long")
	}
}

func TestLiveRanksPaintForTheWholeLobby(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"QueueSkills":{"competitive":{"SeasonalInfoBySeasonID":{"act":{"CompetitiveTier":18,"RankedRating":40,"HighestTierAchieved":21}}}}}`))
	}))
	defer srv.Close()
	c := &riotClient{rankCache: map[string]rankHit{}, rankIcons: map[int]string{}}
	c.rankHTTP = srv.Client()
	c.rankHTTP.Transport = rewriteHost{base: c.rankHTTP.Transport, host: srv.Listener.Addr().String()}
	players := make([]player, 10)
	for i := range players {
		players[i].PUUID = "33333333-3333-3333-3333-33333333333" + string(rune('0'+i))
	}
	start := time.Now()
	c.fillRanks(t.Context(), riotSession{shard: "eu", access: "a", entitlement: "b", version: "1"}, players)
	if time.Since(start) > 2*time.Second {
		t.Fatalf("ranks took %s", time.Since(start))
	}
	c.paintRanks(players)
	for i, p := range players {
		if p.Rank == "" || p.Rank == "Unranked" || p.Peak == "" {
			t.Fatalf("player %d rank %q peak %q", i, p.Rank, p.Peak)
		}
	}
}

func TestEveryLiveStackIsLabeled(t *testing.T) {
	const (
		p1   = "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
		p2   = "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
		p3   = "cccccccc-cccc-cccc-cccc-cccccccccccc"
		solo = "dddddddd-dddd-dddd-dddd-dddddddddddd"
	)
	raw := map[string]string{
		"a1": p1, "a2": p1, "a3": p1,
		"b1": p2, "b2": p2,
		"c1": p3, "c2": p3, "c3": p3, "c4": p3,
		"d1": solo,
	}
	hints := map[string]partyHint{}
	players := make([]player, 0, len(raw))
	for id, party := range raw {
		hints[id] = partyHint{id: party}
		players = append(players, player{PUUID: id})
	}
	grouped := groupsOf(hints)
	markParties(players, grouped)
	got := map[string]string{}
	for _, p := range players {
		got[p.PUUID] = p.Party
	}
	if got["a1"] == "" || got["a1"] != got["a2"] || got["a2"] != got["a3"] {
		t.Fatalf("trio %+v", got)
	}
	if got["b1"] == "" || got["b1"] != got["b2"] || got["b1"] == got["a1"] {
		t.Fatalf("duo %+v", got)
	}
	if got["c1"] == "" || got["c1"] != got["c4"] || got["c1"] == got["a1"] || got["c1"] == got["b1"] {
		t.Fatalf("quad %+v", got)
	}
	if got["d1"] != "" {
		t.Fatalf("solo labeled %q", got["d1"])
	}
}

type rewriteHost struct {
	base http.RoundTripper
	host string
}

func (r rewriteHost) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = r.host
	base := r.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req)
}

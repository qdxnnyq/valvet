package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type rankHit struct {
	tier    int
	rr      int
	peak    int
	at      time.Time
	known   bool
	fetched bool
}

func (c *riotClient) ranksMissing(players []player) bool {
	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if id == "" {
			continue
		}
		hit, ok := c.rankCache[id]
		if rankDone(hit, ok, now) {
			continue
		}
		return true
	}
	return false
}

func (c *riotClient) fillRanks(ctx context.Context, sess riotSession, groups ...[]player) {
	if sess.shard == "" || ctx.Err() != nil {
		return
	}
	now := time.Now()
	need := []string{}
	seen := map[string]bool{}
	c.mu.Lock()
	quiet := c.rankQuiet
	c.mu.Unlock()
	c.mu.Lock()
	for _, group := range groups {
		for _, p := range group {
			id := strings.ToLower(p.PUUID)
			if id == "" || seen[id] {
				continue
			}
			hit, ok := c.rankCache[id]
			if rankDone(hit, ok, now) {
				continue
			}
			if ok && !hit.fetched && !hit.known && now.Sub(hit.at) < 8*time.Second {
				continue
			}
			seen[id] = true
			need = append(need, p.PUUID)
		}
	}
	c.mu.Unlock()
	if len(need) == 0 {
		return
	}
	if len(need) > 24 {
		need = need[:24]
	}
	out := make([]rankHit, len(need))
	var wg sync.WaitGroup
	limit := 5
	sem := make(chan struct{}, limit)
	store := func(id string, hit rankHit) {
		c.mu.Lock()
		if c.rankCache == nil {
			c.rankCache = map[string]rankHit{}
		}
		c.rankCache[strings.ToLower(id)] = hit
		c.mu.Unlock()
	}
	for i, id := range need {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			call, cancel := context.WithTimeout(context.Background(), 4*time.Second)
			defer cancel()
			status, payload, err := c.rankCall(call, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/mmr/v1/players/"+id, nil)
			var body map[string]any
			if err != nil || status >= 400 || json.Unmarshal(payload, &body) != nil {
				return
			}
			tier, rr := tierFromMMR(body)
			peak := peakTier(body)
			hit := rankHit{tier: tier, rr: rr, peak: peak, at: time.Now(), known: tier >= 3 || peak >= 3, fetched: true}
			store(id, hit)
			out[i] = hit
			if !quiet && tier < 3 {
				if t, r := c.tierFromUpdates(call, sess, id); t >= 3 {
					hit.tier = t
					hit.rr = r
					hit.at = time.Now()
					store(id, hit)
					out[i] = hit
				}
			}
		}(i, id)
	}
	wg.Wait()
	c.mu.Lock()
	if c.rankCache == nil {
		c.rankCache = map[string]rankHit{}
	}
	for i, id := range need {
		if out[i].known {
			continue
		}
		c.rankCache[strings.ToLower(id)] = rankHit{at: time.Now()}
	}
	c.mu.Unlock()
	c.saveRanks()
}

func rankDone(hit rankHit, ok bool, now time.Time) bool {
	return ok && hit.fetched && now.Sub(hit.at) < 6*time.Hour
}

func (c *riotClient) kickRanks(sess riotSession, players []player) {
	if sess.shard == "" || len(players) == 0 {
		return
	}
	now := time.Now()
	c.mu.Lock()
	need := false
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if id == "" {
			continue
		}
		hit, ok := c.rankCache[id]
		if rankDone(hit, ok, now) {
			continue
		}
		if ok && !hit.fetched && !hit.known && now.Sub(hit.at) < 8*time.Second {
			continue
		}
		need = true
		break
	}
	if !need || c.rankBusy {
		if need && c.rankBusy {
			c.rankWait = append([]player(nil), players...)
		}
		c.mu.Unlock()
		return
	}
	c.rankBusy = true
	copied := append([]player(nil), players...)
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			left := append([]player(nil), c.rankWait...)
			c.rankWait = nil
			c.rankBusy = false
			c.mu.Unlock()
			if len(left) > 0 {
				c.kickRanks(sess, left)
			}
		}()
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			c.fillRanks(ctx, sess, copied)
			cancel()
			c.mu.Lock()
			copied = append([]player(nil), c.rankWait...)
			c.rankWait = nil
			c.mu.Unlock()
			if len(copied) == 0 {
				return
			}
		}
	}()
}

func (c *riotClient) rankCall(ctx context.Context, sess riotSession, method, url string, body any) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sess.access)
	req.Header.Set("X-Riot-Entitlements-JWT", sess.entitlement)
	req.Header.Set("X-Riot-ClientPlatform", clientPlatform)
	req.Header.Set("X-Riot-ClientVersion", sess.version)
	client := c.rankHTTP
	if client == nil {
		client = c.remote
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return res.StatusCode, payload, nil
}

func (c *riotClient) tierFromUpdates(ctx context.Context, sess riotSession, id string) (int, int) {
	status, payload, err := c.rankCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/mmr/v1/players/"+id+"/competitiveupdates?startIndex=0&endIndex=8", nil)
	if err != nil || status >= 400 {
		return 0, 0
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return 0, 0
	}
	rows := asArr(pick(body, "Matches", "matches"))
	for _, item := range rows {
		row := asMap(item)
		tier := num(pick(row, "TierAfterUpdate", "tierAfterUpdate"))
		if tier >= 3 {
			return tier, num(pick(row, "RankedRatingAfterUpdate", "rankedRatingAfterUpdate"))
		}
	}
	return 0, 0
}

type savedRank struct {
	ID      string `json:"id"`
	Tier    int    `json:"tier"`
	RR      int    `json:"rr"`
	Peak    int    `json:"peak"`
	At      int64  `json:"at"`
	Fetched bool   `json:"fetched,omitempty"`
}

func loadRankCache() map[string]rankHit {
	var rows []savedRank
	out := map[string]rankHit{}
	if _, ok := readCache("ranks", &rows); !ok {
		return out
	}
	now := time.Now()
	for _, row := range rows {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if id == "" || (row.Tier < 3 && row.Peak < 3) {
			continue
		}
		at := time.Unix(row.At, 0)
		if now.Sub(at) > 6*time.Hour {
			continue
		}
		out[id] = rankHit{tier: row.Tier, rr: row.RR, peak: row.Peak, at: at, known: true, fetched: row.Fetched || row.Peak >= 3}
	}
	return out
}

func (c *riotClient) saveRanks() {
	c.mu.Lock()
	rows := make([]savedRank, 0, len(c.rankCache))
	for id, hit := range c.rankCache {
		if !hit.known || (hit.tier < 3 && hit.peak < 3) {
			continue
		}
		rows = append(rows, savedRank{ID: id, Tier: hit.tier, RR: hit.rr, Peak: hit.peak, At: hit.at.Unix(), Fetched: hit.fetched})
	}
	c.mu.Unlock()
	if len(rows) == 0 {
		return
	}
	writeCache("ranks", rows)
}

func (c *riotClient) rememberRank(id string, tier, rr, peak int) {
	if tier < 3 && peak < 3 {
		return
	}
	id = strings.ToLower(id)
	c.mu.Lock()
	if c.rankCache == nil {
		c.rankCache = map[string]rankHit{}
	}
	prev := c.rankCache[id]
	if prev.tier >= 3 && tier < 3 {
		c.mu.Unlock()
		return
	}
	if peak < prev.peak {
		peak = prev.peak
	}
	if rr == 0 && prev.rr > 0 && prev.tier == tier {
		rr = prev.rr
	}
	c.rankCache[id] = rankHit{tier: tier, rr: rr, peak: peak, at: time.Now(), known: true, fetched: prev.fetched}
	c.mu.Unlock()
}

func (c *riotClient) stampPresence(players []player, tiers map[string]int) []player {
	if len(players) == 0 || len(tiers) == 0 {
		return players
	}
	c.mu.Lock()
	icons := c.rankIcons
	c.mu.Unlock()
	for i := range players {
		tier := tiers[strings.ToLower(players[i].PUUID)]
		applyPresenceTier(&players[i], tier, icons)
		if tier >= 3 {
			c.rememberRank(players[i].PUUID, tier, 0, 0)
		}
	}
	return players
}

func applyPresenceTier(p *player, tier int, icons map[int]string) {
	if p == nil || tier < 3 || p.Tier >= 3 {
		return
	}
	p.Tier = tier
	p.Rank = rankName(tier)
	p.RankPeak = false
	if icon := rankArt(tier, icons); icon != "" {
		p.RankIcon = icon
	}
}

func applyRankHit(p *player, hit rankHit, icons map[int]string) {
	if hit.tier >= 3 {
		p.Rank = rankName(hit.tier)
		p.Tier = hit.tier
		p.RR = hit.rr
		p.RankIcon = rankArt(hit.tier, icons)
		p.RankPeak = false
		if hit.peak >= 3 {
			p.Peak = rankName(hit.peak)
			p.PeakIcon = rankArt(hit.peak, icons)
		}
		return
	}
	if hit.peak >= 3 {
		p.Peak = rankName(hit.peak)
		p.PeakIcon = rankArt(hit.peak, icons)
		p.Rank = p.Peak
		p.RankIcon = p.PeakIcon
		p.RankPeak = true
		p.Tier = 0
		p.RR = 0
		return
	}
	if p.Tier >= 3 || (p.Rank != "" && p.Rank != "Unranked") {
		return
	}
	p.Rank = "Unranked"
	p.RankIcon = ""
	p.RankPeak = false
	p.Tier = 0
	p.RR = 0
}

func (c *riotClient) paintRanks(players []player) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i := range players {
		hit, ok := c.rankCache[strings.ToLower(players[i].PUUID)]
		if ok && hit.known && (hit.tier >= 3 || hit.peak >= 3 || players[i].Tier < 3) {
			applyRankHit(&players[i], hit, c.rankIcons)
			continue
		}
		if players[i].Rank == "" && players[i].Tier > 0 {
			players[i].Rank = rankName(players[i].Tier)
		}
		if players[i].Tier > 0 && !strings.HasPrefix(players[i].RankIcon, "/pack/rank/") {
			if icon := rankArt(players[i].Tier, c.rankIcons); icon != "" {
				players[i].RankIcon = icon
			}
		}
	}
}

type profileMatch struct {
	Map     string      `json:"map"`
	Mode    string      `json:"mode"`
	When    string      `json:"when"`
	Agent   string      `json:"agent"`
	Score1  *int        `json:"score1"`
	Score2  *int        `json:"score2"`
	Team1   string      `json:"team1"`
	Team2   string      `json:"team2"`
	Won     *bool       `json:"won"`
	Side    string      `json:"side"`
	Kills   string      `json:"kills,omitempty"`
	Deaths  string      `json:"deaths,omitempty"`
	Assists string      `json:"assists,omitempty"`
	HS      string      `json:"hs,omitempty"`
	ADR     string      `json:"adr,omitempty"`
	Players []player    `json:"players,omitempty"`
	Rounds  []roundBeat `json:"rounds,omitempty"`
}

type profileView struct {
	Name      string         `json:"name"`
	Tag       string         `json:"tag"`
	Rank      string         `json:"rank"`
	RankIcon  string         `json:"rankIcon,omitempty"`
	Peak      string         `json:"peak,omitempty"`
	PeakIcon  string         `json:"peakIcon,omitempty"`
	RR        int            `json:"rr,omitempty"`
	Agent     string         `json:"agent,omitempty"`
	AgentIcon string         `json:"agentIcon,omitempty"`
	Was       string         `json:"was,omitempty"`
	Matches   []profileMatch `json:"matches"`
	Error     string         `json:"error,omitempty"`
}

func rankArt(tier int, icons map[int]string) string {
	if icon := packRankIcon(tier); icon != "" {
		return icon
	}
	if icons != nil {
		return icons[tier]
	}
	return ""
}

func applyProfileRank(view *profileView, hit rankHit, icons map[int]string) {
	if hit.tier >= 3 {
		view.Rank = rankName(hit.tier)
		view.RR = hit.rr
		view.RankIcon = rankArt(hit.tier, icons)
	} else if hit.peak >= 3 {
		view.Rank = rankName(hit.peak)
		view.RankIcon = rankArt(hit.peak, icons)
	}
	if hit.peak >= 3 {
		view.Peak = rankName(hit.peak)
		view.PeakIcon = rankArt(hit.peak, icons)
	}
}

func profileFromRecent(rows []recentMatch, id string) (matches []profileMatch, agent, icon, name, tag string) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return nil, "", "", "", ""
	}
	for _, row := range rows {
		var self player
		found := false
		localTeam := ""
		for _, p := range row.Players {
			if p.You && localTeam == "" {
				localTeam = p.Team
			}
			if !found && strings.EqualFold(p.PUUID, id) {
				self = p
				found = true
			}
		}
		if !found {
			continue
		}
		if name == "" && self.Name != "" && self.Name != "Player" {
			name, tag = self.Name, self.Tag
		}
		if agent == "" && self.Agent != "" {
			agent = self.Agent
			icon = self.AgentIcon
		}
		people := append([]player(nil), row.Players...)
		for i := range people {
			people[i].You = strings.EqualFold(people[i].PUUID, id)
		}
		var won *bool
		if row.Won != nil {
			v := *row.Won
			if localTeam != "" && self.Team != "" && !strings.EqualFold(self.Team, localTeam) {
				v = !v
			}
			won = &v
		}
		matches = append(matches, profileMatch{
			Map: row.Map, Mode: row.Mode, When: row.When,
			Score1: row.Score1, Score2: row.Score2,
			Team1: row.Team1, Team2: row.Team2,
			Won: won, Side: row.Side,
			Players: people, Rounds: row.Rounds,
			Agent: self.Agent, Kills: self.Kills, Deaths: self.Deaths,
			Assists: self.Assists, HS: self.HS, ADR: self.ADR,
		})
		if len(matches) >= 8 {
			break
		}
	}
	return matches, agent, icon, name, tag
}

func (c *riotClient) profile(ctx context.Context, id string) profileView {
	c.mu.Lock()
	sess := c.session
	agents := c.agents
	icons := c.rankIcons
	recent := append([]recentMatch(nil), c.recent...)
	var cached rankHit
	cachedOK := false
	if c.rankCache != nil {
		cached, cachedOK = c.rankCache[strings.ToLower(id)]
	}
	c.mu.Unlock()
	view := profileView{Matches: []profileMatch{}}
	if cachedOK && cached.known {
		applyProfileRank(&view, cached, icons)
	}
	if rows, agent, icon, name, tag := profileFromRecent(recent, id); len(rows) > 0 {
		view.Matches = rows
		view.Agent, view.AgentIcon = agent, icon
		if name != "" {
			view.Name, view.Tag = name, tag
		}
	}
	if sess.shard == "" || sess.access == "" {
		if len(view.Matches) > 0 || view.Name != "" || view.Rank != "" {
			return view
		}
		return profileView{Error: "Open Valorant first, then click the name again."}
	}
	shared := append([]profileMatch(nil), view.Matches...)
	view.Matches = nil
	if view.Name == "" {
		type named struct{ name, tag string }
		namesCh := make(chan named, 1)
		go func() {
			got := named{}
			if book := c.names(ctx, sess, []string{id}); book != nil {
				if pair, ok := book[strings.ToLower(id)]; ok {
					got.name, got.tag = pair[0], pair[1]
				}
			}
			namesCh <- got
		}()
		select {
		case got := <-namesCh:
			if got.name != "" {
				view.Name, view.Tag = got.name, got.tag
			}
		case <-ctx.Done():
		case <-time.After(400 * time.Millisecond):
		}
	}
	if view.Rank == "" {
		status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/mmr/v1/players/"+id, nil)
		if err == nil && status < 400 {
			var body map[string]any
			if json.Unmarshal(payload, &body) == nil {
				tier, rr := tierFromMMR(body)
				peak := peakTier(body)
				applyProfileRank(&view, rankHit{tier: tier, rr: rr, peak: peak, known: true}, icons)
				c.mu.Lock()
				if c.rankCache == nil {
					c.rankCache = map[string]rankHit{}
				}
				c.rankCache[strings.ToLower(id)] = rankHit{tier: tier, rr: rr, peak: peak, at: time.Now(), known: tier >= 3 || peak >= 3}
				c.mu.Unlock()
			}
		}
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/match-history/v1/history/"+id+"?startIndex=0&endIndex=10", nil)
	if err != nil || status >= 400 {
		view.Matches = shared
		if view.Name == "" && view.Rank == "" && len(view.Matches) == 0 {
			view.Error = "Couldn't open that player."
		}
		return view
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		view.Matches = shared
		return view
	}
	ids := []string{}
	for _, item := range asArr(pick(body, "History", "history")) {
		mid := str(pick(asMap(item), "MatchID", "matchId"))
		if mid != "" {
			ids = append(ids, mid)
		}
	}
	if len(ids) > 10 {
		ids = ids[:10]
	}
	type part struct {
		row  *recentMatch
		name string
	}
	parts := make([]part, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 3)
	detailCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for i, mid := range ids {
		wg.Add(1)
		go func(i int, mid string) {
			defer wg.Done()
			defer func() { _ = recover() }()
			sem <- struct{}{}
			defer func() { <-sem }()
			status, payload, err := c.remoteCall(detailCtx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/match-details/v1/matches/"+mid, nil)
			if err != nil || status >= 400 {
				return
			}
			var detail map[string]any
			if json.Unmarshal(payload, &detail) != nil {
				return
			}
			row := finalizeRecent(detail, id, agents)
			label := ""
			for _, p := range roster(detail, nil) {
				if strings.EqualFold(p.PUUID, id) && p.Name != "" && p.Name != "Player" {
					label = p.Name
					if p.Tag != "" {
						label += "#" + p.Tag
					}
				}
			}
			parts[i] = part{row: row, name: label}
		}(i, mid)
	}
	wg.Wait()
	var seen []string
	last := ""
	for _, part := range parts {
		if part.name != "" && !strings.EqualFold(part.name, last) {
			seen = append(seen, part.name)
			last = part.name
		}
		if part.row == nil {
			continue
		}
		var self *player
		for i := range part.row.Players {
			if part.row.Players[i].You {
				self = &part.row.Players[i]
			}
		}
		item := profileMatch{
			Map: part.row.Map, Mode: part.row.Mode, When: part.row.When,
			Score1: part.row.Score1, Score2: part.row.Score2,
			Team1: part.row.Team1, Team2: part.row.Team2,
			Won: part.row.Won, Side: part.row.Side,
			Players: part.row.Players, Rounds: part.row.Rounds,
		}
		if self != nil {
			item.Agent = self.Agent
			item.Kills = self.Kills
			item.Deaths = self.Deaths
			item.Assists = self.Assists
			item.HS = self.HS
			item.ADR = self.ADR
			if view.Agent == "" {
				view.Agent = self.Agent
				view.AgentIcon = self.AgentIcon
			}
		}
		view.Matches = append(view.Matches, item)
	}
	if len(view.Matches) == 0 {
		view.Matches = shared
	}
	view.Was = withoutCurrent(strings.Join(seen, ", "), view.Name, view.Tag)
	return view
}

type rrPoint struct {
	Tier   int    `json:"tier"`
	RR     int    `json:"rr"`
	Rank   string `json:"rank"`
	When   int64  `json:"when,omitempty"`
	Earned int    `json:"earned"`
	ID     string `json:"id,omitempty"`
}

func (c *riotClient) rankSeries(ctx context.Context) map[string]any {
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	empty := map[string]any{"points": []rrPoint{}, "week": 0}
	if sess.shard == "" || sess.puuid == "" || sess.access == "" {
		return empty
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/mmr/v1/players/"+sess.puuid+"/competitiveupdates?startIndex=0&endIndex=20", nil)
	if err != nil || status >= 400 {
		return empty
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return empty
	}
	rows := asArr(pick(body, "Matches", "matches"))
	points := make([]rrPoint, 0, len(rows))
	week := 0
	cutoff := time.Now().Add(-7 * 24 * time.Hour).UnixMilli()
	for i := len(rows) - 1; i >= 0; i-- {
		row := asMap(rows[i])
		tier := num(pick(row, "TierAfterUpdate", "tierAfterUpdate"))
		if tier < 3 {
			continue
		}
		rr := num(pick(row, "RankedRatingAfterUpdate", "rankedRatingAfterUpdate"))
		when := int64(num(pick(row, "MatchStartTime", "matchStartTime")))
		if when > 0 && when < 1_000_000_000_000 {
			when *= 1000
		}
		earned := num(pick(row, "RankedRatingEarned", "rankedRatingEarned"))
		id := str(pick(row, "MatchID", "matchId", "MatchId"))
		points = append(points, rrPoint{Tier: tier, RR: rr, Rank: rankName(tier), When: when, Earned: earned, ID: id})
		if when >= cutoff {
			week += num(pick(row, "RankedRatingEarned", "rankedRatingEarned"))
		}
	}
	return map[string]any{"points": points, "week": week}
}

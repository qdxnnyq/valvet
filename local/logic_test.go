package main

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestDuosDoNotMerge(t *testing.T) {
	now := map[string]string{"a": "red", "b": "red", "c": "red", "d": "red", "me": "blue", "friend": "blue"}
	hints := stackHints(now, [][]string{{"a", "b"}, {"c", "d"}, {"a", "b", "c", "d"}, {"me", "friend"}})
	if hints["a"].id == "" || hints["a"].id != hints["b"].id {
		t.Fatalf("first duo: %+v", hints)
	}
	if hints["c"].id == "" || hints["c"].id != hints["d"].id || hints["c"].id == hints["a"].id {
		t.Fatalf("second duo merged: %+v", hints)
	}
	if hints["me"].id == "" || hints["me"].id != hints["friend"].id || hints["me"].id == hints["a"].id {
		t.Fatalf("own duo: %+v", hints)
	}
}

func TestArchiveShowsOtherStacks(t *testing.T) {
	a := "44444444-4444-4444-4444-444444444444"
	b := "55555555-5555-5555-5555-555555555555"
	me := "11111111-1111-1111-1111-111111111111"
	live := []player{
		{PUUID: a, Team: "Red"},
		{PUUID: b, Team: "Red"},
		{PUUID: me, Team: "Blue"},
	}
	rows := []recentMatch{{
		ID: "m1",
		Players: []player{
			{PUUID: a, Party: "A"},
			{PUUID: b, Party: "A"},
			{PUUID: me, Party: "B"},
		},
	}}
	hints := stacksFromArchive(rows, live)
	if hints[a].id == "" || hints[a].id != hints[b].id {
		t.Fatalf("enemy stack missing: %+v", hints)
	}
	if hints[me].id == hints[a].id {
		t.Fatalf("own player joined the enemy stack: %+v", hints)
	}
}

func TestStacksStaySeparate(t *testing.T) {
	me := "11111111-1111-1111-1111-111111111111"
	friend := "22222222-2222-2222-2222-222222222222"
	random := "33333333-3333-3333-3333-333333333333"
	a := "44444444-4444-4444-4444-444444444444"
	b := "55555555-5555-5555-5555-555555555555"
	hints := map[string]partyHint{}
	fillGroups(hints, map[string]partyHint{
		a: {id: "q"}, b: {id: "q"},
		me: {id: "p"}, friend: {id: "p"},
	})
	roster := map[string]bool{me: true, friend: true, random: true, a: true, b: true}
	forceParty(hints, "p", []string{me, friend, random}, roster)
	if hints[a].id != "q" || hints[b].id != "q" {
		t.Fatalf("other stack lost: %+v", hints)
	}
	if hints[me].id != "p" || hints[friend].id != "p" || hints[random].id != "p" {
		t.Fatalf("own stack: %+v", hints)
	}
	big := map[string]partyHint{}
	for i := 0; i < 6; i++ {
		big[string(rune('a'+i))] = partyHint{id: "all"}
	}
	if len(groupsOf(big)) != 0 {
		t.Fatal("lobby-wide party kept")
	}
}

func TestNewHelpers(t *testing.T) {
	name, tag, ok := splitRiotID(" Danny #EUW ")
	if !ok || name != "Danny" || tag != "EUW" {
		t.Fatalf("%s %s %v", name, tag, ok)
	}
	if shortLevel("Prime Vandal Level 4") != "Level 4" {
		t.Fatal(shortLevel("Prime Vandal Level 4"))
	}
	if shortChroma("Prime Vandal\n(Red)") != "Red" {
		t.Fatal(shortChroma("Prime Vandal\n(Red)"))
	}
	if expressionKind("d5f120f8-ff8c-4aac-92ea-f2b5acbe9475") != "Spray" || expressionKind("other") != "Expression" {
		t.Fatal("expression kind")
	}
	steps := levelXP(map[string]any{"content": map[string]any{"chapters": []any{map[string]any{"levels": []any{map[string]any{"xp": 2000}, map[string]any{"xp": 2500}}}}}})
	if len(steps) != 2 || steps[1] != 2500 {
		t.Fatal(steps)
	}
	if kindByTime(time.Now().Add(5*time.Hour).Format(time.RFC3339)) != "Daily" {
		t.Fatal("daily window")
	}
	week := time.Now().Add(4 * 24 * time.Hour).Format(time.RFC3339)
	soon := time.Now().Add(10 * time.Hour).Format(time.RFC3339)
	if missionWindow(soon, week) != "Daily" {
		t.Fatal("daily before weekly reset")
	}
	if missionWindow(week, week) != "Weekly" {
		t.Fatal("weekly reset")
	}
	code, name := codeFromINI("EAresFloatSettingName::CrosshairInnerLinesLineLength =4\r\nEAresStringSettingName::CrosshairProfileName =Main\r\nEAresIntSettingName::CrosshairColor =5\r\n")
	if name != "Main" || !strings.Contains(code, "0;P;c;5;") || !strings.Contains(code, "0l;4") {
		t.Fatal(name, code)
	}
	if cleanCopy("deal <debuffred>lethal</> damage") != "deal lethal damage" {
		t.Fatal(cleanCopy("deal <debuffred>lethal</> damage"))
	}
}

func TestUnfriendDiff(t *testing.T) {
	prev := map[string]friendNote{"a": {PUUID: "a", Name: "Old", Tag: "EU"}}
	first, gone, ready := diffFriends(nil, false, prev)
	if !ready || len(gone) != 0 || first["a"].Name != "Old" {
		t.Fatal("baseline")
	}
	_, gone, _ = diffFriends(first, true, map[string]friendNote{})
	if len(gone) != 0 {
		t.Fatal("empty list should not wipe friends")
	}
	_, gone, _ = diffFriends(first, true, map[string]friendNote{"b": {PUUID: "b", Name: "New", Tag: "NA"}})
	if len(gone) != 1 || gone[0].Name != "Old" {
		t.Fatal(gone)
	}
}

func TestEnemyOnSelect(t *testing.T) {
	ally := "11111111-1111-1111-1111-111111111111"
	enemy := "22222222-2222-2222-2222-222222222222"
	players := namedPlayers(map[string]any{
		"AllyTeam":  map[string]any{"TeamID": "Blue", "Players": []any{map[string]any{"Subject": ally, "CharacterID": "a", "CompetitiveTier": float64(15)}}},
		"EnemyTeam": map[string]any{"TeamID": "Red", "Players": []any{map[string]any{"Subject": enemy, "CharacterID": "b", "PlayerIdentity": map[string]any{"Incognito": true}}}},
	}, nil)
	if len(players) != 2 {
		t.Fatal(len(players))
	}
	var red string
	for _, p := range players {
		if strings.EqualFold(p.PUUID, enemy) {
			red = p.Team
			if !blankName(p.Name) && p.Name != "Player" {
				t.Fatal(p.Name)
			}
		}
	}
	if !strings.EqualFold(red, "Red") {
		t.Fatal(red)
	}
	tier, _ := tierFromMMR(map[string]any{"Matches": []any{map[string]any{"TierAfterUpdate": float64(18), "RankedRatingAfterUpdate": float64(40)}}})
	if tier != 18 {
		t.Fatal(tier)
	}
	book := namesFromAliases([]byte(`[{"puuid":"22222222-2222-2222-2222-222222222222","game_name":"TenZ","tag_line":"NA1"}]`))
	if book["22222222-2222-2222-2222-222222222222"][0] != "TenZ" {
		t.Fatal(book)
	}
	if prettyServer("aresriot.aws-euc1-prod.eu-gp-frankfurt-1") != "Frankfurt" {
		t.Fatal(prettyServer("aresriot.aws-euc1-prod.eu-gp-frankfurt-1"))
	}
	if prettyMap("/Game/Maps/Ascent/Ascent") != "Ascent" {
		t.Fatal(prettyMap("/Game/Maps/Ascent/Ascent"))
	}
	id, size := findParty(map[string]any{
		"isValid":            true,
		"partyPresenceData":  map[string]any{"partyId": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "partySize": float64(3)},
		"playerPresenceData": map[string]any{"competitiveTier": float64(15)},
	})
	if id != "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" || size != 3 {
		t.Fatal(id, size)
	}
	rawParty := `{"partyId":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb","partySize":2}`
	id, size = findParty(map[string]any{"partyPresenceData": rawParty})
	if id != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" || size != 2 {
		t.Fatal("string party", id, size)
	}
	if !aimedAt(0, 0, 0, 100, 0) {
		t.Fatal("aim ahead")
	}
	if aimedAt(0, 0, 0, 0, 100) {
		t.Fatal("aim side")
	}
	if multiText(map[int]int{2: 4, 3: 1}) != "2:4,3:1" {
		t.Fatal(multiText(map[int]int{2: 4, 3: 1}))
	}
	if num(presencePick(map[string]any{"playerPresenceData": map[string]any{"competitiveTier": float64(15)}}, "competitiveTier", "CompetitiveTier")) != 15 {
		t.Fatal("nested tier")
	}
	badgeID := "33333333-3333-3333-3333-333333333333"
	badged := namedPlayers(map[string]any{
		"Players": []any{map[string]any{
			"Subject": badgeID, "TeamID": "Red", "CharacterID": "b",
			"SeasonalBadgeInfo": map[string]any{"Rank": float64(21)},
		}},
	}, nil)
	if len(badged) != 1 || badged[0].Tier != 21 || badged[0].Rank != "Ascendant 1" {
		t.Fatalf("%+v", badged)
	}
	party := []player{{PUUID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}, {PUUID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}, {PUUID: "cccccccc-cccc-cccc-cccc-cccccccccccc"}}
	markParties(party, map[string]partyHint{
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa": {id: "p1"},
		"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb": {id: "p1"},
		"cccccccc-cccc-cccc-cccc-cccccccccccc": {id: "solo"},
	})
	if party[0].Party == "" || party[0].Party != party[1].Party || party[2].Party != "" || party[0].PartyN != 2 {
		t.Fatal(party)
	}
}

func TestStoreOffers(t *testing.T) {
	c := newRiotClient()
	id := "9c82e19d-4575-0200-1a81-3eacf00cf872"
	c.skins = map[string]agentInfo{id: {Name: "Prime Vandal", Icon: "https://example.test/v.png"}}
	offers := c.storeOffers(map[string]any{
		"SingleItemStoreOffers": []any{
			map[string]any{
				"Rewards": []any{map[string]any{"ItemID": id}},
				"Cost":    map[string]any{"85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741": float64(1775)},
			},
		},
		"SingleItemOffers": []any{id},
	})
	if len(offers) != 1 || offers[0].Name != "Prime Vandal" || offers[0].Price != 1775 || offers[0].Icon == "" {
		t.Fatalf("%+v", offers)
	}
}

func TestStoreHelpers(t *testing.T) {
	if resetLabel(3660) != "1h 1m" {
		t.Fatal(resetLabel(3660))
	}
	if presenceState(map[string]any{"partyState": "MATCHMAKING"}) != "In queue" {
		t.Fatal("queue")
	}
	if presenceState(map[string]any{"sessionLoopState": "INGAME"}) != "In game" {
		t.Fatal("ingame")
	}
	if presenceState(nil) != "Online" {
		t.Fatal("empty")
	}
	_, mode, _, _, _, _, _, _, _, _, _ := activity(map[string]any{
		"sessionLoopState": "INGAME",
		"queueId":          "swiftplay",
		"partySize":        float64(2),
		"maxPartySize":     float64(5),
	}, "", "valorant")
	if mode != "Swiftplay" {
		t.Fatalf("swiftplay %s", mode)
	}
	_, mode, _, _, _, _, _, _, _, _, _ = activity(map[string]any{
		"sessionLoopState":           "INGAME",
		"partyOwnerProvisioningFlow": "CustomGame",
		"customGameName":             "10 mans",
	}, "", "valorant")
	if mode != "Custom" {
		t.Fatalf("custom %s", mode)
	}
	if leagueMode(map[string]any{"gameStatus": "inGame", "gameQueueType": "RANKED_SOLO_5x5"}, "league_of_legends") != "Ranked Solo" {
		t.Fatal("ranked solo")
	}
	if leagueMode(map[string]any{"gameQueueType": "ARAM"}, "lol") != "ARAM" {
		t.Fatal("aram")
	}
	if leagueMode(map[string]any{"queueId": "1700", "gameStatus": "inGame"}, "league_of_legends") != "Arena" {
		t.Fatal("arena")
	}
	if leagueMode(nil, "tft") != "TFT" {
		t.Fatal("tft")
	}
	if vpCost(map[string]any{"85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741": 1775}) != 1775 {
		t.Fatal("vp")
	}
}

func TestDecodePrivateReadsPresenceToken(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"partyId":"11111111-1111-1111-1111-111111111111","competitiveTier":21,"partySize":3}`))
	got := decodePrivate("header." + payload + ".sign")
	if num(got["competitiveTier"]) != 21 {
		t.Fatal(got)
	}
	id, size := findParty(got)
	if id != "11111111-1111-1111-1111-111111111111" || size != 3 {
		t.Fatalf("party %s %d", id, size)
	}
}

func TestPresenceScoresReadsAllyEnemy(t *testing.T) {
	ally, enemy, ok := presenceScores(map[string]any{
		"partyOwnerMatchScoreAllyTeam":  float64(8),
		"partyOwnerMatchScoreEnemyTeam": float64(6),
	})
	if !ok || ally != 8 || enemy != 6 {
		t.Fatalf("scores %d %d %v", ally, enemy, ok)
	}
	ally, enemy, ok = presenceScores(map[string]any{
		"nested": map[string]any{"MatchScoreAllyTeam": float64(3), "MatchScoreEnemyTeam": float64(4)},
	})
	if !ok || ally != 3 || enemy != 4 {
		t.Fatalf("nested %d %d %v", ally, enemy, ok)
	}
}

func TestBuyAndTrades(t *testing.T) {
	if buyWord(4500) != "Full" || buyWord(2500) != "Force" || buyWord(800) != "Eco" {
		t.Fatal(buyWord(4500), buyWord(2500), buyWord(800))
	}
	edges := []killEdge{
		{Round: 1, T: 1000, KID: "a", VID: "b", Gun: "Vandal"},
		{Round: 1, T: 4000, KID: "c", VID: "a", Gun: "Ultimate"},
	}
	players := []player{{PUUID: "a", Team: "Red"}, {PUUID: "b", Team: "Blue"}, {PUUID: "c", Team: "Blue"}}
	markEdgeFlags(edges, players)
	if !edges[0].First || edges[1].First {
		t.Fatalf("first %+v", edges)
	}
	if !edges[1].Trade || !edges[1].Ult {
		t.Fatalf("trade %+v", edges[1])
	}
	when := time.Date(2026, 11, 17, 8, 0, 0, 0, time.UTC)
	line := penaltyLine([]byte(`{"Penalties":[{"QueueRestriction":{"QueueIDs":["competitive"],"Expiry":"2026-11-17T12:00:00Z"}}]}`), when)
	if !strings.Contains(line, "penalty until") {
		t.Fatal(line)
	}
	if penaltyLine([]byte(`{"Penalties":[{"Expiry":"2020-01-01T00:00:00Z"}]}`), when) != "" {
		t.Fatal("expired")
	}
}

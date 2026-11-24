package main

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

type agentInfo struct {
	Name string
	Icon string
	Anim string
}

type player struct {
	Name      string `json:"name"`
	Tag       string `json:"tag"`
	Agent     string `json:"agent"`
	AgentIcon string `json:"agentIcon,omitempty"`
	Rank      string `json:"rank"`
	RankIcon  string `json:"rankIcon,omitempty"`
	Team      string `json:"team"`
	Kills     string `json:"kills,omitempty"`
	Deaths    string `json:"deaths,omitempty"`
	Assists   string `json:"assists,omitempty"`
	HS        string `json:"hs,omitempty"`
	ADR       string `json:"adr,omitempty"`
	ACS       string `json:"acs,omitempty"`
	Damage    string `json:"damage,omitempty"`
	Was       string `json:"was,omitempty"`
	Level     int    `json:"level,omitempty"`
	Tier      int    `json:"tier,omitempty"`
	RR        int    `json:"rr,omitempty"`
	RankPeak  bool   `json:"rankPeak,omitempty"`
	Peak      string `json:"peak,omitempty"`
	PeakIcon  string `json:"peakIcon,omitempty"`
	You       bool   `json:"you,omitempty"`
	Enemy     bool   `json:"enemy,omitempty"`
	Fight     bool   `json:"fight,omitempty"`
	Locked    bool   `json:"locked,omitempty"`
	PUUID     string `json:"puuid,omitempty"`
	Party     string `json:"party,omitempty"`
	PartyN    int    `json:"partyN,omitempty"`
	PartyID   string `json:"-"`
	Score     string `json:"score,omitempty"`
	Plants    string `json:"plants,omitempty"`
	Defuses   string `json:"defuses,omitempty"`
	Guns      string `json:"guns,omitempty"`
	GunHS     string `json:"gunHs,omitempty"`
	Multi     string `json:"multi,omitempty"`
	Aim       string `json:"aim,omitempty"`
	Heads     int    `json:"-"`
	Shots     int    `json:"-"`
}

type spot struct {
	X     float64 `json:"x"`
	Y     float64 `json:"y"`
	Round int     `json:"round,omitempty"`
}

type killEdge struct {
	Round  int     `json:"round"`
	T      int     `json:"t"`
	KID    string  `json:"kid,omitempty"`
	VID    string  `json:"vid,omitempty"`
	Killer string  `json:"killer,omitempty"`
	Victim string  `json:"victim,omitempty"`
	Gun    string  `json:"gun,omitempty"`
	KX     float64 `json:"kx,omitempty"`
	KY     float64 `json:"ky,omitempty"`
	VX     float64 `json:"vx,omitempty"`
	VY     float64 `json:"vy,omitempty"`
	First  bool    `json:"first,omitempty"`
	Trade  bool    `json:"trade,omitempty"`
	Ult    bool    `json:"ult,omitempty"`
}

type roundBeat struct {
	N      int    `json:"n"`
	You    bool   `json:"you"`
	How    string `json:"how,omitempty"`
	Side   string `json:"side,omitempty"`
	Vs     int    `json:"vs,omitempty"`
	ScoreY int    `json:"scoreY,omitempty"`
	ScoreN int    `json:"scoreN,omitempty"`
}

type recentMatch struct {
	ID         string      `json:"id"`
	Map        string      `json:"map"`
	Mode       string      `json:"mode"`
	When       string      `json:"when"`
	Score1     *int        `json:"score1"`
	Score2     *int        `json:"score2"`
	Team1      string      `json:"team1"`
	Team2      string      `json:"team2"`
	Won        *bool       `json:"won"`
	Side       string      `json:"side"`
	Server     string      `json:"server,omitempty"`
	Started    int64       `json:"started,omitempty"`
	Length     int         `json:"length,omitempty"`
	Players    []player    `json:"players"`
	Rounds     []roundBeat `json:"rounds,omitempty"`
	Deaths     []spot      `json:"deaths,omitempty"`
	Kills      []spot      `json:"kills,omitempty"`
	Edges      []killEdge  `json:"edges,omitempty"`
	Buys       []roundBuy  `json:"buys,omitempty"`
	Lines      int         `json:"lines,omitempty"`
	KnownParty bool        `json:"knownParty,omitempty"`
}

type roundBuy struct {
	Round int    `json:"round"`
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Gun   string `json:"gun,omitempty"`
	Value int    `json:"value,omitempty"`
	Buy   string `json:"buy,omitempty"`
	Ult   bool   `json:"ult,omitempty"`
}

type snapshot struct {
	OK           bool          `json:"ok"`
	Error        string        `json:"error,omitempty"`
	Phase        string        `json:"phase"`
	Map          string        `json:"map"`
	Mode         string        `json:"mode"`
	Queue        string        `json:"queue"`
	Server       string        `json:"server,omitempty"`
	Score1       *int          `json:"score1"`
	Score2       *int          `json:"score2"`
	Team1        string        `json:"team1"`
	Team2        string        `json:"team2"`
	You          string        `json:"you"`
	Level        int           `json:"level,omitempty"`
	Players      []player      `json:"players"`
	Recent       []recentMatch `json:"recent"`
	Party        []friend      `json:"party"`
	Friends      []friend      `json:"friends"`
	FriendIDs    []string      `json:"friendIds,omitempty"`
	Records      recordView    `json:"records"`
	Pings        []string      `json:"pings,omitempty"`
	Tray         string        `json:"tray,omitempty"`
	PauseLookups bool          `json:"pauseLookups,omitempty"`
	Store        shop          `json:"store"`
	ReadAt       string        `json:"readAt"`
	LastAct      *actBrief     `json:"lastAct,omitempty"`
	Penalty      string        `json:"penalty,omitempty"`
}

type friend struct {
	Name      string `json:"name"`
	Tag       string `json:"tag"`
	PUUID     string `json:"puuid,omitempty"`
	State     string `json:"state"`
	You       bool   `json:"you,omitempty"`
	Owner     bool   `json:"owner,omitempty"`
	Idle      bool   `json:"idle,omitempty"`
	Mode      string `json:"mode,omitempty"`
	Map       string `json:"map,omitempty"`
	Score     string `json:"score,omitempty"`
	Rank      string `json:"rank,omitempty"`
	RankIcon  string `json:"rankIcon,omitempty"`
	Level     int    `json:"level,omitempty"`
	Card      string `json:"card,omitempty"`
	PartySize int    `json:"partySize,omitempty"`
	MaxParty  int    `json:"maxParty,omitempty"`
	Product   string `json:"product,omitempty"`
}

type storeItem struct {
	Name     string `json:"name"`
	Icon     string `json:"icon,omitempty"`
	Price    int    `json:"price"`
	Was      int    `json:"was,omitempty"`
	Currency string `json:"currency,omitempty"`
	Offer    string `json:"offer,omitempty"`
}

type bundleView struct {
	Name  string      `json:"name"`
	Icon  string      `json:"icon,omitempty"`
	Price int         `json:"price"`
	Was   int         `json:"was,omitempty"`
	Left  string      `json:"left,omitempty"`
	Desc  string      `json:"desc,omitempty"`
	Items []storeItem `json:"items"`
	Offer string      `json:"offer,omitempty"`
}

type shop struct {
	Offers      []storeItem  `json:"offers"`
	Night       []storeItem  `json:"night"`
	Kingdom     []storeItem  `json:"kingdom"`
	KingdomLeft string       `json:"kingdomLeft,omitempty"`
	Bundles     []bundleView `json:"bundles"`
	Esports     []bundleView `json:"esports,omitempty"`
	Resets      string       `json:"resets,omitempty"`
	NightResets string       `json:"nightResets,omitempty"`
	NightEnds   int64        `json:"nightEnds,omitempty"`
	VP          int          `json:"vp,omitempty"`
	Radianite   int          `json:"radianite,omitempty"`
	KC          int          `json:"kc,omitempty"`
	Error       string       `json:"error,omitempty"`
}

type partyHint struct {
	id   string
	size int
}

func pick(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, key := range keys {
		if v, ok := m[key]; ok && v != nil {
			return v
		}
	}
	return nil
}

func asMap(v any) map[string]any {
	m, _ := v.(map[string]any)
	return m
}

func asArr(v any) []any {
	a, _ := v.([]any)
	return a
}

func str(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		if t == math.Trunc(t) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case json.Number:
		return t.String()
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}

func num(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case float32:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	case json.Number:
		n, _ := t.Int64()
		return int(n)
	case string:
		n, _ := strconv.Atoi(strings.TrimSpace(t))
		return n
	case bool:
		if t {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func boolPtr(v bool) *bool { return &v }
func intPtr(v int) *int    { return &v }

func presenceScores(v any) (int, int, bool) {
	var ally, enemy any
	var walk func(any)
	walk = func(node any) {
		m := asMap(node)
		if m == nil {
			if arr := asArr(node); arr != nil {
				for _, child := range arr {
					if ally != nil && enemy != nil {
						return
					}
					walk(child)
				}
			}
			return
		}
		if ally == nil {
			ally = pick(m, "partyOwnerMatchScoreAllyTeam", "PartyOwnerMatchScoreAllyTeam", "matchScoreAllyTeam", "MatchScoreAllyTeam")
		}
		if enemy == nil {
			enemy = pick(m, "partyOwnerMatchScoreEnemyTeam", "PartyOwnerMatchScoreEnemyTeam", "matchScoreEnemyTeam", "MatchScoreEnemyTeam")
		}
		for _, child := range m {
			if ally != nil && enemy != nil {
				return
			}
			walk(child)
		}
	}
	walk(v)
	if ally == nil || enemy == nil {
		return 0, 0, false
	}
	return num(ally), num(enemy), true
}

func matchScores(match map[string]any) (int, int, bool) {
	if ally, enemy, ok := presenceScores(match); ok {
		return ally, enemy, true
	}
	return 0, 0, false
}

func prettyMode(queue string) string {
	key := strings.ToLower(strings.TrimSpace(queue))
	names := map[string]string{
		"competitive": "Competitive", "unrated": "Unrated", "swiftplay": "Swiftplay",
		"spikerush": "Spike Rush", "ggteam": "Escalation", "deathmatch": "Deathmatch",
		"hurm": "Team Deathmatch", "premier": "Premier", "custom": "Custom",
		"onefa": "Replication", "newmap": "New Map", "snowball": "Snowball",
		"skirmish": "Skirmish", "gauntlet": "Gauntlet", "glitched": "Gauntlet",
		"practice": "Practice",
	}
	if name, ok := names[key]; ok {
		return name
	}
	return "Match"
}

func sidesSwapped(queue string, rounds int) bool {
	low := strings.ToLower(queue)
	switch low {
	case "hurm", "ggteam", "practice", "snowball", "deathmatch":
		return false
	}
	if rounds <= 0 {
		return false
	}
	half := 12
	switch low {
	case "spikerush":
		half = 3
	case "swiftplay":
		half = 4
	}
	if rounds < half {
		return false
	}
	if rounds < half*2 {
		return true
	}
	return (rounds-half*2)%2 == 1
}

func ffaQueue(queue string) bool {
	low := strings.ToLower(queue)
	return low == "deathmatch" || low == "practice"
}

func gauntletQueue(queue string) bool {
	low := strings.ToLower(queue)
	return strings.Contains(low, "gauntlet") || strings.Contains(low, "glitched")
}

func rawTeamQueue(queue string) bool {
	switch strings.ToLower(queue) {
	case "hurm", "ggteam", "gauntlet", "glitched", "skirmish", "snowball":
		return true
	default:
		return false
	}
}

func teamSide(team, queue string, rounds int) string {
	if ffaQueue(queue) {
		return prettyMode(queue)
	}
	if rawTeamQueue(queue) {
		if strings.TrimSpace(team) == "" {
			return "Team"
		}
		return team
	}
	low := strings.ToLower(team)
	var side string
	switch low {
	case "red":
		side = "Attackers"
	case "blue":
		side = "Defenders"
	default:
		if strings.TrimSpace(team) == "" {
			return "Lobby"
		}
		return team
	}
	if sidesSwapped(queue, rounds) {
		if side == "Attackers" {
			return "Defenders"
		}
		return "Attackers"
	}
	return side
}

func prettyServer(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	skip := map[string]bool{"prod": true, "aws": true, "gp": true, "aresriot": true, "1": true, "2": true, "3": true, "4": true}
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if skip[part] || len(part) < 4 {
			continue
		}
		if strings.HasPrefix(part, "eu") || strings.HasPrefix(part, "na") || strings.HasPrefix(part, "ap") || strings.HasPrefix(part, "kr") || strings.HasPrefix(part, "br") || strings.HasPrefix(part, "latam") {
			continue
		}
		return strings.ToUpper(part[:1]) + part[1:]
	}
	if text == "" {
		return ""
	}
	return raw
}

func prettyMap(path string) string {
	text := strings.Trim(strings.TrimSpace(path), "/")
	if text == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(text, "\\", "/"), "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	if nice, ok := mapLabel(clean[len(clean)-1]); ok {
		return nice
	}
	if len(clean) >= 2 {
		if nice, ok := mapLabel(clean[len(clean)-2]); ok {
			return nice
		}
	}
	name := clean[len(clean)-1]
	return strings.ToUpper(name[:1]) + name[1:]
}

func mapLabel(name string) (string, bool) {
	known := map[string]string{
		"ascent": "Ascent", "bind": "Bind", "haven": "Haven", "split": "Split", "icebox": "Icebox",
		"breeze": "Breeze", "fracture": "Fracture", "pearl": "Pearl", "lotus": "Lotus", "sunset": "Sunset",
		"abyss": "Abyss", "corrode": "Corrode", "summit": "Summit",
		"bonsai": "Split", "canyon": "Fracture", "duality": "Bind", "foxtrot": "Breeze",
		"infinity": "Abyss", "jam": "Lotus", "juliett": "Sunset", "pitt": "Pearl", "port": "Icebox",
		"triad": "Haven", "rook": "Corrode", "plummet": "Summit",
		"range": "Range", "poveglia": "The Range", "povegliav2": "The Range", "rangev2": "The Range",
		"basictraining": "The Range", "npev2": "Basic Training",
		"hurm_alley": "District", "hurm_bowl": "Kasbah", "hurm_helix": "Drift",
		"hurm_hightide": "Glitch", "hurm_yard": "Piazza",
		"district": "District", "kasbah": "Kasbah", "drift": "Drift", "glitch": "Glitch", "piazza": "Piazza",
	}
	nice, ok := known[strings.ToLower(name)]
	return nice, ok
}

func stacksFromArchive(rows []recentMatch, players []player) map[string]partyHint {
	live := map[string]string{}
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if looksUUID(id) {
			live[id] = p.Team
		}
	}
	for _, row := range rows {
		groups := map[string][]string{}
		for _, p := range row.Players {
			id := strings.ToLower(p.PUUID)
			if !looksUUID(id) || p.Party == "" || live[id] == "" {
				continue
			}
			groups[p.Party] = append(groups[p.Party], id)
		}
		out := map[string]partyHint{}
		for label, ids := range groups {
			if len(ids) < 2 || len(ids) > 5 {
				continue
			}
			team := ""
			ok := true
			seen := map[string]bool{}
			for _, id := range ids {
				if seen[id] || live[id] == "" || (team != "" && live[id] != team) {
					ok = false
					break
				}
				seen[id] = true
				team = live[id]
			}
			if !ok {
				continue
			}
			for id := range seen {
				out[id] = partyHint{id: "arch-" + strings.ToLower(label), size: len(seen)}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return map[string]partyHint{}
}

func partyGrouped(hints map[string]partyHint) bool {
	count := map[string]int{}
	for _, hint := range hints {
		if hint.id == "" {
			continue
		}
		count[hint.id]++
		if count[hint.id] >= 2 {
			return true
		}
	}
	return false
}

func sharedHints(in map[string]partyHint) map[string]partyHint {
	count := map[string]int{}
	for _, hint := range in {
		if hint.id != "" {
			count[hint.id]++
		}
	}
	out := map[string]partyHint{}
	for id, hint := range in {
		if count[hint.id] >= 2 {
			out[id] = hint
		}
	}
	return out
}

func groupsOf(in map[string]partyHint) map[string]partyHint {
	count := map[string]int{}
	for _, hint := range in {
		if hint.id != "" {
			count[hint.id]++
		}
	}
	out := map[string]partyHint{}
	for id, hint := range in {
		n := count[hint.id]
		if n >= 2 && n <= 5 {
			out[id] = hint
		}
	}
	return out
}

func overlayParties(dst, src map[string]partyHint) {
	if dst == nil || len(src) == 0 {
		return
	}
	src = groupsOf(src)
	byID := map[string][]string{}
	for id, hint := range src {
		if hint.id == "" {
			continue
		}
		byID[hint.id] = append(byID[hint.id], id)
	}
	for partyID, members := range byID {
		if len(members) < 2 || len(members) > 5 {
			continue
		}
		for _, id := range members {
			dst[id] = partyHint{id: partyID, size: len(members)}
		}
	}
}

func fillGroups(dst, src map[string]partyHint) {
	if dst == nil || len(src) == 0 {
		return
	}
	src = groupsOf(src)
	byID := map[string][]string{}
	for id, hint := range src {
		byID[hint.id] = append(byID[hint.id], id)
	}
	for partyID, members := range byID {
		blocked := false
		for _, id := range members {
			if prev := dst[id]; prev.id != "" && prev.id != partyID {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		for _, id := range members {
			dst[id] = partyHint{id: partyID, size: len(members)}
		}
	}
}

func forceParty(dst map[string]partyHint, partyID string, members []string, roster map[string]bool) {
	if dst == nil || partyID == "" {
		return
	}
	partyID = strings.ToLower(partyID)
	in := make([]string, 0, len(members))
	seen := map[string]bool{}
	for _, id := range members {
		id = strings.ToLower(id)
		if !looksUUID(id) || !roster[id] || seen[id] {
			continue
		}
		seen[id] = true
		in = append(in, id)
	}
	free := make([]string, 0, len(in))
	for _, id := range in {
		if prev := dst[id]; prev.id != "" && prev.id != partyID {
			continue
		}
		free = append(free, id)
	}
	if len(free) < 2 || len(free) > 5 {
		return
	}
	for _, id := range free {
		dst[id] = partyHint{id: partyID, size: len(free)}
	}
}

func pinMembers(dst map[string]partyHint, partyID string, members []string, roster map[string]bool) {
	if dst == nil || partyID == "" {
		return
	}
	partyID = strings.ToLower(partyID)
	in := make([]string, 0, len(members))
	seen := map[string]bool{}
	for _, id := range members {
		id = strings.ToLower(id)
		if !looksUUID(id) || !roster[id] || seen[id] {
			continue
		}
		if prev := dst[id]; prev.id != "" && prev.id != partyID {
			continue
		}
		seen[id] = true
		in = append(in, id)
	}
	if len(in) < 2 || len(in) > 5 {
		return
	}
	for _, id := range in {
		dst[id] = partyHint{id: partyID, size: len(in)}
	}
}

func partyMemberIDs(body map[string]any) []string {
	if body == nil {
		return nil
	}
	rows := asArr(pick(body, "Members", "members"))
	if len(rows) == 0 {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, item := range rows {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "Subject", "subject", "puuid", "PUUID")))
		if looksUUID(id) {
			out = append(out, id)
		}
	}
	return out
}

func stackHints(nowTeam map[string]string, groups [][]string) map[string]partyHint {
	taken := map[string]bool{}
	out := map[string]partyHint{}
	n := 0
	for _, group := range groups {
		seen := map[string]bool{}
		list := []string{}
		team := ""
		ok := true
		for _, raw := range group {
			id := strings.ToLower(raw)
			if id == "" || seen[id] || taken[id] {
				ok = false
				break
			}
			if nowTeam != nil {
				if nowTeam[id] == "" || (team != "" && nowTeam[id] != team) {
					ok = false
					break
				}
				team = nowTeam[id]
			}
			seen[id] = true
			list = append(list, id)
		}
		if !ok || len(list) < 2 || len(list) > 5 {
			continue
		}
		n++
		label := "stack-" + strconv.Itoa(n)
		for _, id := range list {
			taken[id] = true
			out[id] = partyHint{id: label, size: len(list)}
		}
	}
	return out
}

func markParties(players []player, hints map[string]partyHint) {
	groups := map[string][]int{}
	for i, p := range players {
		hint, ok := hints[strings.ToLower(p.PUUID)]
		if !ok || hint.id == "" {
			continue
		}
		groups[hint.id] = append(groups[hint.id], i)
	}
	ids := make([]string, 0, len(groups))
	for id, idxs := range groups {
		if len(idxs) < 2 || len(idxs) > 5 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		you := func(id string) bool {
			for _, idx := range groups[id] {
				if players[idx].You {
					return true
				}
			}
			return false
		}
		if you(ids[i]) != you(ids[j]) {
			return you(ids[i])
		}
		return ids[i] < ids[j]
	})
	for n, id := range ids {
		label := strconv.Itoa(n + 1)
		if n < 26 {
			label = string(rune('A' + n))
		}
		for _, i := range groups[id] {
			players[i].Party = label
			players[i].PartyN = len(groups[id])
		}
	}
}

func harvestParties(node any, acc map[string]string) {
	switch n := node.(type) {
	case []any:
		for _, item := range n {
			harvestParties(item, acc)
		}
	case map[string]any:
		id := strings.ToLower(playerID(n))
		if !looksUUID(id) {
			id = strings.ToLower(str(pick(n, "Subject", "subject", "puuid", "PUUID")))
		}
		if party := partyKey(n); looksUUID(id) && party != "" {
			acc[id] = party
		}
		for _, value := range n {
			switch child := value.(type) {
			case map[string]any, []any:
				harvestParties(child, acc)
			}
		}
	}
}

func hintsFromIDs(ids map[string]string) map[string]partyHint {
	out := map[string]partyHint{}
	for id, party := range ids {
		if party != "" {
			out[id] = partyHint{id: strings.ToLower(party)}
		}
	}
	return sharedHints(out)
}

func partyKey(m map[string]any) string {
	if m == nil {
		return ""
	}
	if id := str(pick(m, "PartyID", "partyId", "PartyId", "CurrentPartyID", "currentPartyID")); looksUUID(id) {
		return strings.ToLower(id)
	}
	for key, value := range m {
		if !strings.Contains(strings.ToLower(key), "party") {
			continue
		}
		if id := str(value); looksUUID(id) {
			return strings.ToLower(id)
		}
		if child := asMap(value); child != nil {
			if id := str(pick(child, "ID", "id", "PartyID", "partyId")); looksUUID(id) {
				return strings.ToLower(id)
			}
		}
	}
	return ""
}

func partyHintsFrom(players []player) map[string]partyHint {
	hints := map[string]partyHint{}
	for _, p := range players {
		if p.PUUID == "" || p.PartyID == "" {
			continue
		}
		hints[strings.ToLower(p.PUUID)] = partyHint{id: strings.ToLower(p.PartyID)}
	}
	return hints
}

func partyFields(m map[string]any) (string, int) {
	if m == nil {
		return "", 0
	}
	id := str(pick(m, "partyId", "partyID", "PartyID", "PartyId", "CurrentPartyID", "currentPartyID", "party_id"))
	if !looksUUID(id) {
		return "", 0
	}
	return strings.ToLower(id), num(pick(m, "partySize", "PartySize", "party_size"))
}

func jsonObject(v any) map[string]any {
	if m := asMap(v); m != nil {
		return m
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "{") {
		var out map[string]any
		if json.Unmarshal([]byte(s), &out) == nil {
			return out
		}
	}
	return decodePrivate(s)
}

func findParty(v any) (string, int) {
	if m := jsonObject(v); m != nil {
		for _, key := range []string{"partyPresenceData", "PartyPresenceData"} {
			if id, size := partyFields(jsonObject(m[key])); id != "" {
				return id, size
			}
		}
	}
	return walkParty(v)
}

func walkParty(v any) (string, int) {
	if m := jsonObject(v); m != nil {
		if id, size := partyFields(m); id != "" {
			return id, size
		}
		for _, child := range m {
			if id, size := walkParty(child); id != "" {
				return id, size
			}
		}
	}
	if arr := asArr(v); len(arr) > 0 {
		for _, child := range arr {
			if id, size := walkParty(child); id != "" {
				return id, size
			}
		}
	}
	return "", 0
}

func decodePrivate(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "{") {
		var out map[string]any
		if json.Unmarshal([]byte(raw), &out) == nil {
			return out
		}
	}
	if strings.Count(raw, ".") >= 2 {
		parts := strings.Split(raw, ".")
		order := []int{}
		if len(parts) > 1 {
			order = append(order, 1)
		}
		for i := range parts {
			if i != 1 {
				order = append(order, i)
			}
		}
		for _, i := range order {
			if m := decodeB64JSON(parts[i]); m != nil {
				return m
			}
		}
	}
	return decodeB64JSON(raw)
}

func decodeB64JSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var data []byte
	var err error
	padded := raw
	if m := len(padded) % 4; m != 0 {
		padded += strings.Repeat("=", 4-m)
	}
	for _, dec := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		data, err = dec.DecodeString(raw)
		if err == nil {
			break
		}
		data, err = dec.DecodeString(padded)
		if err == nil {
			break
		}
	}
	if err != nil || len(data) == 0 || data[0] != '{' {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}

func isRosterPlayer(m map[string]any) bool {
	id := playerID(m)
	if !looksUUID(id) {
		return false
	}
	return pick(m, "TeamID", "teamId", "Team", "team", "CharacterID", "characterId", "PlayerIdentity", "playerIdentity", "CompetitiveTier", "competitiveTier", "SeasonalBadgeInfo", "seasonalBadgeInfo") != nil
}

func playerID(m map[string]any) string {
	id := str(pick(m, "Subject", "subject"))
	if looksUUID(id) {
		return id
	}
	id = str(pick(asMap(pick(m, "PlayerIdentity", "playerIdentity")), "Subject", "subject"))
	if looksUUID(id) {
		return id
	}
	return ""
}

func countStat(m map[string]any, keys ...string) (int, bool) {
	v := pick(m, keys...)
	if v == nil {
		return 0, false
	}
	if arr, ok := v.([]any); ok {
		return len(arr), true
	}
	return num(v), true
}

func collectPlayers(node any, inheritedTeam string, agents map[string]agentInfo, acc map[string]*player) {
	switch n := node.(type) {
	case []any:
		for _, item := range n {
			collectPlayers(item, inheritedTeam, agents, acc)
		}
	case map[string]any:
		team := str(pick(n, "TeamID", "teamId", "Team", "team"))
		if team == "" || len(team) > 16 {
			team = inheritedTeam
		}
		if isRosterPlayer(n) {
			id := strings.ToLower(playerID(n))
			cur := acc[id]
			if cur == nil {
				cur = &player{PUUID: id}
				acc[id] = cur
			}
			if cur.Team == "" && team != "" {
				cur.Team = team
			}
			if cur.Name == "" {
				cur.Name = str(pick(n, "GameName", "gameName", "DisplayName", "displayName"))
			}
			if cur.Tag == "" {
				cur.Tag = str(pick(n, "TagLine", "tagLine"))
			}
			if cur.PartyID == "" {
				cur.PartyID = partyKey(n)
			}
			ident := asMap(pick(n, "PlayerIdentity", "playerIdentity"))
			if ident != nil {
				if cur.Level == 0 {
					cur.Level = num(pick(ident, "AccountLevel", "accountLevel"))
				}
				if cur.Name == "" {
					cur.Name = str(pick(ident, "GameName", "gameName", "DisplayName", "displayName"))
				}
				if cur.Tag == "" {
					cur.Tag = str(pick(ident, "TagLine", "tagLine"))
				}
				if cur.PUUID == "" {
					cur.PUUID = str(pick(ident, "Subject", "subject"))
				}
			}
			agentID := strings.ToLower(str(pick(n, "CharacterID", "characterId")))
			if cur.Agent == "" && agentID != "" {
				cur.AgentIcon = "https://media.valorant-api.com/agents/" + agentID + "/displayicon.png"
				if info, ok := agents[agentID]; ok && info.Name != "" {
					cur.Agent = info.Name
					if info.Icon != "" {
						cur.AgentIcon = info.Icon
					}
				} else {
					cur.Agent = "Locked"
				}
			}
			state := strings.ToLower(str(pick(n, "CharacterSelectionState", "characterSelectionState")))
			if strings.Contains(state, "lock") {
				cur.Locked = true
			}
			if cur.Rank == "" {
				tier := num(pick(n, "CompetitiveTier", "competitiveTier"))
				if tier == 0 {
					badge := asMap(pick(n, "SeasonalBadgeInfo", "seasonalBadgeInfo"))
					tier = num(pick(badge, "Rank", "rank"))
				}
				if tier > 0 {
					cur.Tier = tier
					cur.Rank = rankName(tier)
				}
			}
			stats := asMap(pick(n, "stats", "Stats", "playerStats", "PlayerStats"))
			if stats != nil && cur.Kills == "" {
				k, kok := countStat(stats, "kills", "Kills")
				d, dok := countStat(stats, "deaths", "Deaths")
				a, aok := countStat(stats, "assists", "Assists")
				if kok || dok || aok {
					if !kok {
						k = 0
					}
					if !dok {
						d = 0
					}
					if !aok {
						a = 0
					}
					cur.Kills = strconv.Itoa(k)
					cur.Deaths = strconv.Itoa(d)
					cur.Assists = strconv.Itoa(a)
					if s, ok := countStat(stats, "score", "Score"); ok {
						cur.Score = strconv.Itoa(s)
					}
				}
			}
			if stats != nil && cur.HS == "" {
				hs, hok := countStat(stats, "headshots", "Headshots", "headShots")
				body, _ := countStat(stats, "bodyshots", "Bodyshots", "bodyShots")
				leg, _ := countStat(stats, "legshots", "Legshots", "legShots")
				shots := hs + body + leg
				if hok && shots > 0 {
					cur.HS = strconv.Itoa(hs*100/shots) + "%"
				}
			}
		}
		for _, v := range n {
			collectPlayers(v, team, agents, acc)
		}
	}
}

func roster(node any, agents map[string]agentInfo) []player {
	acc := map[string]*player{}
	collectPlayers(node, "", agents, acc)
	out := make([]player, 0, len(acc))
	for _, p := range acc {
		if p.Name == "" {
			p.Name = "Player"
		}
		out = append(out, *p)
	}
	return out
}

func applyNames(players []player, book map[string][2]string) {
	for i := range players {
		if pair, ok := book[strings.ToLower(players[i].PUUID)]; ok {
			if players[i].Name == "" || players[i].Name == "Player" || players[i].Name == "Hidden" {
				if pair[0] != "" {
					players[i].Name = pair[0]
					players[i].Tag = pair[1]
				}
			}
		}
	}
}

func labelSides(players []player, queue string, roundsPlayed int) {
	for i := range players {
		players[i].Team = teamSide(players[i].Team, queue, roundsPlayed)
	}
}

func roleWord(raw string) string {
	text := strings.ToLower(raw)
	if strings.Contains(text, "attack") {
		return "Attackers"
	}
	if strings.Contains(text, "defend") {
		return "Defenders"
	}
	return ""
}

func sideMode(queue string) bool {
	return !ffaQueue(queue) && !rawTeamQueue(queue)
}

func applyHistorySides(players []player, detail map[string]any, queue string) {
	rounds := 0
	for _, item := range asArr(pick(detail, "teams", "Teams")) {
		rounds += num(pick(asMap(item), "roundsWon", "numPoints", "RoundsWon"))
	}
	labelSides(players, queue, rounds)
}

func rankName(tier int) string {
	if tier < 3 {
		return "Unranked"
	}
	if tier >= 27 {
		return "Radiant"
	}
	names := []string{"Iron", "Bronze", "Silver", "Gold", "Platinum", "Diamond", "Ascendant", "Immortal"}
	n := tier - 3
	div := n / 3
	if div < 0 || div >= len(names) {
		return "Radiant"
	}
	return names[div] + " " + strconv.Itoa((n%3)+1)
}

func tierFromMMR(body map[string]any) (int, int) {
	if body == nil {
		return 0, 0
	}
	skills := asMap(pick(body, "QueueSkills", "queueSkills"))
	comp := asMap(pick(skills, "competitive", "Competitive"))
	seasons := asMap(pick(comp, "SeasonalInfoBySeasonID", "seasonalInfoBySeasonID"))
	var tier, rr int
	for _, item := range seasons {
		row := asMap(item)
		got := num(pick(row, "Rank", "rank", "CompetitiveTier", "competitiveTier"))
		if got >= tier {
			tier = got
			rr = num(pick(row, "RankedRating", "rankedRating"))
		}
	}
	if tier >= 3 {
		return tier, rr
	}
	latest := asMap(pick(body, "LatestCompetitiveUpdate", "latestCompetitiveUpdate"))
	if t := num(pick(latest, "TierAfterUpdate", "tierAfterUpdate")); t >= 3 {
		return t, num(pick(latest, "RankedRatingAfterUpdate", "rankedRatingAfterUpdate"))
	}
	for _, item := range asArr(pick(body, "Matches", "matches")) {
		row := asMap(item)
		if t := num(pick(row, "TierAfterUpdate", "tierAfterUpdate")); t >= 3 {
			return t, num(pick(row, "RankedRatingAfterUpdate", "rankedRatingAfterUpdate"))
		}
	}
	return 0, 0
}

func peakTier(body map[string]any) int {
	best := 0
	latest := asMap(pick(body, "LatestCompetitiveUpdate", "latestCompetitiveUpdate"))
	if t := num(pick(latest, "TierAfterUpdate", "tierAfterUpdate")); t > best {
		best = t
	}
	skills := asMap(pick(body, "QueueSkills", "queueSkills"))
	comp := asMap(pick(skills, "competitive", "Competitive"))
	seasons := asMap(pick(comp, "SeasonalInfoBySeasonID", "seasonalInfoBySeasonID"))
	for _, item := range seasons {
		row := asMap(item)
		for _, key := range []string{"HighestTierAchieved", "highestTierAchieved", "PeakRank", "peakRank", "CompetitiveTier", "competitiveTier", "Rank", "rank"} {
			if n := num(pick(row, key)); n > best {
				best = n
			}
		}
	}
	if tier, _ := tierFromMMR(body); tier > best {
		best = tier
	}
	return best
}

func edgeLabel(p player) string {
	if p.You {
		return "You"
	}
	if !blankName(p.Name) {
		return p.Name
	}
	if p.Agent != "" && p.Agent != "Picking" && p.Agent != "Locked" {
		return p.Agent
	}
	return "Player"
}

func niceGun(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return ""
	}
	if name, ok := gunNames[strings.ToLower(text)]; ok {
		return name
	}
	if strings.HasSuffix(text, "Ability") {
		return strings.TrimSuffix(text, "Ability")
	}
	return text
}

var gunNames = map[string]string{
	"29a0cfab-485b-f5d5-779a-b59f85e204a8":      "Classic",
	"42da8ccc-40d5-affc-beec-15aa47b42eda":      "Shorty",
	"44d4e95c-4157-0037-81b2-17841bf2e8e3":      "Frenzy",
	"1baa85b4-4c70-1284-64bb-6481dfc3bb4e":      "Ghost",
	"e336c6b8-418d-9340-d77f-7a9e4cfe0702":      "Sheriff",
	"f7e1b454-4ad4-1063-ec0a-159e56b58941":      "Bandit",
	"c4883e50-4494-202c-3ec3-6b8a9284f00b":      "Stinger",
	"462080d1-4035-2937-7c09-27aa2a5c27a7":      "Spectre",
	"910be174-449b-c412-ab22-d0873436b21b":      "Bucky",
	"ec845bf4-4f79-ddda-a3da-0db3774b2794":      "Judge",
	"ee8e8d15-496b-07ac-e5f6-8fae5d4c7b1a":      "Bulldog",
	"ae3de142-4d85-2547-dd26-4e90bed35cf7":      "Guardian",
	"4ade7faa-4cf1-8376-95ef-39884480959b":      "Phantom",
	"9c82e19d-4575-0200-1a81-3eacf00cf872":      "Vandal",
	"a03b24d3-4319-996d-0f8c-94bbfba1dfc7":      "Outlaw",
	"5f0aa613-4418-eb17-f821-079ba2276c3d":      "Marshal",
	"c4883e50-4494-202c-3ec3-6b8a9284f00b-op":   "Operator",
	"a03b24d3-4319-996d-0f8c-94bbfba1dfc7-ares": "Ares",
	"55d8a0f4-4274-ca67-fe2c-06ab45efdf58":      "Ares",
	"63e6c2b6-4a8e-869c-3d4c-e38355226584":      "Odin",
	"2f59173c-4bed-b6c3-2191-dea9b58be9c9":      "Melee",
	"ultimate":                                  "Ultimate",
}

func multiText(counts map[int]int) string {
	keys := make([]int, 0, len(counts))
	for k, n := range counts {
		if n > 0 {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, strconv.Itoa(k)+":"+strconv.Itoa(counts[k]))
	}
	return strings.Join(parts, ",")
}

func aimedAt(x, y, facing, tx, ty float64) bool {
	dx, dy := tx-x, ty-y
	fx, fy := math.Cos(facing), math.Sin(facing)
	return fx*dx+fy*dy > 0.15*math.Hypot(dx, dy)
}

func looksUUID(id string) bool {
	if len(id) != 36 {
		return false
	}
	for i, r := range id {
		switch i {
		case 8, 13, 18, 23:
			if r != '-' {
				return false
			}
		default:
			if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
				return false
			}
		}
	}
	return true
}

func blankName(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "player", "hidden":
		return true
	default:
		return false
	}
}

func splitDisplay(raw string) (string, string) {
	raw = strings.TrimSpace(raw)
	if i := strings.LastIndex(raw, "#"); i > 0 {
		return strings.TrimSpace(raw[:i]), strings.TrimSpace(raw[i+1:])
	}
	return raw, ""
}

func parseNames(payload []byte) map[string][2]string {
	var rows []any
	if json.Unmarshal(payload, &rows) != nil {
		return namesFromAliases(payload)
	}
	out := map[string][2]string{}
	for _, item := range rows {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "Subject", "subject", "puuid", "PUUID")))
		if id == "" {
			continue
		}
		name := str(pick(row, "GameName", "gameName"))
		tag := str(pick(row, "TagLine", "tagLine"))
		if name == "" {
			name, tag = splitDisplay(str(pick(row, "DisplayName", "displayName")))
		}
		out[id] = [2]string{name, tag}
	}
	return out
}

func namedPlayers(match map[string]any, agents map[string]agentInfo) []player {
	var out []player
	addTeam := func(node any) {
		team := asMap(node)
		if len(team) == 0 {
			return
		}
		label := str(pick(team, "TeamID", "teamId"))
		for _, item := range asArr(pick(team, "Players", "players")) {
			row := asMap(item)
			if len(row) == 0 {
				continue
			}
			if str(pick(row, "TeamID", "teamId")) == "" && label != "" {
				row["TeamID"] = label
			}
			out = append(out, roster(row, agents)...)
		}
	}
	addTeam(pick(match, "AllyTeam", "allyTeam"))
	addTeam(pick(match, "EnemyTeam", "enemyTeam"))
	for _, team := range asArr(pick(match, "Teams", "teams")) {
		addTeam(team)
	}
	out = append(out, playersFromToken(str(pick(match, "TeamMatchToken", "teamMatchToken")))...)
	return mergePlayers(roster(match, agents), out)
}

func playersFromToken(raw string) []player {
	parts := strings.Split(raw, ".")
	if len(parts) < 2 {
		return nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}
	var body any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	var out []player
	var walk func(any, string)
	walk = func(v any, team string) {
		switch t := v.(type) {
		case map[string]any:
			if next := str(pick(t, "teamId", "TeamID", "team")); next != "" && len(next) < 24 {
				team = next
			}
			id := str(pick(t, "subject", "Subject", "puuid"))
			if looksUUID(id) {
				out = append(out, player{PUUID: id, Team: team, Name: str(pick(t, "gameName", "GameName"))})
			}
			for _, child := range t {
				walk(child, team)
			}
		case []any:
			for _, child := range t {
				walk(child, team)
			}
		case string:
			if looksUUID(t) && team != "" && len(team) < 24 {
				out = append(out, player{PUUID: t, Team: team, Name: "Player"})
			}
		}
	}
	walk(body, "")
	return out
}

func mergePlayers(groups ...[]player) []player {
	order := []string{}
	by := map[string]player{}
	for _, group := range groups {
		for _, p := range group {
			id := strings.ToLower(p.PUUID)
			if id == "" {
				continue
			}
			prev, ok := by[id]
			if !ok {
				by[id] = p
				order = append(order, id)
				continue
			}
			if blankName(prev.Name) && !blankName(p.Name) {
				prev.Name, prev.Tag = p.Name, p.Tag
			}
			if prev.Team == "" || prev.Team == "Lobby" {
				prev.Team = p.Team
			}
			if prev.Agent == "" || prev.Agent == "Locked" || prev.Agent == "Picking" {
				if p.Agent != "" && p.Agent != "Locked" && p.Agent != "Picking" {
					prev.Agent, prev.AgentIcon = p.Agent, p.AgentIcon
				}
			}
			if prev.Tier == 0 && p.Tier > 0 {
				prev.Tier, prev.Rank = p.Tier, p.Rank
			}
			if prev.PartyID == "" && p.PartyID != "" {
				prev.PartyID = p.PartyID
			}
			if prev.Kills == "" && p.Kills != "" {
				prev.Kills, prev.Deaths, prev.Assists = p.Kills, p.Deaths, p.Assists
			}
			by[id] = prev
		}
	}
	out := make([]player, 0, len(order))
	for _, id := range order {
		out = append(out, by[id])
	}
	return out
}

func namesFromAliases(payload []byte) map[string][2]string {
	var body any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	out := map[string][2]string{}
	var walk func(any, string)
	walk = func(v any, id string) {
		switch t := v.(type) {
		case map[string]any:
			if next := str(pick(t, "puuid", "subject", "Subject")); looksUUID(next) {
				id = strings.ToLower(next)
			}
			name := str(pick(t, "game_name", "gameName", "GameName"))
			tag := str(pick(t, "tag_line", "tagLine", "TagLine"))
			if name == "" {
				name, tag = splitDisplay(str(pick(t, "displayName", "DisplayName")))
			}
			if id != "" && !blankName(name) {
				out[id] = [2]string{name, tag}
			}
			for _, child := range t {
				walk(child, id)
			}
		case []any:
			for _, child := range t {
				walk(child, id)
			}
		}
	}
	walk(body, "")
	return out
}

func parseRegion(logText string) (region, shard string) {
	text := strings.ToLower(logText)
	idx := strings.LastIndex(text, ".a.pvp.net")
	for idx >= 0 {
		start := strings.LastIndex(text[:idx], "glz-")
		if start >= 0 {
			chunk := text[start:idx]
			rest := strings.TrimPrefix(chunk, "glz-")
			parts := strings.Split(rest, ".")
			if len(parts) == 2 && strings.HasSuffix(parts[0], "-1") {
				return strings.TrimSuffix(parts[0], "-1"), parts[1]
			}
		}
		if start <= 0 {
			break
		}
		idx = strings.LastIndex(text[:start], ".a.pvp.net")
	}
	return "", ""
}

func parseClientVersion(logText string) string {
	for _, line := range strings.Split(logText, "\n") {
		low := strings.ToLower(line)
		if strings.Contains(low, "ci server version:") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
		if i := strings.Index(low, "release-"); i >= 0 {
			token := strings.Fields(line[i:])
			if len(token) > 0 && strings.Contains(token[0], "-shipping-") {
				return strings.Trim(token[0], "\",")
			}
		}
	}
	return ""
}

func startedLabel(ms int64) string {
	if ms <= 0 {
		return ""
	}
	if ms < 1_000_000_000_000 {
		ms *= 1000
	}
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}

func queueOf(detail map[string]any) string {
	info := asMap(pick(detail, "matchInfo", "MatchInfo"))
	return str(pick(info, "queueID", "queueId", "QueueID"))
}

func buildRecent(detail map[string]any, self string, agents map[string]agentInfo) *recentMatch {
	info := asMap(pick(detail, "matchInfo", "MatchInfo"))
	if info == nil {
		return nil
	}
	queue := str(pick(info, "queueID", "queueId", "QueueID"))
	players := roster(pick(detail, "players", "Players"), agents)
	if len(players) == 0 {
		return nil
	}
	markParties(players, partyHintsFrom(players))
	applyHistorySides(players, detail, queue)
	var me *player
	for i := range players {
		if self != "" && strings.EqualFold(players[i].PUUID, self) {
			players[i].You = true
			me = &players[i]
		}
	}
	teams := asArr(pick(detail, "teams", "Teams"))
	scoreOf := map[string]int{}
	wonOf := map[string]*bool{}
	for _, item := range teams {
		team := asMap(item)
		id := strings.ToLower(str(pick(team, "teamId", "TeamID")))
		scoreOf[id] = num(pick(team, "roundsWon", "numPoints", "RoundsWon"))
		if _, ok := team["won"]; ok {
			wonOf[id] = boolPtr(team["won"] == true)
		}
		if _, ok := team["Won"]; ok {
			wonOf[id] = boolPtr(team["Won"] == true)
		}
	}
	row := &recentMatch{
		ID:      str(pick(info, "matchId", "MatchID")),
		Map:     prettyMap(str(pick(info, "mapId", "MapID"))),
		Mode:    prettyMode(queue),
		Server:  prettyServer(str(pick(info, "gamePodId", "GamePodID"))),
		Players: players,
		Started: int64(num(pick(info, "gameStartMillis", "GameStartMillis"))),
		Length:  num(pick(info, "gameLengthMillis", "GameLengthMillis")) / 1000,
		When:    startedLabel(int64(num(pick(info, "gameStartMillis", "GameStartMillis")))),
	}
	if row.Started > 0 && row.Started < 1_000_000_000_000 {
		row.Started *= 1000
	}
	blue, red := scoreOf["blue"], scoreOf["red"]
	if me != nil {
		mine := strings.ToLower(me.Team)
		if strings.Contains(strings.ToLower(mine), "attack") {
			row.Team1, row.Team2 = "Attackers", "Defenders"
		} else if strings.Contains(strings.ToLower(mine), "defend") {
			row.Team1, row.Team2 = "Defenders", "Attackers"
		}
		if won, ok := wonOf[colorOf(me.Team)]; ok {
			row.Won = won
		}
	}
	row.Score1, row.Score2 = intPtr(blue), intPtr(red)
	if row.Team1 == "" {
		row.Team1, row.Team2 = "Blue", "Red"
	}
	return row
}

func colorOf(team string) string {
	low := strings.ToLower(team)
	if strings.Contains(low, "red") {
		return "red"
	}
	if strings.Contains(low, "blue") {
		return "blue"
	}
	return low
}

func addCombatStats(detail map[string]any, players []player) {
	by := combatByPlayer(detail)
	for i := range players {
		tot, ok := by[strings.ToLower(players[i].PUUID)]
		if !ok {
			continue
		}
		applyCombat(&players[i], tot)
	}
}

type combat struct {
	dmg, hs, body, leg, rounds, score, kills, deaths, assists int
}

func combatByPlayer(detail map[string]any) map[string]combat {
	out := map[string]combat{}
	for _, item := range asArr(pick(detail, "roundResults", "RoundResults")) {
		round := asMap(item)
		for _, stat := range asArr(pick(round, "playerStats", "PlayerStats")) {
			row := asMap(stat)
			id := strings.ToLower(str(pick(row, "subject", "Subject")))
			if id == "" {
				continue
			}
			cur := out[id]
			cur.rounds++
			cur.kills += len(asArr(pick(row, "kills", "Kills")))
			cur.score += num(pick(row, "score", "Score"))
			for _, hit := range asArr(pick(row, "damage", "Damage")) {
				dmg := asMap(hit)
				cur.dmg += num(pick(dmg, "damage", "Damage"))
				cur.hs += num(pick(dmg, "headshots", "Headshots"))
				cur.body += num(pick(dmg, "bodyshots", "Bodyshots"))
				cur.leg += num(pick(dmg, "legshots", "Legshots"))
			}
			out[id] = cur
		}
	}
	dead := map[string]int{}
	assist := map[string]int{}
	for _, item := range asArr(pick(detail, "roundResults", "RoundResults")) {
		for _, stat := range asArr(pick(asMap(item), "playerStats", "PlayerStats")) {
			for _, kill := range asArr(pick(asMap(stat), "kills", "Kills")) {
				row := asMap(kill)
				dead[strings.ToLower(str(pick(row, "victim", "Victim")))]++
				for _, a := range asArr(pick(row, "assistants", "Assistants")) {
					assist[strings.ToLower(str(a))]++
				}
			}
		}
	}
	for id, n := range dead {
		cur := out[id]
		cur.deaths = n
		out[id] = cur
	}
	for id, n := range assist {
		cur := out[id]
		cur.assists = n
		out[id] = cur
	}
	return out
}

func applyCombat(p *player, tot combat) {
	if p == nil {
		return
	}
	rounds := tot.rounds
	if rounds < 1 {
		rounds = 1
	}
	if p.Kills == "" && (tot.kills > 0 || tot.deaths > 0 || tot.assists > 0) {
		p.Kills = strconv.Itoa(tot.kills)
		p.Deaths = strconv.Itoa(tot.deaths)
		p.Assists = strconv.Itoa(tot.assists)
	}
	if tot.score > 0 {
		if p.Score == "" {
			p.Score = strconv.Itoa(tot.score)
		}
		if p.ACS == "" {
			p.ACS = strconv.Itoa(tot.score / rounds)
		}
	}
	if tot.dmg > 0 {
		p.Damage = strconv.Itoa(tot.dmg)
		p.ADR = strconv.Itoa(tot.dmg / rounds)
	}
	shots := tot.hs + tot.body + tot.leg
	if shots > 0 && p.HS == "" {
		p.HS = strconv.Itoa(tot.hs*100/shots) + "%"
		p.Heads = tot.hs
		p.Shots = shots
	}
}

func aimApply(players []player) {
	for i := range players {
		if players[i].Shots > 0 {
			players[i].Aim = strconv.Itoa(players[i].Heads*100/players[i].Shots) + "%"
		}
	}
}

func matchEdges(detail map[string]any, players []player) []killEdge {
	names := map[string]string{}
	for _, p := range players {
		names[strings.ToLower(p.PUUID)] = edgeLabel(p)
	}
	var edges []killEdge
	for _, item := range asArr(pick(detail, "roundResults", "RoundResults")) {
		round := asMap(item)
		n := num(pick(round, "roundNum", "RoundNum"))
		for _, stat := range asArr(pick(round, "playerStats", "PlayerStats")) {
			row := asMap(stat)
			kid := strings.ToLower(str(pick(row, "subject", "Subject")))
			for _, kill := range asArr(pick(row, "kills", "Kills")) {
				k := asMap(kill)
				vid := strings.ToLower(str(pick(k, "victim", "Victim")))
				loc := asMap(pick(k, "victimLocation", "VictimLocation"))
				kloc := asMap(pick(k, "playerLocations", "PlayerLocations"))
				_ = kloc
				fin := asArr(pick(k, "finishingDamage", "FinishingDamage"))
				gun := ""
				if dmg := asMap(pick(k, "finishingDamage", "FinishingDamage")); dmg != nil {
					gun = niceGun(str(pick(dmg, "damageItem", "DamageItem")))
					if strings.EqualFold(str(pick(dmg, "damageType", "DamageType")), "Ability") && gun == "" {
						gun = "Ability"
					}
				}
				_ = fin
				edges = append(edges, killEdge{
					Round: n, T: num(pick(k, "timeSinceRoundStartMillis", "roundTime")),
					KID: kid, VID: vid, Killer: names[kid], Victim: names[vid], Gun: gun,
					VX: flt(pick(loc, "x", "X")), VY: flt(pick(loc, "y", "Y")),
				})
			}
		}
	}
	markEdgeFlags(edges, players)
	return edges
}

func flt(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case json.Number:
		f, _ := t.Float64()
		return f
	case string:
		f, _ := strconv.ParseFloat(t, 64)
		return f
	default:
		return 0
	}
}

func relabelEdges(row *recentMatch) {
	if row == nil {
		return
	}
	names := map[string]string{}
	for _, p := range row.Players {
		names[strings.ToLower(p.PUUID)] = edgeLabel(p)
	}
	for i := range row.Edges {
		if name := names[strings.ToLower(row.Edges[i].KID)]; name != "" {
			row.Edges[i].Killer = name
		}
		if name := names[strings.ToLower(row.Edges[i].VID)]; name != "" {
			row.Edges[i].Victim = name
		}
	}
}

func matchRounds(detail map[string]any, players []player) []roundBeat {
	var me string
	var mine string
	for _, p := range players {
		if p.You {
			me = strings.ToLower(p.PUUID)
			mine = p.Team
		}
	}
	var out []roundBeat
	ys, ns := 0, 0
	for _, item := range asArr(pick(detail, "roundResults", "RoundResults")) {
		round := asMap(item)
		n := num(pick(round, "roundNum", "RoundNum"))
		winner := str(pick(round, "winningTeam", "WinningTeam"))
		you := mine != "" && (strings.EqualFold(winner, mine) || strings.EqualFold(roleWord(winner), mine) || strings.EqualFold(teamSide(winner, queueOf(detail), n), mine))
		if you {
			ys++
		} else if winner != "" {
			ns++
		}
		out = append(out, roundBeat{N: n, You: you, How: roundHow(round), Side: mine, ScoreY: ys, ScoreN: ns, Vs: roundClutch(round, me)})
	}
	return out
}

func roundHow(round map[string]any) string {
	text := strings.ToLower(str(pick(round, "roundResult", "RoundResult", "roundCeremony", "RoundCeremony")))
	switch {
	case strings.Contains(text, "defuse"):
		return "Defuse"
	case strings.Contains(text, "detonate") || strings.Contains(text, "boom"):
		return "Detonate"
	case strings.Contains(text, "elim"):
		return "Elimination"
	case strings.Contains(text, "time"):
		return "Time"
	default:
		if text == "" {
			return ""
		}
		return text
	}
}

func roundClutch(round map[string]any, me string) int {
	if me == "" {
		return 0
	}
	alive := map[string]bool{}
	team := map[string]string{}
	for _, stat := range asArr(pick(round, "playerStats", "PlayerStats")) {
		row := asMap(stat)
		id := strings.ToLower(str(pick(row, "subject", "Subject")))
		alive[id] = true
	}
	kills := []struct{ kid, vid string }{}
	for _, stat := range asArr(pick(round, "playerStats", "PlayerStats")) {
		row := asMap(stat)
		kid := strings.ToLower(str(pick(row, "subject", "Subject")))
		for _, kill := range asArr(pick(row, "kills", "Kills")) {
			k := asMap(kill)
			kills = append(kills, struct{ kid, vid string }{kid, strings.ToLower(str(pick(k, "victim", "Victim")))})
		}
	}
	_ = team
	mineKills := 0
	for _, k := range kills {
		if k.kid == me {
			mineKills++
		}
	}
	if mineKills >= 2 {
		return mineKills
	}
	return 0
}

func scoreForSide(detail map[string]any, side string) int {
	for _, item := range asArr(pick(detail, "teams", "Teams")) {
		team := asMap(item)
		if strings.EqualFold(str(pick(team, "teamId", "TeamID")), side) {
			return num(pick(team, "roundsWon", "RoundsWon"))
		}
	}
	return 0
}

func scoreBySide(detail map[string]any) (int, int) {
	return scoreForSide(detail, "Blue"), scoreForSide(detail, "Red")
}

func finalizeRecent(detail map[string]any, self string, agents map[string]agentInfo) *recentMatch {
	row := buildRecent(detail, self, agents)
	if row == nil {
		return nil
	}
	addCombatStats(detail, row.Players)
	row.Edges = matchEdges(detail, row.Players)
	row.Rounds = matchRounds(detail, row.Players)
	row.Buys = matchBuys(detail, row.Players)
	row.Lines = len(row.Edges)
	relabelEdges(row)
	markEdgeFlags(row.Edges, row.Players)
	return row
}

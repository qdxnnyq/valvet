package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type recordFace struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
	Side string `json:"side"`
}

type recordGame struct {
	ID     string       `json:"id"`
	Map    string       `json:"map"`
	Mode   string       `json:"mode,omitempty"`
	Agent  string       `json:"agent,omitempty"`
	Won    *bool        `json:"won"`
	Kills  int          `json:"kills"`
	Deaths int          `json:"deaths"`
	ACS    int          `json:"acs"`
	HS     int          `json:"hs"`
	Guns   string       `json:"guns,omitempty"`
	GunHS  string       `json:"gunHs,omitempty"`
	Faces  []recordFace `json:"faces,omitempty"`
	When   string       `json:"when,omitempty"`
	Length int          `json:"length,omitempty"`
}

type recordPerson struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name"`
	Tag     string `json:"tag,omitempty"`
	PUUID   string `json:"puuid,omitempty"`
	Wins    int    `json:"wins"`
	Losses  int    `json:"losses"`
	Games   int    `json:"games"`
	Minutes int    `json:"minutes,omitempty"`
}

type recordAgent struct {
	Name   string `json:"name"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Games  int    `json:"games"`
	Kills  int    `json:"kills"`
	Deaths int    `json:"deaths"`
	HS     int    `json:"hs"`
	ACS    int    `json:"acs"`
}

type recordMap struct {
	Name   string `json:"name"`
	Wins   int    `json:"wins"`
	Losses int    `json:"losses"`
	Games  int    `json:"games"`
}

type recordGun struct {
	Name  string `json:"name"`
	Kills int    `json:"kills"`
	HS    int    `json:"hs"`
	Shots int    `json:"shots"`
}

type recordDay struct {
	Day   string `json:"day"`
	Label string `json:"label"`
	Games int    `json:"games"`
	HS    int    `json:"hs"`
	ACS   int    `json:"acs"`
	Kills int    `json:"kills"`
}

type recordView struct {
	Games        int            `json:"games"`
	Wins         int            `json:"wins"`
	Losses       int            `json:"losses"`
	Kills        int            `json:"kills"`
	Deaths       int            `json:"deaths"`
	KD           string         `json:"kd"`
	HS           string         `json:"hs"`
	ACS          int            `json:"acs"`
	BestKills    int            `json:"bestKills"`
	BestKillsMap string         `json:"bestKillsMap,omitempty"`
	BestACS      int            `json:"bestACS"`
	BestACSMap   string         `json:"bestACSMap,omitempty"`
	BestGun      string         `json:"bestGun,omitempty"`
	BestMap      string         `json:"bestMap,omitempty"`
	BestMapRate  string         `json:"bestMapRate,omitempty"`
	Friends      []recordPerson `json:"friends"`
	Rival        *recordPerson  `json:"rival,omitempty"`
	Duos         []recordPerson `json:"duos"`
	Agents       []recordAgent  `json:"agents"`
	Maps         []recordMap    `json:"maps"`
	Guns         []recordGun    `json:"guns"`
	Days         []recordDay    `json:"days"`
	Note         string         `json:"note,omitempty"`
}

const actSafetyCap = 900

var actClock struct {
	mu    sync.Mutex
	start time.Time
	end   time.Time
	at    time.Time
}

func rememberAct(start, end time.Time) {
	if start.IsZero() || end.IsZero() || !end.After(start) {
		return
	}
	actClock.mu.Lock()
	actClock.start = start
	actClock.end = end
	actClock.at = time.Now()
	actClock.mu.Unlock()
}

func knownAct() (time.Time, time.Time, bool) {
	actClock.mu.Lock()
	defer actClock.mu.Unlock()
	if actClock.start.IsZero() || time.Since(actClock.at) > 12*time.Hour {
		return time.Time{}, time.Time{}, false
	}
	return actClock.start, actClock.end, true
}

func stampMillis(ms int64, when string) int64 {
	if ms > 0 {
		if ms < 1_000_000_000_000 {
			ms *= 1000
		}
		return ms
	}
	if t, err := time.Parse(time.RFC3339, strings.TrimSpace(when)); err == nil {
		return t.UnixMilli()
	}
	return 0
}

func inAct(ms int64, when string) bool {
	start, _, ok := knownAct()
	if !ok {
		return true
	}
	got := stampMillis(ms, when)
	if got == 0 {
		return true
	}
	return got >= start.UnixMilli()
}

func trimActMatches(rows []recentMatch) []recentMatch {
	kept := rows[:0]
	for _, row := range rows {
		if inAct(row.Started, row.When) {
			kept = append(kept, row)
		}
	}
	rows = kept
	if len(rows) > actSafetyCap {
		rows = rows[:actSafetyCap]
	}
	return rows
}

type actBrief struct {
	Games int `json:"games"`
	Wins  int `json:"wins"`
}

func noteLastAct(rows []recentMatch) {
	start, _, ok := knownAct()
	if !ok {
		return
	}
	begin := start.Add(-75 * 24 * time.Hour).UnixMilli()
	floor := start.UnixMilli()
	games, wins := 0, 0
	for _, row := range rows {
		got := stampMillis(row.Started, row.When)
		if got == 0 || got < begin || got >= floor {
			continue
		}
		games++
		if row.Won != nil && *row.Won {
			wins++
		}
	}
	if games == 0 {
		return
	}
	dir := velvetBase()
	if dir == "" {
		return
	}
	path := filepath.Join(dir, "last-act.json")
	var prev actBrief
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &prev)
	}
	if prev.Games >= games {
		return
	}
	raw, err := json.Marshal(actBrief{Games: games, Wins: wins})
	if err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o600)
}

func loadLastAct() (actBrief, bool) {
	var brief actBrief
	raw, err := os.ReadFile(filepath.Join(velvetBase(), "last-act.json"))
	if err != nil || json.Unmarshal(raw, &brief) != nil || brief.Games == 0 {
		return actBrief{}, false
	}
	return brief, true
}

func trimActRecords(rows []recordGame) []recordGame {
	kept := rows[:0]
	for _, row := range rows {
		if inAct(0, row.When) {
			kept = append(kept, row)
		}
	}
	rows = kept
	if len(rows) > actSafetyCap {
		rows = rows[:actSafetyCap]
	}
	return rows
}

func pollRecent(rows []recentMatch) []recentMatch {
	if len(rows) > 36 {
		rows = append([]recentMatch(nil), rows[:36]...)
	} else {
		rows = append([]recentMatch(nil), rows...)
	}
	for i := range rows {
		rows[i].Edges = nil
		rows[i].Buys = nil
	}
	return rows
}

func gunLabel(id string) string {
	name := weaponName(strings.ToLower(strings.TrimSpace(id)))
	if name == "" || name == "Weapon" {
		return ""
	}
	return name
}

func gunsText(guns map[string]int) string {
	type pair struct {
		name string
		n    int
	}
	rows := []pair{}
	for name, n := range guns {
		name = strings.TrimSpace(name)
		if name == "" || n <= 0 {
			continue
		}
		rows = append(rows, pair{name, n})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].n != rows[j].n {
			return rows[i].n > rows[j].n
		}
		return rows[i].name < rows[j].name
	})
	parts := make([]string, len(rows))
	for i, row := range rows {
		parts[i] = row.name + ":" + strconv.Itoa(row.n)
	}
	return strings.Join(parts, ",")
}

func gunHSText(shots map[string][2]int) string {
	type pair struct {
		name  string
		heads int
		shots int
	}
	rows := []pair{}
	for name, pairN := range shots {
		name = strings.TrimSpace(name)
		if name == "" || pairN[1] <= 0 {
			continue
		}
		rows = append(rows, pair{name, pairN[0], pairN[1]})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].shots != rows[j].shots {
			return rows[i].shots > rows[j].shots
		}
		return rows[i].name < rows[j].name
	})
	parts := make([]string, len(rows))
	for i, row := range rows {
		parts[i] = row.name + ":" + strconv.Itoa(row.heads) + "/" + strconv.Itoa(row.shots)
	}
	return strings.Join(parts, ",")
}

func plainInt(v string) int {
	n := 0
	any := false
	for _, r := range v {
		if r < '0' || r > '9' {
			if any {
				break
			}
			continue
		}
		any = true
		n = n*10 + int(r-'0')
		if n > 100000000 {
			return n
		}
	}
	return n
}

func parseGuns(text string) map[string]int {
	out := map[string]int{}
	for _, part := range strings.Split(text, ",") {
		name, count, ok := strings.Cut(part, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			continue
		}
		n := plainInt(count)
		if n <= 0 {
			continue
		}
		out[name] += n
	}
	return out
}

func parseGunHS(text string) map[string][2]int {
	out := map[string][2]int{}
	for _, part := range strings.Split(text, ",") {
		name, rest, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		heads, shots, ok := strings.Cut(rest, "/")
		name = strings.TrimSpace(name)
		if !ok || name == "" {
			continue
		}
		pair := out[name]
		pair[0] += plainInt(heads)
		pair[1] += plainInt(shots)
		if pair[1] > 0 {
			out[name] = pair
		}
	}
	return out
}

func guessLength(mode string, rounds int) int {
	m := strings.ToLower(mode)
	switch {
	case strings.Contains(m, "deathmatch"), strings.Contains(m, "escalation"), strings.Contains(m, "replication"):
		return 10 * 60
	case strings.Contains(m, "spike"), strings.Contains(m, "swift"), strings.Contains(m, "snowball"), strings.Contains(m, "hurm"):
		return 14 * 60
	}
	if rounds > 0 {
		return rounds * 100
	}
	return 35 * 60
}

// lengthMillis treats a stored match length as seconds, unless it is already
// milliseconds. Records, duo time, and the session clock all use this.
func lengthMillis(length int) int {
	if length >= 100000 {
		return length
	}
	if length > 0 {
		return length * 1000
	}
	return 0
}

func statDayStart(t time.Time) time.Time {
	t = t.In(time.Local)
	if t.Hour() < 6 {
		t = t.Add(-6 * time.Hour)
	}
	y, m, d := t.Date()
	return time.Date(y, m, d, 6, 0, 0, 0, t.Location())
}

func gameFromRecent(row recentMatch) recordGame {
	you := player{}
	for _, p := range row.Players {
		if p.You {
			you = p
			break
		}
	}
	faces := []recordFace{}
	for _, p := range row.Players {
		if p.You || strings.TrimSpace(p.PUUID) == "" {
			continue
		}
		side := "enemy"
		if !p.Enemy && (you.Team == "" || p.Team == "" || strings.EqualFold(p.Team, you.Team)) {
			side = "ally"
		}
		faces = append(faces, recordFace{ID: p.PUUID, Name: p.Name, Tag: p.Tag, Side: side})
	}
	length := row.Length
	if length <= 0 {
		length = guessLength(row.Mode, len(row.Rounds))
	}
	return recordGame{
		ID: row.ID, Map: row.Map, Mode: row.Mode, Agent: you.Agent, Won: row.Won,
		Kills: plainInt(you.Kills), Deaths: plainInt(you.Deaths), ACS: plainInt(you.ACS), HS: plainInt(you.HS),
		Guns: you.Guns, GunHS: you.GunHS, Faces: faces, When: row.When, Length: length,
	}
}

func gamesFrom(rows []recentMatch) []recordGame {
	out := make([]recordGame, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.ID) == "" {
			continue
		}
		out = append(out, gameFromRecent(row))
	}
	return out
}

func gameScore(g recordGame) int {
	n := len(g.Faces)
	if g.Map != "" {
		n += 2
	}
	if g.Agent != "" {
		n++
	}
	if g.Guns != "" {
		n++
	}
	if g.GunHS != "" {
		n += 4
	}
	if g.Kills > 0 {
		n++
	}
	if g.Won != nil {
		n++
	}
	return n
}

func preferGame(next, prev recordGame) bool {
	sn, sp := gameScore(next), gameScore(prev)
	if sn != sp {
		return sn > sp
	}
	return next.When > prev.When
}

func mergeRecords(have, add []recordGame) []recordGame {
	by := map[string]recordGame{}
	order := []string{}
	addOne := func(row recordGame) {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if id == "" {
			return
		}
		prev, ok := by[id]
		if !ok {
			order = append(order, id)
			by[id] = row
			return
		}
		if preferGame(row, prev) {
			by[id] = row
		}
	}
	for _, row := range add {
		addOne(row)
	}
	for _, row := range have {
		addOne(row)
	}
	rows := make([]recordGame, 0, len(order))
	for _, id := range order {
		rows = append(rows, by[id])
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].When != rows[j].When {
			return rows[i].When > rows[j].When
		}
		return rows[i].ID > rows[j].ID
	})
	if len(rows) > actSafetyCap {
		rows = rows[:actSafetyCap]
	}
	return trimActRecords(rows)
}

func recordsPath() string {
	return filepath.Join(velvetBase(), "records.json")
}

func migrateTree(prev, next string) {
	info, err := os.Stat(prev)
	if err != nil || !info.IsDir() {
		return
	}
	if _, err := os.Stat(next); os.IsNotExist(err) {
		if os.Rename(prev, next) == nil {
			return
		}
	}
	_ = os.MkdirAll(next, 0o700)
	entries, err := os.ReadDir(prev)
	if err != nil {
		return
	}
	for _, entry := range entries {
		from := filepath.Join(prev, entry.Name())
		to := filepath.Join(next, entry.Name())
		if _, err := os.Stat(to); err == nil {
			continue
		}
		_ = os.Rename(from, to)
	}
	left, _ := os.ReadDir(prev)
	if len(left) == 0 {
		_ = os.Remove(prev)
	}
}

func migrateAppDirs() {
	if base := os.Getenv("LOCALAPPDATA"); base != "" {
		migrateTree(filepath.Join(base, "Velvet"), filepath.Join(base, "Valvet"))
	}
	if base := os.Getenv("APPDATA"); base != "" {
		migrateTree(filepath.Join(base, "Velvet"), filepath.Join(base, "Valvet"))
	}
}

func velvetBase() string {
	migrateAppDirs()
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	next := filepath.Join(base, "Valvet")
	prev := filepath.Join(base, "Velvet")
	if info, err := os.Stat(next); err == nil && info.IsDir() {
		return next
	}
	if info, err := os.Stat(prev); err == nil && info.IsDir() {
		return prev
	}
	_ = os.MkdirAll(next, 0o700)
	return next
}

var recordsMu sync.Mutex

func readRecordFile() []recordGame {
	raw, err := os.ReadFile(recordsPath())
	if err != nil {
		return nil
	}
	var rows []recordGame
	if json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	return rows
}

func writeRecordFile(rows []recordGame) {
	if len(rows) == 0 {
		return
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		return
	}
	path := recordsPath()
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, raw, 0o644)
}

func loadRecords() []recordGame {
	recordsMu.Lock()
	defer recordsMu.Unlock()
	return mergeRecords(nil, readRecordFile())
}

func storeRecords(add []recordGame) {
	if len(add) == 0 {
		return
	}
	recordsMu.Lock()
	defer recordsMu.Unlock()
	writeRecordFile(mergeRecords(readRecordFile(), add))
}

type faceTally struct {
	id, name, tag, when     string
	wins, losses, games, ms int
}

func takeFace(book map[string]*faceTally, face recordFace, won *bool, length int, when string) {
	id := strings.ToLower(strings.TrimSpace(face.ID))
	if id == "" {
		return
	}
	row := book[id]
	if row == nil {
		row = &faceTally{id: face.ID}
		book[id] = row
	}
	row.games++
	row.ms += lengthMillis(length)
	if won != nil {
		if *won {
			row.wins++
		} else {
			row.losses++
		}
	}
	if when >= row.when && strings.TrimSpace(face.Name) != "" && !blankName(face.Name) {
		row.name = face.Name
		row.tag = face.Tag
		row.when = when
	}
	if row.name == "" {
		row.name = strings.TrimSpace(face.Name)
		if row.name == "" {
			row.name = "Player"
		}
		row.tag = face.Tag
	}
}

func peopleFrom(book map[string]*faceTally, minutes bool) []recordPerson {
	rows := []recordPerson{}
	for _, row := range book {
		if row.games < 2 {
			continue
		}
		item := recordPerson{
			ID: row.id, PUUID: row.id, Name: row.name, Tag: row.tag,
			Wins: row.wins, Losses: row.losses, Games: row.games,
		}
		if minutes {
			item.Minutes = row.ms / 60000
		}
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Games != rows[j].Games {
			return rows[i].Games > rows[j].Games
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

func buildRecordView(rows []recordGame, friends map[string]bool) recordView {
	if friends == nil {
		friends = map[string]bool{}
	}
	known := map[string]bool{}
	for id, ok := range friends {
		if ok && id != "" {
			known[strings.ToLower(id)] = true
		}
	}
	view := recordView{
		Friends: []recordPerson{}, Duos: []recordPerson{}, Agents: []recordAgent{},
		Maps: []recordMap{}, Guns: []recordGun{}, Days: []recordDay{}, KD: "—", HS: "—",
	}
	allies := map[string]*faceTally{}
	enemies := map[string]*faceTally{}
	agents := map[string]*struct {
		wins, losses, games, kills, deaths, hsSum, hsN, acsSum, acsN int
	}{}
	maps := map[string]*recordMap{}
	guns := map[string]*struct{ kills, heads, shots int }{}
	days := map[string]*struct {
		label                                  string
		games, kills, hsSum, hsN, acsSum, acsN int
	}{}
	var hsSum, hsN, acsSum, acsN int
	for _, row := range rows {
		view.Games++
		view.Kills += row.Kills
		view.Deaths += row.Deaths
		if row.Won != nil {
			if *row.Won {
				view.Wins++
			} else {
				view.Losses++
			}
		}
		if row.Kills > view.BestKills {
			view.BestKills = row.Kills
			view.BestKillsMap = row.Map
		}
		if row.ACS > view.BestACS {
			view.BestACS = row.ACS
			view.BestACSMap = row.Map
		}
		if row.HS > 0 || row.Kills > 0 {
			hsSum += row.HS
			hsN++
		}
		if row.ACS > 0 {
			acsSum += row.ACS
			acsN++
		}
		if row.Agent != "" {
			ag := agents[row.Agent]
			if ag == nil {
				ag = &struct {
					wins, losses, games, kills, deaths, hsSum, hsN, acsSum, acsN int
				}{}
				agents[row.Agent] = ag
			}
			ag.games++
			ag.kills += row.Kills
			ag.deaths += row.Deaths
			if row.Won != nil {
				if *row.Won {
					ag.wins++
				} else {
					ag.losses++
				}
			}
			if row.HS > 0 || row.Kills > 0 {
				ag.hsSum += row.HS
				ag.hsN++
			}
			if row.ACS > 0 {
				ag.acsSum += row.ACS
				ag.acsN++
			}
		}
		if row.Map != "" && row.Won != nil {
			m := maps[row.Map]
			if m == nil {
				m = &recordMap{Name: row.Map}
				maps[row.Map] = m
			}
			m.Games++
			if *row.Won {
				m.Wins++
			} else {
				m.Losses++
			}
		}
		for name, n := range parseGuns(row.Guns) {
			g := guns[name]
			if g == nil {
				g = &struct{ kills, heads, shots int }{}
				guns[name] = g
			}
			g.kills += n
		}
		for name, pair := range parseGunHS(row.GunHS) {
			g := guns[name]
			if g == nil {
				g = &struct{ kills, heads, shots int }{}
				guns[name] = g
			}
			g.heads += pair[0]
			g.shots += pair[1]
		}
		for _, face := range row.Faces {
			id := strings.ToLower(strings.TrimSpace(face.ID))
			if id == "" {
				continue
			}
			if strings.EqualFold(face.Side, "ally") && known[id] {
				takeFace(allies, face, row.Won, row.Length, row.When)
			} else if strings.EqualFold(face.Side, "enemy") && !known[id] {
				takeFace(enemies, face, row.Won, row.Length, row.When)
			}
		}
		if when, err := time.Parse(time.RFC3339, row.When); err == nil {
			start := statDayStart(when)
			key := start.Format("2006-01-02")
			day := days[key]
			if day == nil {
				day = &struct {
					label                                  string
					games, kills, hsSum, hsN, acsSum, acsN int
				}{label: start.Format("Jan 2")}
				days[key] = day
			}
			day.games++
			day.kills += row.Kills
			if row.HS > 0 || row.Kills > 0 {
				day.hsSum += row.HS
				day.hsN++
			}
			if row.ACS > 0 {
				day.acsSum += row.ACS
				day.acsN++
			}
		}
	}
	if view.Deaths > 0 {
		view.KD = strconv.FormatFloat(float64(view.Kills)/float64(view.Deaths), 'f', 2, 64)
	} else if view.Kills > 0 {
		view.KD = strconv.Itoa(view.Kills) + ".00"
	}
	if hsN > 0 {
		view.HS = strconv.Itoa(hsSum/hsN) + "%"
	}
	if acsN > 0 {
		view.ACS = acsSum / acsN
	}
	view.Friends = peopleFrom(allies, false)
	view.Duos = peopleFrom(allies, true)
	rivals := peopleFrom(enemies, false)
	if len(rivals) > 0 {
		top := rivals[0]
		view.Rival = &top
	}
	for name, ag := range agents {
		hs, acs := 0, 0
		if ag.hsN > 0 {
			hs = ag.hsSum / ag.hsN
		}
		if ag.acsN > 0 {
			acs = ag.acsSum / ag.acsN
		}
		view.Agents = append(view.Agents, recordAgent{
			Name: name, Wins: ag.wins, Losses: ag.losses, Games: ag.games,
			Kills: ag.kills, Deaths: ag.deaths, HS: hs, ACS: acs,
		})
	}
	sort.Slice(view.Agents, func(i, j int) bool {
		if view.Agents[i].Games != view.Agents[j].Games {
			return view.Agents[i].Games > view.Agents[j].Games
		}
		return view.Agents[i].Name < view.Agents[j].Name
	})
	for _, m := range maps {
		view.Maps = append(view.Maps, *m)
	}
	sort.Slice(view.Maps, func(i, j int) bool {
		if view.Maps[i].Games != view.Maps[j].Games {
			return view.Maps[i].Games > view.Maps[j].Games
		}
		if view.Maps[i].Wins != view.Maps[j].Wins {
			return view.Maps[i].Wins > view.Maps[j].Wins
		}
		return view.Maps[i].Name < view.Maps[j].Name
	})
	if len(view.Maps) > 0 {
		best := view.Maps[0]
		view.BestMap = best.Name
		if best.Games > 0 {
			view.BestMapRate = strconv.Itoa(best.Wins*100/best.Games) + "%"
		}
	}
	bestKills := 0
	for name, g := range guns {
		if g.kills > bestKills || (g.kills == bestKills && (view.BestGun == "" || name < view.BestGun)) {
			bestKills = g.kills
			view.BestGun = name
		}
		hs := 0
		if g.shots > 0 {
			hs = g.heads * 100 / g.shots
		}
		view.Guns = append(view.Guns, recordGun{Name: name, Kills: g.kills, HS: hs, Shots: g.shots})
	}
	sort.Slice(view.Guns, func(i, j int) bool {
		if view.Guns[i].Kills != view.Guns[j].Kills {
			return view.Guns[i].Kills > view.Guns[j].Kills
		}
		return view.Guns[i].Name < view.Guns[j].Name
	})
	if len(view.Guns) > 8 {
		view.Guns = view.Guns[:8]
	}
	keys := make([]string, 0, len(days))
	for key := range days {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		day := days[key]
		hs, acs := 0, 0
		if day.hsN > 0 {
			hs = day.hsSum / day.hsN
		}
		if day.acsN > 0 {
			acs = day.acsSum / day.acsN
		}
		view.Days = append(view.Days, recordDay{Day: key, Label: day.label, Games: day.games, HS: hs, ACS: acs, Kills: day.kills})
	}
	if view.Games > 0 {
		view.Note = "Saved on this PC."
	}
	return view
}

func diskFriendIDs() map[string]bool {
	out := map[string]bool{}
	raw, err := os.ReadFile(socialPath())
	if err != nil {
		return out
	}
	var book socialBook
	if json.Unmarshal(raw, &book) != nil {
		return out
	}
	for id := range book.Seen {
		if id != "" {
			out[strings.ToLower(id)] = true
		}
	}
	return out
}

func (c *riotClient) withRecords(out snapshot) snapshot {
	live := out.Phase == "In game" || out.Phase == "Loading lobby" || out.Phase == "Agent select"
	c.mu.Lock()
	if live && c.recordsReady && time.Since(c.recordsAt) < 45*time.Second {
		out.Records = c.recordsView
		out.Pings = append([]string(nil), c.recordsPings...)
		if c.recordsAct != nil {
			got := *c.recordsAct
			out.LastAct = &got
		}
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	friends := diskFriendIDs()
	for _, id := range out.FriendIDs {
		if id != "" {
			friends[strings.ToLower(id)] = true
		}
	}
	for _, person := range out.Friends {
		if person.PUUID != "" {
			friends[strings.ToLower(person.PUUID)] = true
		}
	}
	out.Records = buildRecordView(loadRecords(), friends)
	out.Pings = recentPings()
	if brief, ok := loadLastAct(); ok {
		got := brief
		out.LastAct = &got
	}
	c.mu.Lock()
	c.recordsView = out.Records
	c.recordsPings = append([]string(nil), out.Pings...)
	c.recordsAct = nil
	if out.LastAct != nil {
		got := *out.LastAct
		c.recordsAct = &got
	}
	c.recordsAt = time.Now()
	c.recordsReady = true
	c.mu.Unlock()
	return out
}

func (c *riotClient) matchIDs(ctx context.Context, sess riotSession, limit int) []string {
	return c.matchIDsFor(ctx, sess, sess.puuid, limit)
}

func (c *riotClient) matchIDsFor(ctx context.Context, sess riotSession, puuid string, limit int) []string {
	if sess.shard == "" || puuid == "" || limit < 1 {
		return nil
	}
	if limit > 120 {
		limit = 120
	}
	ids := []string{}
	for start := 0; start < limit; start += 20 {
		end := start + 20
		if end > limit {
			end = limit
		}
		page := c.historyPage(ctx, sess, puuid, start, end)
		if len(page) == 0 {
			break
		}
		ids = append(ids, page...)
		if len(ids) >= limit || len(page) < end-start {
			break
		}
	}
	if len(ids) > limit {
		ids = ids[:limit]
	}
	return ids
}

func (c *riotClient) historyPage(ctx context.Context, sess riotSession, puuid string, start, end int) []string {
	hits := c.historyHits(ctx, sess, puuid, start, end)
	out := make([]string, 0, len(hits))
	for _, hit := range hits {
		if hit.ID != "" {
			out = append(out, hit.ID)
		}
	}
	return out
}

func (c *riotClient) fetchMatches(ctx context.Context, sess riotSession, agents map[string]agentInfo, ids []string) []recentMatch {
	if len(ids) == 0 || sess.shard == "" {
		return nil
	}
	out := make([]*recentMatch, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/match-details/v1/matches/"+id, nil)
			if err != nil || status >= 400 {
				return
			}
			var detail map[string]any
			if json.Unmarshal(payload, &detail) != nil {
				return
			}
			out[i] = finalizeRecent(detail, sess.puuid, agents)
		}(i, id)
	}
	wg.Wait()
	rows := []recentMatch{}
	for _, row := range out {
		if row != nil {
			rows = append(rows, *row)
		}
	}
	if len(rows) > 0 {
		c.fillRosterNames(ctx, sess, matchPlayers(rows))
		for i := range rows {
			relabelEdges(&rows[i])
		}
	}
	return rows
}

func (c *riotClient) replayMatch(ctx context.Context, id string) *recentMatch {
	id = strings.ToLower(strings.TrimSpace(id))
	if !looksUUID(id) {
		return nil
	}
	if pack, ok := loadRoundPack(id); ok && pack.Full && edgesHaveLines(pack.Edges) {
		return pack.asMatch()
	}
	c.mu.Lock()
	sess := c.session
	agents := map[string]agentInfo{}
	for key, info := range c.agents {
		agents[key] = info
	}
	c.mu.Unlock()
	if sess.shard == "" || sess.puuid == "" {
		if pack, ok := loadRoundPack(id); ok {
			return pack.asMatch()
		}
		return nil
	}
	rows := c.fetchMatches(ctx, sess, agents, []string{id})
	if len(rows) == 0 {
		if pack, ok := loadRoundPack(id); ok {
			return pack.asMatch()
		}
		return nil
	}
	saveRoundPack(rows[0])
	return &rows[0]
}

func (c *riotClient) growRecords(ctx context.Context, sess riotSession, agents map[string]agentInfo) {
	have := map[string]bool{}
	thin := map[string]bool{}
	for _, row := range loadRecords() {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if id == "" {
			continue
		}
		have[id] = true
		if row.GunHS == "" {
			thin[id] = true
		}
	}
	need := []string{}
	for _, id := range c.matchIDs(ctx, sess, 80) {
		key := strings.ToLower(id)
		if have[key] && !thin[key] {
			continue
		}
		need = append(need, id)
		if len(need) >= 8 {
			break
		}
	}
	if len(need) == 0 {
		return
	}
	rows := c.fetchMatches(ctx, sess, agents, need)
	if len(rows) == 0 {
		return
	}
	storeRecords(gamesFrom(rows))
}

func (c *riotClient) kickWatch(sess riotSession) {
	if sess.shard == "" || sess.puuid == "" {
		return
	}
	book := loadSocial()
	if len(book.Watch) == 0 {
		return
	}
	c.mu.Lock()
	if c.watchBusy || (!c.watchAt.IsZero() && time.Since(c.watchAt) < 8*time.Minute) {
		c.mu.Unlock()
		return
	}
	c.watchBusy = true
	idx := c.watchCursor % len(book.Watch)
	c.watchCursor++
	c.mu.Unlock()
	target := book.Watch[idx]
	go func() {
		defer func() {
			c.mu.Lock()
			c.watchBusy = false
			c.watchAt = time.Now()
			c.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		puuid := target.PUUID
		if puuid == "" {
			if _, _, id := c.knownPlayer(target.Name, target.Tag); id != "" {
				puuid = id
				rememberWatchPUUID(target.Name, target.Tag, id)
			}
		}
		if puuid == "" {
			return
		}
		id := c.latestMatchID(ctx, sess, puuid)
		if id == "" || sawMatch(puuid, id) {
			return
		}
		label := strings.TrimSpace(target.Name + "#" + target.Tag)
		mapName := ""
		var finished int64
		if rows := c.fetchMatches(ctx, sess, c.agents, []string{id}); len(rows) > 0 {
			mapName = rows[0].Map
			finished = rows[0].Started
			if rows[0].Length > 0 && finished > 0 {
				finished += int64(lengthMillis(rows[0].Length))
			}
		}
		if finished == 0 {
			for _, hit := range c.historyHits(ctx, sess, puuid, 0, 1) {
				if strings.EqualFold(hit.ID, id) {
					finished = hit.Start
					if finished > 0 && finished < 1_000_000_000_000 {
						finished *= 1000
					}
				}
			}
		}
		if finished == 0 {
			finished = time.Now().UnixMilli()
		}
		notePing(id, label, mapName, finished)
	}()
}

func (c *riotClient) latestMatchID(ctx context.Context, sess riotSession, puuid string) string {
	ids := c.historyPage(ctx, sess, puuid, 0, 1)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

type savedPing struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	At   int64  `json:"at"`
}

type watchSeenFile struct {
	Last map[string]string `json:"last"`
}

var pingMu sync.Mutex

func watchSeenPath() string { return filepath.Join(velvetBase(), "watch-seen.json") }
func pingsPath() string     { return filepath.Join(velvetBase(), "pings.json") }

func loadWatchSeen() watchSeenFile {
	var seen watchSeenFile
	raw, err := os.ReadFile(watchSeenPath())
	if err != nil || json.Unmarshal(raw, &seen) != nil || seen.Last == nil {
		seen.Last = map[string]string{}
	}
	return seen
}

func saveWatchSeen(seen watchSeenFile) {
	raw, err := json.Marshal(seen)
	if err != nil {
		return
	}
	_ = os.MkdirAll(velvetBase(), 0o755)
	_ = os.WriteFile(watchSeenPath(), raw, 0o644)
}

func loadPings() []savedPing {
	var rows []savedPing
	raw, err := os.ReadFile(pingsPath())
	if err != nil || json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	return rows
}

func savePings(rows []savedPing) {
	raw, err := json.Marshal(rows)
	if err != nil {
		return
	}
	_ = os.MkdirAll(velvetBase(), 0o755)
	_ = os.WriteFile(pingsPath(), raw, 0o644)
}

func sawMatch(puuid, matchID string) bool {
	puuid = strings.ToLower(strings.TrimSpace(puuid))
	matchID = strings.ToLower(strings.TrimSpace(matchID))
	if puuid == "" || matchID == "" {
		return true
	}
	pingMu.Lock()
	defer pingMu.Unlock()
	seen := loadWatchSeen()
	prev := strings.ToLower(seen.Last[puuid])
	if prev == matchID {
		return true
	}
	first := prev == ""
	seen.Last[puuid] = matchID
	saveWatchSeen(seen)
	return first
}

func notePing(matchID, name, mapName string, finished int64) {
	matchID = strings.TrimSpace(matchID)
	name = strings.TrimSpace(name)
	mapName = strings.ReplaceAll(strings.TrimSpace(mapName), "|", " ")
	if matchID == "" || name == "" {
		return
	}
	text := name + "|" + mapName + "|" + strconv.FormatInt(finished, 10)
	pingMu.Lock()
	defer pingMu.Unlock()
	rows := append([]savedPing{{ID: matchID, Text: text, At: time.Now().Unix()}}, loadPings()...)
	if len(rows) > 20 {
		rows = rows[:20]
	}
	savePings(rows)
}

func recentPings() []string {
	pingMu.Lock()
	defer pingMu.Unlock()
	cutoff := time.Now().Add(-6 * time.Hour).Unix()
	out := []string{}
	for _, row := range loadPings() {
		if row.At < cutoff || row.ID == "" {
			continue
		}
		out = append(out, row.ID+"|"+row.Text)
		if len(out) >= 8 {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

const matchArchiveCap = actSafetyCap

type histHit struct {
	ID    string
	Start int64
}

func (c *riotClient) historyHits(ctx context.Context, sess riotSession, puuid string, start, end int) []histHit {
	if sess.shard == "" || puuid == "" || end <= start {
		return nil
	}
	rawURL := "https://pd." + sess.shard + ".a.pvp.net/match-history/v1/history/" + puuid + "?startIndex=" + strconv.Itoa(start) + "&endIndex=" + strconv.Itoa(end)
	status, payload, err := c.remoteCall(ctx, sess, "GET", rawURL, nil)
	if err != nil || status >= 400 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	hits := []histHit{}
	for _, item := range asArr(pick(body, "History", "history")) {
		row := asMap(item)
		id := str(pick(row, "MatchID", "matchId"))
		if id == "" {
			continue
		}
		hits = append(hits, histHit{ID: id, Start: int64(num(pick(row, "GameStartTime", "gameStartTime")))})
	}
	return hits
}

func loadMatchArchive() []recentMatch {
	var rows []recentMatch
	if _, ok := readCache("matches", &rows); !ok || len(rows) == 0 {
		return nil
	}
	return mergeMatches(nil, rows)
}

func saveMatchArchive(rows []recentMatch) {
	rows = mergeMatches(nil, rows)
	if len(rows) == 0 {
		return
	}
	writeCache("matches", rows)
	archivePastActs(rows)
}

func archivePastActs(rows []recentMatch) {
	start, _, ok := knownAct()
	if !ok {
		return
	}
	floor := start.UnixMilli()
	var past []recentMatch
	for _, row := range rows {
		got := stampMillis(row.Started, row.When)
		if got > 0 && got < floor {
			past = append(past, row)
		}
	}
	if len(past) == 0 {
		return
	}
	dir := filepath.Join(velvetBase(), "acts")
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	name := start.UTC().Format("2006-01-02") + "-before.json"
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return
	}
	raw, err := json.Marshal(past)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, raw, 0o600)
}

func loadPastAct() []recentMatch {
	dir := filepath.Join(velvetBase(), "acts")
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) == 0 {
		return nil
	}
	name := ""
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if entry.Name() > name {
			name = entry.Name()
		}
	}
	if name == "" {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil
	}
	var rows []recentMatch
	if json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	return rows
}

func mergeMatches(have, add []recentMatch) []recentMatch {
	by := map[string]recentMatch{}
	order := []string{}
	addOne := func(row recentMatch) {
		id := strings.ToLower(strings.TrimSpace(row.ID))
		if id == "" {
			return
		}
		prev, ok := by[id]
		if !ok {
			order = append(order, id)
			by[id] = row
			return
		}
		if matchRicher(row, prev) {
			by[id] = keepParty(row, prev)
			return
		}
		by[id] = keepParty(prev, row)
	}
	for _, row := range add {
		addOne(row)
	}
	for _, row := range have {
		addOne(row)
	}
	rows := make([]recentMatch, 0, len(order))
	for _, id := range order {
		rows = append(rows, by[id])
	}
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Started != rows[j].Started {
			return rows[i].Started > rows[j].Started
		}
		return rows[i].When > rows[j].When
	})
	if len(rows) > actSafetyCap {
		rows = rows[:actSafetyCap]
	}
	noteLastAct(rows)
	rows = trimActMatches(rows)
	parkLines(rows)
	return rows
}

func keepParty(dst, src recentMatch) recentMatch {
	if dst.KnownParty || !src.KnownParty {
		return dst
	}
	by := map[string]player{}
	for _, p := range src.Players {
		id := strings.ToLower(p.PUUID)
		if id != "" && p.Party != "" {
			by[id] = p
		}
	}
	if len(by) == 0 {
		return dst
	}
	for i := range dst.Players {
		id := strings.ToLower(dst.Players[i].PUUID)
		if prev, ok := by[id]; ok {
			dst.Players[i].Party = prev.Party
			dst.Players[i].PartyN = prev.PartyN
		}
	}
	dst.KnownParty = true
	return dst
}

func matchRicher(a, b recentMatch) bool {
	weight := func(m recentMatch) int {
		n := len(m.Players)*2 + len(m.Rounds)*3 + len(m.Edges)
		if len(m.Kills) > 0 || len(m.Deaths) > 0 {
			n += 8
		}
		if m.Won != nil {
			n += 6
		}
		if m.Score1 != nil {
			n += 2
		}
		if m.KnownParty {
			n += 2
		}
		return n
	}
	return weight(a) > weight(b)
}

func (c *riotClient) growArchive(ctx context.Context, sess riotSession, agents map[string]agentInfo) {
	if sess.shard == "" || sess.puuid == "" {
		return
	}
	start, _ := c.ensureAct(ctx)
	floor := start.UnixMilli()
	c.mu.Lock()
	if c.archiveDone && c.archiveFloor == floor {
		c.mu.Unlock()
		return
	}
	if c.archiveFloor != floor {
		c.archiveDone = false
		c.archiveAt = 0
		c.archiveFloor = floor
	}
	have := map[string]bool{}
	for _, row := range c.recent {
		if row.ID != "" {
			have[strings.ToLower(row.ID)] = true
		}
	}
	cursor := c.archiveAt
	c.mu.Unlock()
	need := []string{}
	done := false
	for page := 0; page < 1 && len(need) < 4; page++ {
		if floor == 0 && cursor >= 120 {
			done = true
			break
		}
		hits := c.historyHits(ctx, sess, sess.puuid, cursor, cursor+20)
		if len(hits) == 0 {
			done = true
			break
		}
		pageOld := false
		for _, hit := range hits {
			if floor > 0 && hit.Start > 0 && hit.Start < floor {
				pageOld = true
				continue
			}
			key := strings.ToLower(hit.ID)
			if have[key] {
				continue
			}
			have[key] = true
			need = append(need, hit.ID)
		}
		cursor += len(hits)
		if pageOld || len(hits) < 20 {
			done = true
			break
		}
	}
	if len(need) > 12 {
		need = need[:12]
	}
	var rows []recentMatch
	if len(need) > 0 {
		rows = c.fetchMatches(ctx, sess, agents, need)
	}
	c.mu.Lock()
	c.archiveAt = cursor
	if done && len(need) == 0 {
		c.archiveDone = true
		c.archiveFloor = floor
	}
	if len(rows) > 0 {
		c.recent = mergeMatches(c.recent, rows)
	} else if floor > 0 {
		c.recent = mergeMatches(c.recent, nil)
	}
	merged := append([]recentMatch(nil), c.recent...)
	c.mu.Unlock()
	if len(merged) > 0 {
		saveMatchArchive(merged)
	}
	if len(rows) > 0 {
		storeRecords(gamesFrom(rows))
	}
}

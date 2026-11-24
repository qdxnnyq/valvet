package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const clientPlatform = "ew0KCSJwbGF0Zm9ybVR5cGUiOiAiUEMiLA0KCSJwbGF0Zm9ybU9TIjogIldpbmRvd3MiLA0KCSJwbGF0Zm9ybU9TVmVyc2lvbiI6ICIxMC4wLjE5MDQyLjEuMjU2LjY0Yml0IiwNCgkicGxhdGZvcm1DaGlwc2V0IjogIlVua25vd24iDQp9"

var errClosed = errors.New("closed")
var errMissing = errors.New("missing")

type riotClient struct {
	local        *http.Client
	remote       *http.Client
	mu           sync.Mutex
	readMu       sync.Mutex
	snap         snapshot
	last         time.Time
	histAt       time.Time
	histBusy     bool
	watchBusy    bool
	watchAt      time.Time
	watchCursor  int
	archiveAt    int
	archiveDone  bool
	archiveFloor int64
	recent       []recentMatch
	agents       map[string]agentInfo
	rankIcons    map[int]string
	career       map[string]career
	careerBusy   bool
	rankCache    map[string]rankHit
	rankHTTP     *http.Client
	rankBusy     bool
	rankWait     []player
	knownNames   map[string][2]string
	session      riotSession
	skins        map[string]agentInfo
	skinsAt      time.Time
	cards        map[string]string
	cardsAt      time.Time
	skinLook     map[string]agentInfo
	gunNames     map[string]gear
	levelNames   map[string]string
	chromaNames  map[string]string
	chromaArt    map[string]string
	chromaSwatch map[string]string
	buddies      map[string]flairInfo
	flair        map[string]flairInfo
	flairAt      time.Time
	savedLoadout loadoutView
	loadoutBody  []byte
	skinBook     map[string]skinRecord
	levelOwner   map[string]string
	chromaOwner  map[string]string
	levelAnim    map[string]string
	catalogAt    time.Time
	guns         []gear
	gunsAt       time.Time
	shop         shop
	shopAt       time.Time
	bundleArt    map[string]agentInfo
	bundleNote   map[string]string
	bundleAt     time.Time
	rewards      map[string]agentInfo
	rewardsAt    time.Time
	agentBook    []agentCard
	agentBookAt  time.Time
	gearBase     []gearAgent
	gearBaseAt   time.Time
	offerByItem  map[string]string
	offerAt      time.Time
	borders      []borderBit
	bordersAt    time.Time
	masteryRows  []masteryAgent
	masteryErr   string
	masteryAt    time.Time
	acctLevel    int
	levelAt      time.Time
	cardSnap     cardView
	cardAt       time.Time
	cardBusy     bool
	partySeen    map[string]partyHint
	stackLockID  string
	stackLock    map[string]partyHint
	stackTried   string
	stackTriedAt time.Time
	stackDoneID  string
	stackPassID  string
	stackPasses  int
	stackRows    []recentMatch
	stackRowsAt  time.Time
	queueHist    map[string]histCache
	friendIDs    []string
	friendNames  map[string]friendNote
	liveOnly     bool
	gotAgents    time.Time
	scoreScan    time.Time
	partyAt      time.Time
	stackBusy    bool
	version      string
	versionAt    time.Time
	rankSaved    time.Time
	asleep       bool
	skipStore    bool
	skipBook     bool
	penalty      string
	penaltyAt    time.Time
}

func newRiotClient() *riotClient {
	recent := loadMatchArchive()
	if recent == nil {
		recent = []recentMatch{}
	}
	hadLines := false
	for _, row := range recent {
		if len(row.Edges) > 0 || len(row.Buys) > 0 {
			hadLines = true
			break
		}
	}
	parkLines(recent)
	if hadLines {
		saveMatchArchive(recent)
	}
	return &riotClient{
		local: &http.Client{Timeout: 8 * time.Second, Transport: &http.Transport{
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxConnsPerHost:     2,
			MaxIdleConnsPerHost: 2,
		}},
		remote: &http.Client{Timeout: 12 * time.Second, Transport: &http.Transport{
			MaxConnsPerHost:     6,
			MaxIdleConnsPerHost: 4,
		}},
		rankHTTP: &http.Client{Timeout: 6 * time.Second, Transport: &http.Transport{
			MaxConnsPerHost:     8,
			MaxIdleConnsPerHost: 8,
		}},
		agents:    map[string]agentInfo{},
		rankIcons: map[int]string{},
		career:    map[string]career{},
		rankCache: loadRankCache(),
		skins:     map[string]agentInfo{},
		queueHist: map[string]histCache{},
		recent:    append([]recentMatch(nil), recent...),
		snap: snapshot{
			Error:   "Open Valorant. This window reads the live match on this PC.",
			Players: []player{},
			Recent:  append([]recentMatch(nil), recent...),
		},
	}
}

func (c *riotClient) setLiveOnly(on bool) {
	c.mu.Lock()
	c.liveOnly = on
	c.mu.Unlock()
}

type riotSession struct {
	puuid, access, entitlement, name, tag, region, shard, version string
}

func sessionFrom(ent map[string]any) riotSession {
	return riotSession{
		puuid:       str(pick(ent, "subject", "Subject")),
		access:      str(pick(ent, "accessToken", "AccessToken")),
		entitlement: str(pick(ent, "token", "Token", "entitlementsToken")),
	}
}

func labelYou(name, tag string) string {
	if name == "" {
		return ""
	}
	if tag == "" {
		return name
	}
	return name + "#" + tag
}

func (c *riotClient) signedInRiot() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return labelYou(c.session.name, c.session.tag)
}

func stampSelfLevel(party []friend, level int) {
	if level <= 0 {
		return
	}
	for i := range party {
		if party[i].You && party[i].Level == 0 {
			party[i].Level = level
		}
	}
}

func selfLevel(party []friend) int {
	for _, person := range party {
		if person.You && person.Level > 0 {
			return person.Level
		}
	}
	return 0
}

func (c *riotClient) playerLevel(ctx context.Context, sess riotSession, priv map[string]any, party []friend) int {
	level := num(presencePick(priv, "accountLevel", "AccountLevel"))
	if level == 0 {
		level = selfLevel(party)
	}
	c.mu.Lock()
	cached := c.acctLevel
	fresh := cached > 0 && time.Since(c.levelAt) < 10*time.Minute
	c.mu.Unlock()
	if level > 0 && (cached == 0 || !fresh) {
		c.mu.Lock()
		c.acctLevel = level
		c.levelAt = time.Now()
		c.mu.Unlock()
		return level
	}
	if fresh {
		if level > cached {
			return level
		}
		return cached
	}
	if got := c.accountXP(ctx, sess); got > 0 {
		return got
	}
	return level
}

func (c *riotClient) accountXP(ctx context.Context, sess riotSession) int {
	if sess.shard == "" || sess.puuid == "" {
		return 0
	}
	call, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	status, payload, err := c.remoteCall(call, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/account-xp/v1/players/"+sess.puuid, nil)
	if err != nil || status >= 400 {
		return 0
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return 0
	}
	n := num(pick(asMap(pick(body, "Progress", "progress")), "Level", "level"))
	if n <= 0 {
		n = num(pick(body, "Level", "level"))
	}
	if n > 0 {
		c.mu.Lock()
		c.acctLevel = n
		c.levelAt = time.Now()
		c.mu.Unlock()
	}
	return n
}

func normalizeRegion(raw string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "na", "northamerica":
		return "na", "na"
	case "eu", "europe":
		return "eu", "eu"
	case "ap", "asia", "apac":
		return "ap", "ap"
	case "kr", "korea":
		return "kr", "kr"
	case "br", "brazil":
		return "br", "br"
	case "latam", "latinamerica":
		return "latam", "latam"
	default:
		return "", ""
	}
}

func readLockfile() (string, string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", "", errClosed
	}
	b, err := os.ReadFile(filepath.Join(base, "Riot Games", "Riot Client", "Config", "lockfile"))
	if err != nil {
		return "", "", errClosed
	}
	parts := strings.Split(strings.TrimSpace(string(b)), ":")
	if len(parts) < 5 || parts[2] == "" || parts[3] == "" {
		return "", "", errClosed
	}
	return parts[2], parts[3], nil
}

func readTail(path string, n int64) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return ""
	}
	if info.Size() > n {
		_, _ = f.Seek(info.Size()-n, io.SeekStart)
	}
	b, _ := io.ReadAll(f)
	return string(b)
}

func (c *riotClient) localJSON(ctx context.Context, method, url, auth string, body any) (map[string]any, error) {
	status, payload, err := c.do(ctx, c.local, method, url, auth, "", body)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, errMissing
	}
	if status >= 400 {
		return nil, errors.New("local status")
	}
	var out map[string]any
	if json.Unmarshal(payload, &out) != nil {
		return nil, errors.New("bad json")
	}
	return out, nil
}

func (c *riotClient) do(ctx context.Context, client *http.Client, method, url, auth, version string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	if strings.HasPrefix(auth, "Basic ") || strings.HasPrefix(auth, "Bearer ") {
		req.Header.Set("Authorization", auth)
	}
	if version != "" {
		req.Header.Set("X-Riot-ClientVersion", version)
		req.Header.Set("X-Riot-ClientPlatform", clientPlatform)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return res.StatusCode, payload, nil
}

func waitRiot(ctx context.Context, gap time.Duration) bool {
	for {
		riotPace.mu.Lock()
		now := time.Now()
		if now.Before(riotPace.cool) {
			riotPace.mu.Unlock()
			return false
		}
		if !riotPace.busy && !now.Before(riotPace.next) {
			riotPace.busy = true
			riotPace.next = now.Add(gap)
			riotPace.mu.Unlock()
			return true
		}
		wait := riotPace.next.Sub(now)
		if wait < 30*time.Millisecond {
			wait = 30 * time.Millisecond
		}
		riotPace.mu.Unlock()
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return false
		case <-timer.C:
		}
	}
}

func doneRiot() {
	riotPace.mu.Lock()
	riotPace.busy = false
	riotPace.mu.Unlock()
}

func coolRiot(d time.Duration) {
	riotPace.mu.Lock()
	until := time.Now().Add(d)
	if until.After(riotPace.cool) {
		riotPace.cool = until
	}
	riotPace.mu.Unlock()
}

var riotPace struct {
	mu   sync.Mutex
	next time.Time
	cool time.Time
	busy bool
}

func riotGap(url string) time.Duration {
	if strings.Contains(url, "match-history") || strings.Contains(url, "match-details") {
		return 400 * time.Millisecond
	}
	return 0
}

func (c *riotClient) remoteCall(ctx context.Context, sess riotSession, method, url string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, reader)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sess.access)
	req.Header.Set("X-Riot-Entitlements-JWT", sess.entitlement)
	req.Header.Set("X-Riot-ClientPlatform", clientPlatform)
	req.Header.Set("X-Riot-ClientVersion", sess.version)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if gap := riotGap(url); gap > 0 {
		if !waitRiot(ctx, gap) {
			return 0, nil, ctx.Err()
		}
		defer doneRiot()
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if res.StatusCode == http.StatusTooManyRequests && riotGap(url) > 0 {
		coolRiot(90 * time.Second)
	}
	return res.StatusCode, payload, nil
}

func liveScores(self map[string]any) (ally, enemy int) {
	ally, enemy, _ = presenceScores(self)
	return ally, enemy
}

func (c *riotClient) selfPresence(ctx context.Context, port, auth, puuid string) map[string]any {
	body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/chat/v4/presences", auth, nil)
	if err != nil {
		return nil
	}
	for _, item := range asArr(pick(body, "presences")) {
		row := asMap(item)
		if strings.EqualFold(str(pick(row, "puuid")), puuid) {
			return decodePrivate(str(pick(row, "private")))
		}
	}
	return nil
}

func (c *riotClient) matchDoc(ctx context.Context, port, auth string, sess riotSession, kind string) map[string]any {
	if m := c.localMatch(ctx, port, auth, kind, sess.puuid); m != nil {
		return m
	}
	return c.glzMatch(ctx, sess, kind)
}

func (c *riotClient) localMatch(ctx context.Context, port, auth, kind, puuid string) map[string]any {
	if port == "" || puuid == "" {
		return nil
	}
	body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/"+kind+"/v1/players/"+puuid, auth, nil)
	if err != nil {
		return nil
	}
	id := str(pick(body, "MatchID", "matchID", "MatchId"))
	if id == "" {
		return nil
	}
	match, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/"+kind+"/v1/matches/"+id, auth, nil)
	if err != nil || len(match) == 0 {
		return nil
	}
	return match
}

func (c *riotClient) glzMatch(ctx context.Context, sess riotSession, kind string) map[string]any {
	base := "https://glz-" + sess.region + "-1." + sess.shard + ".a.pvp.net/" + kind + "/v1"
	status, payload, err := c.remoteCall(ctx, sess, "GET", base+"/players/"+sess.puuid, nil)
	if err != nil || status == http.StatusNotFound || status >= 400 {
		return nil
	}
	var player map[string]any
	if json.Unmarshal(payload, &player) != nil {
		return nil
	}
	id := str(pick(player, "MatchID", "matchID"))
	if id == "" {
		return nil
	}
	status, payload, err = c.remoteCall(ctx, sess, "GET", base+"/matches/"+id, nil)
	if err != nil || status >= 400 {
		return nil
	}
	var match map[string]any
	if json.Unmarshal(payload, &match) != nil {
		return nil
	}
	return match
}

func (c *riotClient) glzByID(ctx context.Context, sess riotSession, kind, id string) map[string]any {
	if id == "" || sess.region == "" || sess.shard == "" {
		return nil
	}
	url := "https://glz-" + sess.region + "-1." + sess.shard + ".a.pvp.net/" + kind + "/v1/matches/" + id
	status, payload, err := c.remoteCall(ctx, sess, "GET", url, nil)
	if err != nil || status >= 400 {
		return nil
	}
	var match map[string]any
	if json.Unmarshal(payload, &match) != nil {
		return nil
	}
	return match
}

func (c *riotClient) rememberNames(book map[string][2]string) {
	if len(book) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.knownNames == nil {
		c.knownNames = map[string][2]string{}
	}
	for id, pair := range book {
		if !blankName(pair[0]) {
			c.knownNames[strings.ToLower(id)] = pair
		}
	}
}

func (c *riotClient) applyKnownNames(players []player) {
	c.mu.Lock()
	book := c.knownNames
	c.mu.Unlock()
	applyNames(players, book)
}

func (c *riotClient) resolveHidden(ctx context.Context, sess riotSession, players []player) {
	ids := []string{}
	for i := range players {
		if blankName(players[i].Name) && players[i].PUUID != "" {
			ids = append(ids, players[i].PUUID)
		}
	}
	if len(ids) == 0 {
		return
	}
	book := c.names(ctx, sess, ids)
	if len(book) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.knownNames == nil {
		c.knownNames = map[string][2]string{}
	}
	for i := range players {
		pair, ok := book[strings.ToLower(players[i].PUUID)]
		if !ok || blankName(pair[0]) {
			continue
		}
		players[i].Name = pair[0]
		players[i].Tag = pair[1]
		c.knownNames[strings.ToLower(players[i].PUUID)] = pair
	}
}

func (c *riotClient) names(ctx context.Context, sess riotSession, ids []string) map[string][2]string {
	clean := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		clean = append(clean, id)
	}
	if len(clean) == 0 {
		return nil
	}
	status, payload, err := c.remoteCall(ctx, sess, "PUT", "https://pd."+sess.shard+".a.pvp.net/name-service/v2/players", clean)
	if err != nil || status >= 400 {
		return nil
	}
	return parseNames(payload)
}

var vtlName = regexp.MustCompile(`(?i)([^\s<>"#|]{2,24})#([A-Za-z0-9]{2,6})`)

func (c *riotClient) stampLivePresence(ctx context.Context, port, auth string, players []player) map[string]partyHint {
	hints := map[string]partyHint{}
	c.mu.Lock()
	icons := c.rankIcons
	c.mu.Unlock()
	byID := map[string]int{}
	for i := range players {
		if id := strings.ToLower(players[i].PUUID); id != "" {
			byID[id] = i
		}
	}
	grouped := map[string][]map[string]any{}
	for _, p := range c.gatherPresences(ctx, port, auth) {
		id := strings.ToLower(str(pick(p, "puuid", "Puuid", "PUUID", "subject", "Subject")))
		if !looksUUID(id) {
			continue
		}
		grouped[id] = append(grouped[id], p)
	}
	for id, list := range grouped {
		idx, ok := byID[id]
		if !ok {
			continue
		}
		for _, row := range list {
			priv := presenceBlob(row)
			if priv == nil {
				continue
			}
			if players[idx].Level == 0 {
				if level := num(presencePick(priv, "accountLevel", "AccountLevel")); level > 0 {
					players[idx].Level = level
				}
			}
			if tier := num(presencePick(priv, "competitiveTier", "CompetitiveTier")); tier >= 3 && players[idx].Tier < 3 {
				applyPresenceTier(&players[idx], tier, icons)
				c.rememberRank(id, tier, 0, 0)
			}
			party, size := findParty(priv)
			if party != "" {
				hints[id] = partyHint{id: strings.ToLower(party), size: size}
			}
		}
	}
	return hints
}

func (c *riotClient) kickStacks(sess riotSession, port, auth, matchID string, players []player, presence map[string]partyHint, mine, me string) {
	c.mu.Lock()
	if c.stackBusy {
		c.mu.Unlock()
		return
	}
	if matchID != "" && c.stackTried == matchID && time.Since(c.stackTriedAt) < 8*time.Second {
		c.mu.Unlock()
		return
	}
	c.stackBusy = true
	c.stackTried = matchID
	c.stackTriedAt = time.Now()
	copied := append([]player(nil), players...)
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			c.stackBusy = false
			c.mu.Unlock()
		}()
		found := map[string]partyHint{}
		roster := map[string]bool{}
		for _, p := range copied {
			if id := strings.ToLower(p.PUUID); looksUUID(id) {
				roster[id] = true
			}
		}
		raw := map[string]partyHint{}
		keep := func(src map[string]partyHint) {
			for id, hint := range src {
				if hint.id == "" || raw[id].id != "" {
					continue
				}
				raw[id] = partyHint{id: strings.ToLower(hint.id), size: hint.size}
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		for _, kind := range []string{"pregame", "core-game"} {
			if doc := c.glzMatch(ctx, sess, kind); doc != nil {
				got := map[string]string{}
				harvestParties(doc, got)
				for id, party := range got {
					if party != "" && raw[strings.ToLower(id)].id == "" {
						raw[strings.ToLower(id)] = partyHint{id: strings.ToLower(party)}
					}
				}
				fillGroups(found, hintsFromIDs(got))
			}
		}
		cancel()
		keep(partyHintsFrom(copied))
		ids, heard := c.partyIDs(sess, copied)
		remote := map[string]partyHint{}
		for id, party := range ids {
			if party != "" {
				remote[id] = partyHint{id: strings.ToLower(party)}
			}
		}
		keep(remote)
		keep(presence)
		missing := make([]player, 0, len(copied))
		for _, p := range copied {
			if raw[strings.ToLower(p.PUUID)].id == "" {
				missing = append(missing, p)
			}
		}
		for id, party := range c.localPartyIDs(port, auth, missing) {
			if party != "" && raw[id].id == "" {
				raw[id] = partyHint{id: strings.ToLower(party)}
			}
		}
		if mine == "" && me != "" {
			if hint, ok := raw[me]; ok && looksUUID(hint.id) {
				mine = hint.id
			} else if hint, ok := found[me]; ok {
				mine = hint.id
			}
		}
		members := c.myPartyMembers(context.Background(), port, auth, me)
		if mine != "" && looksUUID(mine) {
			if extraMembers := c.fetchPartyMembers(sess, port, auth, mine); len(extraMembers) >= 2 {
				members = extraMembers
			}
		}
		if mine == "" && len(members) >= 2 {
			mine = "mine"
		}
		forceParty(found, mine, members, roster)
		parties := map[string]bool{}
		for _, hint := range raw {
			if looksUUID(hint.id) && hint.id != strings.ToLower(mine) {
				parties[hint.id] = true
			}
		}
		pinned := 0
		for partyID := range parties {
			if pinned >= 6 {
				break
			}
			pinned++
			extraMembers := c.fetchPartyMembers(sess, port, auth, partyID)
			pinMembers(found, partyID, extraMembers, roster)
		}
		fillGroups(found, raw)
		fillGroups(found, c.archiveStacks(copied))
		grouped := groupsOf(found)
		c.rememberParties(grouped)
		if matchID == "" {
			return
		}
		c.mu.Lock()
		if c.stackPassID != matchID {
			c.stackPassID = matchID
			c.stackPasses = 0
		}
		c.stackPasses++
		passes := c.stackPasses
		c.mu.Unlock()
		rosterN := len(roster)
		presenceN := 0
		for id := range roster {
			if _, ok := presence[id]; ok && presence[id].id != "" {
				presenceN++
			}
		}
		seen := rosterN > 0 && presenceN*3 >= rosterN*2
		foreign := foreignStack(grouped, me, mine)
		if passes >= 10 || (passes >= 2 && seen && (foreign || heard*4 >= rosterN*3)) {
			c.mu.Lock()
			c.stackDoneID = matchID
			c.mu.Unlock()
			if partyGrouped(grouped) {
				c.saveLocked(matchID, grouped)
			}
		}
	}()
}

func (c *riotClient) myPartyMembers(ctx context.Context, port, auth, me string) []string {
	me = strings.ToLower(me)
	if !looksUUID(me) {
		return nil
	}
	groups := map[string][]string{}
	seen := map[string]bool{}
	for _, part := range c.chatParts(ctx, port, auth) {
		id := strings.ToLower(part.id)
		cid := strings.ToLower(part.cid)
		if !looksUUID(id) || cid == "" || seen[id+"|"+cid] {
			continue
		}
		if strings.Contains(cid, "coregame") || strings.Contains(cid, "conference") {
			continue
		}
		if !strings.Contains(cid, "party") && !strings.Contains(cid, "parties") {
			continue
		}
		seen[id+"|"+cid] = true
		groups[cid] = append(groups[cid], id)
	}
	for _, ids := range groups {
		hasMe := false
		for _, id := range ids {
			if id == me {
				hasMe = true
				break
			}
		}
		if hasMe && len(ids) >= 2 && len(ids) <= 5 {
			return ids
		}
	}
	return nil
}

func (c *riotClient) localPartyMembers(ctx context.Context, port, auth, partyID string) []string {
	partyID = strings.ToLower(partyID)
	if port == "" || !looksUUID(partyID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 400*time.Millisecond)
	defer cancel()
	body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/parties/v1/parties/"+partyID, auth, nil)
	if err != nil {
		return nil
	}
	return partyMemberIDs(body)
}

func (c *riotClient) fetchPartyMembers(sess riotSession, port, auth, partyID string) []string {
	partyID = strings.ToLower(partyID)
	if !looksUUID(partyID) {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	if port != "" && auth != "" {
		if body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/parties/v1/parties/"+partyID, auth, nil); err == nil {
			if ids := partyMemberIDs(body); len(ids) >= 2 && len(ids) <= 5 {
				return ids
			}
		}
	}
	if sess.region == "" || sess.shard == "" {
		return nil
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://glz-"+sess.region+"-1."+sess.shard+".a.pvp.net/parties/v1/parties/"+partyID, nil)
	if err != nil || status >= 400 || len(payload) == 0 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	ids := partyMemberIDs(body)
	if len(ids) < 2 || len(ids) > 5 {
		return nil
	}
	return ids
}

func (c *riotClient) archiveStacks(players []player) map[string]partyHint {
	c.mu.Lock()
	if c.stackRows == nil || time.Since(c.stackRowsAt) > 20*time.Second {
		c.mu.Unlock()
		rows := loadMatchArchive()
		if rows == nil {
			rows = []recentMatch{}
		}
		c.mu.Lock()
		c.stackRows = rows
		c.stackRowsAt = time.Now()
	}
	rows := c.stackRows
	c.mu.Unlock()
	return stacksFromArchive(rows, players)
}

func (c *riotClient) stacksDone(matchID string) bool {
	if matchID == "" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.stackDoneID == matchID
}

func partyIDFrom(body map[string]any) string {
	if id := str(pick(body, "CurrentPartyID", "currentPartyID", "PartyID", "partyID")); looksUUID(id) {
		return strings.ToLower(id)
	}
	id, _ := findParty(body)
	if looksUUID(id) {
		return strings.ToLower(id)
	}
	return ""
}

func (c *riotClient) rememberParties(hints map[string]partyHint) {
	if len(hints) == 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.partySeen == nil {
		c.partySeen = map[string]partyHint{}
	}
	for id, hint := range hints {
		if hint.id != "" {
			c.partySeen[id] = hint
		}
	}
}

func (c *riotClient) cachedParties(players []player) map[string]partyHint {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]partyHint{}
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if hint, ok := c.partySeen[id]; ok && hint.id != "" {
			out[id] = hint
		}
	}
	return out
}

func foreignStack(hints map[string]partyHint, me, mine string) bool {
	me = strings.ToLower(me)
	mine = strings.ToLower(mine)
	count := map[string]int{}
	self := map[string]bool{}
	for id, hint := range hints {
		if hint.id == "" {
			continue
		}
		count[hint.id]++
		if id == me {
			self[hint.id] = true
		}
	}
	for id, n := range count {
		if n < 2 || n > 5 || self[id] || id == mine || id == "mine" {
			continue
		}
		return true
	}
	return false
}

var partyHTTP = &http.Client{Timeout: 2 * time.Second, Transport: &http.Transport{
	TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
	MaxConnsPerHost:     4,
	MaxIdleConnsPerHost: 4,
}}

func (c *riotClient) localPartyIDs(port, auth string, players []player) map[string]string {
	out := map[string]string{}
	if port == "" || auth == "" || len(players) == 0 {
		return out
	}
	ids := make([]string, 0, len(players))
	seen := map[string]bool{}
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if !looksUUID(id) || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, 2)
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
			defer cancel()
			status, payload, err := c.do(ctx, partyHTTP, "GET", "https://127.0.0.1:"+port+"/parties/v1/players/"+id, auth, "", nil)
			if err != nil || status >= 400 || len(payload) == 0 {
				return
			}
			var body map[string]any
			if json.Unmarshal(payload, &body) != nil {
				return
			}
			party := partyIDFrom(body)
			if party == "" {
				return
			}
			mu.Lock()
			out[id] = party
			mu.Unlock()
		}(id)
	}
	wg.Wait()
	return out
}

type rosterRow struct {
	PUUID string `json:"puuid"`
	Tier  int    `json:"tier,omitempty"`
	RR    int    `json:"rr,omitempty"`
	Rank  string `json:"rank,omitempty"`
	Party string `json:"party,omitempty"`
	Name  string `json:"name,omitempty"`
	Tag   string `json:"tag,omitempty"`
}

type rosterFile struct {
	Match   string      `json:"match"`
	At      int64       `json:"at"`
	Players []rosterRow `json:"players"`
}

func (c *riotClient) restoreRoster(matchID string) map[string]partyHint {
	matchID = strings.ToLower(strings.TrimSpace(matchID))
	if matchID == "" {
		return nil
	}
	var file rosterFile
	if _, ok := readCache("roster", &file); !ok || !strings.EqualFold(file.Match, matchID) {
		return nil
	}
	if file.At > 0 && time.Since(time.Unix(file.At, 0)) > 3*time.Hour {
		return nil
	}
	hints := map[string]partyHint{}
	for _, row := range file.Players {
		id := strings.ToLower(row.PUUID)
		if !looksUUID(id) {
			continue
		}
		if row.Tier >= 3 {
			c.rememberRank(id, row.Tier, row.RR, 0)
		}
		if row.Party != "" {
			hints[id] = partyHint{id: strings.ToLower(row.Party)}
		}
	}
	return hints
}

func (c *riotClient) saveRoster(matchID string, players []player, hints map[string]partyHint) {
	matchID = strings.ToLower(strings.TrimSpace(matchID))
	if matchID == "" || len(players) == 0 {
		return
	}
	rows := make([]rosterRow, 0, len(players))
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if !looksUUID(id) {
			continue
		}
		party := ""
		if hint, ok := hints[id]; ok {
			party = hint.id
		}
		if p.Tier < 3 && party == "" {
			continue
		}
		rows = append(rows, rosterRow{PUUID: id, Tier: p.Tier, RR: p.RR, Rank: p.Rank, Party: party, Name: p.Name, Tag: p.Tag})
	}
	if len(rows) == 0 {
		return
	}
	writeCache("roster", rosterFile{Match: matchID, At: time.Now().Unix(), Players: rows})
}

func (c *riotClient) noteRecentParties(matchID string, hints map[string]partyHint) {
	matchID = strings.ToLower(strings.TrimSpace(matchID))
	if matchID == "" || !partyGrouped(hints) {
		return
	}
	c.mu.Lock()
	changed := false
	for i := range c.recent {
		if !strings.EqualFold(c.recent[i].ID, matchID) {
			continue
		}
		markParties(c.recent[i].Players, hints)
		if !c.recent[i].KnownParty {
			c.recent[i].KnownParty = true
			changed = true
		}
	}
	var rows []recentMatch
	if changed {
		rows = append([]recentMatch(nil), c.recent...)
	}
	c.mu.Unlock()
	if changed {
		saveMatchArchive(rows)
	}
}

func (c *riotClient) applySavedParties(rows []recentMatch) {
	if len(rows) == 0 {
		return
	}
	var file rosterFile
	if _, ok := readCache("roster", &file); !ok || strings.TrimSpace(file.Match) == "" {
		return
	}
	hints := map[string]partyHint{}
	for _, row := range file.Players {
		id := strings.ToLower(row.PUUID)
		if looksUUID(id) && row.Party != "" {
			hints[id] = partyHint{id: strings.ToLower(row.Party)}
		}
	}
	if !partyGrouped(hints) {
		return
	}
	for i := range rows {
		if !strings.EqualFold(rows[i].ID, file.Match) {
			continue
		}
		markParties(rows[i].Players, hints)
		rows[i].KnownParty = true
	}
	c.noteRecentParties(file.Match, hints)
}

func (c *riotClient) flushRanks() {
	c.mu.Lock()
	if time.Since(c.rankSaved) < 5*time.Second {
		c.mu.Unlock()
		return
	}
	c.rankSaved = time.Now()
	c.mu.Unlock()
	c.saveRanks()
}

func (c *riotClient) partyIDs(sess riotSession, players []player) (map[string]string, int) {
	out := map[string]string{}
	if sess.region == "" || sess.shard == "" || len(players) == 0 {
		return out, 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	ids := make([]string, 0, len(players))
	seen := map[string]bool{}
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if !looksUUID(id) || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	found := make([]string, len(ids))
	heard := make([]bool, len(ids))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 6)
	for i, id := range ids {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			call, stop := context.WithTimeout(ctx, 6*time.Second)
			defer stop()
			status, payload, err := c.remoteCall(call, sess, "GET", "https://glz-"+sess.region+"-1."+sess.shard+".a.pvp.net/parties/v1/players/"+id, nil)
			if err != nil || status >= 400 || len(payload) == 0 {
				return
			}
			var body map[string]any
			if json.Unmarshal(payload, &body) != nil {
				return
			}
			heard[i] = true
			found[i] = partyIDFrom(body)
		}(i, id)
	}
	wg.Wait()
	n := 0
	for i, id := range ids {
		if heard[i] {
			n++
		}
		if found[i] != "" {
			out[id] = found[i]
		}
	}
	return out, n
}

func (c *riotClient) lockedStacks(matchID string) map[string]partyHint {
	if matchID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stackLockID != matchID || len(c.stackLock) == 0 {
		return nil
	}
	out := map[string]partyHint{}
	for id, hint := range c.stackLock {
		out[id] = hint
	}
	return out
}

func (c *riotClient) saveLocked(matchID string, hints map[string]partyHint) {
	if matchID == "" || !partyGrouped(hints) {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	out := map[string]partyHint{}
	for id, hint := range hints {
		if hint.id != "" {
			out[id] = hint
		}
	}
	c.stackLockID = matchID
	c.stackLock = out
}

func (c *riotClient) foldChat(ctx context.Context, port, auth, self, ourColor string, match map[string]any, players []player) []player {
	parts := c.chatParts(ctx, port, auth)
	tiers := map[string]int{}
	if body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/chat/v4/presences", auth, nil); err == nil {
		for _, item := range asArr(pick(body, "presences")) {
			row := asMap(item)
			id := str(pick(row, "puuid"))
			name := str(pick(row, "game_name", "gameName"))
			tag := str(pick(row, "game_tag", "gameTag"))
			if looksUUID(id) && !blankName(name) {
				parts = append(parts, chatPart{id: id, name: name, tag: tag})
			}
			if priv := decodePrivate(str(pick(row, "private"))); priv != nil && looksUUID(id) {
				if tier := num(presencePick(priv, "competitiveTier", "CompetitiveTier")); tier >= 3 {
					tiers[strings.ToLower(id)] = tier
				}
			}
		}
	}
	stamp := func(list []player) []player { return c.stampPresence(list, tiers) }
	if len(parts) == 0 {
		return stamp(players)
	}
	names := map[string][2]string{}
	for _, part := range parts {
		if !blankName(part.name) {
			names[strings.ToLower(part.id)] = [2]string{part.name, part.tag}
		}
	}
	applyNames(players, names)
	c.rememberNames(names)
	teamMUC := strings.ToLower(str(pick(match, "TeamMUCName", "teamMUCName", "MUCName", "mucName")))
	allMUC := strings.ToLower(str(pick(match, "AllMUCName", "allMUCName")))
	if teamMUC == "" && allMUC == "" {
		return stamp(players)
	}
	ally := ourColor
	for _, p := range players {
		if self != "" && strings.EqualFold(p.PUUID, self) && p.Team != "" {
			ally = p.Team
		}
	}
	enemy := "Red"
	if strings.EqualFold(ally, "Red") {
		enemy = "Blue"
	}
	have := map[string]bool{}
	for _, p := range players {
		have[strings.ToLower(p.PUUID)] = true
	}
	var extra []player
	for _, part := range parts {
		id := strings.ToLower(part.id)
		cid := strings.ToLower(part.cid)
		if id == "" || have[id] || blankName(part.name) || cid == "" {
			continue
		}
		team := ""
		switch cid {
		case allMUC:
			if allMUC == "" || allMUC == teamMUC {
				continue
			}
			team = enemy
		case teamMUC:
			team = ally
		default:
			continue
		}
		if team == "" {
			continue
		}
		extra = append(extra, player{PUUID: part.id, Name: part.name, Tag: part.tag, Team: team, Agent: "Picking"})
		have[id] = true
		if len(players)+len(extra) >= 10 {
			break
		}
	}
	return stamp(mergePlayers(players, extra))
}

type chatPart struct{ id, cid, name, tag string }

func (c *riotClient) chatParts(ctx context.Context, port, auth string) []chatPart {
	var out []chatPart
	for _, path := range []string{"/chat/v5/participants", "/chat/v4/participants", "/chat/v6/participants"} {
		body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+path, auth, nil)
		if err != nil {
			continue
		}
		rows := asArr(pick(body, "participants", "Participants"))
		if len(rows) == 0 {
			rows = asArr(body["participants"])
		}
		for _, item := range rows {
			row := asMap(item)
			id := str(pick(row, "puuid", "pid"))
			if !looksUUID(id) {
				continue
			}
			name := str(pick(row, "game_name", "gameName", "name"))
			tag := str(pick(row, "game_tag", "gameTag"))
			if tag == "" && strings.Contains(name, "#") {
				name, tag = splitDisplay(name)
			}
			out = append(out, chatPart{id: id, cid: str(pick(row, "cid")), name: name, tag: tag})
		}
		if len(out) > 0 {
			return out
		}
	}
	return out
}

func (c *riotClient) fillMissingNames(ctx context.Context, sess riotSession, players []player) {
	missing := []string{}
	for _, p := range players {
		if p.PUUID != "" && blankName(p.Name) {
			missing = append(missing, p.PUUID)
		}
	}
	if len(missing) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()
	book := c.aliasNames(ctx, sess, missing)
	applyNames(players, book)
	c.rememberNames(book)
	still := []int{}
	for i := range players {
		if blankName(players[i].Name) && players[i].PUUID != "" {
			still = append(still, i)
		}
	}
	if len(still) == 0 {
		return
	}
	hits := make([][2]string, len(players))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)
	for _, i := range still {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			hits[i][0], hits[i][1] = nameFromVTL(ctx, c.remote, id)
		}(i, players[i].PUUID)
	}
	wg.Wait()
	found := map[string][2]string{}
	for _, i := range still {
		if blankName(hits[i][0]) {
			continue
		}
		players[i].Name = hits[i][0]
		players[i].Tag = hits[i][1]
		found[strings.ToLower(players[i].PUUID)] = hits[i]
	}
	c.rememberNames(found)
}

func (c *riotClient) aliasNames(ctx context.Context, sess riotSession, ids []string) map[string][2]string {
	if sess.access == "" || len(ids) == 0 {
		return nil
	}
	q := url.Values{}
	for _, id := range ids {
		q.Add("puuid", id)
	}
	tries := []struct {
		method, url string
		body        any
	}{
		{"GET", "https://api.account.riotgames.com/aliases/v1/aliases?" + q.Encode(), nil},
		{"POST", "https://api.account.riotgames.com/aliases/v1/aliases", ids},
		{"PUT", "https://api.account.riotgames.com/aliases/v1/aliases", ids},
	}
	for _, try := range tries {
		status, payload, err := c.do(ctx, c.remote, try.method, try.url, "Bearer "+sess.access, "", try.body)
		if err != nil || status >= 400 {
			continue
		}
		if book := namesFromAliases(payload); len(book) > 0 {
			return book
		}
	}
	return nil
}

func nameFromVTL(ctx context.Context, client *http.Client, id string) (string, string) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://vtl.lol/id/"+id, nil)
	if err != nil {
		return "", ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/json")
	res, err := client.Do(req)
	if err != nil {
		return "", ""
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return "", ""
	}
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 200<<10))
	if book := namesFromAliases(payload); len(book) > 0 {
		for _, pair := range book {
			if !blankName(pair[0]) {
				return pair[0], pair[1]
			}
		}
	}
	if m := vtlName.FindSubmatch(payload); len(m) == 3 {
		return string(m[1]), string(m[2])
	}
	return "", ""
}

func (c *riotClient) kickHistory(sess riotSession) {
	if sess.shard == "" || sess.puuid == "" {
		return
	}
	c.mu.Lock()
	if c.histBusy || (!c.histAt.IsZero() && time.Since(c.histAt) < 3*time.Minute) {
		c.mu.Unlock()
		return
	}
	agents := c.agents
	c.histBusy = true
	c.mu.Unlock()
	go func() {
		defer func() {
			c.mu.Lock()
			c.histBusy = false
			c.mu.Unlock()
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
		defer cancel()
		rows := c.fetchMatches(ctx, sess, agents, c.matchIDs(ctx, sess, 20))
		c.mu.Lock()
		merged := mergeMatches(c.recent, rows)
		if len(merged) > 0 {
			c.recent = merged
			c.histAt = time.Now()
		} else {
			c.histAt = time.Now().Add(-12 * time.Second)
		}
		c.mu.Unlock()
		if len(merged) > 0 {
			saveMatchArchive(merged)
		}
		if len(rows) > 0 {
			storeRecords(gamesFrom(rows))
		}
		growCtx, growCancel := context.WithTimeout(context.Background(), 50*time.Second)
		c.growArchive(growCtx, sess, agents)
		c.growRecords(growCtx, sess, agents)
		growCancel()
	}()
}

func (c *riotClient) history(ctx context.Context, sess riotSession, agents map[string]agentInfo, queue string, limit int) []recentMatch {
	end := limit + 2
	if end < 8 {
		end = 8
	}
	if end > 20 {
		end = 20
	}
	url := "https://pd." + sess.shard + ".a.pvp.net/match-history/v1/history/" + sess.puuid + "?startIndex=0&endIndex=" + strconv.Itoa(end)
	if queue != "" {
		url += "&queue=" + queue
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", url, nil)
	if err != nil || status >= 400 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	type job struct{ id string }
	jobs := []job{}
	for _, item := range asArr(pick(body, "History", "history")) {
		id := str(pick(asMap(item), "MatchID", "matchId"))
		if id != "" {
			jobs = append(jobs, job{id})
		}
	}
	if len(jobs) > limit {
		jobs = jobs[:limit]
	}
	out := make([]*recentMatch, len(jobs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 4)
	for i, job := range jobs {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			defer func() { _ = recover() }()
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
		}(i, job.id)
	}
	wg.Wait()
	rows := []recentMatch{}
	for _, row := range out {
		if row == nil {
			continue
		}
		rows = append(rows, *row)
	}
	c.fillRosterNames(ctx, sess, matchPlayers(rows))
	for i := range rows {
		relabelEdges(&rows[i])
	}
	return rows
}

func matchPlayers(rows []recentMatch) []profileMatch {
	out := make([]profileMatch, len(rows))
	for i := range rows {
		out[i].Players = rows[i].Players
	}
	return out
}

func (c *riotClient) fillRosterNames(ctx context.Context, sess riotSession, matches []profileMatch) {
	c.mu.Lock()
	known := c.knownNames
	c.mu.Unlock()
	if len(known) > 0 {
		for i := range matches {
			applyNames(matches[i].Players, known)
		}
	}
	missing := []string{}
	seen := map[string]bool{}
	for _, match := range matches {
		for _, p := range match.Players {
			id := strings.ToLower(p.PUUID)
			if id == "" || seen[id] || !blankName(p.Name) {
				continue
			}
			seen[id] = true
			missing = append(missing, p.PUUID)
		}
	}
	if len(missing) == 0 {
		return
	}
	book := c.names(ctx, sess, missing)
	if len(book) > 0 {
		for i := range matches {
			applyNames(matches[i].Players, book)
		}
		c.rememberNames(book)
	}
	left := []player{}
	index := map[string][][2]int{}
	for i := range matches {
		for j := range matches[i].Players {
			p := matches[i].Players[j]
			id := strings.ToLower(p.PUUID)
			if id == "" || !blankName(p.Name) {
				continue
			}
			index[id] = append(index[id], [2]int{i, j})
			if len(left) < 12 {
				left = append(left, p)
			}
		}
	}
	if len(left) == 0 {
		return
	}
	c.fillMissingNames(ctx, sess, left)
	for _, p := range left {
		if blankName(p.Name) {
			continue
		}
		for _, at := range index[strings.ToLower(p.PUUID)] {
			matches[at[0]].Players[at[1]].Name = p.Name
			matches[at[0]].Players[at[1]].Tag = p.Tag
		}
	}
}

func (c *riotClient) ensureVersion(ctx context.Context) {
	if c.version != "" && time.Since(c.versionAt) < time.Hour {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://valorant-api.com/v1/version", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var body map[string]any
	if json.NewDecoder(res.Body).Decode(&body) != nil {
		return
	}
	if ver := str(pick(asMap(pick(body, "data")), "riotClientVersion")); ver != "" {
		c.version = ver
		c.versionAt = time.Now()
	}
}

func (c *riotClient) ensureAgents(ctx context.Context) {
	if len(c.agents) > 0 && time.Since(c.gotAgents) < 12*time.Hour {
		return
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://valorant-api.com/v1/agents?isPlayableCharacter=true", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	var body map[string]any
	if json.NewDecoder(res.Body).Decode(&body) != nil {
		return
	}
	next := map[string]agentInfo{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "uuid")))
		name := str(pick(row, "displayName"))
		if id != "" && name != "" {
			next[id] = agentInfo{Name: name, Icon: str(pick(row, "displayIcon"))}
		}
	}
	if len(next) > 0 {
		c.mu.Lock()
		c.agents = next
		c.gotAgents = time.Now()
		c.mu.Unlock()
	}
}

package main

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (c *riotClient) busyPhase() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch c.snap.Phase {
	case "In game", "Loading lobby", "Agent select":
		return true
	default:
		return false
	}
}

func (c *riotClient) noteIdle(asleep, skipStore, skipBook bool) {
	c.mu.Lock()
	c.asleep = asleep
	c.skipStore = skipStore
	c.skipBook = skipBook
	c.mu.Unlock()
}

func (c *riotClient) state(ctx context.Context) snapshot {
	c.mu.Lock()
	if c.asleep && time.Since(c.last) < 45*time.Second && !c.last.IsZero() && (c.snap.OK || c.snap.Error != "") {
		out := c.snap
		players := append([]player(nil), out.Players...)
		c.mu.Unlock()
		c.applyKnownNames(players)
		c.paintRanks(players)
		out.Players = players
		out.Recent = pollRecent(c.recent)
		return c.withRecords(out)
	}
	minGap := 2 * time.Second
	switch c.snap.Phase {
	case "Loading lobby":
		minGap = 12 * time.Second
	case "In game":
		minGap = 8 * time.Second
	case "Agent select":
		minGap = 4 * time.Second
	}
	if time.Since(c.last) < minGap && !c.last.IsZero() {
		out := c.snap
		players := append([]player(nil), out.Players...)
		c.mu.Unlock()
		c.applyKnownNames(players)
		c.paintRanks(players)
		out.Players = players
		out.Recent = pollRecent(c.recent)
		return c.withRecords(out)
	}
	c.last = time.Now()
	c.mu.Unlock()

	c.readMu.Lock()
	next := c.read(ctx)
	c.readMu.Unlock()

	c.mu.Lock()
	if next.OK || c.snap.You == "" {
		if (next.Phase == "In game" || next.Phase == "Loading lobby" || next.Phase == "Agent select") && len(next.Players) == 0 && len(c.snap.Players) > 0 {
			prevLive := c.snap.Phase == "In game" || c.snap.Phase == "Loading lobby" || c.snap.Phase == "Agent select"
			sameMap := next.Map == "" || next.Map == "Unknown map" || next.Map == "Lobby" || c.snap.Map == "" || next.Map == c.snap.Map
			if prevLive && sameMap {
				next.Players = append([]player(nil), c.snap.Players...)
				if next.Map == "" || next.Map == "Unknown map" || next.Map == "Lobby" {
					next.Map = c.snap.Map
				}
				if next.Mode == "" {
					next.Mode = c.snap.Mode
				}
				if next.Score1 == nil {
					next.Score1, next.Score2 = c.snap.Score1, c.snap.Score2
				}
			}
		}
		if next.Level == 0 {
			if c.acctLevel > 0 {
				next.Level = c.acctLevel
			} else if c.snap.Level > 0 {
				next.Level = c.snap.Level
			}
		}
		if len(next.Store.Bundles) == 0 && len(c.snap.Store.Bundles) > 0 {
			next.Store.Bundles = append([]bundleView(nil), c.snap.Store.Bundles...)
		}
		stampSelfLevel(next.Party, next.Level)
		c.snap = next
	} else {
		c.snap.Error = next.Error
		c.snap.OK = false
		c.snap.Phase = ""
		c.snap.Players = []player{}
		if next.You != "" {
			c.snap.You = next.You
		}
		if next.Level > 0 {
			c.snap.Level = next.Level
		}
		if len(next.Party) > 0 || len(next.Friends) > 0 {
			stampSelfLevel(next.Party, c.snap.Level)
			c.snap.Party = next.Party
			c.snap.Friends = next.Friends
		}
		if len(next.Store.Offers) > 0 || next.Store.Error != "" || len(next.Store.Bundles) > 0 {
			if len(next.Store.Bundles) == 0 && len(c.snap.Store.Bundles) > 0 {
				next.Store.Bundles = append([]bundleView(nil), c.snap.Store.Bundles...)
			}
			c.snap.Store = next.Store
		}
	}
	c.snap.Recent = pollRecent(c.recent)
	c.snap.ReadAt = time.Now().Format(time.RFC3339)
	if c.snap.Players == nil {
		c.snap.Players = []player{}
	}
	if c.snap.Recent == nil {
		c.snap.Recent = []recentMatch{}
	}
	if c.snap.Party == nil {
		c.snap.Party = []friend{}
	}
	if c.snap.Friends == nil {
		c.snap.Friends = []friend{}
	}
	if c.snap.Store.Offers == nil {
		c.snap.Store.Offers = []storeItem{}
	}
	out := c.snap
	c.mu.Unlock()
	return c.withRecords(out)
}

func (c *riotClient) read(ctx context.Context) snapshot {
	port, password, err := readLockfile()
	if err != nil {
		appLog("Riot client closed")
		return snapshot{Error: "Open the Riot Client. Friend status works from the client even when Valorant is closed.", Players: []player{}, Recent: c.recent, Party: []friend{}, Friends: []friend{}}
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("riot:"+password))
	var sess riotSession
	if chat, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/chat/v1/session", auth, nil); err == nil {
		sess.puuid = str(pick(chat, "puuid", "subject"))
		sess.name = str(pick(chat, "game_name", "gameName"))
		sess.tag = str(pick(chat, "game_tag", "tagLine"))
	}
	ent, entErr := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/entitlements/v1/token", auth, nil)
	if entErr == nil {
		got := sessionFrom(ent)
		if got.puuid != "" {
			sess.puuid = got.puuid
		}
		sess.access = got.access
		sess.entitlement = got.entitlement
	}
	party, friends := []friend{}, []friend{}
	c.mu.Lock()
	heldPhase := c.snap.Phase
	useLobby := (heldPhase == "In game" || heldPhase == "Loading lobby" || heldPhase == "Agent select") && c.lobbyOK && time.Since(c.lobbyAt) < 20*time.Second
	if useLobby {
		party = append([]friend(nil), c.lobbyParty...)
		friends = append([]friend(nil), c.lobbyFriends...)
	}
	c.mu.Unlock()
	if !useLobby {
		party, friends = c.lobby(ctx, port, auth, sess)
		c.mu.Lock()
		c.lobbyParty = append([]friend(nil), party...)
		c.lobbyFriends = append([]friend(nil), friends...)
		c.lobbyAt = time.Now()
		c.lobbyOK = true
		c.mu.Unlock()
	}
	clientOnly := func() snapshot {
		c.mu.Lock()
		level := selfLevel(party)
		if level == 0 {
			level = c.acctLevel
		}
		bundles := append([]bundleView(nil), c.shop.Bundles...)
		c.mu.Unlock()
		stampSelfLevel(party, level)
		return snapshot{
			OK:      true,
			Phase:   "Riot Client",
			You:     labelYou(sess.name, sess.tag),
			Level:   level,
			Players: []player{},
			Recent:  append([]recentMatch(nil), c.recent...),
			Party:   party,
			Friends: friends,
			Store:   shop{Offers: []storeItem{}, Night: []storeItem{}, Bundles: bundles, Kingdom: []storeItem{}},
		}
	}
	if entErr != nil || sess.puuid == "" || sess.access == "" || sess.entitlement == "" {
		return clientOnly()
	}
	c.mu.Lock()
	liveOnly := c.liveOnly
	skipStore := c.skipStore
	skipBook := c.skipBook
	prevPhase := c.snap.Phase
	prevRegion, prevShard, prevVer := c.session.region, c.session.shard, c.session.version
	logAt := c.logAt
	c.mu.Unlock()
	hot := prevPhase == "In game" || prevPhase == "Loading lobby" || prevPhase == "Agent select" || prevPhase == "In queue"
	region, shard := "", ""
	if (hot || time.Since(logAt) < 3*time.Minute) && prevRegion != "" && prevShard != "" {
		region, shard = prevRegion, prevShard
		if prevVer != "" {
			sess.version = prevVer
		} else {
			sess.version = c.version
		}
	} else {
		if !hot {
			c.ensureVersion(ctx)
		}
		logText := readTail(filepath.Join(os.Getenv("LOCALAPPDATA"), "VALORANT", "Saved", "Logs", "ShooterGame.log"), 180_000)
		region, shard = parseRegion(logText)
		if region == "" {
			if locale, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+"/riotclient/region-locale", auth, nil); err == nil {
				region, shard = normalizeRegion(str(pick(locale, "region")))
			}
		}
		if ver := parseClientVersion(logText); strings.Contains(ver, "shipping") {
			sess.version = ver
		} else if sess.version == "" {
			sess.version = c.version
		}
		c.mu.Lock()
		c.logAt = time.Now()
		c.mu.Unlock()
	}
	if region == "" || shard == "" {
		return clientOnly()
	}
	sess.region, sess.shard = region, shard
	c.mu.Lock()
	c.session = sess
	c.mu.Unlock()
	if !hot {
		c.ensureAgents(ctx)
		c.ensureRanks(ctx)
	}
	store := shop{Offers: []storeItem{}, Night: []storeItem{}, Bundles: []bundleView{}, Kingdom: []storeItem{}}
	if hot || liveOnly || skipStore {
		c.mu.Lock()
		store = c.shop
		c.mu.Unlock()
	} else {
		store = c.dailyStore(ctx, sess)
	}

	priv := c.selfPresence(ctx, port, auth, sess.puuid)
	loop := strings.ToUpper(strings.TrimSpace(str(pick(priv, "sessionLoopState", "SessionLoopState"))))
	queue := str(pick(priv, "queueId", "queueID", "QueueID"))
	mapPath := str(pick(priv, "matchMap", "MatchMap"))
	if mapPath == "" {
		mapPath = str(pick(priv, "partyOwnerMatchMap", "PartyOwnerMatchMap"))
	}
	ally, enemy := liveScores(priv)
	ourColor := str(pick(priv, "partyOwnerMatchCurrentTeam", "PartyOwnerMatchCurrentTeam"))

	var match map[string]any
	phase := "In menu"
	switch loop {
	case "PREGAME":
		phase = "Agent select"
		match = c.localMatch(ctx, port, auth, "pregame", sess.puuid)
	case "INGAME":
		phase = "In game"
		match = c.localMatch(ctx, port, auth, "core-game", sess.puuid)
	default:
		if m := c.localMatch(ctx, port, auth, "pregame", sess.puuid); m != nil {
			phase = "Agent select"
			match = m
		} else if loop != "MENUS" {
			if m := c.localMatch(ctx, port, auth, "core-game", sess.puuid); m != nil {
				phase = "In game"
				match = m
			}
		}
	}
	if match == nil && (loop == "PREGAME" || loop == "INGAME") && sess.region != "" && sess.shard != "" {
		kind := "pregame"
		if loop == "INGAME" {
			kind = "core-game"
		}
		if m := c.glzMatch(ctx, sess, kind); m != nil {
			match = m
			if loop == "PREGAME" {
				phase = "Agent select"
			} else if phase != "Loading lobby" {
				phase = "In game"
			}
		}
	}
	if match != nil {
		if q := str(pick(match, "QueueID", "queueID")); q != "" {
			queue = q
		}
		if m := str(pick(match, "MapID", "mapID")); m != "" {
			mapPath = m
		}
		if state := strings.ToLower(str(pick(match, "PregameState", "State"))); state != "" {
			if strings.Contains(state, "character_select_active") {
				phase = "Agent select"
			} else if strings.Contains(state, "provision") || strings.Contains(state, "finished") {
				phase = "Loading lobby"
			}
		}
	}
	if match == nil {
		state, _, _, _, _, _, _, _, _, _, _ := activity(priv, "", "valorant")
		switch state {
		case "In queue":
			phase = "In queue"
		case "In lobby":
			phase = "In lobby"
		}
	}
	players := []player{}
	if match != nil {
		players = namedPlayers(match, c.agents)
		if len(players) < 10 && phase != "In game" && phase != "Loading lobby" {
			players = c.foldChat(ctx, port, auth, sess.puuid, ourColor, match, players)
		}
		if phase != "In game" && phase != "Loading lobby" {
			ids := make([]string, 0, len(players))
			needNames := false
			for _, p := range players {
				ids = append(ids, p.PUUID)
				if blankName(p.Name) {
					needNames = true
				}
			}
			if needNames {
				nameCtx, cancelNames := context.WithTimeout(ctx, 1500*time.Millisecond)
				book := c.names(nameCtx, sess, ids)
				cancelNames()
				applyNames(players, book)
				c.rememberNames(book)
			}
		}
		c.applyKnownNames(players)
		if ourColor == "" {
			for _, p := range players {
				if sess.puuid != "" && strings.EqualFold(p.PUUID, sess.puuid) {
					ourColor = p.Team
				}
			}
		}
		if a, e, ok := matchScores(match); ok && a+e >= ally+enemy {
			ally, enemy = a, e
		}
		rounds := ally + enemy
		if phase == "In game" && rounds == 0 && strings.Contains(strings.ToLower(str(pick(match, "State", "state"))), "provision") {
			phase = "Loading lobby"
		}
		for i := range players {
			raw := players[i].Team
			players[i].Enemy = ourColor != "" && !strings.EqualFold(raw, ourColor)
			if sess.puuid != "" && strings.EqualFold(players[i].PUUID, sess.puuid) {
				players[i].You = true
				players[i].Enemy = false
			}
			if players[i].Agent == "" {
				players[i].Agent = "Picking"
			}
		}
		labelSides(players, queue, rounds)
		addCombatStats(match, players)
		if ffaQueue(queue) {
			for i := range players {
				players[i].Enemy = false
			}
		}
		c.decorate(players)
		sort.Slice(players, func(i, j int) bool {
			if players[i].You != players[j].You {
				return players[i].You
			}
			if players[i].Enemy != players[j].Enemy {
				return !players[i].Enemy
			}
			return players[i].Name < players[j].Name
		})
	}
	live := phase == "Agent select" || phase == "In game" || phase == "Loading lobby"
	matchID := ""
	if match != nil {
		matchID = strings.ToLower(str(pick(match, "ID", "id", "MatchID", "matchId")))
	}
	presenceHints := map[string]partyHint{}
	if live && len(players) > 0 {
		presenceHints = c.stampLivePresence(ctx, port, auth, players)
		if matchID != "" {
			for id, hint := range c.restoreRoster(matchID) {
				if _, ok := presenceHints[id]; !ok && hint.id != "" {
					presenceHints[id] = hint
				}
			}
		}
		for i := range players {
			if players[i].Tier >= 3 {
				c.rememberRank(players[i].PUUID, players[i].Tier, players[i].RR, 0)
			}
		}
		if live {
			c.flushRanks()
		}
	}
	c.paintRanks(players)
	c.mu.Lock()
	if phase == "In game" || phase == "Loading lobby" {
		if c.snap.Phase != "In game" && c.snap.Phase != "Loading lobby" {
			c.gameAt = time.Now()
		}
	} else if phase != "Agent select" {
		c.gameAt = time.Time{}
	}
	c.rankQuiet = live
	gameAge := time.Since(c.gameAt)
	c.mu.Unlock()
	rankWait := phase == "Loading lobby" || (phase == "In game" && gameAge < 25*time.Second)
	if live && !rankWait && c.ranksMissing(players) {
		c.kickRanks(sess, players)
	}
	c.mu.Lock()
	recent := append([]recentMatch(nil), c.recent...)
	c.mu.Unlock()
	if live || liveOnly {
		recent = nil
	}
	if !live {
		for i := range recent {
			c.decorate(recent[i].Players)
			c.paintRanks(recent[i].Players)
		}
	}
	if !live && !liveOnly && len(recent) > 0 {
		flat := []player{}
		for i := range recent {
			flat = append(flat, recent[i].Players...)
		}
		c.mu.Lock()
		c.rankQuiet = false
		c.mu.Unlock()
		c.kickRanks(sess, flat)
	}
	if !live && !skipBook {
		c.kickHistory(sess)
	}
	if sess.shard != "" && !live {
		c.kickWatch(sess)
	}
	youSide := ""
	for _, p := range players {
		if p.You {
			youSide = p.Team
		}
	}
	server := ""
	if match != nil {
		server = prettyServer(str(pick(match, "GamePodID", "gamePodId", "GamePodId")))
	}
	if match != nil {
		got := map[string]string{}
		harvestParties(match, got)
		for i := range players {
			id := strings.ToLower(players[i].PUUID)
			if players[i].PartyID == "" && got[id] != "" {
				players[i].PartyID = got[id]
			}
		}
	}
	// Parties for both sides, and your own stack, are filled on every live poll.
	// Pause stat lookups does not skip this.
	hints := map[string]partyHint{}
	if len(players) > 0 {
		roster := map[string]bool{}
		var me string
		for _, p := range players {
			id := strings.ToLower(p.PUUID)
			if looksUUID(id) {
				roster[id] = true
			}
			if p.You {
				me = id
			}
		}
		if locked := c.lockedStacks(matchID); len(locked) > 0 {
			fillGroups(hints, locked)
		}
		fillGroups(hints, c.cachedParties(players))
		fillGroups(hints, c.archiveStacks(players))
		known := map[string]partyHint{}
		if match != nil {
			got := map[string]string{}
			harvestParties(match, got)
			fillGroups(known, hintsFromIDs(got))
		}
		fillGroups(known, partyHintsFrom(players))
		fillGroups(known, presenceHints)
		overlayParties(hints, known)
		mine := ""
		if me != "" {
			if hint, ok := presenceHints[me]; ok {
				mine = hint.id
			} else if hint, ok := hints[me]; ok {
				mine = hint.id
			}
		}
		members := c.myPartyMembers(ctx, port, auth, me)
		if mine != "" {
			if extra := c.localPartyMembers(ctx, port, auth, mine); len(extra) >= 2 {
				members = extra
			}
		}
		if mine == "" && len(members) >= 2 {
			mine = "mine"
		}
		forceParty(hints, mine, members, roster)
		if !c.stacksDone(matchID) && phase != "In game" && phase != "Loading lobby" {
			c.kickStacks(sess, port, auth, matchID, players, presenceHints, mine, me)
		}
		if matchID != "" && partyGrouped(hints) && c.stacksDone(matchID) {
			c.saveLocked(matchID, hints)
		}
	}
	c.rememberParties(groupsOf(hints))
	markParties(players, hints)
	if live && matchID != "" {
		c.saveRoster(matchID, players, hints)
		c.flushRanks()
	}
	c.applySavedParties(recent)
	out := snapshot{
		OK:      true,
		Phase:   phase,
		Map:     prettyMap(mapPath),
		Mode:    prettyMode(queue),
		Queue:   strings.ToLower(queue),
		Server:  server,
		You:     labelYou(sess.name, sess.tag),
		Level:   c.playerLevel(ctx, sess, priv, party),
		Players: players,
		Recent:  recent,
		Party:   []friend{},
		Friends: []friend{},
		Store:   shop{Offers: []storeItem{}, Night: []storeItem{}, Bundles: []bundleView{}, Kingdom: []storeItem{}},
	}
	if out.Map == "Unknown map" && phase == "In menu" {
		out.Map = ""
		out.Mode = ""
	}
	if sideMode(queue) && (phase == "In game" || phase == "Loading lobby" || phase == "Agent select") && len(players) > 0 {
		out.Team1, out.Team2 = "Attackers", "Defenders"
		if youSide == "Attackers" {
			out.Score1, out.Score2 = intPtr(ally), intPtr(enemy)
		} else if youSide == "Defenders" {
			out.Score1, out.Score2 = intPtr(enemy), intPtr(ally)
		} else {
			out.Score1, out.Score2 = intPtr(ally), intPtr(enemy)
			out.Team1, out.Team2 = "Your team", "Other team"
		}
	} else if phase == "In game" || phase == "Loading lobby" {
		out.Score1, out.Score2 = intPtr(ally), intPtr(enemy)
		out.Team1, out.Team2 = "Score", ""
	}
	if out.You == "" {
		for _, p := range players {
			if p.You && p.Name != "" && p.Name != "Player" {
				out.You = labelYou(p.Name, p.Tag)
			}
		}
	}
	out.Party = party
	out.Friends = friends
	out.Store = store
	if live {
		c.mu.Lock()
		out.Penalty = c.penalty
		c.mu.Unlock()
	} else {
		out.Penalty = c.penaltyText(ctx)
	}
	stampSelfLevel(out.Party, out.Level)
	c.mu.Lock()
	out.FriendIDs = append([]string(nil), c.friendIDs...)
	c.mu.Unlock()
	return out
}

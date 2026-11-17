package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func presencePick(priv map[string]any, keys ...string) any {
	if v := pick(priv, keys...); v != nil {
		return v
	}
	if priv == nil {
		return nil
	}
	for _, nest := range []string{"playerPresenceData", "PlayerPresenceData", "private", "data", "dat", "body"} {
		if m := asMap(priv[nest]); m != nil {
			if v := pick(m, keys...); v != nil {
				return v
			}
		}
	}
	return nil
}

func cardArt(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if strings.HasPrefix(id, "http") {
		return id
	}
	if len(id) < 32 {
		return ""
	}
	return "https://media.valorant-api.com/playercards/" + id + "/wideart.png"
}

func activity(priv map[string]any, availability, product string) (state, mode, mapName, score string, idle, owner bool, size, max, tier, level int, card string) {
	if priv == nil {
		if availability == "away" || availability == "idle" {
			return "AFK", "", "", "", true, false, 0, 0, 0, 0, ""
		}
		return "Online", "", "", "", false, false, 0, 0, 0, 0, ""
	}
	loop := strings.ToUpper(str(presencePick(priv, "sessionLoopState", "SessionLoopState", "session_loop_state")))
	if loop == "" {
		loop = strings.ToUpper(str(presencePick(priv, "partyOwnerSessionLoopState", "PartyOwnerSessionLoopState")))
	}
	partyState := strings.ToUpper(str(presencePick(priv, "partyState", "PartyState", "party_state")))
	switch {
	case partyState == "MATCHMAKING" || loop == "MATCHMAKING":
		state = "In queue"
	case loop == "PREGAME":
		state = "Agent select"
	case loop == "INGAME":
		state = "In game"
	case loop == "MENUS":
		state = "In lobby"
	default:
		state = "Online"
	}
	idle, _ = presencePick(priv, "isIdle", "IsIdle").(bool)
	if (availability == "away" || availability == "idle" || idle) && state != "In game" && state != "Agent select" && state != "In queue" {
		state = "AFK"
	}
	mode = prettyMode(str(presencePick(priv, "queueId", "queueID", "QueueID", "partyOwnerQueueId", "PartyOwnerQueueId")))
	if mode == "Match" {
		flow := strings.ToLower(str(presencePick(priv, "partyOwnerProvisioningFlow", "PartyOwnerProvisioningFlow", "provisioningFlow", "ProvisioningFlow")))
		custom := strings.TrimSpace(str(presencePick(priv, "customGameName", "CustomGameName")))
		if strings.Contains(flow, "custom") || strings.Contains(partyState, "CUSTOM") || custom != "" {
			mode = "Custom"
		} else {
			mode = ""
		}
	}
	if state == "In game" && mode == "" {
		appLog("no queue in presence")
	}
	mapPath := str(presencePick(priv, "matchMap", "MatchMap"))
	if mapPath == "" {
		mapPath = str(presencePick(priv, "partyOwnerMatchMap", "PartyOwnerMatchMap"))
	}
	mapName = prettyMap(mapPath)
	if mapName == "" || mapName == "Unknown map" {
		mapName = ""
	}
	if state == "In game" || state == "Agent select" {
		ally := num(presencePick(priv, "partyOwnerMatchScoreAllyTeam", "PartyOwnerMatchScoreAllyTeam"))
		enemy := num(presencePick(priv, "partyOwnerMatchScoreEnemyTeam", "PartyOwnerMatchScoreEnemyTeam"))
		if ally > 0 || enemy > 0 {
			score = fmt.Sprintf("%d–%d", ally, enemy)
		}
	} else {
		mapName = ""
	}
	owner, _ = presencePick(priv, "isPartyOwner", "IsPartyOwner").(bool)
	size = num(presencePick(priv, "partySize", "PartySize"))
	max = num(presencePick(priv, "maxPartySize", "MaxPartySize"))
	tier = num(presencePick(priv, "competitiveTier", "CompetitiveTier"))
	level = num(presencePick(priv, "accountLevel", "AccountLevel"))
	card = cardArt(str(presencePick(priv, "playerCardId", "PlayerCardID", "playerCardID")))
	return
}

func leagueProduct(product string) bool {
	p := strings.ToLower(strings.TrimSpace(product))
	switch p {
	case "league_of_legends", "league of legends", "leagueclient", "leagueclientux", "lol", "lion", "tft", "teamfight_tactics", "teamfight tactics":
		return true
	default:
		return false
	}
}

func leagueClient(p map[string]any) bool {
	if p == nil {
		return false
	}
	state := strings.ToLower(str(pick(p, "state", "show")))
	product := strings.ToLower(strings.TrimSpace(str(pick(p, "product", "productName", "Product", "resource", "Resource"))))
	if state == "mobile" || state == "offline" || strings.Contains(product, "mobile") || strings.Contains(product, "wild") {
		return false
	}
	if product == "valorant" || strings.Contains(product, "valorant") {
		return false
	}
	if leagueProduct(product) || strings.Contains(product, "league") || product == "lion" || product == "lol" || strings.Contains(product, "teamfight") || product == "tft" {
		return true
	}
	blob := strings.ToLower(str(pick(p, "pid")) + " " + str(pick(p, "summary")) + " " + str(pick(p, "basic")) + " " + str(pick(p, "details")) + " " + str(pick(p, "msg")))
	if strings.Contains(blob, "lol.pvp") || strings.Contains(blob, "@lol") || strings.Contains(blob, "league of legends") || strings.Contains(blob, "league_of_legends") || strings.Contains(blob, "teamfight") {
		return true
	}
	priv := presenceBlob(p)
	if priv != nil && str(presencePick(priv, "sessionLoopState", "SessionLoopState", "queueId", "queueID", "QueueID")) == "" && leagueField(priv) != nil {
		return true
	}
	return false
}

func riotMobilePresence(p map[string]any) bool {
	if p == nil {
		return false
	}
	chat := strings.ToLower(strings.TrimSpace(str(pick(p, "state", "show"))))
	if chat == "mobile" {
		return true
	}
	plat := strings.ToLower(strings.TrimSpace(str(pick(p, "platform", "Platform"))))
	switch plat {
	case "android", "ios", "iphone", "ipad", "mobile":
		return true
	}
	blob := strings.ToLower(strings.Join([]string{
		str(pick(p, "product", "productName", "Product")),
		str(pick(p, "patchline", "patchLine", "Patchline")),
		str(pick(p, "resource", "Resource")),
		str(pick(p, "pid")),
		plat,
	}, " "))
	return strings.Contains(blob, "mobile")
}

func presenceBucket(p map[string]any) (string, string) {
	if riotMobilePresence(p) {
		return "riot", "Mobile"
	}
	product := strings.ToLower(str(pick(p, "product", "productName", "Product")))
	chat := strings.ToLower(str(pick(p, "state", "show")))
	switch {
	case product == "valorant" || (valorantPresence(p) && !leagueClient(p)):
		return "valorant", ""
	case leagueClient(p):
		return "league", leagueStatus(p)
	default:
		switch {
		case chat == "away" || chat == "idle":
			return "riot", "AFK"
		default:
			return "riot", "In Riot"
		}
	}
}

func flattenPresence(p map[string]any) []map[string]any {
	games := asMap(pick(p, "games", "Games"))
	if games == nil {
		return []map[string]any{p}
	}
	out := []map[string]any{}
	for name, raw := range games {
		child := asMap(raw)
		if child == nil {
			continue
		}
		next := map[string]any{}
		for key, value := range p {
			if key == "games" || key == "Games" {
				continue
			}
			next[key] = value
		}
		for key, value := range child {
			next[key] = value
		}
		if str(pick(next, "product", "productName", "Product")) == "" {
			next["product"] = name
		}
		out = append(out, next)
	}
	if len(out) == 0 {
		return []map[string]any{p}
	}
	return out
}

func classifyPresences(rows []map[string]any) (string, string, map[string]any) {
	type hit struct {
		p     map[string]any
		label string
	}
	var league, val, riot, mobile *hit
	keep := func(cur *hit, p map[string]any, label string) *hit {
		if cur == nil || presenceRank(p) >= presenceRank(cur.p) {
			return &hit{p: p, label: label}
		}
		return cur
	}
	for _, p := range rows {
		if p == nil {
			continue
		}
		if strings.ToLower(str(pick(p, "state", "show"))) == "offline" {
			continue
		}
		bucket, label := presenceBucket(p)
		switch bucket {
		case "league":
			league = keep(league, p, label)
		case "valorant":
			val = keep(val, p, label)
		default:
			if label == "Mobile" {
				mobile = keep(mobile, p, label)
			} else {
				riot = keep(riot, p, label)
			}
		}
	}
	switch {
	case league != nil:
		return "league", league.label, league.p
	case val != nil:
		return "valorant", "", val.p
	case riot != nil:
		return "riot", riot.label, riot.p
	case mobile != nil:
		return "riot", "Mobile", mobile.p
	}
	if len(rows) > 0 && rows[0] != nil {
		return "riot", "In Riot", rows[0]
	}
	return "riot", "In Riot", map[string]any{}
}

func presenceBlob(p map[string]any) map[string]any {
	for _, key := range []string{"private", "Private", "privateJwt", "private_jwt"} {
		if blob := decodePrivate(str(pick(p, key))); blob != nil {
			return blob
		}
	}
	return nil
}

func leagueField(priv map[string]any) any {
	if priv == nil {
		return nil
	}
	if v := presencePick(priv, "gameStatus", "GameStatus", "championId", "ChampionId", "gameQueueType", "GameQueueType", "rankedLeagueTier", "RankedLeagueTier", "skinname", "SkinName"); v != nil {
		return v
	}
	for _, nest := range []string{"dat", "data", "body", "player", "private"} {
		if m := asMap(priv[nest]); m != nil {
			if v := presencePick(m, "gameStatus", "GameStatus", "championId", "ChampionId", "gameQueueType", "rankedLeagueTier"); v != nil {
				return v
			}
		}
	}
	return nil
}

func presenceState(priv map[string]any) string {
	state, _, _, _, _, _, _, _, _, _, _ := activity(priv, "", "valorant")
	return state
}

func friendFromPresence(p map[string]any, self string, icons map[int]string) (friend, string) {
	priv := decodePrivate(str(pick(p, "private")))
	state, mode, mapName, score, idle, owner, size, max, tier, level, card := activity(priv, strings.ToLower(str(pick(p, "state"))), str(pick(p, "product")))
	puuid := str(pick(p, "puuid"))
	person := friend{
		Name:      str(pick(p, "game_name", "gameName")),
		Tag:       str(pick(p, "game_tag", "gameTag")),
		PUUID:     puuid,
		State:     state,
		You:       self != "" && strings.EqualFold(puuid, self),
		Owner:     owner,
		Idle:      idle,
		Mode:      mode,
		Map:       mapName,
		Score:     score,
		Level:     level,
		Card:      card,
		PartySize: size,
		MaxParty:  max,
		Product:   strings.ToLower(str(pick(p, "product"))),
	}
	if tier > 0 {
		person.Rank = rankName(tier)
		person.RankIcon = icons[tier]
	}
	return person, str(pick(priv, "partyId", "partyID"))
}

func betterPresence(a, b map[string]any) map[string]any {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	if presenceRank(b) > presenceRank(a) {
		return b
	}
	return a
}

func presenceRank(p map[string]any) int {
	state := strings.ToLower(str(pick(p, "state")))
	product := strings.ToLower(str(pick(p, "product")))
	score := 0
	if state != "offline" && state != "" {
		score = 20
	}
	switch {
	case product == "valorant" || product == "":
		score += 5
	case leagueProduct(product):
		if leagueClient(p) {
			score += 8
		}
	default:
		score++
	}
	return score
}

func (c *riotClient) friendSet(ctx context.Context, port, auth string) map[string]bool {
	for _, path := range []string{"/chat/v4/friends", "/chat/v5/friends", "/chat/v6/friends"} {
		body, err := c.localJSON(ctx, "GET", "https://127.0.0.1:"+port+path, auth, nil)
		if err != nil || pick(body, "friends", "Friends") == nil {
			continue
		}
		ids := map[string]bool{}
		notes := map[string]friendNote{}
		for _, item := range asArr(pick(body, "friends", "Friends")) {
			row := asMap(item)
			id := str(pick(row, "puuid"))
			if !looksUUID(id) {
				pid := str(pick(row, "pid"))
				if i := strings.Index(pid, "@"); i > 0 {
					id = pid[:i]
				}
			}
			id = strings.ToLower(id)
			if !looksUUID(id) {
				continue
			}
			ids[id] = true
			notes[id] = friendNote{
				PUUID: id,
				Name:  str(pick(row, "game_name", "gameName", "name")),
				Tag:   str(pick(row, "game_tag", "gameTag", "tagLine")),
			}
		}
		c.observeFriends(notes)
		c.mu.Lock()
		c.friendNames = notes
		c.mu.Unlock()
		return ids
	}
	return nil
}

func (c *riotClient) gatherPresences(ctx context.Context, port, auth string) []map[string]any {
	paths := []string{"/chat/v4/presences", "/chat/v5/presences", "/chat/v6/presences"}
	var mu sync.Mutex
	var out []map[string]any
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			call, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
			defer cancel()
			body, err := c.localJSON(call, "GET", "https://127.0.0.1:"+port+path, auth, nil)
			if err != nil {
				return
			}
			mu.Lock()
			for _, item := range asArr(pick(body, "presences", "Presences")) {
				if p := asMap(item); p != nil {
					out = append(out, flattenPresence(p)...)
				}
			}
			mu.Unlock()
		}(path)
	}
	wg.Wait()
	return out
}

func (c *riotClient) lobby(ctx context.Context, port, auth string, sess riotSession) ([]friend, []friend) {
	grouped := map[string][]map[string]any{}
	for _, p := range c.gatherPresences(ctx, port, auth) {
		id := strings.ToLower(str(pick(p, "puuid", "Puuid", "PUUID", "subject", "Subject")))
		if !looksUUID(id) {
			continue
		}
		grouped[id] = append(grouped[id], p)
	}
	allowed := c.friendSet(ctx, port, auth)
	if allowed == nil {
		allowed = map[string]bool{}
		for id := range grouped {
			allowed[id] = true
		}
	}
	if sess.puuid != "" {
		allowed[strings.ToLower(sess.puuid)] = true
	}
	type row struct {
		friend
		party string
	}
	rows := []row{}
	selfParty := ""
	seen := map[string]bool{}
	for id := range allowed {
		if seen[id] {
			continue
		}
		seen[id] = true
		list := grouped[id]
		if len(list) == 0 {
			continue
		}
		bucket, label, p := classifyPresences(list)
		if p == nil {
			continue
		}
		if strings.ToLower(str(pick(p, "state", "show"))) == "offline" && !strings.EqualFold(id, sess.puuid) {
			continue
		}
		person, party := friendFromPresence(p, sess.puuid, c.rankIcons)
		switch bucket {
		case "league":
			person.Product = "league"
			person.State = label
			person.Mode = ""
			person.Map = ""
			person.Score = ""
			person.Card = ""
			size, max := leagueParty(presenceBlob(p))
			person.PartySize = size
			person.MaxParty = max
			party = ""
			prod := strings.ToLower(str(pick(p, "product", "productName", "Product")))
			if label == "In game" || label == "In queue" {
				person.Mode = leagueMode(presenceBlob(p), prod)
			}
		case "riot":
			person.Product = "riot"
			person.Card = ""
			person.Map = ""
			person.Score = ""
			person.Mode = ""
			person.PartySize = 0
			party = ""
			if person.Idle && label == "In Riot" {
				label = "AFK"
			}
			person.State = label
		default:
			person.Product = "valorant"
		}
		if person.You {
			selfParty = party
			if person.Name == "" {
				person.Name = sess.name
				person.Tag = sess.tag
			}
		}
		if person.Name == "" && person.PUUID == "" {
			continue
		}
		rows = append(rows, row{friend: person, party: party})
	}
	c.mu.Lock()
	namedFriends := c.friendNames
	c.mu.Unlock()
	have := map[string]bool{}
	for _, person := range rows {
		have[strings.ToLower(person.PUUID)] = true
	}
	for id, note := range namedFriends {
		if have[id] || strings.EqualFold(id, sess.puuid) || strings.TrimSpace(note.Name) == "" {
			continue
		}
		rows = append(rows, row{friend: friend{
			Name:    note.Name,
			Tag:     note.Tag,
			PUUID:   id,
			State:   "Offline",
			Product: "riot",
		}})
	}
	if selfParty != "" {
		if extra := c.partyRoster(ctx, sess, selfParty); len(extra) > 0 {
			by := map[string]friend{}
			for _, person := range rows {
				by[strings.ToLower(person.PUUID)] = person.friend
			}
			seen := map[string]bool{}
			for i := range extra {
				id := strings.ToLower(extra[i].PUUID)
				seen[id] = true
				prev, ok := by[id]
				if !ok {
					continue
				}
				extra[i].State = prev.State
				extra[i].Mode = prev.Mode
				extra[i].Map = prev.Map
				extra[i].Score = prev.Score
				extra[i].Idle = prev.Idle
				extra[i].Owner = extra[i].Owner || prev.Owner
				if prev.PartySize > 0 {
					extra[i].PartySize = prev.PartySize
				}
				if prev.MaxParty > 0 {
					extra[i].MaxParty = prev.MaxParty
				}
				if extra[i].Rank == "" {
					extra[i].Rank = prev.Rank
					extra[i].RankIcon = prev.RankIcon
				}
				if extra[i].Level == 0 {
					extra[i].Level = prev.Level
				}
				if extra[i].Card == "" {
					extra[i].Card = prev.Card
				}
				if extra[i].Name == "" || extra[i].Name == "Player" {
					extra[i].Name, extra[i].Tag = prev.Name, prev.Tag
				}
			}
			for _, person := range rows {
				id := strings.ToLower(person.PUUID)
				if person.party == selfParty && !seen[id] {
					extra = append(extra, person.friend)
				}
			}
			online := []friend{}
			for _, person := range rows {
				if person.party == selfParty || person.You {
					continue
				}
				online = append(online, person.friend)
			}
			sortFriends(extra)
			sortFriends(online)
			c.applyCardArt(extra)
			c.applyCardArt(online)
			return extra, online
		}
	}
	party := []friend{}
	online := []friend{}
	for _, person := range rows {
		if person.You || (selfParty != "" && person.party == selfParty) {
			party = append(party, person.friend)
			continue
		}
		online = append(online, person.friend)
	}
	sortFriends(party)
	sortFriends(online)
	c.applyCardArt(party)
	c.applyCardArt(online)
	return party, online
}

func sortFriends(people []friend) {
	rank := func(state string) int {
		switch state {
		case "In game":
			return 0
		case "Agent select":
			return 1
		case "In queue":
			return 2
		case "In lobby":
			return 3
		case "Away":
			return 5
		default:
			return 4
		}
	}
	better := func(a, b friend) bool {
		if a.You != b.You {
			return a.You
		}
		if rank(a.State) != rank(b.State) {
			return rank(a.State) < rank(b.State)
		}
		return a.Name < b.Name
	}
	for i := 1; i < len(people); i++ {
		for j := i; j > 0 && better(people[j], people[j-1]); j-- {
			people[j], people[j-1] = people[j-1], people[j]
		}
	}
}

func (c *riotClient) ensureCards(ctx context.Context) {
	c.mu.Lock()
	fresh := len(c.cards) > 0 && time.Since(c.cardsAt) < 12*time.Hour
	c.mu.Unlock()
	if fresh {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/playercards", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return
	}
	next := map[string]string{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "uuid")))
		art := str(pick(row, "wideArt"))
		if art == "" {
			art = str(pick(row, "smallArt"))
		}
		if id != "" && art != "" {
			next[id] = art
		}
	}
	if len(next) == 0 {
		return
	}
	c.mu.Lock()
	c.cards = next
	c.cardsAt = time.Now()
	c.mu.Unlock()
}

func (c *riotClient) applyCardArt(people []friend) {
	for i := range people {
		people[i].Card = cardArt(people[i].Card)
	}
}

func (c *riotClient) partyRoster(ctx context.Context, sess riotSession, partyID string) []friend {
	if sess.region == "" || sess.shard == "" || partyID == "" {
		return nil
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://glz-"+sess.region+"-1."+sess.shard+".a.pvp.net/parties/v1/parties/"+partyID, nil)
	if err != nil || status >= 400 {
		return nil
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	members := asArr(pick(body, "Members", "members"))
	out := make([]friend, 0, len(members))
	missing := []string{}
	for _, item := range members {
		m := asMap(item)
		id := str(pick(m, "Subject", "subject"))
		name := str(pick(m, "GameName", "gameName"))
		tag := str(pick(m, "TagLine", "tagLine"))
		if name == "" {
			missing = append(missing, id)
		}
		idty := asMap(pick(m, "PlayerIdentity", "playerIdentity"))
		tier := num(pick(m, "CompetitiveTier", "competitiveTier"))
		person := friend{
			Name:  name,
			Tag:   tag,
			PUUID: id,
			State: "In lobby",
			You:   sess.puuid != "" && strings.EqualFold(id, sess.puuid),
			Owner: func() bool { b, _ := pick(m, "IsOwner", "isOwner").(bool); return b }(),
			Level: num(pick(idty, "AccountLevel", "accountLevel")),
			Card:  cardArt(str(pick(idty, "PlayerCardID", "playerCardId"))),
		}
		if tier > 0 {
			person.Rank = rankName(tier)
			person.RankIcon = c.rankIcons[tier]
		}
		out = append(out, person)
	}
	size := num(pick(body, "PartySize", "partySize"))
	if size == 0 {
		size = len(out)
	}
	max := num(pick(body, "MaxPartySize", "maxPartySize"))
	if max == 0 {
		max = 5
	}
	for i := range out {
		out[i].PartySize = size
		out[i].MaxParty = max
	}
	if book := c.names(ctx, sess, missing); book != nil {
		for i := range out {
			if pair, ok := book[strings.ToLower(out[i].PUUID)]; ok && out[i].Name == "" {
				out[i].Name, out[i].Tag = pair[0], pair[1]
			}
		}
	}
	for i := range out {
		if out[i].Name == "" {
			out[i].Name = "Player"
		}
	}
	return out
}

func leagueMode(priv map[string]any, product string) string {
	prod := strings.ToLower(product)
	tft := strings.Contains(prod, "tft") || strings.Contains(prod, "teamfight")
	if priv == nil {
		if tft {
			return "TFT"
		}
		return ""
	}
	raw := strings.TrimSpace(str(presencePick(priv, "gameQueueType", "GameQueueType", "queueType", "QueueType")))
	if raw == "" {
		raw = strings.TrimSpace(str(presencePick(priv, "queueId", "queueID", "QueueID")))
	}
	if raw == "" {
		raw = strings.TrimSpace(str(presencePick(priv, "gameMode", "GameMode")))
	}
	mode := prettyLeague(raw)
	if mode == "" && tft {
		return "TFT"
	}
	return mode
}

func prettyLeague(queue string) string {
	key := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(queue), " ", "_"))
	switch key {
	case "", "NONE", "0", "INVALID", "CLASSIC", "MATCHED", "PRACTICETOOL", "PRACTICE_TOOL":
		return ""
	case "RANKED_SOLO_5X5", "RANKED_SOLO", "420":
		return "Ranked Solo"
	case "RANKED_FLEX_SR", "RANKED_FLEX", "440":
		return "Ranked Flex"
	case "NORMAL", "NORMAL_DRAFT", "NORMAL_DRAFT_PICK", "400":
		return "Draft"
	case "NORMAL_BLIND", "NORMAL_BLIND_PICK", "430":
		return "Blind"
	case "ARAM", "450", "ARAM_UNRANKED_5X5":
		return "ARAM"
	case "URF", "900", "1900", "ARURF", "ARURF_DOMINION":
		return "URF"
	case "CHERRY", "ARENA", "1700":
		return "Arena"
	case "CLASH", "700":
		return "Clash"
	case "NEXUS_BLITZ", "1300", "2300":
		return "Nexus Blitz"
	case "ONE_FOR_ALL", "1020":
		return "One for All"
	case "ULTBOOK", "ULTIMATE_SPELLBOOK", "1400":
		return "Ultimate Spellbook"
	case "SWIFTPLAY", "480":
		return "Swiftplay"
	case "QUICKPLAY", "490":
		return "Quickplay"
	case "RANKED_TFT", "RANKED_TFT_TURBO", "1100":
		return "TFT Ranked"
	case "RANKED_TFT_DOUBLE_UP", "1160":
		return "TFT Double Up"
	case "NORMAL_TFT", "1090", "TFT", "TEAMFIGHT_TACTICS":
		return "TFT"
	case "HYPER_ROLL", "TFT_HYPER_ROLL", "1130":
		return "TFT Hyper Roll"
	default:
		if strings.Contains(key, "TFT") || strings.Contains(key, "TEAMFIGHT") {
			if strings.Contains(key, "DOUBLE") {
				return "TFT Double Up"
			}
			if strings.Contains(key, "RANKED") {
				return "TFT Ranked"
			}
			if strings.Contains(key, "HYPER") {
				return "TFT Hyper Roll"
			}
			return "TFT"
		}
		if _, err := strconv.Atoi(key); err == nil {
			return ""
		}
		parts := strings.Fields(strings.ReplaceAll(strings.ToLower(key), "_", " "))
		for i, part := range parts {
			if part == "" {
				continue
			}
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
		return strings.Join(parts, " ")
	}
}

func leagueStatus(p map[string]any) string {
	chat := strings.ToLower(str(pick(p, "state")))
	priv := presenceBlob(p)
	idle, _ := presencePick(priv, "isIdle", "IsIdle").(bool)
	if chat == "away" || idle {
		return "AFK"
	}
	status := strings.ToLower(str(leagueField(priv)))
	if status == "" {
		status = strings.ToLower(str(presencePick(priv, "gameStatus", "GameStatus")))
	}
	switch status {
	case "ingame", "in_game", "championselect", "champion_select", "hosting", "gamestart", "spectating", "inchampionselect":
		return "In game"
	case "inqueue", "in_queue", "checkedin", "searching":
		return "In queue"
	}
	if chat == "dnd" {
		return "Online"
	}
	return "Online"
}

func leagueParty(priv map[string]any) (int, int) {
	if priv == nil {
		return 0, 0
	}
	size := num(presencePick(priv, "partySize", "PartySize", "summonerCount"))
	max := num(presencePick(priv, "maxPartySize", "MaxPartySize"))
	raw := str(presencePick(priv, "pty", "party"))
	if raw != "" && strings.HasPrefix(strings.TrimSpace(raw), "{") {
		var body map[string]any
		if json.Unmarshal([]byte(raw), &body) == nil {
			if n := len(asArr(pick(body, "summoners", "members", "Members"))); n > size {
				size = n
			}
		}
	}
	if size > 0 && max == 0 {
		max = 5
	}
	if size < 0 {
		size = 0
	}
	return size, max
}

func valorantPresence(p map[string]any) bool {
	if strings.EqualFold(str(pick(p, "product")), "valorant") {
		return true
	}
	priv := decodePrivate(str(pick(p, "private")))
	return str(presencePick(priv, "sessionLoopState", "SessionLoopState")) != "" || str(presencePick(priv, "queueId", "queueID", "QueueID")) != ""
}

func vpCost(cost map[string]any) int {
	for key, value := range cost {
		if strings.EqualFold(key, "85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741") {
			return num(value)
		}
	}
	return 0
}

func kcCost(cost map[string]any) int {
	for key, value := range cost {
		if strings.EqualFold(key, "85ca954a-41f2-ce94-9b45-8ca3dd39a00d") {
			return num(value)
		}
	}
	return 0
}

func resetLabel(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	hours := seconds / 3600
	mins := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dm", mins)
}

func nightEndsAt(seconds int) int64 {
	if seconds <= 0 {
		return 0
	}
	return time.Now().Add(time.Duration(seconds) * time.Second).Unix()
}

func (c *riotClient) ensureSkins(ctx context.Context) {
	if len(c.skins) > 0 && time.Since(c.skinsAt) < 12*time.Hour {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/weapons/skinlevels", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return
	}
	next := map[string]agentInfo{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "uuid")))
		if id == "" {
			continue
		}
		next[id] = agentInfo{Name: str(pick(row, "displayName")), Icon: str(pick(row, "displayIcon"))}
	}
	if len(next) > 0 {
		c.mu.Lock()
		c.skins = next
		c.skinsAt = time.Now()
		c.mu.Unlock()
	}
}

func offerID(offer map[string]any) string {
	for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
		id := strings.ToLower(str(pick(asMap(reward), "ItemID", "ItemId", "itemId")))
		if id != "" {
			return id
		}
	}
	return strings.ToLower(str(pick(offer, "OfferID", "offerId")))
}

func skinArt(id string, info agentInfo) (string, string) {
	if info.Name == "" {
		return "", ""
	}
	return info.Name, info.Icon
}

func (c *riotClient) storeOffers(panel map[string]any) []storeItem {
	prices := map[string]int{}
	for _, item := range asArr(pick(panel, "SingleItemStoreOffers", "singleItemStoreOffers")) {
		offer := asMap(item)
		id := offerID(offer)
		if id != "" {
			prices[id] = vpCost(asMap(pick(offer, "Cost", "cost")))
		}
	}
	offers := []storeItem{}
	seen := map[string]bool{}
	add := func(id string, price int) {
		if id == "" || seen[id] {
			return
		}
		name, icon := skinArt(id, c.skins[id])
		if name == "" {
			return
		}
		if price == 0 {
			price = prices[id]
		}
		seen[id] = true
		offers = append(offers, storeItem{Name: name, Icon: icon, Price: price})
	}
	for _, item := range asArr(pick(panel, "SingleItemOffers", "singleItemOffers")) {
		switch row := item.(type) {
		case string:
			add(strings.ToLower(row), 0)
		default:
			offer := asMap(item)
			add(offerID(offer), vpCost(asMap(pick(offer, "Cost", "cost"))))
		}
	}
	if len(offers) == 0 {
		for id, price := range prices {
			add(id, price)
			if len(offers) >= 4 {
				break
			}
		}
	}
	return offers
}

func (c *riotClient) itemLook(id string) (string, string) {
	id = strings.ToLower(id)
	if id == "" {
		return "", ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if info, ok := c.bundleArt[id]; ok && info.Name != "" {
		icon := info.Icon
		if icon == "" {
			icon = "https://media.valorant-api.com/bundles/" + id + "/displayicon.png"
		}
		return info.Name, icon
	}
	if info, ok := c.skins[id]; ok && info.Name != "" {
		return info.Name, info.Icon
	}
	if info, ok := c.skinLook[id]; ok && info.Name != "" {
		return info.Name, info.Icon
	}
	if info, ok := c.buddies[id]; ok && info.Name != "" {
		icon := info.Icon
		if icon == "" {
			icon = "https://media.valorant-api.com/buddylevels/" + id + "/displayicon.png"
		}
		return info.Name, icon
	}
	if info, ok := c.flair[id]; ok && info.Name != "" {
		icon := info.Icon
		if icon == "" {
			icon = info.Anim
		}
		return info.Name, icon
	}
	if info, ok := c.rewards[id]; ok && info.Name != "" {
		icon := info.Icon
		if info.Anim != "" {
			icon = info.Anim
		}
		return info.Name, icon
	}
	return "", ""
}

func (c *riotClient) kingdomOffers(body map[string]any) ([]storeItem, string) {
	acc := asMap(pick(body, "AccessoryStore", "accessoryStore"))
	rows := asArr(pick(acc, "AccessoryStoreOffers", "accessoryStoreOffers"))
	left := resetLabel(num(pick(acc, "AccessoryStoreRemainingDurationInSeconds", "accessoryStoreRemainingDurationInSeconds")))
	offers := []storeItem{}
	seen := map[string]bool{}
	add := func(row map[string]any) {
		offer := asMap(pick(row, "Offer", "offer"))
		if len(offer) == 0 {
			offer = row
		}
		id := offerID(offer)
		if id == "" || seen[id] {
			return
		}
		price := kcCost(asMap(pick(offer, "Cost", "cost")))
		if price == 0 {
			price = kcCost(asMap(pick(row, "Cost", "cost")))
		}
		name, icon := "", ""
		for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
			rid := strings.ToLower(str(pick(asMap(reward), "ItemID", "ItemId", "itemId")))
			if rid == "" {
				continue
			}
			n, ic := c.itemLook(rid)
			if n != "" {
				name, icon = n, ic
				break
			}
		}
		if name == "" {
			name, icon = c.itemLook(id)
		}
		if name == "" && price == 0 {
			return
		}
		if name == "" {
			name = "Kingdom offer"
		}
		seen[id] = true
		offers = append(offers, storeItem{Name: name, Icon: icon, Price: price, Currency: "KC", Offer: strings.ToLower(str(pick(offer, "OfferID", "offerId")))})
	}
	for _, item := range rows {
		add(asMap(item))
	}
	if offers == nil {
		offers = []storeItem{}
	}
	return offers, left
}

func skinWishName(name string) string {
	name = strings.TrimSpace(name)
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "standard") || strings.HasPrefix(lower, "random") || name == "" {
		return ""
	}
	if i := strings.Index(lower, " level "); i > 0 {
		if _, err := strconv.Atoi(strings.TrimSpace(lower[i+7:])); err == nil {
			return strings.TrimSpace(name[:i])
		}
	}
	return name
}

func (c *riotClient) kcCatalog(ctx context.Context) []storeItem {
	c.ensureCosmetics(ctx)
	c.ensureSkins(ctx)
	c.mu.Lock()
	buddies := c.buddies
	flair := c.flair
	skins := c.skins
	c.mu.Unlock()
	seen := map[string]bool{}
	out := []storeItem{}
	add := func(name, icon, kind string) {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || icon == "" || seen[key] {
			return
		}
		seen[key] = true
		out = append(out, storeItem{Name: name, Icon: icon, Currency: kind})
	}
	for _, info := range skins {
		add(skinWishName(info.Name), info.Icon, "Skin")
	}
	for _, info := range buddies {
		add(info.Name, info.Icon, "KC")
	}
	for _, info := range flair {
		if info.Kind == "Title" {
			continue
		}
		add(info.Name, info.Icon, "KC")
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func (c *riotClient) dailyStore(ctx context.Context, sess riotSession) shop {
	c.mu.Lock()
	cached := c.shop
	hasOffers := len(cached.Offers) > 0
	hasBundles := len(cached.Bundles) > 0
	age := time.Since(c.shopAt)
	c.mu.Unlock()
	keep := func(msg string) shop {
		if !hasOffers && !hasBundles {
			return shop{Offers: []storeItem{}, Night: []storeItem{}, Bundles: []bundleView{}, Kingdom: []storeItem{}, Error: msg}
		}
		out := cached
		if out.Offers == nil {
			out.Offers = []storeItem{}
		}
		if out.Night == nil {
			out.Night = []storeItem{}
		}
		if out.Bundles == nil {
			out.Bundles = []bundleView{}
		}
		if out.Kingdom == nil {
			out.Kingdom = []storeItem{}
		}
		if msg != "" {
			out.Error = msg
		}
		return out
	}
	if hasOffers && age < 10*time.Minute && (hasBundles || age < 40*time.Second) {
		return cached
	}
	if sess.shard == "" || sess.puuid == "" || sess.access == "" {
		return keep("Open Valorant, then open the store tab again.")
	}
	c.ensureSkins(ctx)
	status, payload, err := c.remoteCall(ctx, sess, "POST", "https://pd."+sess.shard+".a.pvp.net/store/v3/storefront/"+sess.puuid, map[string]any{})
	if err != nil || status == 404 || status == 405 {
		status, payload, err = c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v2/storefront/"+sess.puuid, nil)
	}
	if err != nil || status >= 400 {
		msg := "Couldn't read the store."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't read the store (%d).", status)
		}
		return keep(msg)
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return keep("The store response was unreadable.")
	}
	panel := asMap(pick(body, "SkinsPanelLayout", "skinsPanelLayout"))
	offers := c.storeOffers(panel)
	night := []storeItem{}
	bonus := asMap(pick(body, "BonusStore", "bonusStore"))
	for _, item := range asArr(pick(bonus, "BonusStoreOffers", "bonusStoreOffers")) {
		row := asMap(item)
		offer := asMap(pick(row, "Offer", "offer"))
		if len(offer) == 0 {
			offer = row
		}
		price := vpCost(asMap(pick(row, "DiscountCosts", "discountCosts")))
		if price == 0 {
			price = vpCost(asMap(pick(offer, "DiscountCosts", "discountCosts")))
		}
		was := vpCost(asMap(pick(offer, "Cost", "cost")))
		id := offerID(offer)
		name, icon := skinArt(id, c.skins[id])
		if name == "" {
			continue
		}
		night = append(night, storeItem{Name: name, Icon: icon, Price: price, Was: was})
	}
	vp, rad, kc := c.wallet(ctx, sess)
	c.ensureBundles(ctx)
	c.ensureCatalog(ctx)
	c.ensureCosmetics(ctx)
	_ = c.rewardBook(ctx)
	kingdom, kingdomLeft := c.kingdomOffers(body)
	bundles := c.featuredBundles(body)
	esports := c.esportsOffers(body)
	if len(bundles) == 0 && hasBundles {
		bundles = append([]bundleView(nil), cached.Bundles...)
	}
	if len(esports) == 0 && len(cached.Esports) > 0 {
		esports = append([]bundleView(nil), cached.Esports...)
	}
	next := shop{
		Offers:      offers,
		Night:       night,
		Kingdom:     kingdom,
		KingdomLeft: kingdomLeft,
		Bundles:     bundles,
		Esports:     esports,
		Resets:      resetLabel(num(pick(panel, "SingleItemOffersRemainingDurationInSeconds"))),
		NightResets: resetLabel(num(pick(bonus, "BonusStoreRemainingDurationInSeconds"))),
		NightEnds:   nightEndsAt(num(pick(bonus, "BonusStoreRemainingDurationInSeconds"))),
		VP:          vp,
		Radianite:   rad,
		KC:          kc,
	}
	if len(next.Offers) == 0 && len(next.Bundles) == 0 && len(next.Esports) == 0 {
		next.Error = "The store came back empty."
		return keep(next.Error)
	}
	if len(next.Offers) == 0 && hasOffers {
		next.Offers = cached.Offers
		if next.Resets == "" {
			next.Resets = cached.Resets
		}
	}
	c.mu.Lock()
	c.shop = next
	c.shopAt = time.Now()
	c.mu.Unlock()
	return next
}

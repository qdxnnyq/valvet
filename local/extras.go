package main

import (
	"bytes"
	"compress/flate"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

type searchHit struct {
	Name    string `json:"name"`
	Tag     string `json:"tag"`
	PUUID   string `json:"puuid"`
	Error   string `json:"error,omitempty"`
	VTL     string `json:"vtl,omitempty"`
	Tracker string `json:"tracker,omitempty"`
}

type ability struct {
	Slot string `json:"slot"`
	Name string `json:"name"`
	Icon string `json:"icon,omitempty"`
	Desc string `json:"desc"`
}

type agentCard struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon,omitempty"`
	Role      string    `json:"role"`
	Desc      string    `json:"desc"`
	Abilities []ability `json:"abilities"`
}

type missionView struct {
	Title    string `json:"title"`
	Kind     string `json:"kind"`
	Progress int    `json:"progress"`
	Goal     int    `json:"goal"`
	Done     bool   `json:"done"`
	XP       int    `json:"xp"`
	Ends     string `json:"ends,omitempty"`
}

type passView struct {
	Name     string        `json:"name"`
	Level    int           `json:"level"`
	XP       int           `json:"xp"`
	Next     int           `json:"next"`
	Icon     string        `json:"icon,omitempty"`
	Free     bool          `json:"free,omitempty"`
	Missions []missionView `json:"missions"`
	Rewards  []passReward  `json:"rewards"`
	Tracks   []passTrack   `json:"tracks,omitempty"`
	Ends     string        `json:"ends,omitempty"`
	Error    string        `json:"error,omitempty"`
}

type passTrack struct {
	Name    string       `json:"name"`
	Kind    string       `json:"kind"`
	Level   int          `json:"level"`
	XP      int          `json:"xp"`
	Next    int          `json:"next"`
	Icon    string       `json:"icon,omitempty"`
	Free    bool         `json:"free,omitempty"`
	Rewards []passReward `json:"rewards"`
}

type passReward struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	Icon  string `json:"icon,omitempty"`
	Anim  string `json:"anim,omitempty"`
	Got   bool   `json:"got"`
	Free  bool   `json:"free,omitempty"`
}

func splitRiotID(q string) (string, string, bool) {
	q = strings.TrimSpace(q)
	i := strings.LastIndex(q, "#")
	if i <= 0 || i == len(q)-1 {
		return "", "", false
	}
	name := strings.TrimSpace(q[:i])
	tag := strings.TrimSpace(q[i+1:])
	if name == "" || tag == "" {
		return "", "", false
	}
	return name, tag, true
}

func shortLevel(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.Index(name, "Level "); i >= 0 {
		rest := name[i:]
		if n := strings.IndexAny(rest, "\n("); n >= 0 {
			rest = rest[:n]
		}
		return strings.TrimSpace(rest)
	}
	return name
}

func shortChroma(name string) string {
	name = strings.TrimSpace(name)
	if i := strings.LastIndex(name, "("); i >= 0 && strings.Contains(name[i:], ")") {
		inner := strings.TrimSpace(strings.Trim(name[i:], "()"))
		if inner != "" {
			return inner
		}
	}
	if name == "" || strings.Contains(strings.ToLower(name), "standard") || strings.Contains(strings.ToLower(name), "base") {
		return ""
	}
	return name
}

func missionKind(kind string) string {
	kind = strings.ToLower(kind)
	switch {
	case strings.Contains(kind, "daily"):
		return "Daily"
	case strings.Contains(kind, "week"):
		return "Weekly"
	default:
		return ""
	}
}

func (c *riotClient) lookupPlayer(ctx context.Context, query string) searchHit {
	name, tag, ok := splitRiotID(query)
	if !ok {
		return searchHit{Error: "Use Name#Tag"}
	}
	if n, t, id := c.knownPlayer(name, tag); id != "" {
		return searchHit{Name: n, Tag: t, PUUID: id}
	}
	port, password, err := readLockfile()
	if err == nil {
		auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("riot:"+password))
		q := url.Values{}
		q.Set("gameName", name)
		q.Set("tagLine", tag)
		paths := []string{
			"/player-account/aliases/v1/lookup?" + q.Encode(),
			"/player-account/aliases/v2/lookup?" + q.Encode(),
			"/riotclient/player-account/aliases/v1/lookup?" + q.Encode(),
		}
		for _, path := range paths {
			status, payload, err := c.do(ctx, c.local, "GET", "https://127.0.0.1:"+port+path, auth, "", nil)
			if err == nil && status < 400 {
				if n, t, id := aliasFrom(payload); id != "" {
					if n == "" {
						n = name
					}
					if t == "" {
						t = tag
					}
					return searchHit{Name: n, Tag: t, PUUID: id}
				}
			}
		}
		body := []map[string]string{{"gameName": name, "tagLine": tag}}
		for _, path := range []string{"/player-account/aliases/v2/lookup", "/player-account/aliases/v1/lookup"} {
			status, payload, err := c.do(ctx, c.local, "PUT", "https://127.0.0.1:"+port+path, auth, "", body)
			if err == nil && status < 400 {
				if n, t, id := aliasFrom(payload); id != "" {
					if n == "" {
						n = name
					}
					if t == "" {
						t = tag
					}
					return searchHit{Name: n, Tag: t, PUUID: id}
				}
			}
		}
	}
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	if sess.access != "" {
		account := "https://api.account.riotgames.com/aliases/v1/lookup?gameName=" + url.QueryEscape(name) + "&tagLine=" + url.QueryEscape(tag)
		if status, payload, err := c.do(ctx, c.remote, "GET", account, "Bearer "+sess.access, "", nil); err == nil && status < 400 {
			if n, t, id := aliasFrom(payload); id != "" {
				if n == "" {
					n = name
				}
				if t == "" {
					t = tag
				}
				return searchHit{Name: n, Tag: t, PUUID: id}
			}
		}
	}
	hit := searchHit{Name: name, Tag: tag, Error: "No local match for " + name + "#" + tag + ". Opened the tracker page."}
	hit.VTL = "https://vtl.lol/"
	hit.Tracker = "https://tracker.gg/valorant/profile/riot/" + url.PathEscape(name+"#"+tag)
	return hit
}

func (c *riotClient) knownPlayer(name, tag string) (string, string, string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	wantN, wantT := strings.ToLower(name), strings.ToLower(tag)
	check := func(n, t, id string) (string, string, string, bool) {
		if id != "" && strings.ToLower(n) == wantN && strings.ToLower(t) == wantT {
			return n, t, id, true
		}
		return "", "", "", false
	}
	for _, person := range c.snap.Friends {
		if n, t, id, ok := check(person.Name, person.Tag, person.PUUID); ok {
			return n, t, id
		}
	}
	for _, person := range c.snap.Party {
		if n, t, id, ok := check(person.Name, person.Tag, person.PUUID); ok {
			return n, t, id
		}
	}
	for _, match := range c.recent {
		for _, person := range match.Players {
			if n, t, id, ok := check(person.Name, person.Tag, person.PUUID); ok {
				return n, t, id
			}
		}
	}
	return "", "", ""
}

func aliasFrom(payload []byte) (string, string, string) {
	var body any
	if json.Unmarshal(payload, &body) != nil {
		return "", "", ""
	}
	row := asMap(body)
	if len(row) == 0 {
		if arr := asArr(body); len(arr) > 0 {
			row = asMap(arr[0])
		}
	}
	if nested := asMap(pick(row, "alias", "Alias")); len(nested) > 0 {
		row = nested
	}
	id := str(pick(row, "puuid", "Puuid", "subject", "Subject"))
	name := str(pick(row, "gameName", "GameName"))
	tag := str(pick(row, "tagLine", "TagLine"))
	return name, tag, id
}

func (c *riotClient) playableAgents(ctx context.Context) []agentCard {
	c.mu.Lock()
	if len(c.agentBook) > 0 && time.Since(c.agentBookAt) < 12*time.Hour {
		out := append([]agentCard(nil), c.agentBook...)
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	var disk []agentCard
	if age, ok := readCache("agents", &disk); ok && len(disk) > 0 {
		c.mu.Lock()
		c.agentBook = disk
		c.agentBookAt = time.Now().Add(-age)
		out := append([]agentCard(nil), disk...)
		c.mu.Unlock()
		if age > 12*time.Hour {
			cacheOnce("agents", func() { c.fetchAgents(context.Background()) })
		}
		return out
	}
	return c.fetchAgents(ctx)
}

func (c *riotClient) fetchAgents(ctx context.Context) []agentCard {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/agents?isPlayableCharacter=true", nil)
	if err != nil {
		return []agentCard{}
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return []agentCard{}
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return []agentCard{}
	}
	out := []agentCard{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		if play, ok := row["isPlayableCharacter"].(bool); ok && !play {
			continue
		}
		role := asMap(pick(row, "role"))
		card := agentCard{
			ID:   strings.ToLower(str(pick(row, "uuid"))),
			Name: str(pick(row, "displayName")),
			Icon: str(pick(row, "displayIcon")),
			Role: str(pick(role, "displayName")),
			Desc: cleanCopy(str(pick(row, "description"))),
		}
		for _, raw := range asArr(pick(row, "abilities")) {
			ab := asMap(raw)
			card.Abilities = append(card.Abilities, ability{
				Slot: str(pick(ab, "slot")),
				Name: str(pick(ab, "displayName")),
				Icon: str(pick(ab, "displayIcon")),
				Desc: cleanCopy(str(pick(ab, "description"))),
			})
		}
		if card.Name != "" {
			out = append(out, card)
		}
	}
	if len(out) > 0 {
		c.mu.Lock()
		c.agentBook = out
		c.agentBookAt = time.Now()
		c.mu.Unlock()
		writeCache("agents", out)
	}
	return out
}

func (c *riotClient) ensureBundles(ctx context.Context) {
	c.mu.Lock()
	fresh := len(c.bundleArt) > 0 && time.Since(c.bundleAt) < 12*time.Hour
	c.mu.Unlock()
	if fresh {
		return
	}
	var disk struct {
		Art   map[string]agentInfo `json:"art"`
		Notes map[string]string    `json:"notes"`
	}
	if age, ok := readCache("bundles", &disk); ok && len(disk.Art) > 0 {
		c.mu.Lock()
		if len(c.bundleArt) == 0 {
			c.bundleArt = disk.Art
			c.bundleNote = disk.Notes
			c.bundleAt = time.Now().Add(-age)
		}
		c.mu.Unlock()
		if age > 12*time.Hour {
			cacheOnce("bundles", func() {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				c.fetchBundles(ctx)
			})
		}
		return
	}
	c.fetchBundles(ctx)
}

func (c *riotClient) fetchBundles(ctx context.Context) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/bundles", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return
	}
	next := map[string]agentInfo{}
	notes := map[string]string{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "uuid")))
		icon := str(pick(row, "displayIcon"))
		if icon == "" {
			icon = str(pick(row, "verticalPromoImage"))
		}
		if id != "" {
			next[id] = agentInfo{Name: str(pick(row, "displayName")), Icon: icon}
			notes[id] = str(pick(row, "description"))
		}
	}
	if len(next) == 0 {
		return
	}
	c.mu.Lock()
	c.bundleArt = next
	c.bundleNote = notes
	c.bundleAt = time.Now()
	c.mu.Unlock()
	writeCache("bundles", map[string]any{"art": next, "notes": notes})
}

func (c *riotClient) featuredBundles(body map[string]any) []bundleView {
	c.mu.Lock()
	art := c.bundleArt
	c.mu.Unlock()
	raw := bundleRows(body)
	out := []bundleView{}
	seen := map[string]bool{}
	for _, item := range raw {
		row := asMap(item)
		if len(row) == 0 {
			continue
		}
		id := strings.ToLower(str(pick(row, "DataAssetID", "dataAssetID", "ID", "id")))
		if id != "" && seen[id] {
			continue
		}
		if id != "" {
			seen[id] = true
		}
		info := art[id]
		name := info.Name
		if name == "" {
			name = str(pick(row, "DisplayName", "displayName", "Name", "name"))
		}
		if name == "" {
			name = "Featured bundle"
		}
		icon := info.Icon
		if icon == "" && id != "" {
			icon = "https://media.valorant-api.com/bundles/" + id + "/displayicon.png"
		}
		price := vpCost(asMap(pick(row, "TotalDiscountedCost", "totalDiscountedCost")))
		was := vpCost(asMap(pick(row, "TotalBaseCost", "totalBaseCost")))
		items := c.bundleItems(row)
		for _, piece := range items {
			if price == 0 {
				price += piece.Price
			}
		}
		left := resetLabel(num(pick(row, "DurationRemainingInSeconds", "durationRemainingInSeconds")))
		if left == "" {
			featured := asMap(pick(body, "FeaturedBundle", "featuredBundle"))
			left = resetLabel(num(pick(featured, "BundleRemainingDurationInSeconds", "bundleRemainingDurationInSeconds")))
		}
		out = append(out, bundleView{Name: name, Icon: icon, Price: price, Was: was, Left: left, Items: items})
	}
	return out
}

func bundleRows(body map[string]any) []any {
	featured := asMap(pick(body, "FeaturedBundle", "featuredBundle"))
	raw := asArr(pick(featured, "Bundles", "bundles"))
	if one := asMap(pick(featured, "Bundle", "bundle")); len(one) > 0 {
		raw = append([]any{one}, raw...)
	}
	if len(asArr(pick(featured, "Items", "items"))) > 0 || str(pick(featured, "DataAssetID", "dataAssetID")) != "" {
		raw = append(raw, featured)
	}
	if len(raw) == 0 {
		if theme := asMap(pick(body, "FeaturedTheme", "featuredTheme")); len(theme) > 0 {
			raw = append(raw, theme)
		}
	}
	if arr := asArr(pick(body, "FeaturedBundle", "featuredBundle")); len(arr) > 0 {
		raw = append(raw, arr...)
	}
	if len(raw) == 0 {
		for _, plugin := range asArr(pick(body, "PluginStores", "pluginStores")) {
			row := asMap(plugin)
			offers := asMap(pick(row, "PluginOffers", "pluginOffers"))
			list := asArr(pick(offers, "StoreOffers", "storeOffers", "Offers", "offers"))
			if len(list) == 0 {
				list = asArr(pick(row, "Offers", "offers"))
			}
			if len(list) == 0 {
				continue
			}
			raw = append(raw, map[string]any{
				"DisplayName":                "Featured",
				"DurationRemainingInSeconds": num(pick(offers, "RemainingDurationInSeconds", "remainingDurationInSeconds")),
				"ItemOffers":                 list,
			})
		}
	}
	return raw
}

func (c *riotClient) bundleItems(row map[string]any) []storeItem {
	items := []storeItem{}
	seen := map[string]bool{}
	add := func(id, typeID string, cost, base int) {
		id = strings.ToLower(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		name, icon := c.itemLook(id)
		if name == "" {
			name, icon = cosmeticFallback(id, typeID)
		}
		if name == "" {
			name = "Bundle item"
		}
		items = append(items, storeItem{Name: name, Icon: icon, Price: cost, Was: base})
	}
	for _, rawItem := range asArr(pick(row, "Items", "items")) {
		offer := asMap(rawItem)
		item := asMap(pick(offer, "Item", "item"))
		add(str(pick(item, "ItemID", "itemId")), str(pick(item, "ItemTypeID", "itemTypeId")), num(pick(offer, "DiscountedPrice", "discountedPrice")), num(pick(offer, "BasePrice", "basePrice")))
	}
	for _, rawOffer := range asArr(pick(row, "ItemOffers", "itemOffers")) {
		wrap := asMap(rawOffer)
		offer := asMap(pick(wrap, "Offer", "offer"))
		if len(offer) == 0 {
			offer = wrap
		}
		cost := vpCost(asMap(pick(offer, "Cost", "cost")))
		for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
			bit := asMap(reward)
			add(str(pick(bit, "ItemID", "itemId")), str(pick(bit, "ItemTypeID", "itemTypeId")), cost, cost)
			cost = 0
		}
	}
	return items
}

func cosmeticFallback(id, typeID string) (string, string) {
	id = strings.ToLower(id)
	if id == "" {
		return "", ""
	}
	switch strings.ToLower(typeID) {
	case "e7c63390-eda7-46e0-bb7a-a6abdacd2433":
		return "Skin", "https://media.valorant-api.com/weaponskinlevels/" + id + "/displayicon.png"
	case "3ad1b2b2-acdb-4524-852f-954a76ddae0a":
		return "Chroma", "https://media.valorant-api.com/weaponskinchromas/" + id + "/fullrender.png"
	case "dd3bf334-87f3-40bd-b043-682a57a8dc3a":
		return "Buddy", "https://media.valorant-api.com/buddies/" + id + "/displayicon.png"
	case "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475":
		return "Spray", "https://media.valorant-api.com/sprays/" + id + "/fulltransparenticon.png"
	case "3f296c07-64c3-494c-923b-fe692a4fa1bd":
		return "Player card", cardArt(id)
	case "de7caa6b-adf7-4588-bbd1-143831e786c6":
		return "Title", ""
	default:
		return "", "https://media.valorant-api.com/weaponskinlevels/" + id + "/displayicon.png"
	}
}

func (c *riotClient) fetchContracts(ctx context.Context, sess riotSession) (int, []byte, error) {
	if ver := parseClientVersion(readTail(filepath.Join(os.Getenv("LOCALAPPDATA"), "VALORANT", "Saved", "Logs", "ShooterGame.log"), 700_000)); strings.Contains(ver, "shipping") {
		sess.version = ver
	}
	id := sess.puuid
	raw := "https://pd." + sess.shard + ".a.pvp.net/contracts/v1/contracts/" + id
	status, payload, err := c.contractCall(ctx, sess, raw)
	if err == nil && status < 400 && json.Valid(payload) {
		return status, payload, nil
	}
	c.mu.Lock()
	cached := c.version
	c.mu.Unlock()
	if cached != "" && cached != sess.version {
		sess.version = cached
		if status2, payload2, err2 := c.contractCall(ctx, sess, raw); err2 == nil && status2 < 400 && json.Valid(payload2) {
			return status2, payload2, nil
		}
	}
	return status, payload, err
}

func (c *riotClient) contractCall(ctx context.Context, sess riotSession, raw string) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Authorization", "Bearer "+sess.access)
	req.Header.Set("X-Riot-Entitlements-JWT", sess.entitlement)
	req.Header.Set("X-Riot-ClientPlatform", clientPlatform)
	req.Header.Set("X-Riot-ClientVersion", sess.version)
	req.Header.Set("Accept-Encoding", "identity")
	res, err := c.remote.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	return res.StatusCode, payload, nil
}

func (c *riotClient) battlepass(ctx context.Context) passView {
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	if sess.shard == "" || sess.puuid == "" {
		return passView{Error: "Open Valorant to read the battle pass.", Missions: []missionView{}}
	}
	defs := c.contractBook(ctx)
	missions := c.missionBook(ctx)
	status, payload, err := c.fetchContracts(ctx, sess)
	if err != nil || status >= 400 {
		msg := "Couldn't read the battle pass."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't read the battle pass (%d).", status)
		}
		return passView{Error: msg, Missions: []missionView{}}
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return passView{Error: "Couldn't read the battle pass.", Missions: []missionView{}}
	}
	var chosen map[string]any
	var chosenDef map[string]any
	chosenID := ""
	best := -1
	active := c.activeSeason(ctx)
	type passCand struct {
		id, rel string
		row     map[string]any
		def     map[string]any
		level   int
	}
	var cands []passCand
	for _, item := range asArr(pick(body, "Contracts", "contracts")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "ContractDefinitionID", "contractDefinitionID")))
		def := defs[id]
		if !isPass(def) {
			continue
		}
		content := asMap(pick(def, "content"))
		cands = append(cands, passCand{
			id:    id,
			rel:   strings.ToLower(str(pick(content, "relationUuid"))),
			row:   row,
			def:   def,
			level: contractLevel(row),
		})
	}
	for _, cand := range cands {
		if active != "" && cand.rel == active && !isFreePass(cand.def) {
			chosen = cand.row
			chosenDef = cand.def
			chosenID = cand.id
			best = cand.level
			break
		}
	}
	if chosen == nil {
		for _, cand := range cands {
			if cand.level >= best {
				best = cand.level
				chosen = cand.row
				chosenDef = cand.def
				chosenID = cand.id
			}
		}
	}
	view := passView{Missions: []missionView{}, Tracks: []passTrack{}}
	if chosen != nil {
		view.Name = str(pick(chosenDef, "displayName"))
		if view.Name == "" {
			view.Name = "Battle pass"
		}
		view.Icon = str(pick(chosenDef, "displayIcon"))
		view.Free = isFreePass(chosenDef)
		view.Level = contractLevel(chosen)
		view.XP = contractXP(chosen)
		steps := levelXP(chosenDef)
		if view.Level >= 0 && view.Level < len(steps) {
			view.Next = steps[view.Level]
		}
		view.Rewards = c.passRewards(ctx, chosenDef, view.Level)
		for _, cand := range cands {
			if cand.id == chosenID || cand.id == "" {
				continue
			}
			if !isFreePass(cand.def) && passSize(cand.def) < 8 {
				continue
			}
			view.Tracks = append(view.Tracks, c.trackFrom(ctx, cand.row, cand.def))
		}
	} else {
		view.Name = "Battle pass"
		view.Error = "No active battle pass on this account."
	}
	meta := asMap(pick(body, "MissionMetadata", "missionMetadata"))
	weeklyReset := expiryStamp(pick(meta, "WeeklyRefillTime", "weeklyRefillTime", "WeeklyCheckpoint", "weeklyCheckpoint"))
	for _, item := range missionList(body) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "ID", "id", "MissionID", "missionID", "DefinitionID")))
		def := missions[id]
		ends := expiryStamp(pick(row, "ExpirationTime", "expirationTime"))
		window := missionWindow(ends, weeklyReset)
		if window == "Daily" {
			for objID := range asMap(pick(row, "Objectives", "objectives")) {
				if alt := missions["daily:"+strings.ToLower(objID)]; len(alt) > 0 {
					def = alt
					break
				}
			}
		}
		if len(def) == 0 {
			for objID := range asMap(pick(row, "Objectives", "objectives")) {
				if found := missions[strings.ToLower(objID)]; len(found) > 0 {
					def = found
					break
				}
			}
		}
		rawType := strings.ToLower(str(pick(def, "type")))
		if strings.Contains(rawType, "tutorial") || strings.Contains(rawType, "npe") || strings.Contains(rawType, "bte") {
			continue
		}
		kind := window
		if kind == "" {
			kind = missionKind(rawType)
		}
		xp := num(pick(def, "xpGrant"))
		if kind == "" && xp > 0 && xp <= 4500 {
			kind = "Daily"
		}
		if kind == "" {
			kind = "Weekly"
		}
		goal := num(pick(def, "progressToComplete"))
		if goal == 0 {
			for _, obj := range asArr(pick(def, "objectives")) {
				goal += num(pick(asMap(obj), "value"))
			}
		}
		progress := 0
		for _, value := range asMap(pick(row, "Objectives", "objectives")) {
			progress += num(value)
		}
		done, _ := pick(row, "Complete", "complete").(bool)
		if done && goal > 0 {
			progress = goal
		}
		if goal == 0 {
			goal = 1
		}
		title := str(pick(def, "title"))
		if title == "" {
			title = kind + " mission"
		}
		view.Missions = append(view.Missions, missionView{
			Title:    title,
			Kind:     kind,
			Progress: progress,
			Goal:     goal,
			Done:     done,
			XP:       xp,
			Ends:     untilLabel(ends),
		})
	}
	view.Ends = c.actEnds(ctx)
	return view
}

var markup = regexp.MustCompile(`</?[^<>]*>`)

func cleanCopy(text string) string {
	text = markup.ReplaceAllString(text, "")
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&", "&")
	return strings.TrimSpace(text)
}

type savedCrosshair struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Current bool   `json:"current,omitempty"`
}

var crosshairCode = regexp.MustCompile(`0;[A-Za-z];[0-9A-Za-z;.\-]{12,}`)

func (c *riotClient) gameCrosshairs(ctx context.Context) ([]savedCrosshair, string) {
	return []savedCrosshair{}, ""
}

func (c *riotClient) preferenceText(ctx context.Context, token string) string {
	status, payload, err := c.do(ctx, c.remote, "GET", "https://playerpreferences.riotgames.com/playerPref/v3/getPreference/Ares.PlayerSettings", "Bearer "+token, "", nil)
	if err != nil || status >= 400 || len(payload) == 0 {
		return ""
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return fileText(payload)
	}
	data := str(pick(body, "data"))
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil || len(raw) == 0 {
		return string(payload)
	}
	reader := flate.NewReader(bytes.NewReader(raw))
	defer reader.Close()
	out, err := io.ReadAll(io.LimitReader(reader, 8<<20))
	if err != nil || len(out) == 0 {
		return string(payload)
	}
	return string(out)
}

func missionList(body map[string]any) []any {
	out := []any{}
	seen := map[string]bool{}
	add := func(v any) {
		rows := asArr(v)
		if len(rows) == 0 {
			for id, item := range asMap(v) {
				row := asMap(item)
				if len(row) == 0 {
					continue
				}
				if str(pick(row, "ID", "id")) == "" {
					row["ID"] = id
				}
				rows = append(rows, row)
			}
		}
		for _, item := range rows {
			row := asMap(item)
			if len(row) == 0 || asMap(pick(row, "Objectives", "objectives")) == nil && expiryStamp(pick(row, "ExpirationTime", "expirationTime")) == "" {
				continue
			}
			id := strings.ToLower(str(pick(row, "ID", "id", "MissionID", "missionID")))
			if id == "" || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, row)
		}
	}
	for _, key := range []string{"Missions", "missions", "DailyMissions", "dailyMissions", "WeeklyMissions", "weeklyMissions"} {
		add(body[key])
	}
	return out
}

func expiryStamp(v any) string {
	if s := str(v); s != "" {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 1_000_000_000 {
			if n > 10_000_000_000 {
				n /= 1000
			}
			return time.Unix(n, 0).UTC().Format(time.RFC3339)
		}
		return s
	}
	n := num(v)
	if n > 1_000_000_000 {
		if n > 10_000_000_000 {
			n /= 1000
		}
		return time.Unix(int64(n), 0).UTC().Format(time.RFC3339)
	}
	return ""
}

func kindByTime(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		t, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return ""
		}
	}
	if t.Year() < 2020 {
		return ""
	}
	left := time.Until(t)
	if left < -6*time.Hour {
		return ""
	}
	if left <= 40*time.Hour {
		return "Daily"
	}
	if left <= 8*24*time.Hour {
		return "Weekly"
	}
	return ""
}

func missionWindow(ends, weekly string) string {
	end := parseAnyTime(ends)
	week := parseAnyTime(weekly)
	if end.Year() >= 2020 && week.Year() >= 2020 && end.Before(week.Add(-12*time.Hour)) {
		return "Daily"
	}
	return kindByTime(ends)
}

func parseAnyTime(raw string) time.Time {
	raw = strings.TrimSpace(expiryStamp(raw))
	if raw == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (c *riotClient) ensureAct(ctx context.Context) (time.Time, time.Time) {
	if start, end, ok := knownAct(); ok {
		return start, end
	}
	now := time.Now()
	var start, end time.Time
	for _, row := range c.apiBook(ctx, "https://valorant-api.com/v1/seasons") {
		s, e := parseStamp(str(pick(row, "startTime"))), parseStamp(str(pick(row, "endTime")))
		if s.IsZero() || e.IsZero() || now.Before(s) || !now.Before(e) {
			continue
		}
		if strings.EqualFold(str(pick(row, "type")), "act") {
			start, end = s, e
			break
		}
	}
	if !start.IsZero() && end.After(start) {
		rememberAct(start, end)
	}
	return start, end
}

func (c *riotClient) actEnds(ctx context.Context) string {
	_, end := c.ensureAct(ctx)
	if end.IsZero() {
		return ""
	}
	return end.UTC().Format(time.RFC3339)
}

func (c *riotClient) activeSeason(ctx context.Context) string {
	now := time.Now()
	for id, row := range c.apiBook(ctx, "https://valorant-api.com/v1/seasons") {
		start, end := parseStamp(str(pick(row, "startTime"))), parseStamp(str(pick(row, "endTime")))
		if start.IsZero() || end.IsZero() || now.Before(start) || !now.Before(end) {
			continue
		}
		if strings.EqualFold(str(pick(row, "type")), "act") {
			return id
		}
	}
	return ""
}

func parseStamp(raw string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}
	}
	return t
}

func (c *riotClient) passRewards(ctx context.Context, def map[string]any, level int) []passReward {
	c.ensureSkins(ctx)
	c.ensureCatalog(ctx)
	book := c.rewardBook(ctx)
	var all []passReward
	n := 0
	for _, chapter := range asArr(pick(asMap(pick(def, "content")), "chapters")) {
		for _, row := range asArr(pick(asMap(chapter), "levels")) {
			n++
			reward := asMap(pick(asMap(row), "reward"))
			id := strings.ToLower(str(pick(reward, "uuid")))
			kind := str(pick(reward, "type"))
			info := book[id]
			if info.Name == "" {
				c.mu.Lock()
				info = c.skins[id]
				c.mu.Unlock()
			}
			name := info.Name
			if name == "" {
				name = prettyReward(kind)
			}
			icon := info.Icon
			if icon == "" {
				icon = rewardIcon(kind, id)
			}
			anim := info.Anim
			if anim == "" {
				c.mu.Lock()
				anim = c.levelAnim[id]
				c.mu.Unlock()
			}
			all = append(all, passReward{Level: n, Name: name, Icon: icon, Anim: anim, Got: n <= level})
		}
		for _, raw := range asArr(pick(asMap(chapter), "freeRewards")) {
			reward := asMap(raw)
			if nested := asMap(pick(reward, "reward")); len(nested) > 0 {
				reward = nested
			}
			id := strings.ToLower(str(pick(reward, "uuid")))
			if id == "" {
				continue
			}
			kind := str(pick(reward, "type"))
			info := book[id]
			if info.Name == "" {
				c.mu.Lock()
				info = c.skins[id]
				c.mu.Unlock()
			}
			name := info.Name
			if name == "" {
				name = prettyReward(kind)
			}
			icon := info.Icon
			if icon == "" {
				icon = rewardIcon(kind, id)
			}
			all = append(all, passReward{Level: n, Name: name, Icon: icon, Anim: info.Anim, Free: true, Got: n <= level})
		}
	}
	if len(all) == 0 {
		return []passReward{}
	}
	return all
}

func prettyReward(kind string) string {
	switch kind {
	case "EquippableSkinLevel":
		return "Weapon skin"
	case "EquippableCharmLevel", "EquippableCharm":
		return "Gun buddy"
	case "Spray":
		return "Spray"
	case "PlayerCard":
		return "Player card"
	case "Title":
		return "Title"
	case "Currency":
		return "Currency"
	default:
		if kind == "" {
			return "Reward"
		}
		return kind
	}
}

func rewardIcon(kind, id string) string {
	if len(id) < 32 {
		return ""
	}
	switch kind {
	case "EquippableSkinLevel":
		return "https://media.valorant-api.com/weaponskinlevels/" + id + "/displayicon.png"
	case "Spray":
		return "https://media.valorant-api.com/sprays/" + id + "/displayicon.png"
	case "PlayerCard":
		return "https://media.valorant-api.com/playercards/" + id + "/smallart.png"
	case "Currency":
		return "https://media.valorant-api.com/currencies/" + id + "/displayicon.png"
	case "EquippableCharmLevel", "EquippableCharm":
		return "https://media.valorant-api.com/buddies/" + id + "/displayicon.png"
	default:
		return ""
	}
}

func (c *riotClient) rewardBook(ctx context.Context) map[string]agentInfo {
	c.mu.Lock()
	if c.rewards != nil && time.Since(c.rewardsAt) < 12*time.Hour {
		out := c.rewards
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	book := map[string]agentInfo{}
	put := func(id, name, icon, anim string) {
		id = strings.ToLower(id)
		if id == "" {
			return
		}
		if prev, ok := book[id]; ok {
			if icon == "" {
				icon = prev.Icon
			}
			if anim == "" {
				anim = prev.Anim
			}
			if name == "" {
				name = prev.Name
			}
		}
		if name == "" && icon == "" && anim == "" {
			return
		}
		book[id] = agentInfo{Name: name, Icon: icon, Anim: anim}
	}
	for _, raw := range []string{
		"https://valorant-api.com/v1/sprays",
		"https://valorant-api.com/v1/playercards",
		"https://valorant-api.com/v1/buddies",
		"https://valorant-api.com/v1/playertitles",
		"https://valorant-api.com/v1/currencies",
		"https://valorant-api.com/v1/flex",
	} {
		for _, row := range c.apiBook(ctx, raw) {
			name := str(pick(row, "displayName", "titleText"))
			icon := str(pick(row, "fullTransparentIcon", "displayIcon", "smallArt", "wideArt"))
			anim := str(pick(row, "animationGif", "streamedVideo"))
			put(str(pick(row, "uuid")), name, icon, anim)
			for _, level := range asArr(pick(row, "levels")) {
				lv := asMap(level)
				put(str(pick(lv, "uuid")), name, fallback(str(pick(lv, "displayIcon")), icon), fallback(str(pick(lv, "streamedVideo")), anim))
			}
		}
	}
	c.mu.Lock()
	c.rewards = book
	c.rewardsAt = time.Now()
	c.mu.Unlock()
	return book
}

func markCurrent(rows []savedCrosshair, name, code string) []savedCrosshair {
	for i := range rows {
		if code != "" && rows[i].Code == code {
			rows[i].Current = true
		}
	}
	any := false
	for _, row := range rows {
		if row.Current {
			any = true
			break
		}
	}
	if !any && name != "" {
		for i := range rows {
			if strings.EqualFold(rows[i].Name, name) {
				rows[i].Current = true
			}
		}
	}
	front, rest := []savedCrosshair{}, []savedCrosshair{}
	for _, row := range rows {
		if row.Current {
			front = append(front, row)
		} else {
			rest = append(rest, row)
		}
	}
	return append(front, rest...)
}

func harvestCrosshairs(text string, add func(string, string)) {
	if code, name := codeFromINI(text); code != "" {
		add(name, code)
	}
	var body any
	if json.Unmarshal([]byte(text), &body) == nil {
		walkProfiles(body, add)
	}
	for _, code := range crosshairCode.FindAllString(text, 24) {
		add("", code)
	}
}

func walkProfiles(v any, add func(string, string)) {
	switch t := v.(type) {
	case map[string]any:
		key := str(pick(t, "settingEnum", "setting"))
		val := str(pick(t, "value"))
		if strings.Contains(key, "SavedCrosshairProfileData") && val != "" {
			var nested any
			if json.Unmarshal([]byte(val), &nested) == nil {
				walkProfiles(nested, add)
			}
		}
		if _, ok := t["Profiles"]; ok || t["ProfileName"] != nil || t["Primary"] != nil {
			addProfiles(t, add)
		}
		for _, child := range t {
			walkProfiles(child, add)
		}
	case []any:
		for _, child := range t {
			walkProfiles(child, add)
		}
	case string:
		if strings.Contains(t, "InnerLines") || strings.Contains(t, "ProfileName") {
			var nested any
			if json.Unmarshal([]byte(t), &nested) == nil {
				walkProfiles(nested, add)
			}
		}
	}
}

func addProfiles(v any, add func(string, string)) {
	row := asMap(v)
	if profiles := asArr(pick(row, "Profiles", "profiles")); len(profiles) > 0 {
		for _, item := range profiles {
			addProfiles(item, add)
		}
		return
	}
	if list := asMap(pick(row, "ProfileList")); len(list) > 0 {
		addProfiles(list, add)
		return
	}
	if inner := asMap(pick(row, "CrosshairProfile")); len(inner) > 0 {
		row = inner
	}
	code := encodeCrosshair(row)
	if code == "" {
		return
	}
	add(str(pick(row, "ProfileName", "profileName")), code)
}

func encodeCrosshair(profile map[string]any) string {
	primary := asMap(pick(profile, "Primary", "primary"))
	if len(primary) == 0 {
		primary = profile
	}
	if nested := asMap(pick(primary, "ProfileSettings")); len(nested) > 0 {
		primary = nested
	}
	inner := asMap(pick(primary, "InnerLines", "innerLines"))
	outer := asMap(pick(primary, "OuterLines", "outerLines"))
	if len(inner) == 0 && len(outer) == 0 && iniFloat(pick(primary, "LineLength")) == 0 {
		return ""
	}
	parts := []string{"0", "P", "c", strconv.Itoa(colorIndex(pick(primary, "Color", "color")))}
	parts = append(parts, "h", bitOf(pick(primary, "bHasOutline")))
	if n := iniFloat(pick(primary, "OutlineThickness")); n > 0 {
		parts = append(parts, "t", trimFloat(n))
	}
	if n := iniFloat(pick(primary, "OutlineOpacity")); n > 0 {
		parts = append(parts, "o", trimFloat(n))
	}
	parts = append(parts, "d", bitOf(pick(primary, "bDisplayCenterDot")))
	if n := iniFloat(pick(primary, "CenterDotSize")); n > 0 {
		parts = append(parts, "z", trimFloat(n))
	}
	parts = append(parts, lineCode("0", inner)...)
	parts = append(parts, lineCode("1", outer)...)
	return strings.Join(parts, ";")
}

func lineCode(prefix string, lines map[string]any) []string {
	if len(lines) == 0 {
		return nil
	}
	opacity := iniFloat(pick(lines, "Opacity", "opacity"))
	if opacity == 0 {
		opacity = 1
	}
	return []string{
		prefix + "b", bitOf(pick(lines, "bShowLines")),
		prefix + "t", trimFloat(iniFloat(pick(lines, "LineThickness"))),
		prefix + "l", trimFloat(iniFloat(pick(lines, "LineLength"))),
		prefix + "o", trimFloat(iniFloat(pick(lines, "LineOffset"))),
		prefix + "a", trimFloat(opacity),
		prefix + "m", bitOf(pick(lines, "bShowMovementError")),
		prefix + "f", bitOf(pick(lines, "bShowShootingError")),
	}
}

func colorIndex(v any) int {
	switch strings.ToLower(strings.TrimSpace(str(v))) {
	case "", "white", "0":
		return 0
	case "green", "1":
		return 1
	case "yellowgreen", "2":
		return 2
	case "greenyellow", "3":
		return 3
	case "yellow", "4":
		return 4
	case "cyan", "5":
		return 5
	case "pink", "6":
		return 6
	case "red", "7":
		return 7
	default:
		n, err := strconv.Atoi(str(v))
		if err == nil {
			return n
		}
		return 0
	}
}

func bitOf(v any) string {
	switch t := v.(type) {
	case bool:
		if t {
			return "1"
		}
		return "0"
	default:
		s := strings.ToLower(strings.TrimSpace(str(v)))
		if s == "true" || s == "1" {
			return "1"
		}
		return "0"
	}
}

func iniFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case string:
		n, _ := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return n
	default:
		return 0
	}
}

func trimFloat(n float64) string {
	s := strconv.FormatFloat(n, 'f', 2, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func codeFromINI(text string) (string, string) {
	length := iniValue(text, "CrosshairInnerLinesLineLength")
	if length == "" {
		return "", ""
	}
	profile := map[string]any{
		"Color":             firstINI(text, "CrosshairColor", "CrosshairPrimaryColor"),
		"bHasOutline":       iniValue(text, "CrosshairHasOutline"),
		"OutlineThickness":  iniValue(text, "CrosshairOutlineThickness"),
		"OutlineOpacity":    iniValue(text, "CrosshairOutlineOpacity"),
		"bDisplayCenterDot": firstINI(text, "CrosshairDisplayCenterDot", "CrosshairCenterDot"),
		"CenterDotSize":     iniValue(text, "CrosshairCenterDotSize"),
		"InnerLines": map[string]any{
			"bShowLines":         true,
			"LineThickness":      iniValue(text, "CrosshairInnerLinesLineThickness"),
			"LineLength":         length,
			"LineOffset":         iniValue(text, "CrosshairInnerLinesLineOffset"),
			"Opacity":            iniValue(text, "CrosshairInnerLinesOpacity"),
			"bShowMovementError": iniValue(text, "CrosshairInnerLinesShowMovementError"),
			"bShowShootingError": iniValue(text, "CrosshairInnerLinesShowShootingError"),
		},
		"OuterLines": map[string]any{
			"bShowLines":         iniValue(text, "CrosshairOuterLinesShowLines"),
			"LineThickness":      iniValue(text, "CrosshairOuterLinesLineThickness"),
			"LineLength":         iniValue(text, "CrosshairOuterLinesLineLength"),
			"LineOffset":         iniValue(text, "CrosshairOuterLinesLineOffset"),
			"Opacity":            iniValue(text, "CrosshairOuterLinesOpacity"),
			"bShowMovementError": iniValue(text, "CrosshairOuterLinesShowMovementError"),
			"bShowShootingError": iniValue(text, "CrosshairOuterLinesShowShootingError"),
		},
	}
	name := iniValue(text, "CrosshairProfileName")
	if name == "" {
		name = "Current"
	}
	return encodeCrosshair(profile), name
}

func firstINI(text string, keys ...string) string {
	for _, key := range keys {
		if value := iniValue(text, key); value != "" {
			return value
		}
	}
	return ""
}

func iniValue(text, key string) string {
	i := strings.Index(text, key)
	if i < 0 {
		return ""
	}
	rest := strings.TrimLeft(text[i+len(key):], " \t=")
	if cut := strings.IndexAny(rest, "\r\n"); cut >= 0 {
		rest = rest[:cut]
	}
	return strings.Trim(strings.TrimSpace(rest), `"`)
}

func fileText(b []byte) string {
	if len(b) >= 2 && b[0] == 0xFF && b[1] == 0xFE {
		return utf16LE(b[2:])
	}
	nulls := 0
	limit := len(b)
	if limit > 400 {
		limit = 400
	}
	for i := 1; i < limit; i += 2 {
		if b[i] == 0 {
			nulls++
		}
	}
	if nulls > 40 {
		return utf16LE(b)
	}
	return string(b)
}

func utf16LE(b []byte) string {
	if len(b)%2 == 1 {
		b = b[:len(b)-1]
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = uint16(b[i*2]) | uint16(b[i*2+1])<<8
	}
	return string(utf16.Decode(units))
}

func fallback(value, other string) string {
	if value == "" {
		return other
	}
	return value
}

func isPass(def map[string]any) bool {
	if len(def) == 0 {
		return false
	}
	content := asMap(pick(def, "content"))
	rel := strings.ToLower(str(pick(content, "relationType")))
	name := strings.ToLower(str(pick(def, "displayName")))
	switch rel {
	case "agent", "story", "npe", "tutorial":
		return false
	case "season", "event":
		return true
	}
	return strings.Contains(name, "battle") || strings.Contains(name, "pass")
}

func isFreePass(def map[string]any) bool {
	if len(def) == 0 {
		return false
	}
	content := asMap(pick(def, "content"))
	name := strings.ToLower(str(pick(def, "displayName")))
	if strings.Contains(name, "free") {
		return true
	}
	rel := strings.ToLower(str(pick(content, "relationType")))
	if rel == "event" && num(pick(content, "premiumVPCost")) == 0 {
		return true
	}
	if rel == "season" && num(pick(content, "premiumVPCost")) == 0 && strings.Contains(name, "pass") {
		return true
	}
	return false
}

func passSize(def map[string]any) int {
	n := 0
	for _, chapter := range asArr(pick(asMap(pick(def, "content")), "chapters")) {
		n += len(asArr(pick(asMap(chapter), "levels")))
	}
	return n
}

func (c *riotClient) trackFrom(ctx context.Context, row, def map[string]any) passTrack {
	name := str(pick(def, "displayName"))
	if name == "" {
		name = "Pass"
	}
	level := contractLevel(row)
	steps := levelXP(def)
	next := 0
	if level >= 0 && level < len(steps) {
		next = steps[level]
	}
	free := isFreePass(def)
	kind := "Event pass"
	if free {
		kind = "Free pass"
	}
	return passTrack{
		Name: name, Kind: kind, Level: level, XP: contractXP(row), Next: next,
		Icon: str(pick(def, "displayIcon")), Free: free,
		Rewards: c.passRewards(ctx, def, level),
	}
}

func contractLevel(row map[string]any) int {
	level := num(pick(row, "ProgressionLevelReached", "progressionLevelReached"))
	if level > 0 {
		return level
	}
	prog := asMap(pick(row, "ContractProgression", "contractProgression"))
	if level = num(pick(prog, "ProgressionLevelReached")); level > 0 {
		return level
	}
	return num(pick(asMap(pick(prog, "HighestRewardedLevel")), "Level"))
}

func contractXP(row map[string]any) int {
	if xp := num(pick(row, "ProgressionTowardsNextLevel", "progressionTowardsNextLevel")); xp > 0 {
		return xp
	}
	prog := asMap(pick(row, "ContractProgression", "contractProgression"))
	return num(pick(asMap(pick(prog, "HighestRewardedLevel")), "ProgressionTowardsNextLevel"))
}

func levelXP(def map[string]any) []int {
	var out []int
	content := asMap(pick(def, "content"))
	for _, chapter := range asArr(pick(content, "chapters")) {
		for _, level := range asArr(pick(asMap(chapter), "levels")) {
			out = append(out, num(pick(asMap(level), "xp")))
		}
	}
	return out
}

func untilLabel(raw string) string {
	if raw == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return ""
	}
	seconds := int(time.Until(t).Seconds())
	if seconds <= 0 {
		return "Ended"
	}
	return resetLabel(seconds)
}

func (c *riotClient) contractBook(ctx context.Context) map[string]map[string]any {
	return c.apiBook(ctx, "https://valorant-api.com/v1/contracts")
}

func (c *riotClient) missionBook(ctx context.Context) map[string]map[string]any {
	book := c.apiBook(ctx, "https://valorant-api.com/v1/missions")
	for _, row := range book {
		for _, obj := range asArr(pick(row, "objectives")) {
			id := strings.ToLower(str(pick(asMap(obj), "objectiveUuid")))
			if id == "" {
				continue
			}
			if missionKind(str(pick(row, "type"))) == "Daily" {
				book["daily:"+id] = row
			}
			if _, ok := book[id]; !ok {
				book[id] = row
			}
		}
	}
	return book
}

func (c *riotClient) apiBook(ctx context.Context, rawURL string) map[string]map[string]any {
	name := "book-" + cacheKey(rawURL)
	var cached map[string]map[string]any
	if age, ok := readCache(name, &cached); ok && len(cached) > 0 {
		if age > 12*time.Hour {
			cacheOnce(name, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
				defer cancel()
				if fresh := c.fetchAPIBook(ctx, rawURL); len(fresh) > 0 {
					writeCache(name, fresh)
				}
			})
		}
		return cached
	}
	fresh := c.fetchAPIBook(ctx, rawURL)
	if len(fresh) > 0 {
		writeCache(name, fresh)
	}
	return fresh
}

func (c *riotClient) fetchAPIBook(ctx context.Context, rawURL string) map[string]map[string]any {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return map[string]map[string]any{}
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return map[string]map[string]any{}
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 12<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return map[string]map[string]any{}
	}
	out := map[string]map[string]any{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "uuid")))
		if id != "" {
			out[id] = row
		}
	}
	return out
}

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type histCache struct {
	rows []recentMatch
	at   time.Time
}

type gear struct {
	Name      string `json:"name"`
	Skin      string `json:"skin"`
	ID        string `json:"id,omitempty"`
	Level     string `json:"level,omitempty"`
	Chroma    string `json:"chroma,omitempty"`
	Icon      string `json:"icon,omitempty"`
	Swatch    string `json:"swatch,omitempty"`
	Buddy     string `json:"buddy,omitempty"`
	BuddyIcon string `json:"buddyIcon,omitempty"`
	Group     string `json:"group"`
	Anim      string `json:"anim,omitempty"`
}

type cosmetic struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
	Icon string `json:"icon,omitempty"`
	Anim string `json:"anim,omitempty"`
	Slot string `json:"slot,omitempty"`
}

type loadoutView struct {
	Guns        []gear     `json:"guns"`
	Card        string     `json:"card,omitempty"`
	Title       string     `json:"title,omitempty"`
	Level       int        `json:"level,omitempty"`
	Sprays      []cosmetic `json:"sprays"`
	Expressions []cosmetic `json:"expressions"`
	Error       string     `json:"error,omitempty"`
	Value       int        `json:"value,omitempty"`
}

type skinRecord struct {
	Name    string
	Weapon  string
	Icon    string
	Levels  []string
	Chromas []chromaBit
}

type chromaBit struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon,omitempty"`
	Swatch string `json:"swatch,omitempty"`
}

type ownedSkin struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Icon    string      `json:"icon,omitempty"`
	Level   string      `json:"level"`
	Chromas []chromaBit `json:"chromas"`
}

type ownedBuddy struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Icon     string `json:"icon,omitempty"`
	Instance string `json:"instance,omitempty"`
	Type     string `json:"type,omitempty"`
}

type flairInfo struct {
	Name string
	Icon string
	Anim string
	Kind string
}

func expressionKind(typeID string) string {
	switch strings.ToLower(typeID) {
	case "", "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475":
		if typeID == "" {
			return ""
		}
		return "Spray"
	default:
		return "Expression"
	}
}

func knownQueue(queue string) bool {
	switch queue {
	case "", "competitive", "unrated", "swiftplay", "spikerush", "deathmatch", "hurm", "ggteam", "premier", "onefa", "custom", "snowball", "skirmish", "skirmishascension", "abilitydraft", "abilitydraftarena", "gauntlet", "aros", "dodgeball", "fortcollins":
		return true
	default:
		return false
	}
}

func (c *riotClient) matches(ctx context.Context, queue string) []recentMatch {
	if !knownQueue(queue) {
		return []recentMatch{}
	}
	c.mu.Lock()
	if hit, ok := c.queueHist[queue]; ok && len(hit.rows) > 0 && time.Since(hit.at) < 3*time.Minute {
		rows := append([]recentMatch(nil), hit.rows...)
		c.mu.Unlock()
		for i := range rows {
			c.decorate(rows[i].Players)
			c.paintRanks(rows[i].Players)
		}
		return rows
	}
	want := prettyMode(queue)
	saved := []recentMatch{}
	for _, row := range c.recent {
		if queue == "" || strings.EqualFold(row.Mode, want) {
			saved = append(saved, row)
		}
	}
	sess := c.session
	agents := map[string]agentInfo{}
	for key, info := range c.agents {
		agents[key] = info
	}
	c.mu.Unlock()
	for i := range saved {
		c.decorate(saved[i].Players)
		c.paintRanks(saved[i].Players)
	}
	if sess.shard != "" && len(saved) > 0 {
		c.mu.Lock()
		quiet := c.rankQuiet
		c.mu.Unlock()
		if !quiet {
			flat := []player{}
			n := len(saved)
			if n > 2 {
				n = 2
			}
			for i := 0; i < n; i++ {
				flat = append(flat, saved[i].Players...)
			}
			c.kickRanks(sess, flat)
		}
	}
	if sess.shard == "" || len(saved) > 0 {
		if sess.shard != "" {
			c.kickHistory(sess)
		}
		return saved
	}
	var rows []recentMatch
	func() {
		defer func() { _ = recover() }()
		rows = c.history(ctx, sess, agents, queue, 8)
	}()
	if rows == nil {
		return saved
	}
	merged := mergeMatches(saved, rows)
	if merged == nil {
		merged = []recentMatch{}
	}
	c.mu.Lock()
	c.queueHist[queue] = histCache{rows: merged, at: time.Now()}
	c.mu.Unlock()
	return merged
}

func weaponGroup(category string) string {
	switch {
	case strings.Contains(category, "Sidearm"), category == "Classic", category == "Shorty", category == "Frenzy", category == "Ghost", category == "Bandit", category == "Sheriff":
		return "Sidearms"
	case strings.Contains(category, "SMG"), category == "Stinger", category == "Spectre":
		return "SMGs"
	case strings.Contains(category, "Shotgun"), category == "Bucky", category == "Judge":
		return "Shotguns"
	case strings.Contains(category, "Rifle"), category == "Bulldog", category == "Guardian", category == "Phantom", category == "Vandal", category == "Warden":
		return "Rifles"
	case strings.Contains(category, "Sniper"), category == "Marshal", category == "Outlaw", category == "Operator":
		return "Snipers"
	case strings.Contains(category, "Heavy"), category == "Ares", category == "Odin":
		return "Heavy"
	case strings.Contains(category, "Melee"), category == "Melee":
		return "Melee"
	default:
		return "Other"
	}
}

func weaponName(id string) string {
	switch id {
	case "29a0cfab-485b-f5d5-779a-b59f85e204a8":
		return "Classic"
	case "42da8ccc-40d5-affc-beec-15aa47b42eda":
		return "Shorty"
	case "44d4e95c-4157-0037-81b2-17841bf2e8e3":
		return "Frenzy"
	case "1baa85b4-4c70-1284-64bb-6481dfc3bb4e":
		return "Ghost"
	case "410b2e0b-4ceb-1321-1727-20858f7f3477":
		return "Bandit"
	case "e336c6b8-418d-9340-d77f-7a9e4cfe0702":
		return "Sheriff"
	case "f7e1b454-4ad4-1063-ec0a-159e56b58941":
		return "Stinger"
	case "462080d1-4035-2937-7c09-27aa2a5c27a7":
		return "Spectre"
	case "910be174-449b-c412-ab22-d0873436b21b":
		return "Bucky"
	case "ec845bf4-4f79-ddda-a3da-0db3774b2794":
		return "Judge"
	case "ae3de142-4d85-2547-dd26-4e90bed35cf7":
		return "Bulldog"
	case "4ade7faa-4cf1-8376-95ef-39884480959b":
		return "Guardian"
	case "ee8e8d15-496b-07ac-e5f6-8fae5d4c7b1a":
		return "Phantom"
	case "9c82e19d-4575-0200-1a81-3eacf00cf872":
		return "Vandal"
	case "8db0a1bf-4a50-832a-4566-faaaa6d250ca":
		return "Warden"
	case "c4883e50-4494-202c-3ec3-6b8a9284f00b":
		return "Marshal"
	case "5f0aaf7a-4289-3998-d5ff-eb9a5cf7ef5c":
		return "Outlaw"
	case "a03b24d3-4319-996d-0f8c-94bbfba1dfc7":
		return "Operator"
	case "55d8a0f4-4274-ca67-fe2c-06ab45efdf58":
		return "Ares"
	case "63e6c2b6-4a8e-869c-3d4c-e38355226584":
		return "Odin"
	case "2f59173c-4bed-b6c3-2191-dea9b58be9c7":
		return "Melee"
	default:
		return "Weapon"
	}
}

func (c *riotClient) ensureCatalog(ctx context.Context) {
	c.mu.Lock()
	fresh := len(c.skinLook) > 0 && len(c.skinBook) > 0 && time.Since(c.catalogAt) < 12*time.Hour
	c.mu.Unlock()
	if fresh {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/weapons", nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return
	}
	look := map[string]agentInfo{}
	guns := map[string]gear{}
	levels := map[string]string{}
	chromas := map[string]string{}
	chromaArt := map[string]string{}
	chromaSwatch := map[string]string{}
	book := map[string]skinRecord{}
	levelOwner := map[string]string{}
	chromaOwner := map[string]string{}
	levelAnim := map[string]string{}
	for _, item := range asArr(pick(body, "data")) {
		weapon := asMap(item)
		wid := strings.ToLower(str(pick(weapon, "uuid")))
		name := str(pick(weapon, "displayName"))
		guns[wid] = gear{Name: name, Group: weaponGroup(str(pick(weapon, "category")))}
		for _, skinItem := range asArr(pick(weapon, "skins")) {
			skin := asMap(skinItem)
			id := strings.ToLower(str(pick(skin, "uuid")))
			if id == "" {
				continue
			}
			icon := str(pick(skin, "displayIcon"))
			if icon == "" {
				for _, chroma := range asArr(pick(skin, "chromas")) {
					icon = str(pick(asMap(chroma), "displayIcon", "fullRender"))
					if icon != "" {
						break
					}
				}
			}
			look[id] = agentInfo{Name: str(pick(skin, "displayName")), Icon: icon}
			rec := skinRecord{Name: str(pick(skin, "displayName")), Weapon: wid, Icon: icon}
			for _, levelItem := range asArr(pick(skin, "levels")) {
				level := asMap(levelItem)
				lid := strings.ToLower(str(pick(level, "uuid")))
				if lid != "" {
					levels[lid] = str(pick(level, "displayName"))
					levelOwner[lid] = id
					rec.Levels = append(rec.Levels, lid)
					if video := str(pick(level, "streamedVideo")); video != "" {
						levelAnim[lid] = video
					}
				}
			}
			for _, chromaItem := range asArr(pick(skin, "chromas")) {
				chroma := asMap(chromaItem)
				cid := strings.ToLower(str(pick(chroma, "uuid")))
				if cid == "" {
					continue
				}
				chromas[cid] = str(pick(chroma, "displayName"))
				art := str(pick(chroma, "fullRender", "displayIcon"))
				if art != "" {
					chromaArt[cid] = art
				}
				swatch := str(pick(chroma, "swatch"))
				if swatch != "" {
					chromaSwatch[cid] = swatch
				}
				chromaOwner[cid] = id
				rec.Chromas = append(rec.Chromas, chromaBit{ID: cid, Name: shortChroma(str(pick(chroma, "displayName"))), Icon: art, Swatch: swatch})
			}
			book[id] = rec
		}
	}
	if len(look) > 0 {
		c.mu.Lock()
		c.skinLook = look
		c.gunNames = guns
		c.levelNames = levels
		c.chromaNames = chromas
		c.chromaArt = chromaArt
		c.chromaSwatch = chromaSwatch
		c.skinBook = book
		c.levelOwner = levelOwner
		c.chromaOwner = chromaOwner
		c.levelAnim = levelAnim
		c.catalogAt = time.Now()
		c.mu.Unlock()
	}
}

func (c *riotClient) loadout(ctx context.Context) loadoutView {
	c.mu.Lock()
	sess := c.session
	if len(c.savedLoadout.Guns) > 0 && time.Since(c.gunsAt) < 2*time.Minute {
		view := c.savedLoadout
		view.Guns = append([]gear(nil), c.savedLoadout.Guns...)
		view.Sprays = append([]cosmetic(nil), c.savedLoadout.Sprays...)
		view.Expressions = append([]cosmetic(nil), c.savedLoadout.Expressions...)
		c.mu.Unlock()
		return view
	}
	c.mu.Unlock()
	if sess.shard == "" || sess.puuid == "" {
		return loadoutView{Error: "Open Valorant first.", Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	c.ensureCatalog(ctx)
	c.ensureCosmetics(ctx)
	c.mu.Lock()
	look := c.skinLook
	names := c.gunNames
	levels := c.levelNames
	chromas := c.chromaNames
	chromaArt := c.chromaArt
	swatches := c.chromaSwatch
	buddies := c.buddies
	flair := c.flair
	videos := c.levelAnim
	c.mu.Unlock()
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/personalization/v3/players/"+sess.puuid+"/playerloadout", nil)
	if err != nil || status == 404 || status == 405 {
		status, payload, err = c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/personalization/v2/players/"+sess.puuid+"/playerloadout", nil)
	}
	if err != nil || status >= 400 {
		msg := "Couldn't read your loadout."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't read your loadout (%d).", status)
		}
		return loadoutView{Error: msg, Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return loadoutView{Error: "Couldn't read your loadout.", Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	order := []string{"Classic", "Shorty", "Frenzy", "Ghost", "Bandit", "Sheriff", "Stinger", "Spectre", "Bucky", "Judge", "Bulldog", "Guardian", "Phantom", "Vandal", "Warden", "Marshal", "Outlaw", "Operator", "Ares", "Odin", "Melee"}
	rank := map[string]int{}
	for i, name := range order {
		rank[name] = i
	}
	guns := []gear{}
	for _, item := range asArr(pick(body, "Guns", "guns")) {
		row := asMap(item)
		wid := strings.ToLower(str(pick(row, "ID", "id")))
		weapon := names[wid]
		if weapon.Name == "" {
			weapon.Name = weaponName(wid)
			weapon.Group = weaponGroup(weapon.Name)
		}
		skinID := strings.ToLower(str(pick(row, "SkinID", "skinId")))
		levelID := strings.ToLower(str(pick(row, "SkinLevelID", "skinLevelID", "skinLevelId")))
		chromaID := strings.ToLower(str(pick(row, "ChromaID", "chromaId")))
		skin := look[skinID]
		if skin.Name == "" {
			skin = look[levelID]
		}
		skinName := skin.Name
		if skinName == "" {
			skinName = "Standard"
		}
		icon := chromaArt[chromaID]
		if icon == "" && chromaID != "" {
			icon = "https://media.valorant-api.com/weaponskinchromas/" + chromaID + "/fullrender.png"
		}
		if icon == "" {
			icon = skin.Icon
		}
		if icon == "" && levelID != "" {
			icon = "https://media.valorant-api.com/weaponskinlevels/" + levelID + "/displayicon.png"
		}
		charmID := strings.ToLower(str(pick(row, "CharmLevelID", "charmLevelID", "CharmID", "charmId")))
		buddy := buddies[charmID]
		buddyIcon := buddy.Icon
		if buddyIcon == "" && charmID != "" {
			buddyIcon = "https://media.valorant-api.com/buddylevels/" + charmID + "/displayicon.png"
		}
		guns = append(guns, gear{
			ID: wid, Name: weapon.Name, Skin: skinName, Level: shortLevel(levels[levelID]), Chroma: shortChroma(chromas[chromaID]),
			Icon: icon, Swatch: swatches[chromaID], Buddy: buddy.Name, BuddyIcon: buddyIcon, Group: weapon.Group, Anim: videos[levelID],
		})
	}
	for i := 1; i < len(guns); i++ {
		for j := i; j > 0; j-- {
			ai, bi := rank[guns[j-1].Name], rank[guns[j].Name]
			if _, ok := rank[guns[j-1].Name]; !ok {
				ai = 100
			}
			if _, ok := rank[guns[j].Name]; !ok {
				bi = 100
			}
			if ai <= bi {
				break
			}
			guns[j-1], guns[j] = guns[j], guns[j-1]
		}
	}
	sprays, expressions := equippedFlair(body, flair)
	ident := asMap(pick(body, "Identity", "identity"))
	cardID := strings.ToLower(str(pick(ident, "PlayerCardID", "playerCardID")))
	titleID := strings.ToLower(str(pick(ident, "PlayerTitleID", "playerTitleID")))
	card := flair[cardID].Icon
	if card == "" && cardID != "" {
		card = cardArt(cardID)
	}
	view := loadoutView{
		Guns: guns, Card: card, Title: flair[titleID].Name, Level: num(pick(ident, "AccountLevel", "accountLevel")),
		Sprays: sprays, Expressions: expressions,
	}
	if view.Level == 0 {
		view.Level = c.accountXP(ctx, sess)
	}
	view.Value = c.skinValue(ctx, sess)
	if len(guns) == 0 {
		view.Error = "No weapons came back from the client."
		return view
	}
	c.mu.Lock()
	c.guns = guns
	c.savedLoadout = view
	c.loadoutBody = append([]byte(nil), payload...)
	c.gunsAt = time.Now()
	c.mu.Unlock()
	return view
}

var (
	vpPriceMu sync.Mutex
	vpPrices  map[string]int
	vpPriceAt time.Time
)

func (c *riotClient) skinValue(ctx context.Context, sess riotSession) int {
	if sess.shard == "" || sess.puuid == "" {
		return 0
	}
	c.ensureCatalog(ctx)
	levels, err := c.ownedIDs(ctx, sess, "e7c63390-eda7-46e0-bb7a-a6abdacd2433")
	if err != nil || len(levels) == 0 {
		return 0
	}
	prices := c.vpByItem(ctx, sess)
	c.mu.Lock()
	owner := c.levelOwner
	c.mu.Unlock()
	best := map[string]int{}
	for id := range levels {
		id = strings.ToLower(id)
		skin := owner[id]
		if skin == "" {
			skin = id
		}
		if prices[id] > best[skin] {
			best[skin] = prices[id]
		}
	}
	sum := 0
	for _, vp := range best {
		sum += vp
	}
	return sum
}

func (c *riotClient) vpByItem(ctx context.Context, sess riotSession) map[string]int {
	vpPriceMu.Lock()
	if len(vpPrices) > 0 && time.Since(vpPriceAt) < 6*time.Hour {
		out := vpPrices
		vpPriceMu.Unlock()
		return out
	}
	vpPriceMu.Unlock()
	if sess.shard == "" {
		return map[string]int{}
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v1/offers/", nil)
	if err != nil || status >= 400 {
		return map[string]int{}
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return map[string]int{}
	}
	out := map[string]int{}
	for _, item := range asArr(pick(body, "Offers", "offers")) {
		offer := asMap(item)
		vp := vpCost(asMap(pick(offer, "Cost", "cost")))
		if vp <= 0 {
			continue
		}
		for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
			id := strings.ToLower(str(pick(asMap(reward), "ItemID", "itemId")))
			if id != "" && vp > out[id] {
				out[id] = vp
			}
		}
	}
	if len(out) > 0 {
		vpPriceMu.Lock()
		vpPrices = out
		vpPriceAt = time.Now()
		vpPriceMu.Unlock()
	}
	return out
}

func equippedFlair(body map[string]any, flair map[string]flairInfo) ([]cosmetic, []cosmetic) {
	sprays := []cosmetic{}
	expressions := []cosmetic{}
	add := func(id, kind, slot string) {
		id = strings.ToLower(id)
		if id == "" {
			return
		}
		info := flair[id]
		name := info.Name
		icon := info.Icon
		if info.Kind == "Spray" || kind == "Spray" {
			if name == "" {
				name = "Spray"
			}
			if icon == "" {
				icon = "https://media.valorant-api.com/sprays/" + id + "/fulltransparenticon.png"
			}
			sprays = append(sprays, cosmetic{Name: name, Kind: "Spray", Icon: icon, Anim: info.Anim, Slot: slot})
			return
		}
		if name == "" {
			name = "Expression"
		}
		if icon == "" {
			icon = "https://media.valorant-api.com/flex/" + id + "/displayicon.png"
		}
		expressions = append(expressions, cosmetic{Name: name, Kind: "Expression", Icon: icon, Slot: slot})
	}
	rows := asArr(pick(body, "ActiveExpressions", "activeExpressions"))
	if sprayBox := asMap(pick(body, "Sprays", "sprays")); sprayBox != nil {
		rows = append(rows, asArr(pick(sprayBox, "SpraySelections", "spraySelections"))...)
	}
	rows = append(rows, asArr(pick(body, "Sprays", "sprays"))...)
	if expr := asMap(pick(body, "Expressions", "expressions")); expr != nil {
		rows = append(rows, asArr(pick(expr, "AESSelections", "aesSelections"))...)
	}
	seen := map[string]bool{}
	n := 0
	for _, item := range rows {
		row := asMap(item)
		id := str(pick(row, "AssetID", "assetId", "SprayID", "sprayId"))
		if id == "" || seen[strings.ToLower(id)] {
			continue
		}
		seen[strings.ToLower(id)] = true
		n++
		kind := expressionKind(str(pick(row, "TypeID", "typeId")))
		if kind == "" {
			if _, ok := row["SprayID"]; ok || str(pick(row, "sprayId")) != "" {
				kind = "Spray"
			} else if flair[strings.ToLower(id)].Kind != "" {
				kind = flair[strings.ToLower(id)].Kind
			} else {
				kind = "Expression"
			}
		}
		slot := str(pick(row, "EquipSlotID", "equipSlotID", "SocketID", "socketId"))
		if strings.EqualFold(slot, "0814b2fe-4512-60a4-5288-1fbdcec6ca48") {
			slot = "Pre-round"
		} else {
			slot = kind + " " + strconv.Itoa(n)
		}
		add(id, kind, slot)
	}
	if sprays == nil {
		sprays = []cosmetic{}
	}
	if expressions == nil {
		expressions = []cosmetic{}
	}
	return sprays, expressions
}

func (c *riotClient) ensureCosmetics(ctx context.Context) {
	c.mu.Lock()
	fresh := len(c.flair) > 0 && time.Since(c.flairAt) < 12*time.Hour
	c.mu.Unlock()
	if fresh {
		return
	}
	buddies := map[string]flairInfo{}
	flair := map[string]flairInfo{}
	c.pullCosmetics(ctx, "https://valorant-api.com/v1/buddies", func(row map[string]any) {
		name := str(pick(row, "displayName"))
		icon := str(pick(row, "displayIcon"))
		id := strings.ToLower(str(pick(row, "uuid")))
		if id != "" {
			buddies[id] = flairInfo{Name: name, Icon: icon, Kind: "Buddy"}
		}
		for _, levelItem := range asArr(pick(row, "levels")) {
			level := asMap(levelItem)
			lid := strings.ToLower(str(pick(level, "uuid")))
			art := str(pick(level, "displayIcon"))
			if art == "" {
				art = icon
			}
			if lid != "" {
				buddies[lid] = flairInfo{Name: name, Icon: art, Kind: "Buddy"}
			}
		}
	})
	c.pullCosmetics(ctx, "https://valorant-api.com/v1/sprays", func(row map[string]any) {
		id := strings.ToLower(str(pick(row, "uuid")))
		icon := str(pick(row, "fullTransparentIcon", "displayIcon", "fullIcon"))
		name := str(pick(row, "displayName"))
		info := flairInfo{Name: name, Icon: icon, Anim: str(pick(row, "animationGif")), Kind: "Spray"}
		if id != "" {
			flair[id] = info
		}
		for _, levelItem := range asArr(pick(row, "levels")) {
			level := asMap(levelItem)
			lid := strings.ToLower(str(pick(level, "uuid")))
			if lid != "" {
				flair[lid] = info
			}
		}
	})
	c.pullCosmetics(ctx, "https://valorant-api.com/v1/flex", func(row map[string]any) {
		id := strings.ToLower(str(pick(row, "uuid")))
		if id != "" {
			flair[id] = flairInfo{Name: str(pick(row, "displayName")), Icon: str(pick(row, "displayIcon")), Kind: "Expression"}
		}
	})
	c.pullCosmetics(ctx, "https://valorant-api.com/v1/playercards", func(row map[string]any) {
		id := strings.ToLower(str(pick(row, "uuid")))
		icon := str(pick(row, "wideArt", "largeArt", "displayIcon"))
		if id != "" {
			flair[id] = flairInfo{Name: str(pick(row, "displayName")), Icon: icon, Kind: "Card"}
		}
	})
	c.pullCosmetics(ctx, "https://valorant-api.com/v1/playertitles", func(row map[string]any) {
		id := strings.ToLower(str(pick(row, "uuid")))
		title := str(pick(row, "titleText", "displayName"))
		if id != "" && title != "" {
			flair[id] = flairInfo{Name: title, Kind: "Title"}
		}
	})
	if len(flair) == 0 && len(buddies) == 0 {
		return
	}
	c.mu.Lock()
	c.buddies = buddies
	c.flair = flair
	c.flairAt = time.Now()
	c.mu.Unlock()
}

func (c *riotClient) pullCosmetics(ctx context.Context, rawURL string, visit func(map[string]any)) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return
	}
	for _, item := range asArr(pick(body, "data")) {
		if row := asMap(item); row != nil {
			visit(row)
		}
	}
}

func (c *riotClient) ownedIDs(ctx context.Context, sess riotSession, itemType string) (map[string]string, error) {
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v1/entitlements/"+sess.puuid+"/"+itemType, nil)
	if err != nil || status >= 400 {
		if status > 0 {
			return nil, fmt.Errorf("owned items %d", status)
		}
		return nil, err
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil, fmt.Errorf("owned items")
	}
	out := map[string]string{}
	for _, item := range asArr(pick(body, "Entitlements", "entitlements")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "ItemID", "itemId")))
		if id == "" {
			continue
		}
		out[id] = str(pick(row, "InstanceID", "instanceId"))
	}
	return out, nil
}

func (c *riotClient) wardrobe(ctx context.Context, weaponID string) map[string]any {
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	empty := map[string]any{"skins": []ownedSkin{}, "buddies": []ownedBuddy{}}
	if sess.shard == "" || sess.puuid == "" {
		empty["error"] = "Open Valorant first."
		return empty
	}
	c.ensureCatalog(ctx)
	c.ensureCosmetics(ctx)
	weaponID = strings.ToLower(weaponID)
	levels, err := c.ownedIDs(ctx, sess, "e7c63390-eda7-46e0-bb7a-a6abdacd2433")
	if err != nil {
		empty["error"] = "Couldn't read owned skins."
		return empty
	}
	chromas, _ := c.ownedIDs(ctx, sess, "3ad1b2b2-acdb-4524-852f-954a76ddae0a")
	buddies, _ := c.ownedIDs(ctx, sess, "dd3bf334-87f3-40bd-b043-682a57a8dc3a")
	c.mu.Lock()
	book := c.skinBook
	knownBuddies := c.buddies
	c.mu.Unlock()
	skins := []ownedSkin{}
	for id, rec := range book {
		if rec.Weapon != weaponID {
			continue
		}
		level := ""
		for _, lid := range rec.Levels {
			if _, ok := levels[lid]; ok {
				level = lid
			}
		}
		standard := strings.Contains(strings.ToLower(rec.Name), "standard")
		if level == "" && !standard {
			continue
		}
		if level == "" && len(rec.Levels) > 0 {
			level = rec.Levels[0]
		}
		ownedColors := []chromaBit{}
		for _, chroma := range rec.Chromas {
			if _, ok := chromas[chroma.ID]; ok || standard || len(chromas) == 0 {
				ownedColors = append(ownedColors, chroma)
			}
		}
		if len(ownedColors) == 0 {
			ownedColors = rec.Chromas
		}
		skins = append(skins, ownedSkin{ID: id, Name: rec.Name, Icon: rec.Icon, Level: level, Chromas: ownedColors})
	}
	for i := 1; i < len(skins); i++ {
		for j := i; j > 0 && skins[j].Name < skins[j-1].Name; j-- {
			skins[j], skins[j-1] = skins[j-1], skins[j]
		}
	}
	buddyRows := []ownedBuddy{}
	seenBuddy := map[string]bool{}
	for id, instance := range buddies {
		info := knownBuddies[id]
		icon := info.Icon
		if icon == "" {
			icon = "https://media.valorant-api.com/buddylevels/" + id + "/displayicon.png"
		}
		name := info.Name
		if name == "" {
			name = "Buddy"
		}
		if seenBuddy[id] {
			continue
		}
		seenBuddy[id] = true
		buddyRows = append(buddyRows, ownedBuddy{ID: id, Name: name, Icon: icon, Instance: instance})
	}
	if skins == nil {
		skins = []ownedSkin{}
	}
	if buddyRows == nil {
		buddyRows = []ownedBuddy{}
	}
	return map[string]any{"skins": skins, "buddies": buddyRows}
}

func (c *riotClient) flairOptions(ctx context.Context) map[string]any {
	c.mu.Lock()
	sess := c.session
	raw := append([]byte(nil), c.loadoutBody...)
	c.mu.Unlock()
	empty := map[string]any{"sprays": []ownedBuddy{}, "flex": []ownedBuddy{}, "slots": []cosmetic{}}
	if sess.shard == "" {
		empty["error"] = "Open Valorant first."
		return empty
	}
	if len(raw) == 0 {
		_ = c.loadout(ctx)
		c.mu.Lock()
		raw = append([]byte(nil), c.loadoutBody...)
		c.mu.Unlock()
	}
	c.ensureCosmetics(ctx)
	_ = c.rewardBook(ctx)
	owned := c.allOwned(ctx, sess)
	c.mu.Lock()
	flair := c.flair
	rewards := c.rewards
	c.mu.Unlock()
	flexIDs := map[string]bool{}
	for _, row := range c.apiBook(ctx, "https://valorant-api.com/v1/flex") {
		flexIDs[strings.ToLower(str(pick(row, "uuid")))] = true
	}
	sprayRows := []ownedBuddy{}
	flexRows := []ownedBuddy{}
	seenItem := map[string]bool{}
	for typeID, items := range owned {
		for id := range items {
			if seenItem[id] {
				continue
			}
			seenItem[id] = true
			info := flair[id]
			reward := agentInfo{}
			if rewards != nil {
				reward = rewards[id]
			}
			name := info.Name
			if name == "" {
				name = reward.Name
			}
			icon := info.Anim
			if icon == "" {
				icon = info.Icon
			}
			if icon == "" {
				icon = reward.Anim
			}
			if icon == "" {
				icon = reward.Icon
			}
			isFlex := flexIDs[id] || strings.EqualFold(typeID, "03f085d6-4a3d-4126-9a82-f3f1c1d3c1e0")
			isSpray := info.Kind == "Spray" || strings.EqualFold(typeID, "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475") || strings.EqualFold(typeID, "290f8769-97c6-492a-a1a8-caacf3d5b325")
			if !isFlex && !isSpray {
				continue
			}
			if isFlex && !isSpray {
				if name == "" {
					name = "Flex"
				}
				if icon == "" {
					icon = "https://media.valorant-api.com/flex/" + id + "/displayicon.png"
				}
				flexRows = append(flexRows, ownedBuddy{ID: id, Name: name, Icon: icon, Type: typeID})
				continue
			}
			if name == "" {
				name = "Spray"
			}
			if icon == "" {
				icon = "https://media.valorant-api.com/sprays/" + id + "/fulltransparenticon.png"
			}
			sprayRows = append(sprayRows, ownedBuddy{ID: id, Name: name, Icon: icon, Type: "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475"})
		}
	}
	slots := []cosmetic{}
	var body map[string]any
	if json.Unmarshal(raw, &body) == nil {
		for i, item := range asArr(pick(body, "ActiveExpressions", "activeExpressions")) {
			row := asMap(item)
			id := strings.ToLower(str(pick(row, "AssetID", "assetId")))
			info := flair[id]
			if info.Name == "" && rewards != nil {
				info = flairInfo{Name: rewards[id].Name, Icon: rewards[id].Icon, Anim: rewards[id].Anim}
			}
			kind := expressionKind(str(pick(row, "TypeID", "typeId")))
			icon := info.Anim
			if icon == "" {
				icon = info.Icon
			}
			slots = append(slots, cosmetic{Name: fallback(info.Name, kind), Kind: kind, Icon: icon, Slot: strconv.Itoa(i)})
		}
	}
	if slots == nil {
		slots = []cosmetic{}
	}
	if sprayRows == nil {
		sprayRows = []ownedBuddy{}
	}
	if flexRows == nil {
		flexRows = []ownedBuddy{}
	}
	return map[string]any{"sprays": sprayRows, "flex": flexRows, "slots": slots}
}

func (c *riotClient) allOwned(ctx context.Context, sess riotSession) map[string]map[string]string {
	out := map[string]map[string]string{}
	add := func(typeID string, rows []any) {
		typeID = strings.ToLower(typeID)
		if out[typeID] == nil {
			out[typeID] = map[string]string{}
		}
		for _, item := range rows {
			row := asMap(item)
			id := strings.ToLower(str(pick(row, "ItemID", "itemId")))
			if id == "" {
				continue
			}
			out[typeID][id] = str(pick(row, "InstanceID", "instanceId"))
		}
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v1/entitlements/"+sess.puuid, nil)
	if err == nil && status < 400 {
		var body map[string]any
		if json.Unmarshal(payload, &body) == nil {
			groups := asArr(pick(body, "EntitlementsByTypes", "entitlementsByTypes"))
			if len(groups) == 0 {
				add(str(pick(body, "ItemTypeID", "itemTypeId")), asArr(pick(body, "Entitlements", "entitlements")))
			}
			for _, group := range groups {
				row := asMap(group)
				add(str(pick(row, "ItemTypeID", "itemTypeId")), asArr(pick(row, "Entitlements", "entitlements")))
			}
		}
	}
	if len(out) == 0 {
		for _, typeID := range []string{"d5f120f8-ff8c-4aac-92ea-f2b5acbe9475", "290f8769-97c6-492a-a1a8-caacf3d5b325", "03f085d6-4a3d-4126-9a82-f3f1c1d3c1e0"} {
			ids, err := c.ownedIDs(ctx, sess, typeID)
			if err != nil {
				continue
			}
			out[typeID] = ids
		}
	}
	return out
}

func (c *riotClient) equipFlair(ctx context.Context, slot int, assetID, kind, typeHint string) loadoutView {
	c.mu.Lock()
	sess := c.session
	raw := append([]byte(nil), c.loadoutBody...)
	c.mu.Unlock()
	if len(raw) == 0 {
		_ = c.loadout(ctx)
		c.mu.Lock()
		raw = append([]byte(nil), c.loadoutBody...)
		sess = c.session
		c.mu.Unlock()
	}
	if sess.shard == "" || len(raw) == 0 {
		return loadoutView{Error: "Open Valorant first.", Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return loadoutView{Error: "Couldn't change that.", Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	typeID := "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475"
	if kind == "flex" {
		typeID = "03f085d6-4a3d-4126-9a82-f3f1c1d3c1e0"
	}
	if looksUUID(typeHint) {
		typeID = strings.ToLower(typeHint)
	}
	rows := asArr(pick(body, "ActiveExpressions", "activeExpressions"))
	if slot < 0 {
		slot = 0
	}
	for len(rows) <= slot {
		rows = append(rows, map[string]any{"AssetID": assetID, "TypeID": typeID})
	}
	row := asMap(rows[slot])
	if row == nil {
		row = map[string]any{}
	}
	if existing := str(pick(row, "TypeID", "typeId")); kind == "flex" && existing != "" && !strings.EqualFold(existing, typeID) && !strings.EqualFold(existing, "d5f120f8-ff8c-4aac-92ea-f2b5acbe9475") {
		typeID = existing
	}
	row["AssetID"] = strings.ToLower(assetID)
	row["TypeID"] = typeID
	if pick(row, "DynamicOptions", "dynamicOptions") == nil {
		row["DynamicOptions"] = map[string]any{}
	}
	rows[slot] = row
	body["ActiveExpressions"] = rows
	status, _, err := c.remoteCall(ctx, sess, "PUT", "https://pd."+sess.shard+".a.pvp.net/personalization/v3/players/"+sess.puuid+"/playerloadout", body)
	if err != nil || status >= 400 {
		msg := "Couldn't save that."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't save that (%d).", status)
		}
		return loadoutView{Error: msg, Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	c.mu.Lock()
	c.gunsAt = time.Time{}
	c.loadoutBody = nil
	c.mu.Unlock()
	return c.loadout(ctx)
}

func (c *riotClient) equip(ctx context.Context, weaponID, skinID, levelID, chromaID, buddyID, instanceID string) loadoutView {
	c.mu.Lock()
	sess := c.session
	raw := append([]byte(nil), c.loadoutBody...)
	c.mu.Unlock()
	if sess.shard == "" || len(raw) == 0 {
		return c.loadout(ctx)
	}
	c.ensureCatalog(ctx)
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return loadoutView{Error: "Couldn't change the loadout.", Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	c.mu.Lock()
	book := c.skinBook
	levelOwner := c.levelOwner
	chromaOwner := c.chromaOwner
	c.mu.Unlock()
	weaponID = strings.ToLower(weaponID)
	for _, item := range asArr(pick(body, "Guns", "guns")) {
		row := asMap(item)
		if strings.ToLower(str(pick(row, "ID", "id"))) != weaponID {
			continue
		}
		if skinID != "" {
			rec := book[strings.ToLower(skinID)]
			if levelID == "" && len(rec.Levels) > 0 {
				levelID = rec.Levels[len(rec.Levels)-1]
			}
			if chromaID == "" && len(rec.Chromas) > 0 {
				chromaID = rec.Chromas[0].ID
			}
			row["SkinID"] = strings.ToLower(skinID)
			if levelID != "" {
				row["SkinLevelID"] = strings.ToLower(levelID)
			}
			if chromaID != "" {
				row["ChromaID"] = strings.ToLower(chromaID)
			}
		} else if chromaID != "" {
			row["ChromaID"] = strings.ToLower(chromaID)
			if owner := chromaOwner[strings.ToLower(chromaID)]; owner != "" {
				row["SkinID"] = owner
			}
		}
		if buddyID != "" {
			row["CharmID"] = strings.ToLower(buddyID)
			row["CharmLevelID"] = strings.ToLower(buddyID)
			if instanceID != "" {
				row["CharmInstanceID"] = instanceID
			}
		}
		_ = levelOwner
		break
	}
	status, _, err := c.remoteCall(ctx, sess, "PUT", "https://pd."+sess.shard+".a.pvp.net/personalization/v3/players/"+sess.puuid+"/playerloadout", body)
	if err != nil || status >= 400 {
		msg := "Couldn't save that loadout."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't save that loadout (%d).", status)
		}
		return loadoutView{Error: msg, Guns: []gear{}, Sprays: []cosmetic{}, Expressions: []cosmetic{}}
	}
	c.mu.Lock()
	c.gunsAt = time.Time{}
	c.loadoutBody = nil
	c.mu.Unlock()
	return c.loadout(ctx)
}

func (c *riotClient) wallet(ctx context.Context, sess riotSession) (int, int, int) {
	if sess.shard == "" || sess.puuid == "" {
		return 0, 0, 0
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v1/wallet/"+sess.puuid, nil)
	if err != nil || status >= 400 {
		return 0, 0, 0
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return 0, 0, 0
	}
	balances := asMap(pick(body, "Balances", "balances"))
	vp := num(balances["85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741"])
	rad := num(balances["e59aa87c-4cbf-517a-5983-6e81511be9b7"])
	kc := num(balances["85ca954a-41f2-ce94-9b45-8ca3dd39a00d"])
	return vp, rad, kc
}

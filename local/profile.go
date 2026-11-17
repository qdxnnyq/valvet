package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type gearLevel struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Icon  string `json:"icon,omitempty"`
	Anim  string `json:"anim,omitempty"`
	Cost  int    `json:"cost"`
	Got   bool   `json:"got"`
	Next  bool   `json:"next,omitempty"`
	Item  string `json:"item,omitempty"`
	Offer string `json:"offer,omitempty"`
}

type gearAgent struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Icon     string      `json:"icon,omitempty"`
	Level    int         `json:"level"`
	Max      int         `json:"max"`
	Owned    bool        `json:"owned"`
	Contract string      `json:"contract,omitempty"`
	Levels   []gearLevel `json:"levels,omitempty"`
}

type borderBit struct {
	ID    string
	Start int
	Art   string
}

type cardView struct {
	Name   string         `json:"name"`
	Tag    string         `json:"tag"`
	Level  int            `json:"level"`
	Title  string         `json:"title"`
	Card   string         `json:"card"`
	Border string         `json:"border"`
	KC     int            `json:"kc"`
	Agents []masteryAgent `json:"agents"`
	Error  string         `json:"error,omitempty"`
}

func (c *riotClient) gearCatalog(ctx context.Context) []gearAgent {
	c.mu.Lock()
	if len(c.gearBase) > 0 && time.Since(c.gearBaseAt) < 12*time.Hour {
		out := c.gearBase
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/contracts", nil)
	if err != nil {
		return nil
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	c.ensureAgents(ctx)
	_ = c.rewardBook(ctx)
	c.ensureCards(ctx)
	c.mu.Lock()
	agents := c.agents
	rewards := c.rewards
	cards := c.cards
	c.mu.Unlock()
	out := []gearAgent{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		content := asMap(pick(row, "content"))
		if !strings.EqualFold(str(pick(content, "relationType")), "Agent") {
			continue
		}
		id := strings.ToLower(str(pick(content, "relationUuid")))
		if id == "" {
			continue
		}
		agent := gearAgent{ID: id, Name: agents[id].Name, Icon: agents[id].Icon, Contract: strings.ToLower(str(pick(row, "uuid")))}
		if agent.Name == "" {
			agent.Name = strings.TrimSuffix(str(pick(row, "displayName")), " Gear")
		}
		n := 0
		for _, chapter := range asArr(pick(content, "chapters")) {
			for _, level := range asArr(pick(asMap(chapter), "levels")) {
				n++
				lv := asMap(level)
				reward := asMap(pick(lv, "reward"))
				rid := strings.ToLower(str(pick(reward, "uuid")))
				kind := str(pick(reward, "type"))
				info := rewards[rid]
				icon := info.Icon
				if strings.EqualFold(kind, "PlayerCard") && cards[rid] != "" {
					icon = cards[rid]
				}
				if icon == "" {
					icon = rewardIcon(kind, rid)
				}
				name := info.Name
				if name == "" {
					name = prettyReward(kind)
				}
				agent.Levels = append(agent.Levels, gearLevel{
					Level: n,
					Name:  name,
					Kind:  prettyReward(kind),
					Icon:  icon,
					Anim:  info.Anim,
					Cost:  num(pick(lv, "doughCost")),
					Item:  rid,
				})
			}
		}
		agent.Max = len(agent.Levels)
		if agent.Max > 0 {
			out = append(out, agent)
		}
	}
	if len(out) > 0 {
		c.mu.Lock()
		c.gearBase = out
		c.gearBaseAt = time.Now()
		c.mu.Unlock()
	}
	return out
}

func (c *riotClient) offerBook(ctx context.Context, sess riotSession) map[string]string {
	c.mu.Lock()
	if len(c.offerByItem) > 0 && time.Since(c.offerAt) < 6*time.Hour {
		out := c.offerByItem
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	if sess.shard == "" {
		return map[string]string{}
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/store/v1/offers/", nil)
	if err != nil || status >= 400 {
		return map[string]string{}
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return map[string]string{}
	}
	out := map[string]string{}
	for _, item := range asArr(pick(body, "Offers", "offers")) {
		offer := asMap(item)
		oid := strings.ToLower(str(pick(offer, "OfferID", "offerId")))
		if kcCost(asMap(pick(offer, "Cost", "cost"))) == 0 || oid == "" {
			continue
		}
		for _, reward := range asArr(pick(offer, "Rewards", "rewards")) {
			id := strings.ToLower(str(pick(asMap(reward), "ItemID", "itemId")))
			if id != "" {
				out[id] = oid
			}
		}
	}
	if len(out) > 0 {
		c.mu.Lock()
		c.offerByItem = out
		c.offerAt = time.Now()
		c.mu.Unlock()
	}
	return out
}

func (c *riotClient) borderBook(ctx context.Context) []borderBit {
	c.mu.Lock()
	if len(c.borders) > 0 && time.Since(c.bordersAt) < 12*time.Hour {
		out := c.borders
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/levelborders", nil)
	if err != nil {
		return nil
	}
	res, err := c.remote.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return nil
	}
	out := []borderBit{}
	for _, item := range asArr(pick(body, "data")) {
		row := asMap(item)
		art := str(pick(row, "smallPlayerCardAppearance"))
		if art == "" {
			art = str(pick(row, "levelNumberAppearance"))
		}
		out = append(out, borderBit{ID: strings.ToLower(str(pick(row, "uuid"))), Start: num(pick(row, "startingLevel")), Art: art})
	}
	if len(out) > 0 {
		c.mu.Lock()
		c.borders = out
		c.bordersAt = time.Now()
		c.mu.Unlock()
	}
	return out
}

func (c *riotClient) contractLevels(ctx context.Context, sess riotSession) map[string]int {
	out := map[string]int{}
	if sess.shard == "" || sess.puuid == "" {
		return out
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/contracts/v1/contracts/"+sess.puuid, nil)
	if err != nil || status >= 400 {
		return out
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return out
	}
	for _, item := range asArr(pick(body, "Contracts", "contracts")) {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "ContractDefinitionID", "contractDefinitionID")))
		prog := asMap(pick(row, "ContractProgression", "contractProgression"))
		out[id] = num(pick(prog, "ProgressionLevelReached", "progressionLevelReached"))
	}
	return out
}

func (c *riotClient) gearFor(ctx context.Context, sess riotSession, agentID string) []gearAgent {
	base := c.gearCatalog(ctx)
	if len(base) == 0 {
		return nil
	}
	levels := c.contractLevels(ctx, sess)
	owned, _ := c.ownedIDs(ctx, sess, "01bb38e1-da47-4e6a-9b3d-945fe4655707")
	offers := c.offerBook(ctx, sess)
	out := []gearAgent{}
	for _, agent := range base {
		if agentID != "" && !strings.EqualFold(agent.ID, agentID) {
			continue
		}
		copyAgent := agent
		copyAgent.Levels = append([]gearLevel(nil), agent.Levels...)
		reached := levels[agent.Contract]
		if _, ok := owned[agent.ID]; ok || reached > 0 {
			copyAgent.Owned = true
		}
		copyAgent.Level = reached
		for n := range copyAgent.Levels {
			copyAgent.Levels[n].Got = n < reached
			copyAgent.Levels[n].Next = copyAgent.Owned && n == reached
			copyAgent.Levels[n].Offer = offers[copyAgent.Levels[n].Item]
		}
		out = append(out, copyAgent)
	}
	return out
}

func (c *riotClient) cardProfile(ctx context.Context) cardView {
	c.mu.Lock()
	cached := c.cardSnap
	age := time.Since(c.cardAt)
	busy := c.cardBusy
	has := !c.cardAt.IsZero() && (cached.Name != "" || cached.Level > 0 || cached.Card != "")
	c.mu.Unlock()
	if has && age < 45*time.Second {
		return cached
	}
	if has {
		if !busy {
			go c.refreshCard()
		}
		return cached
	}
	return c.buildCard(ctx)
}

func (c *riotClient) refreshCard() {
	c.mu.Lock()
	if c.cardBusy {
		c.mu.Unlock()
		return
	}
	c.cardBusy = true
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.cardBusy = false
		c.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_ = c.buildCard(ctx)
}

func (c *riotClient) buildCard(ctx context.Context) cardView {
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	if sess.puuid == "" {
		return cardView{Error: "Open the Riot Client first.", Agents: []masteryAgent{}}
	}
	loadout := c.loadout(ctx)
	borders := c.borderBook(ctx)
	border := ""
	var body map[string]any
	c.mu.Lock()
	raw := append([]byte(nil), c.loadoutBody...)
	c.mu.Unlock()
	if json.Unmarshal(raw, &body) == nil {
		ident := asMap(pick(body, "Identity", "identity"))
		want := strings.ToLower(str(pick(ident, "PreferredLevelBorderID", "preferredLevelBorderID")))
		for _, bit := range borders {
			if want != "" && bit.ID == want {
				border = bit.Art
			}
		}
	}
	if border == "" {
		best := -1
		for _, bit := range borders {
			if bit.Start <= loadout.Level && bit.Start >= best {
				best = bit.Start
				border = bit.Art
			}
		}
	}
	_, _, kc := c.wallet(ctx, sess)
	level := loadout.Level
	if level == 0 {
		level = c.accountXP(ctx, sess)
	}
	agents, masteryErr := c.masteryFor(ctx, sess, "")
	if agents == nil {
		agents = []masteryAgent{}
	}
	errText := loadout.Error
	if masteryErr != "" {
		errText = masteryErr
	}
	view := cardView{
		Name: sess.name, Tag: sess.tag, Level: level, Title: loadout.Title,
		Card: loadout.Card, Border: border, KC: kc, Agents: agents, Error: errText,
	}
	if view.Name != "" || view.Card != "" || view.Level > 0 {
		c.mu.Lock()
		c.cardSnap = view
		c.cardAt = time.Now()
		c.mu.Unlock()
	}
	return view
}

func (c *riotClient) buy(ctx context.Context, offerID string, price int, currency string) string {
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	offerID = strings.ToLower(strings.TrimSpace(offerID))
	if sess.shard == "" || sess.puuid == "" {
		return "Open Valorant first."
	}
	if offerID == "" || price <= 0 {
		return "That item can't be bought from here."
	}
	cur := "85ca954a-41f2-ce94-9b45-8ca3dd39a00d"
	if strings.EqualFold(currency, "VP") {
		cur = "85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741"
	}
	body := map[string]any{
		"offerId":            offerID,
		"purchaseCurrencyId": cur,
		"purchasePrice":      price,
		"quantity":           1,
	}
	status, payload, err := c.remoteCall(ctx, sess, "POST", "https://pd."+sess.shard+".a.pvp.net/store/v1/purchase/"+sess.puuid, body)
	if err != nil || status == 404 || status == 405 {
		status, payload, err = c.remoteCall(ctx, sess, "POST", "https://pd."+sess.shard+".a.pvp.net/store/v1/purchase", body)
	}
	if err != nil || status >= 400 {
		msg := "Couldn't buy that."
		if status > 0 {
			msg = fmt.Sprintf("Couldn't buy that (%d).", status)
		}
		text := strings.TrimSpace(string(payload))
		if text != "" && len(text) < 180 {
			msg = text
		}
		return msg
	}
	c.mu.Lock()
	c.shopAt = time.Time{}
	c.masteryAt = time.Time{}
	c.mu.Unlock()
	forgetOwnedCache()
	return ""
}

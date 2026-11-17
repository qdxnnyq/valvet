package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type masteryReward struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Icon  string `json:"icon,omitempty"`
	Anim  string `json:"anim,omitempty"`
	Cost  int    `json:"cost"`
	Got   bool   `json:"got"`
	Open  bool   `json:"open"`
	Item  string `json:"item,omitempty"`
	Offer string `json:"offer,omitempty"`
}

type masteryAgent struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	Icon     string          `json:"icon,omitempty"`
	Owned    bool            `json:"owned"`
	Act      int             `json:"act"`
	ActMax   int             `json:"actMax"`
	Over     int             `json:"over"`
	Lifetime int             `json:"lifetime"`
	Points   int             `json:"points"`
	Gear     int             `json:"gear,omitempty"`
	Accent   string          `json:"accent"`
	Rewards  []masteryReward `json:"rewards"`
}

type mprog struct {
	Act      int
	Lifetime int
	Points   int
	Rewards  []masteryReward
}

func (c *riotClient) masteryFor(ctx context.Context, sess riotSession, agentID string) ([]masteryAgent, string) {
	c.mu.Lock()
	if !c.masteryAt.IsZero() {
		age := time.Since(c.masteryAt)
		fresh := (c.masteryErr == "" && age < 45*time.Second) || (c.masteryErr != "" && age < 8*time.Second)
		if fresh {
			rows, msg := c.masteryRows, c.masteryErr
			c.mu.Unlock()
			return filterMastery(rows, agentID), msg
		}
	}
	c.mu.Unlock()
	rows, msg := c.loadMastery(ctx, sess)
	c.mu.Lock()
	c.masteryRows = rows
	c.masteryErr = msg
	c.masteryAt = time.Now()
	c.mu.Unlock()
	return filterMastery(rows, agentID), msg
}

func filterMastery(rows []masteryAgent, agentID string) []masteryAgent {
	if rows == nil {
		rows = []masteryAgent{}
	}
	if agentID == "" {
		out := make([]masteryAgent, len(rows))
		copy(out, rows)
		return out
	}
	want := strings.ToLower(agentID)
	for _, row := range rows {
		if strings.EqualFold(row.ID, want) {
			return []masteryAgent{row}
		}
	}
	return []masteryAgent{}
}

func (c *riotClient) loadMastery(ctx context.Context, sess riotSession) ([]masteryAgent, string) {
	if sess.shard == "" || sess.puuid == "" {
		return []masteryAgent{}, "Open the Riot Client first."
	}
	call, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	c.ensureAgents(call)
	_ = c.rewardBook(call)
	c.mu.Lock()
	known := map[string]agentInfo{}
	for id, info := range c.agents {
		known[id] = info
	}
	book := c.rewards
	c.mu.Unlock()
	paths := masteryPaths(sess.puuid)
	blobs := make([]any, 0, len(paths))
	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, path := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()
			status, body := c.pdAny(call, sess, path)
			if status < 200 || status >= 300 || body == nil {
				return
			}
			mu.Lock()
			blobs = append(blobs, body)
			mu.Unlock()
		}(path)
	}
	wg.Wait()
	defs := map[string]string{}
	rewardBookByAgent := map[string][]masteryReward{}
	for _, blob := range blobs {
		links, rewards := masteryCatalog(blob)
		for id, agent := range links {
			defs[id] = agent
		}
		for agent, rows := range rewards {
			rewardBookByAgent[agent] = append(rewardBookByAgent[agent], rows...)
		}
	}
	prog := collectMastery(blobs, defs)
	if len(prog) == 0 && len(rewardBookByAgent) == 0 {
		return []masteryAgent{}, "Valorant didn't return Agent Mastery. Open an agent's mastery screen once with Valorant running, then open Profile again."
	}
	offers := c.offerBook(call, sess)
	owned, _ := c.ownedIDs(call, sess, "01bb38e1-da47-4e6a-9b3d-945fe4655707")
	generic := true
	for _, rows := range rewardBookByAgent {
		if len(rows) > 0 {
			generic = false
		}
	}
	for _, hit := range prog {
		if hit != nil && len(hit.Rewards) > 0 {
			generic = false
		}
	}
	ids := map[string]bool{}
	for id := range known {
		ids[id] = true
	}
	for id := range prog {
		ids[id] = true
	}
	for id := range rewardBookByAgent {
		ids[id] = true
	}
	out := []masteryAgent{}
	for id := range ids {
		if !looksUUID(id) {
			continue
		}
		info := known[id]
		row := masteryAgent{ID: id, Name: info.Name, Icon: info.Icon, ActMax: 10, Rewards: []masteryReward{}}
		if hit := prog[id]; hit != nil {
			row.Act = hit.Act
			row.Lifetime = hit.Lifetime
			row.Points = hit.Points
			row.Rewards = append(row.Rewards, hit.Rewards...)
		}
		row.Rewards = append(row.Rewards, rewardBookByAgent[id]...)
		if len(row.Rewards) == 0 && generic {
			row.Rewards = append(row.Rewards, defaultMasteryTrack()...)
		}
		row.Rewards = decorateRewards(row.Rewards, book, offers, row.Lifetime)
		if _, ok := owned[id]; ok || row.Act > 0 || row.Lifetime > 0 || row.Points > 0 {
			row.Owned = true
		}
		if row.Act > 10 {
			row.Over = row.Act - 10
		}
		row.Accent = accentLabel(row.Act)
		if row.Name == "" {
			row.Name = "Agent"
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Lifetime != out[j].Lifetime {
			return out[i].Lifetime > out[j].Lifetime
		}
		if out[i].Act != out[j].Act {
			return out[i].Act > out[j].Act
		}
		return out[i].Name < out[j].Name
	})
	kept := []masteryAgent{}
	for _, row := range out {
		if row.Act > 0 || row.Lifetime > 0 || row.Points > 0 {
			kept = append(kept, row)
		}
	}
	if len(kept) == 0 {
		return []masteryAgent{}, "Valorant didn't return Act or Lifetime levels. Open an agent's mastery screen once, then open Profile again."
	}
	return kept, ""
}

func hasProgress(rows []masteryAgent) bool {
	for _, row := range rows {
		if row.Act > 0 || row.Lifetime > 0 || row.Points > 0 {
			return true
		}
	}
	return false
}

func (c *riotClient) gearMastery(ctx context.Context, sess riotSession, known map[string]agentInfo) []masteryAgent {
	base := c.gearFor(ctx, sess, "")
	if len(base) == 0 {
		return nil
	}
	out := []masteryAgent{}
	for _, agent := range base {
		if !agent.Owned && agent.Level == 0 {
			continue
		}
		info := known[agent.ID]
		name := agent.Name
		icon := agent.Icon
		if info.Name != "" {
			name = info.Name
		}
		if info.Icon != "" {
			icon = info.Icon
		}
		rewards := []masteryReward{}
		for _, level := range agent.Levels {
			rewards = append(rewards, masteryReward{
				Level: level.Level, Name: level.Name, Kind: level.Kind, Icon: level.Icon, Anim: level.Anim,
				Cost: level.Cost, Got: level.Got, Open: level.Got || level.Next, Item: level.Item, Offer: level.Offer,
			})
		}
		row := masteryAgent{
			ID: agent.ID, Name: name, Icon: icon, Owned: agent.Owned, ActMax: 10,
			Gear: agent.Level, Rewards: rewards, Accent: accentLabel(0),
		}
		if row.Name == "" {
			row.Name = "Agent"
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Lifetime != out[j].Lifetime {
			return out[i].Lifetime > out[j].Lifetime
		}
		return out[i].Name < out[j].Name
	})
	return out
}

var masteryPath = regexp.MustCompile(`(?i)https://pd\.[a-z0-9.-]+\.a\.pvp\.net(/[a-z0-9_./{}-]{6,180})`)

func masteryPaths(puuid string) []string {
	base := []string{
		"/contracts/v1/contracts/" + puuid,
		"/contracts/v2/contracts/" + puuid,
		"/contract-definitions/v3/definitions",
		"/contract-definitions/v2/definitions",
		"/mastery/v1/players/" + puuid,
		"/mastery/v2/players/" + puuid,
		"/mastery/v1/players/" + puuid + "/agents",
		"/agent-mastery/v1/players/" + puuid,
		"/agent-mastery/v2/players/" + puuid,
		"/agentmastery/v1/players/" + puuid,
		"/ares-mastery/v1/players/" + puuid,
		"/character-mastery/v1/players/" + puuid,
		"/player-mastery/v1/players/" + puuid,
		"/career/v1/players/" + puuid,
		"/career/v1/players/" + puuid + "/agents",
		"/agent-progression/v1/players/" + puuid,
		"/progression/v1/players/" + puuid,
		"/personalization/v3/players/" + puuid + "/mastery",
		"/personalization/v2/players/" + puuid + "/mastery",
		"/personalization/v4/players/" + puuid + "/mastery",
	}
	seen := map[string]bool{}
	out := []string{}
	add := func(path string) {
		path = strings.TrimSpace(path)
		if i := strings.IndexAny(path, "?#"); i >= 0 {
			path = path[:i]
		}
		if path == "" || seen[path] {
			return
		}
		low := strings.ToLower(path)
		if strings.Contains(low, "store/") || strings.Contains(low, "name-service") || strings.Contains(low, "match-history") || strings.Contains(low, "match-details") {
			return
		}
		seen[path] = true
		out = append(out, path)
	}
	for _, path := range base {
		add(path)
	}
	for _, path := range masteryPath.FindAllStringSubmatch(valorantLogText(), 120) {
		got := path[1]
		low := strings.ToLower(got)
		if !strings.Contains(low, "master") && !strings.Contains(low, "career") && !strings.Contains(low, "progress") && !strings.Contains(low, "agent") {
			continue
		}
		add(got)
	}
	return out
}

func valorantLogText() string {
	root := filepath.Join(os.Getenv("LOCALAPPDATA"), "VALORANT", "Saved", "Logs")
	names, _ := filepath.Glob(filepath.Join(root, "ShooterGame*.log"))
	if len(names) == 0 {
		names = []string{filepath.Join(root, "ShooterGame.log")}
	}
	var b strings.Builder
	for _, name := range names {
		b.WriteString(readTail(name, 800_000))
		b.WriteByte('\n')
		if b.Len() > 2_000_000 {
			break
		}
	}
	return b.String()
}

func (c *riotClient) pdAny(ctx context.Context, sess riotSession, path string) (int, any) {
	call, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	status, payload, err := c.remoteCall(call, sess, "GET", "https://pd."+sess.shard+".a.pvp.net"+path, nil)
	if err != nil || status >= 400 || len(payload) == 0 {
		return status, nil
	}
	var body any
	if json.Unmarshal(payload, &body) != nil {
		return status, nil
	}
	return status, body
}

func defaultMasteryTrack() []masteryReward {
	return []masteryReward{
		{Level: 5, Name: "Quotes, sprays, titles, cards", Kind: "Lifetime"},
		{Level: 10, Name: "Background, frame, stats", Kind: "Lifetime"},
		{Level: 15, Name: "Skin, quotes, sprays", Kind: "Lifetime"},
		{Level: 20, Name: "Agent ID pieces", Kind: "Lifetime"},
		{Level: 25, Name: "Mastery pose", Kind: "Lifetime"},
		{Level: 30, Name: "Animated player card", Kind: "Lifetime"},
	}
}

func accentLabel(act int) string {
	switch {
	case act >= 10:
		return "Accent 10"
	case act >= 7:
		return "Accent 7"
	case act >= 4:
		return "Accent 4"
	default:
		return "No accent"
	}
}

func decorateRewards(rows []masteryReward, book map[string]agentInfo, offers map[string]string, lifetime int) []masteryReward {
	seen := map[string]bool{}
	out := []masteryReward{}
	for _, row := range rows {
		if row.Level <= 0 {
			continue
		}
		row.Item = strings.ToLower(row.Item)
		key := strings.Join([]string{itoaSmall(row.Level), row.Item, strings.ToLower(row.Name)}, "|")
		if seen[key] {
			continue
		}
		seen[key] = true
		if info, ok := book[row.Item]; ok {
			if row.Name == "" || strings.EqualFold(row.Name, row.Kind) {
				row.Name = info.Name
			}
			if row.Icon == "" {
				row.Icon = info.Icon
			}
			if row.Anim == "" {
				row.Anim = info.Anim
			}
		}
		if row.Name == "" {
			row.Name = prettyReward(row.Kind)
		}
		if row.Icon == "" {
			row.Icon = rewardIcon(row.Kind, row.Item)
		}
		if row.Kind != "" {
			row.Kind = prettyReward(row.Kind)
		}
		row.Open = lifetime >= row.Level
		if row.Offer == "" {
			row.Offer = offers[row.Item]
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Level < out[j].Level })
	return out
}

func itoaSmall(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func collectMastery(blobs []any, defs map[string]string) map[string]*mprog {
	out := map[string]*mprog{}
	for _, blob := range blobs {
		walkMastery(blob, out, "", "", defs)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func walkMastery(v any, out map[string]*mprog, hint, parent string, defs map[string]string) {
	switch t := v.(type) {
	case []any:
		for _, item := range t {
			walkMastery(item, out, hint, parent, defs)
		}
	case map[string]any:
		next := hint
		if id := agentIDFrom(t); looksUUID(id) {
			next = id
		}
		if def := strings.ToLower(str(pick(t, "ContractDefinitionID", "contractDefinitionID", "DefinitionID", "definitionID"))); defs[def] != "" {
			next = defs[def]
		}
		act, aok := fuzzyLevel(t, "act")
		life, lok := fuzzyLevel(t, "life")
		pts, pok := fuzzyLevel(t, "point")
		if n, ok := nestedLevel(parent, "act", t); ok {
			act, aok = n, true
		}
		if n, ok := nestedLevel(parent, "life", t); ok {
			life, lok = n, true
		}
		if next != "" && (aok || lok || pok) {
			row := ensureProg(out, next)
			if aok && act > row.Act {
				row.Act = act
			}
			if lok && life > row.Lifetime {
				row.Lifetime = life
			}
			if pok && pts > row.Points {
				row.Points = pts
			}
			row.Rewards = append(row.Rewards, rewardsFrom(t)...)
		}
		if next != "" && !lok {
			if reached := pick(t, "ProgressionLevelReached", "progressionLevelReached"); reached != nil && num(reached) > 0 {
				row := ensureProg(out, next)
				if num(reached) > row.Lifetime {
					row.Lifetime = num(reached)
				}
			}
		}
		for key, child := range t {
			childHint := next
			if looksUUID(key) {
				if cm, ok := child.(map[string]any); ok {
					_, ca := fuzzyLevel(cm, "act")
					_, cl := fuzzyLevel(cm, "life")
					_, cp := fuzzyLevel(cm, "point")
					if ca || cl || cp || nestedMap(key) {
						childHint = strings.ToLower(key)
					}
				}
			}
			walkMastery(child, out, childHint, key, defs)
		}
	}
}

func nestedMap(key string) bool {
	switch strings.ToLower(key) {
	case "act", "actlevel", "actprogress", "actprogression", "currentact", "lifetime", "lifetimelevel", "career", "careerlevel":
		return true
	default:
		return false
	}
}

func nestedLevel(parent, kind string, m map[string]any) (int, bool) {
	p := strings.ToLower(parent)
	match := false
	switch kind {
	case "act":
		match = p == "act" || p == "actlevel" || p == "actprogress" || p == "actprogression" || p == "currentact"
	case "life":
		match = p == "lifetime" || p == "lifetimelevel" || p == "career" || p == "careerlevel"
	}
	if !match {
		return 0, false
	}
	if v := pick(m, "Level", "level", "Rank", "rank", "Value", "value"); v != nil {
		return num(v), true
	}
	return 0, false
}

func ensureProg(out map[string]*mprog, id string) *mprog {
	id = strings.ToLower(id)
	if out[id] == nil {
		out[id] = &mprog{}
	}
	return out[id]
}

func agentIDFrom(m map[string]any) string {
	return strings.ToLower(str(pick(m, "AgentID", "agentID", "agentId", "CharacterID", "characterID", "characterId", "AgentUuid", "agentUuid")))
}

func fuzzyLevel(m map[string]any, kind string) (int, bool) {
	for key, value := range m {
		lk := strings.ToLower(key)
		switch kind {
		case "act":
			if strings.Contains(lk, "lifetime") || strings.Contains(lk, "career") || strings.Contains(lk, "contract") {
				continue
			}
			if (strings.Contains(lk, "act") && (strings.Contains(lk, "level") || strings.Contains(lk, "lvl") || strings.Contains(lk, "rank"))) || lk == "act" || lk == "actlevel" {
				return num(value), true
			}
		case "life":
			if (strings.Contains(lk, "lifetime") || strings.Contains(lk, "career")) && (strings.Contains(lk, "level") || strings.Contains(lk, "lvl") || lk == "lifetime" || lk == "career") {
				return num(value), true
			}
		case "point":
			if lk == "mp" || lk == "masterypoints" || lk == "masteryxp" {
				return num(value), true
			}
			if strings.Contains(lk, "master") && (strings.Contains(lk, "point") || strings.Contains(lk, "xp") || strings.Contains(lk, "progress")) {
				return num(value), true
			}
		}
	}
	return 0, false
}

func masteryCatalog(v any) (map[string]string, map[string][]masteryReward) {
	links := map[string]string{}
	rewards := map[string][]masteryReward{}
	var walk func(any)
	walk = func(node any) {
		switch t := node.(type) {
		case []any:
			for _, item := range t {
				walk(item)
			}
		case map[string]any:
			if agent, def, ok := masteryNode(t); ok {
				if def != "" {
					links[def] = agent
				}
				rewards[agent] = append(rewards[agent], rewardsFrom(t)...)
			}
			for _, child := range t {
				walk(child)
			}
		}
	}
	walk(v)
	if len(links) == 0 && len(rewards) == 0 {
		return nil, nil
	}
	return links, rewards
}

func masteryNode(m map[string]any) (string, string, bool) {
	content := asMap(pick(m, "Content", "content"))
	if content == nil {
		content = m
	}
	rel := strings.ToLower(str(pick(content, "RelationType", "relationType")))
	name := strings.ToLower(str(pick(m, "Name", "DisplayName", "displayName", "name")))
	if strings.Contains(name, "gear") && !strings.Contains(name, "master") {
		return "", "", false
	}
	if rel == "agent" || rel == "season" || rel == "event" {
		return "", "", false
	}
	if !strings.Contains(name, "master") && !strings.Contains(rel, "master") {
		return "", "", false
	}
	agent := strings.ToLower(str(pick(content, "RelationUuid", "relationUuid", "RelationID", "RelationId", "relationID")))
	if !looksUUID(agent) {
		return "", "", false
	}
	def := strings.ToLower(str(pick(m, "ID", "Id", "uuid", "UUID", "DefinitionID", "definitionID")))
	return agent, def, true
}

func rewardsFrom(m map[string]any) []masteryReward {
	var out []masteryReward
	for key, value := range m {
		lk := strings.ToLower(key)
		if strings.Contains(lk, "reward") || strings.Contains(lk, "track") || lk == "levels" || lk == "chapters" || lk == "items" {
			out = append(out, flattenRewards(value)...)
		}
	}
	return out
}

func flattenRewards(v any) []masteryReward {
	switch t := v.(type) {
	case []any:
		var out []masteryReward
		for _, item := range t {
			out = append(out, flattenRewards(item)...)
		}
		return out
	case map[string]any:
		if row, ok := rewardFrom(t); ok {
			return []masteryReward{row}
		}
		var out []masteryReward
		for _, child := range t {
			out = append(out, flattenRewards(child)...)
		}
		return out
	default:
		return nil
	}
}

func rewardFrom(m map[string]any) (masteryReward, bool) {
	level, ok := unlockLevel(m)
	if !ok || level <= 0 {
		return masteryReward{}, false
	}
	reward := asMap(pick(m, "Reward", "reward"))
	item := strings.ToLower(str(pick(m, "ItemID", "ItemId", "itemId", "RewardID", "rewardId")))
	if item == "" && reward != nil {
		item = strings.ToLower(str(pick(reward, "ItemID", "ItemId", "uuid", "UUID")))
	}
	name := str(pick(m, "Name", "DisplayName", "displayName", "Title"))
	kind := str(pick(m, "Type", "ItemType", "itemType", "Kind", "kind"))
	if reward != nil {
		if name == "" {
			name = str(pick(reward, "Name", "DisplayName", "type", "Type"))
		}
		if kind == "" {
			kind = str(pick(reward, "Type", "type", "ItemTypeID", "itemTypeId"))
		}
	}
	cost := kcFrom(m)
	if cost == 0 && reward != nil {
		cost = kcFrom(reward)
	}
	if name == "" && item == "" && kind == "" {
		return masteryReward{}, false
	}
	row := masteryReward{Level: level, Name: name, Kind: kind, Cost: cost, Item: item, Offer: strings.ToLower(str(pick(m, "OfferID", "offerId")))}
	for _, key := range []string{"Purchased", "Owned", "IsOwned", "Unlocked", "purchased", "owned"} {
		if got, isBool := m[key].(bool); isBool && got {
			row.Got = true
		}
	}
	return row, true
}

func unlockLevel(m map[string]any) (int, bool) {
	for key, value := range m {
		lk := strings.ToLower(key)
		switch {
		case strings.Contains(lk, "lifetime") && strings.Contains(lk, "level"):
			return num(value), true
		case lk == "requiredlevel" || lk == "unlocklevel" || lk == "level" || lk == "tier" || lk == "milestone":
			return num(value), true
		}
	}
	return 0, false
}

func kcFrom(m map[string]any) int {
	if n := kcCost(asMap(pick(m, "Cost", "cost"))); n > 0 {
		return n
	}
	for _, key := range []string{"KingdomCredits", "kingdomCredits", "Price", "price", "KC", "doughCost"} {
		if n := num(pick(m, key)); n > 0 {
			return n
		}
	}
	return 0
}

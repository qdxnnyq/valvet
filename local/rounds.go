package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func roundPlayerName(detail map[string]any, id, me string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return ""
	}
	if me != "" && id == me {
		return "You"
	}
	for _, item := range asArr(pick(detail, "players", "Players")) {
		row := asMap(item)
		if !strings.EqualFold(str(pick(row, "subject", "Subject")), id) {
			continue
		}
		if name := strings.TrimSpace(str(pick(row, "gameName", "GameName"))); name != "" {
			return name
		}
	}
	return ""
}

func buyWord(value int) string {
	if value >= 3900 {
		return "Full"
	}
	if value >= 2000 {
		return "Force"
	}
	if value > 0 {
		return "Eco"
	}
	return ""
}

func matchBuys(detail map[string]any, players []player) []roundBuy {
	names := map[string]string{}
	for _, p := range players {
		id := strings.ToLower(strings.TrimSpace(p.PUUID))
		if id == "" {
			continue
		}
		names[id] = edgeLabel(p)
	}
	var out []roundBuy
	for _, item := range asArr(pick(detail, "roundResults", "RoundResults")) {
		round := asMap(item)
		n := num(pick(round, "roundNum", "RoundNum"))
		for _, stat := range asArr(pick(round, "playerStats", "PlayerStats")) {
			row := asMap(stat)
			id := strings.ToLower(str(pick(row, "subject", "Subject")))
			econ := asMap(pick(row, "economy", "Economy"))
			value := num(pick(econ, "loadoutValue", "LoadoutValue"))
			gun := niceGun(str(pick(econ, "weapon", "Weapon")))
			if value == 0 && gun == "" {
				continue
			}
			casts := asMap(pick(row, "abilityCasts", "AbilityCasts"))
			ult := num(pick(casts, "ultimateCasts", "UltimateCasts")) > 0
			out = append(out, roundBuy{
				Round: n, ID: id, Name: names[id], Gun: gun, Value: value, Buy: buyWord(value), Ult: ult,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func markEdgeFlags(edges []killEdge, players []player) {
	team := map[string]string{}
	for _, p := range players {
		id := strings.ToLower(strings.TrimSpace(p.PUUID))
		if id != "" && p.Team != "" {
			team[id] = p.Team
		}
	}
	byRound := map[int][]int{}
	for i, edge := range edges {
		byRound[edge.Round] = append(byRound[edge.Round], i)
	}
	for _, idxs := range byRound {
		sort.Slice(idxs, func(a, b int) bool { return edges[idxs[a]].T < edges[idxs[b]].T })
		if len(idxs) > 0 {
			edges[idxs[0]].First = true
		}
		for n, i := range idxs {
			if edges[i].Gun == "Ultimate" {
				edges[i].Ult = true
			}
			for _, j := range idxs[:n] {
				prev := edges[j]
				if edges[i].T-prev.T > 5000 || edges[i].T < prev.T {
					continue
				}
				if !strings.EqualFold(edges[i].VID, prev.KID) {
					continue
				}
				if team[edges[i].KID] != "" && team[edges[i].KID] == team[prev.VID] {
					edges[i].Trade = true
				}
			}
		}
	}
}

func penaltyLine(raw []byte, now time.Time) string {
	var body map[string]any
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	var soon time.Time
	queue := ""
	for _, item := range asArr(pick(body, "Penalties", "penalties")) {
		row := asMap(item)
		exp := penaltyExpiry(row)
		if exp.IsZero() || !exp.After(now) {
			continue
		}
		if !soon.IsZero() && !exp.Before(soon) {
			continue
		}
		soon = exp
		queue = ""
		block := asMap(pick(row, "QueueRestriction", "queueRestriction"))
		for _, id := range asArr(pick(block, "QueueIDs", "queueIDs")) {
			if name := prettyMode(str(id)); name != "" && name != "Match" {
				queue = name
				break
			}
		}
	}
	if soon.IsZero() {
		return ""
	}
	label := "Penalty"
	if queue != "" {
		label = queue + " penalty"
	}
	mins := int(soon.Sub(now).Minutes())
	if mins < 1 {
		mins = 1
	}
	return label + " · " + strconv.Itoa(mins) + " min"
}

func penaltyExpiry(row map[string]any) time.Time {
	block := asMap(pick(row, "QueueRestriction", "queueRestriction"))
	for _, key := range []any{pick(block, "Expiry", "expiry"), pick(row, "Expiry", "expiry")} {
		text := strings.TrimSpace(str(key))
		if text == "" {
			continue
		}
		if when, err := time.Parse(time.RFC3339, text); err == nil {
			return when
		}
	}
	return time.Time{}
}

func (c *riotClient) penaltyText(ctx context.Context) string {
	c.mu.Lock()
	sess := c.session
	if time.Since(c.penaltyAt) < 3*time.Minute {
		line := c.penalty
		c.mu.Unlock()
		return line
	}
	c.penaltyAt = time.Now()
	c.mu.Unlock()
	if sess.shard == "" || sess.puuid == "" || sess.access == "" {
		return ""
	}
	call, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	status, payload, err := c.remoteCall(call, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/restrictions/v3/penalties", nil)
	line := ""
	if err == nil && status < 400 && len(payload) > 0 {
		line = penaltyLine(payload, time.Now())
	}
	c.mu.Lock()
	c.penalty = line
	c.mu.Unlock()
	return line
}

type roundPack struct {
	ID      string      `json:"id"`
	Edges   []killEdge  `json:"edges,omitempty"`
	Rounds  []roundBeat `json:"rounds,omitempty"`
	Buys    []roundBuy  `json:"buys,omitempty"`
	Players []player    `json:"players,omitempty"`
	Full    bool        `json:"full,omitempty"`
}

func roundPackPath(id string) string {
	id = strings.ToLower(strings.TrimSpace(id))
	return filepath.Join(velvetBase(), "rounds", id+".json")
}

func saveRoundPack(row recentMatch) {
	id := strings.ToLower(strings.TrimSpace(row.ID))
	if id == "" || (len(row.Edges) == 0 && len(row.Buys) == 0) {
		return
	}
	pack := roundPack{ID: id, Edges: row.Edges, Rounds: row.Rounds, Buys: row.Buys, Players: row.Players, Full: row.Lines > 0}
	raw, err := json.Marshal(pack)
	if err != nil {
		return
	}
	dir := filepath.Join(velvetBase(), "rounds")
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	_ = os.WriteFile(roundPackPath(id), raw, 0o600)
}

func loadRoundPack(id string) (roundPack, bool) {
	raw, err := os.ReadFile(roundPackPath(id))
	if err != nil {
		return roundPack{}, false
	}
	var pack roundPack
	if json.Unmarshal(raw, &pack) != nil || len(pack.Edges) == 0 {
		return roundPack{}, false
	}
	return pack, true
}

func (pack roundPack) asMatch() *recentMatch {
	row := recentMatch{ID: pack.ID, Edges: pack.Edges, Rounds: pack.Rounds, Buys: pack.Buys, Players: pack.Players}
	if pack.Full {
		row.Lines = 1
	}
	return &row
}

func parkLines(rows []recentMatch) {
	for i := range rows {
		if len(rows[i].Edges) == 0 && len(rows[i].Buys) == 0 {
			continue
		}
		saveRoundPack(rows[i])
		rows[i].Edges = nil
		rows[i].Buys = nil
	}
}

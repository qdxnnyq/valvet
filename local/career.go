package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type career struct {
	HS       string
	ADR      string
	ACS      string
	Was      string
	Rank     string
	RankIcon string
	Tier     int
	Tried    time.Time
}

type nameHit struct {
	name string
	tag  string
	when int64
}

func withoutCurrent(was, name, tag string) string {
	current := strings.ToLower(name)
	tagged := current
	if tag != "" {
		tagged = strings.ToLower(name + "#" + tag)
	}
	var keep []string
	for _, part := range strings.Split(was, ", ") {
		low := strings.ToLower(strings.TrimSpace(part))
		if low == "" || low == current || low == tagged {
			continue
		}
		keep = append(keep, strings.TrimSpace(part))
	}
	return strings.Join(keep, ", ")
}

func (c *riotClient) decorate(players []player) {
	for i := range players {
		id := strings.ToLower(players[i].PUUID)
		if info, ok := c.career[id]; ok {
			if players[i].HS == "" {
				players[i].HS = info.HS
			}
			if players[i].ADR == "" {
				players[i].ADR = info.ADR
			}
			if players[i].ACS == "" {
				players[i].ACS = info.ACS
			}
			if players[i].Was == "" {
				players[i].Was = withoutCurrent(info.Was, players[i].Name, players[i].Tag)
			}
			if (players[i].Rank == "" || players[i].Rank == "Unranked") && info.Rank != "" {
				players[i].Rank = info.Rank
				players[i].Tier = info.Tier
				players[i].RankIcon = info.RankIcon
			}
		}
		if players[i].RankIcon == "" && players[i].Tier > 0 {
			players[i].RankIcon = c.rankIcons[players[i].Tier]
		}
	}
}

func (c *riotClient) queueCareer(ctx context.Context, sess riotSession, players []player) {
	if sess.shard == "" {
		return
	}
	c.mu.Lock()
	if c.careerBusy {
		c.mu.Unlock()
		return
	}
	var ids []string
	for _, p := range players {
		if p.PUUID == "" {
			continue
		}
		if info, ok := c.career[strings.ToLower(p.PUUID)]; ok {
			if info.HS != "" && time.Since(info.Tried) < 15*time.Minute {
				continue
			}
			if info.HS == "" && time.Since(info.Tried) < 45*time.Second {
				continue
			}
		}
		ids = append(ids, p.PUUID)
	}
	if len(ids) == 0 {
		c.mu.Unlock()
		return
	}
	if len(ids) > 16 {
		ids = ids[:16]
	}
	c.careerBusy = true
	c.mu.Unlock()
	go c.fillCareer(sess, ids)
	_ = ctx
}

func (c *riotClient) fillCareer(sess riotSession, ids []string) {
	defer func() {
		c.mu.Lock()
		c.careerBusy = false
		c.mu.Unlock()
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 22*time.Second)
	defer cancel()
	next := map[string]career{}
	for _, id := range ids {
		next[strings.ToLower(id)] = c.careerFromRecent(ctx, sess, id)
	}

	c.mu.Lock()
	for id, info := range next {
		if prev, ok := c.career[id]; ok {
			if info.HS == "" {
				info.HS = prev.HS
			}
			if info.ADR == "" {
				info.ADR = prev.ADR
			}
			if info.ACS == "" {
				info.ACS = prev.ACS
			}
			if info.Rank == "" && prev.Rank != "" {
				info.Rank = prev.Rank
				info.Tier = prev.Tier
				info.RankIcon = prev.RankIcon
			}
		}
		if info.Tier > 0 && info.RankIcon == "" {
			info.RankIcon = c.rankIcons[info.Tier]
		}
		info.Tried = time.Now()
		c.career[id] = info
	}
	c.mu.Unlock()
}

func (c *riotClient) careerFromRecent(ctx context.Context, sess riotSession, id string) career {
	if info, ok := c.careerLocal(id); ok {
		return info
	}
	var info career
	if sess.shard == "" || id == "" {
		return info
	}
	status, payload, err := c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/match-history/v1/history/"+id+"?startIndex=0&endIndex=2", nil)
	if err != nil || status >= 400 {
		return info
	}
	var body map[string]any
	if json.Unmarshal(payload, &body) != nil {
		return info
	}
	mids := []string{}
	for _, item := range asArr(pick(body, "History", "history")) {
		mid := str(pick(asMap(item), "MatchID", "matchId"))
		if mid == "" {
			continue
		}
		mids = append(mids, mid)
		break
	}
	var tot combat
	for _, mid := range mids {
		status, payload, err = c.remoteCall(ctx, sess, "GET", "https://pd."+sess.shard+".a.pvp.net/match-details/v1/matches/"+mid, nil)
		if err != nil || status >= 400 {
			continue
		}
		var detail map[string]any
		if json.Unmarshal(payload, &detail) != nil {
			continue
		}
		part, ok := combatByPlayer(detail)[strings.ToLower(id)]
		if !ok {
			continue
		}
		tot.dmg += part.dmg
		tot.hs += part.hs
		tot.body += part.body
		tot.leg += part.leg
		tot.rounds += part.rounds
		tot.score += part.score
	}
	if tot.rounds > 0 {
		var tmp player
		applyCombat(&tmp, tot)
		info.HS, info.ADR, info.ACS = tmp.HS, tmp.ADR, tmp.ACS
	}
	return info
}

func (c *riotClient) careerLocal(id string) (career, bool) {
	var info career
	id = strings.ToLower(id)
	c.mu.Lock()
	rows := append([]recentMatch(nil), c.recent...)
	c.mu.Unlock()
	var hs, adr, acs, n int
	for _, row := range rows {
		for _, p := range row.Players {
			if strings.ToLower(p.PUUID) != id {
				continue
			}
			got := false
			if v, ok := statNum(p.HS); ok {
				hs += v
				got = true
			}
			if v, ok := statNum(p.ADR); ok {
				adr += v
				got = true
			}
			if v, ok := statNum(p.ACS); ok {
				acs += v
				got = true
			}
			if !got {
				break
			}
			n++
			break
		}
		if n >= 2 {
			break
		}
	}
	if n == 0 {
		return info, false
	}
	info.HS = strconv.Itoa(hs/n) + "%"
	info.ADR = strconv.Itoa(adr / n)
	info.ACS = strconv.Itoa(acs / n)
	return info, true
}

func statNum(raw string) (int, bool) {
	raw = strings.TrimSpace(strings.TrimSuffix(raw, "%"))
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	return n, err == nil
}

func (c *riotClient) ensureRanks(ctx context.Context) {
	if len(c.rankIcons) > 0 {
		return
	}
	defer func() {
		if len(c.rankIcons) == 0 {
			c.mu.Lock()
			if len(c.rankIcons) == 0 {
				c.rankIcons = localRankIcons()
			}
			c.mu.Unlock()
		}
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://valorant-api.com/v1/competitivetiers", nil)
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
	tables := asArr(pick(body, "data"))
	if len(tables) == 0 {
		return
	}
	latest := asMap(tables[len(tables)-1])
	icons := map[int]string{}
	for _, item := range asArr(pick(latest, "tiers")) {
		row := asMap(item)
		tier := num(pick(row, "tier"))
		icon := str(pick(row, "largeIcon"))
		if tier > 0 && icon != "" {
			icons[tier] = icon
		}
	}
	if len(icons) > 0 {
		c.rankIcons = icons
	}
}

package main

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

type premierMate struct {
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
	Role string `json:"role,omitempty"`
}

type premierView struct {
	Name     string        `json:"name,omitempty"`
	Tag      string        `json:"tag,omitempty"`
	Division string        `json:"division,omitempty"`
	Points   int           `json:"points,omitempty"`
	When     string        `json:"when,omitempty"`
	Maps     string        `json:"maps,omitempty"`
	Roster   []premierMate `json:"roster,omitempty"`
	Error    string        `json:"error,omitempty"`
}

func (c *riotClient) premierTeam(ctx context.Context) premierView {
	sess := c.session
	if sess.puuid == "" || sess.shard == "" || sess.access == "" {
		return premierView{Error: "Open the Riot Client"}
	}
	urls := []string{
		"https://pd." + sess.shard + ".a.pvp.net/premier/v1/teams/player/" + sess.puuid,
		"https://pd." + sess.shard + ".a.pvp.net/premier/v1/player/" + sess.puuid,
	}
	var payload []byte
	for _, url := range urls {
		status, body, err := c.remoteCall(ctx, sess, "GET", url, nil)
		if err != nil || status >= 400 || len(body) < 8 {
			continue
		}
		payload = body
		break
	}
	if len(payload) == 0 {
		return premierView{Error: "No Premier team on this client"}
	}
	var root any
	if json.Unmarshal(payload, &root) != nil {
		return premierView{Error: "No Premier team on this client"}
	}
	view := premierView{}
	readPremier(root, &view)
	if view.Name == "" && len(view.Roster) == 0 {
		return premierView{Error: "No Premier team on this client"}
	}
	return view
}

func readPremier(node any, view *premierView) {
	switch n := node.(type) {
	case []any:
		for _, item := range n {
			readPremier(item, view)
		}
	case map[string]any:
		if view.Name == "" {
			view.Name = str(pick(n, "teamName", "TeamName"))
		}
		if view.Name == "" && len(asArr(pick(n, "members", "Members", "roster", "Roster"))) > 0 {
			view.Name = str(pick(n, "name", "Name"))
		}
		if view.Tag == "" {
			view.Tag = str(pick(n, "teamTag", "TeamTag", "tag", "Tag"))
		}
		if view.Division == "" {
			view.Division = str(pick(n, "divisionName", "DivisionName", "division", "Division"))
		}
		if view.Points == 0 {
			view.Points = num(pick(n, "points", "Points", "score", "Score"))
		}
		if view.When == "" {
			view.When = premierWhen(pick(n, "nextMatchTime", "scheduledMatchTime", "matchStartTime", "startTime"))
		}
		if view.Maps == "" {
			view.Maps = premierMapList(pick(n, "mapPool", "MapPool"))
		}
		for _, item := range asArr(pick(n, "members", "Members", "roster", "Roster")) {
			row := asMap(item)
			if row == nil {
				continue
			}
			name := str(pick(row, "gameName", "GameName", "name", "Name"))
			tag := str(pick(row, "tagLine", "TagLine", "tag", "Tag"))
			role := str(pick(row, "role", "Role"))
			if name == "" && tag == "" && role == "" {
				continue
			}
			view.Roster = append(view.Roster, premierMate{Name: name, Tag: tag, Role: role})
		}
		for _, value := range n {
			switch child := value.(type) {
			case map[string]any, []any:
				readPremier(child, view)
			}
		}
	}
	if len(view.Roster) > 10 {
		view.Roster = view.Roster[:10]
	}
	seen := map[string]bool{}
	kept := view.Roster[:0]
	for _, mate := range view.Roster {
		key := strings.ToLower(mate.Name + "#" + mate.Tag + "#" + mate.Role)
		if key == "##" || seen[key] {
			continue
		}
		seen[key] = true
		kept = append(kept, mate)
	}
	view.Roster = kept
}

func premierWhen(node any) string {
	raw := strings.TrimSpace(str(node))
	if raw == "" {
		return ""
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
		if n < 1_000_000_000_000 {
			n *= 1000
		}
		return time.UnixMilli(n).Local().Format("Mon 15:04")
	}
	if t, err := time.Parse(time.RFC3339, raw); err == nil {
		return t.Local().Format("Mon 15:04")
	}
	if len(raw) > 42 {
		return ""
	}
	return raw
}

func premierMapList(node any) string {
	items := asArr(node)
	if len(items) == 0 {
		if row := asMap(node); row != nil {
			items = asArr(pick(row, "maps", "Maps", "mapPool", "MapPool"))
		}
	}
	names := []string{}
	for _, item := range items {
		name := str(item)
		if name == "" {
			row := asMap(item)
			name = str(pick(row, "name", "Name", "map", "Map"))
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		names = append(names, prettyMap(name))
		if len(names) == 7 {
			break
		}
	}
	return strings.Join(names, ", ")
}

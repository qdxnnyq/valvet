package main

import (
	"encoding/base64"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

func stacksFromArchive(rows []recentMatch, players []player) map[string]partyHint {
	live := map[string]string{}
	for _, p := range players {
		id := strings.ToLower(p.PUUID)
		if looksUUID(id) {
			live[id] = p.Team
		}
	}
	for _, row := range rows {
		groups := map[string][]string{}
		for _, p := range row.Players {
			id := strings.ToLower(p.PUUID)
			if !looksUUID(id) || p.Party == "" || live[id] == "" {
				continue
			}
			groups[p.Party] = append(groups[p.Party], id)
		}
		out := map[string]partyHint{}
		for label, ids := range groups {
			if len(ids) < 2 || len(ids) > 5 {
				continue
			}
			team := ""
			ok := true
			seen := map[string]bool{}
			for _, id := range ids {
				if seen[id] || live[id] == "" || (team != "" && live[id] != team) {
					ok = false
					break
				}
				seen[id] = true
				team = live[id]
			}
			if !ok {
				continue
			}
			for id := range seen {
				out[id] = partyHint{id: "arch-" + strings.ToLower(label), size: len(seen)}
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return map[string]partyHint{}
}

func partyGrouped(hints map[string]partyHint) bool {
	count := map[string]int{}
	for _, hint := range hints {
		if hint.id == "" {
			continue
		}
		count[hint.id]++
		if count[hint.id] >= 2 {
			return true
		}
	}
	return false
}

func sharedHints(in map[string]partyHint) map[string]partyHint {
	count := map[string]int{}
	for _, hint := range in {
		if hint.id != "" {
			count[hint.id]++
		}
	}
	out := map[string]partyHint{}
	for id, hint := range in {
		if count[hint.id] >= 2 {
			out[id] = hint
		}
	}
	return out
}

func groupsOf(in map[string]partyHint) map[string]partyHint {
	count := map[string]int{}
	for _, hint := range in {
		if hint.id != "" {
			count[hint.id]++
		}
	}
	out := map[string]partyHint{}
	for id, hint := range in {
		n := count[hint.id]
		if n >= 2 && n <= 5 {
			out[id] = hint
		}
	}
	return out
}

func overlayParties(dst, src map[string]partyHint) {
	if dst == nil || len(src) == 0 {
		return
	}
	src = groupsOf(src)
	byID := map[string][]string{}
	for id, hint := range src {
		if hint.id == "" {
			continue
		}
		byID[hint.id] = append(byID[hint.id], id)
	}
	for partyID, members := range byID {
		if len(members) < 2 || len(members) > 5 {
			continue
		}
		for _, id := range members {
			dst[id] = partyHint{id: partyID, size: len(members)}
		}
	}
}

func fillGroups(dst, src map[string]partyHint) {
	if dst == nil || len(src) == 0 {
		return
	}
	src = groupsOf(src)
	byID := map[string][]string{}
	for id, hint := range src {
		byID[hint.id] = append(byID[hint.id], id)
	}
	for partyID, members := range byID {
		blocked := false
		for _, id := range members {
			if prev := dst[id]; prev.id != "" && prev.id != partyID {
				blocked = true
				break
			}
		}
		if blocked {
			continue
		}
		for _, id := range members {
			dst[id] = partyHint{id: partyID, size: len(members)}
		}
	}
}

func forceParty(dst map[string]partyHint, partyID string, members []string, roster map[string]bool) {
	if dst == nil || partyID == "" {
		return
	}
	partyID = strings.ToLower(partyID)
	in := make([]string, 0, len(members))
	seen := map[string]bool{}
	for _, id := range members {
		id = strings.ToLower(id)
		if !looksUUID(id) || !roster[id] || seen[id] {
			continue
		}
		seen[id] = true
		in = append(in, id)
	}
	free := make([]string, 0, len(in))
	for _, id := range in {
		if prev := dst[id]; prev.id != "" && prev.id != partyID {
			continue
		}
		free = append(free, id)
	}
	if len(free) < 2 || len(free) > 5 {
		return
	}
	for _, id := range free {
		dst[id] = partyHint{id: partyID, size: len(free)}
	}
}

func pinMembers(dst map[string]partyHint, partyID string, members []string, roster map[string]bool) {
	if dst == nil || partyID == "" {
		return
	}
	partyID = strings.ToLower(partyID)
	in := make([]string, 0, len(members))
	seen := map[string]bool{}
	for _, id := range members {
		id = strings.ToLower(id)
		if !looksUUID(id) || !roster[id] || seen[id] {
			continue
		}
		if prev := dst[id]; prev.id != "" && prev.id != partyID {
			continue
		}
		seen[id] = true
		in = append(in, id)
	}
	if len(in) < 2 || len(in) > 5 {
		return
	}
	for _, id := range in {
		dst[id] = partyHint{id: partyID, size: len(in)}
	}
}

func partyMemberIDs(body map[string]any) []string {
	if body == nil {
		return nil
	}
	rows := asArr(pick(body, "Members", "members"))
	if len(rows) == 0 {
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, item := range rows {
		row := asMap(item)
		id := strings.ToLower(str(pick(row, "Subject", "subject", "puuid", "PUUID")))
		if looksUUID(id) {
			out = append(out, id)
		}
	}
	return out
}

func stackHints(nowTeam map[string]string, groups [][]string) map[string]partyHint {
	taken := map[string]bool{}
	out := map[string]partyHint{}
	n := 0
	for _, group := range groups {
		seen := map[string]bool{}
		list := []string{}
		team := ""
		ok := true
		for _, raw := range group {
			id := strings.ToLower(raw)
			if id == "" || seen[id] || taken[id] {
				ok = false
				break
			}
			if nowTeam != nil {
				if nowTeam[id] == "" || (team != "" && nowTeam[id] != team) {
					ok = false
					break
				}
				team = nowTeam[id]
			}
			seen[id] = true
			list = append(list, id)
		}
		if !ok || len(list) < 2 || len(list) > 5 {
			continue
		}
		n++
		label := "stack-" + strconv.Itoa(n)
		for _, id := range list {
			taken[id] = true
			out[id] = partyHint{id: label, size: len(list)}
		}
	}
	return out
}

func markParties(players []player, hints map[string]partyHint) {
	groups := map[string][]int{}
	for i, p := range players {
		hint, ok := hints[strings.ToLower(p.PUUID)]
		if !ok || hint.id == "" {
			continue
		}
		groups[hint.id] = append(groups[hint.id], i)
	}
	ids := make([]string, 0, len(groups))
	for id, idxs := range groups {
		if len(idxs) < 2 || len(idxs) > 5 {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		you := func(id string) bool {
			for _, idx := range groups[id] {
				if players[idx].You {
					return true
				}
			}
			return false
		}
		if you(ids[i]) != you(ids[j]) {
			return you(ids[i])
		}
		return ids[i] < ids[j]
	})
	for n, id := range ids {
		label := strconv.Itoa(n + 1)
		if n < 26 {
			label = string(rune('A' + n))
		}
		for _, i := range groups[id] {
			players[i].Party = label
			players[i].PartyN = len(groups[id])
		}
	}
}

func harvestParties(node any, acc map[string]string) {
	switch n := node.(type) {
	case []any:
		for _, item := range n {
			harvestParties(item, acc)
		}
	case map[string]any:
		id := strings.ToLower(playerID(n))
		if !looksUUID(id) {
			id = strings.ToLower(str(pick(n, "Subject", "subject", "puuid", "PUUID")))
		}
		if party := partyKey(n); looksUUID(id) && party != "" {
			acc[id] = party
		}
		for _, value := range n {
			switch child := value.(type) {
			case map[string]any, []any:
				harvestParties(child, acc)
			}
		}
	}
}

func hintsFromIDs(ids map[string]string) map[string]partyHint {
	out := map[string]partyHint{}
	for id, party := range ids {
		if party != "" {
			out[id] = partyHint{id: strings.ToLower(party)}
		}
	}
	return sharedHints(out)
}

func partyKey(m map[string]any) string {
	if m == nil {
		return ""
	}
	if id := str(pick(m, "PartyID", "partyId", "PartyId", "CurrentPartyID", "currentPartyID")); looksUUID(id) {
		return strings.ToLower(id)
	}
	for key, value := range m {
		if !strings.Contains(strings.ToLower(key), "party") {
			continue
		}
		if id := str(value); looksUUID(id) {
			return strings.ToLower(id)
		}
		if child := asMap(value); child != nil {
			if id := str(pick(child, "ID", "id", "PartyID", "partyId")); looksUUID(id) {
				return strings.ToLower(id)
			}
		}
	}
	return ""
}

func partyHintsFrom(players []player) map[string]partyHint {
	hints := map[string]partyHint{}
	for _, p := range players {
		if p.PUUID == "" || p.PartyID == "" {
			continue
		}
		hints[strings.ToLower(p.PUUID)] = partyHint{id: strings.ToLower(p.PartyID)}
	}
	return hints
}

func partyFields(m map[string]any) (string, int) {
	if m == nil {
		return "", 0
	}
	id := str(pick(m, "partyId", "partyID", "PartyID", "PartyId", "CurrentPartyID", "currentPartyID", "party_id"))
	if !looksUUID(id) {
		return "", 0
	}
	return strings.ToLower(id), num(pick(m, "partySize", "PartySize", "party_size"))
}

func jsonObject(v any) map[string]any {
	if m := asMap(v); m != nil {
		return m
	}
	s, ok := v.(string)
	if !ok {
		return nil
	}
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if strings.HasPrefix(s, "{") {
		var out map[string]any
		if json.Unmarshal([]byte(s), &out) == nil {
			return out
		}
	}
	return decodePrivate(s)
}

func findParty(v any) (string, int) {
	if m := jsonObject(v); m != nil {
		for _, key := range []string{"partyPresenceData", "PartyPresenceData"} {
			if id, size := partyFields(jsonObject(m[key])); id != "" {
				return id, size
			}
		}
	}
	return walkParty(v)
}

func walkParty(v any) (string, int) {
	if m := jsonObject(v); m != nil {
		if id, size := partyFields(m); id != "" {
			return id, size
		}
		for _, child := range m {
			if id, size := walkParty(child); id != "" {
				return id, size
			}
		}
	}
	if arr := asArr(v); len(arr) > 0 {
		for _, child := range arr {
			if id, size := walkParty(child); id != "" {
				return id, size
			}
		}
	}
	return "", 0
}

func decodePrivate(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	if strings.HasPrefix(raw, "{") {
		var out map[string]any
		if json.Unmarshal([]byte(raw), &out) == nil {
			return out
		}
	}
	if strings.Count(raw, ".") >= 2 {
		parts := strings.Split(raw, ".")
		order := []int{}
		if len(parts) > 1 {
			order = append(order, 1)
		}
		for i := range parts {
			if i != 1 {
				order = append(order, i)
			}
		}
		for _, i := range order {
			if m := decodeB64JSON(parts[i]); m != nil {
				return m
			}
		}
	}
	return decodeB64JSON(raw)
}

func decodeB64JSON(raw string) map[string]any {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var data []byte
	var err error
	padded := raw
	if m := len(padded) % 4; m != 0 {
		padded += strings.Repeat("=", 4-m)
	}
	for _, dec := range []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.RawStdEncoding, base64.StdEncoding} {
		data, err = dec.DecodeString(raw)
		if err == nil {
			break
		}
		data, err = dec.DecodeString(padded)
		if err == nil {
			break
		}
	}
	if err != nil || len(data) == 0 || data[0] != '{' {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}

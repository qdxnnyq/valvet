package main

import (
	"encoding/json"
	"strings"
)

func prettyMode(queue string) string {
	key := strings.ToLower(strings.TrimSpace(queue))
	names := map[string]string{
		"competitive": "Competitive", "unrated": "Unrated", "swiftplay": "Swiftplay",
		"spikerush": "Spike Rush", "ggteam": "Escalation", "deathmatch": "Deathmatch",
		"hurm": "Team Deathmatch", "premier": "Premier", "custom": "Custom",
		"onefa": "Replication", "newmap": "New Map", "snowball": "Snowball",
		"skirmish": "Skirmish", "gauntlet": "Gauntlet", "glitched": "Gauntlet",
		"practice": "Practice",
	}
	if name, ok := names[key]; ok {
		return name
	}
	return "Match"
}

func sidesSwapped(queue string, rounds int) bool {
	low := strings.ToLower(queue)
	switch low {
	case "hurm", "ggteam", "practice", "snowball", "deathmatch":
		return false
	}
	if rounds <= 0 {
		return false
	}
	half := 12
	switch low {
	case "spikerush":
		half = 3
	case "swiftplay":
		half = 4
	}
	if rounds < half {
		return false
	}
	if rounds < half*2 {
		return true
	}
	return (rounds-half*2)%2 == 1
}

func ffaQueue(queue string) bool {
	low := strings.ToLower(queue)
	return low == "deathmatch" || low == "practice"
}

func gauntletQueue(queue string) bool {
	low := strings.ToLower(queue)
	return strings.Contains(low, "gauntlet") || strings.Contains(low, "glitched")
}

func rawTeamQueue(queue string) bool {
	switch strings.ToLower(queue) {
	case "hurm", "ggteam", "gauntlet", "glitched", "skirmish", "snowball":
		return true
	default:
		return false
	}
}

func teamSide(team, queue string, rounds int) string {
	if ffaQueue(queue) {
		return prettyMode(queue)
	}
	if rawTeamQueue(queue) {
		if strings.TrimSpace(team) == "" {
			return "Team"
		}
		return team
	}
	low := strings.ToLower(team)
	var side string
	switch low {
	case "red":
		side = "Attackers"
	case "blue":
		side = "Defenders"
	default:
		if strings.TrimSpace(team) == "" {
			return "Lobby"
		}
		return team
	}
	if sidesSwapped(queue, rounds) {
		if side == "Attackers" {
			return "Defenders"
		}
		return "Attackers"
	}
	return side
}

func prettyServer(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	parts := strings.FieldsFunc(text, func(r rune) bool { return r == '.' || r == '-' || r == '_' })
	skip := map[string]bool{"prod": true, "aws": true, "gp": true, "aresriot": true, "1": true, "2": true, "3": true, "4": true}
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if skip[part] || len(part) < 4 {
			continue
		}
		if strings.HasPrefix(part, "eu") || strings.HasPrefix(part, "na") || strings.HasPrefix(part, "ap") || strings.HasPrefix(part, "kr") || strings.HasPrefix(part, "br") || strings.HasPrefix(part, "latam") {
			continue
		}
		return strings.ToUpper(part[:1]) + part[1:]
	}
	if text == "" {
		return ""
	}
	return raw
}

func prettyMap(path string) string {
	text := strings.Trim(strings.TrimSpace(path), "/")
	if text == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(text, "\\", "/"), "/")
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			clean = append(clean, part)
		}
	}
	if len(clean) == 0 {
		return ""
	}
	if nice, ok := mapLabel(clean[len(clean)-1]); ok {
		return nice
	}
	if len(clean) >= 2 {
		if nice, ok := mapLabel(clean[len(clean)-2]); ok {
			return nice
		}
	}
	name := clean[len(clean)-1]
	return strings.ToUpper(name[:1]) + name[1:]
}

var mapNames = map[string]string{
	"ascent": "Ascent", "bind": "Bind", "haven": "Haven", "split": "Split", "icebox": "Icebox",
	"breeze": "Breeze", "fracture": "Fracture", "pearl": "Pearl", "lotus": "Lotus", "sunset": "Sunset",
	"abyss": "Abyss", "corrode": "Corrode", "summit": "Summit",
	"bonsai": "Split", "canyon": "Fracture", "duality": "Bind", "foxtrot": "Breeze",
	"infinity": "Abyss", "jam": "Lotus", "juliett": "Sunset", "pitt": "Pearl", "port": "Icebox",
	"triad": "Haven", "rook": "Corrode", "plummet": "Summit",
	"range": "Range", "poveglia": "The Range", "povegliav2": "The Range", "rangev2": "The Range",
	"basictraining": "The Range", "npev2": "Basic Training",
	"hurm_alley": "District", "hurm_bowl": "Kasbah", "hurm_helix": "Drift",
	"hurm_hightide": "Glitch", "hurm_yard": "Piazza",
	"district": "District", "kasbah": "Kasbah", "drift": "Drift", "glitch": "Glitch", "piazza": "Piazza",
}

func mapNamesJSON() []byte {
	raw, err := json.Marshal(mapNames)
	if err != nil || len(raw) == 0 {
		return []byte("{}")
	}
	return raw
}

func mapLabel(name string) (string, bool) {
	nice, ok := mapNames[strings.ToLower(name)]
	return nice, ok
}

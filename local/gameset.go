package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type gameRow struct {
	Group string `json:"group"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type gameView struct {
	Rows   []gameRow `json:"rows"`
	Text   string    `json:"text"`
	Source string    `json:"source,omitempty"`
	Error  string    `json:"error,omitempty"`
}

func (c *riotClient) gameSettings(ctx context.Context) gameView {
	riotText, gameText := readValorantConfig(filepath.Join(os.Getenv("LOCALAPPDATA"), "VALORANT", "Saved", "Config"))
	c.mu.Lock()
	sess := c.session
	c.mu.Unlock()
	account := ""
	if sess.access != "" && !strings.Contains(strings.ToLower(riotText), "mousesensitivity") {
		account = c.preferenceText(ctx, sess.access)
	}
	source := ""
	switch {
	case strings.TrimSpace(riotText) != "":
		source = "Newest Riot profile on this PC"
	case strings.TrimSpace(account) != "":
		source = "Riot account"
	}
	rows := extractGameRowsFrom(riotText, gameText, account)
	if len(rows) == 0 {
		return gameView{Rows: []gameRow{}, Error: "Open Valorant once on this PC. Graphics and sensitivity come from the newest Riot profile, not the Unreal default."}
	}
	var lines []string
	group := ""
	for _, row := range rows {
		if row.Group != group {
			if group != "" {
				lines = append(lines, "")
			}
			lines = append(lines, row.Group)
			group = row.Group
		}
		lines = append(lines, row.Label+": "+row.Value)
	}
	return gameView{Rows: rows, Text: strings.Join(lines, "\n"), Source: source}
}

func readValorantConfig(root string) (string, string) {
	type hit struct {
		dir  string
		when time.Time
		body string
	}
	var best *hit
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() || info.Size() > 2<<20 {
			return nil
		}
		if !strings.EqualFold(info.Name(), "RiotUserSettings.ini") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		body := fileText(b)
		if !strings.Contains(strings.ToLower(body), "mousesensitivity") {
			return nil
		}
		if best == nil || info.ModTime().After(best.when) {
			best = &hit{dir: filepath.Dir(path), when: info.ModTime(), body: body}
		}
		return nil
	})
	if best == nil {
		return "", ""
	}
	game := ""
	if b, err := os.ReadFile(filepath.Join(best.dir, "GameUserSettings.ini")); err == nil {
		game = fileText(b)
	}
	return best.body, game
}

type settingWant struct {
	key   string
	label string
	group string
	kind  string
}

func extractGameRows(text string) []gameRow {
	return extractGameRowsFrom(text, "", "")
}

func extractGameRowsFrom(riot, game, account string) []gameRow {
	if strings.TrimSpace(riot) == "" && strings.TrimSpace(game) == "" && strings.TrimSpace(account) == "" {
		return nil
	}
	wants := []settingWant{
		{"MouseSensitivity", "Sensitivity", "Mouse", "sens"},
		{"ScopedSensitivityMultiplier", "Scoped sensitivity", "Mouse", "sens"},
		{"ScopedSensitivity", "Scoped sensitivity", "Mouse", "sens"},
		{"ADSSensitivityMultiplier", "ADS sensitivity", "Mouse", "sens"},
		{"ADSSensitivity", "ADS sensitivity", "Mouse", "sens"},
		{"ZoomedSensitivity", "Zoom sensitivity", "Mouse", "sens"},
		{"FocusModeSensitivity", "Focus sensitivity", "Mouse", "sens"},
		{"ResolutionSizeX", "Resolution width", "Graphics", "skip"},
		{"ResolutionSizeY", "Resolution height", "Graphics", "skip"},
		{"LastConfirmedFullscreenMode", "Display mode", "Graphics", "screen"},
		{"FullscreenMode", "Display mode", "Graphics", "screen"},
		{"FrameRateLimit", "Frame limit", "Graphics", "fps"},
		{"bUseVSync", "VSync", "Graphics", "bool"},
		{"sg.ViewDistanceQuality", "View distance", "Graphics", "quality"},
		{"sg.AntiAliasingQuality", "Anti-aliasing", "Graphics", "quality"},
		{"sg.ShadowQuality", "Shadows", "Graphics", "quality"},
		{"sg.GlobalIlluminationQuality", "Global illumination", "Graphics", "quality"},
		{"sg.ReflectionQuality", "Reflections", "Graphics", "quality"},
		{"sg.PostProcessQuality", "Post processing", "Graphics", "quality"},
		{"sg.TextureQuality", "Textures", "Graphics", "quality"},
		{"sg.EffectsQuality", "Effects", "Graphics", "quality"},
		{"sg.FoliageQuality", "Foliage", "Graphics", "quality"},
		{"sg.ShadingQuality", "Shading", "Graphics", "quality"},
		{"bUseNVIDIAReflex", "NVIDIA Reflex", "Graphics", "bool"},
		{"ReflexMode", "NVIDIA Reflex", "Graphics", "reflex"},
	}
	aresLocal := aresValues(riot)
	aresAccount := aresValues(account)
	found := map[string]string{}
	for _, want := range wants {
		if want.kind == "skip" {
			continue
		}
		if _, ok := found[want.label]; ok {
			continue
		}
		raw := ""
		if want.kind == "sens" {
			key := strings.ToLower(want.key)
			// The slider is EAresFloatSettingName::MouseSensitivity in
			// RiotUserSettings.ini. GameUserSettings.ini also has
			// MouseSensitivity, and that one is the Unreal default.
			if v := aresLocal[key]; v != "" {
				raw = v
			} else if v := aresAccount[key]; v != "" {
				raw = v
			}
		} else {
			raw = valueAfterKey(game, want.key)
			if raw == "" {
				raw = valueAfterKey(account, want.key)
			}
		}
		if raw == "" {
			continue
		}
		value := formatSetting(want.kind, raw)
		if value == "" {
			continue
		}
		found[want.label] = value
	}
	if x, y := confirmedRes(game); x != "" && y != "" {
		found["Resolution"] = trimNum(x) + " × " + trimNum(y)
	} else if x, y := confirmedRes(account); x != "" && y != "" {
		found["Resolution"] = trimNum(x) + " × " + trimNum(y)
	}
	order := []settingWant{
		{"", "Sensitivity", "Mouse", ""},
		{"", "Scoped sensitivity", "Mouse", ""},
		{"", "ADS sensitivity", "Mouse", ""},
		{"", "Zoom sensitivity", "Mouse", ""},
		{"", "Focus sensitivity", "Mouse", ""},
		{"", "Resolution", "Graphics", ""},
		{"", "Display mode", "Graphics", ""},
		{"", "Frame limit", "Graphics", ""},
		{"", "VSync", "Graphics", ""},
		{"", "NVIDIA Reflex", "Graphics", ""},
		{"", "View distance", "Graphics", ""},
		{"", "Anti-aliasing", "Graphics", ""},
		{"", "Shadows", "Graphics", ""},
		{"", "Global illumination", "Graphics", ""},
		{"", "Reflections", "Graphics", ""},
		{"", "Post processing", "Graphics", ""},
		{"", "Textures", "Graphics", ""},
		{"", "Effects", "Graphics", ""},
		{"", "Foliage", "Graphics", ""},
		{"", "Shading", "Graphics", ""},
	}
	var out []gameRow
	for _, want := range order {
		value := found[want.label]
		if value == "" {
			continue
		}
		out = append(out, gameRow{Group: want.group, Label: want.label, Value: value})
	}
	return out
}

func confirmedRes(text string) (string, string) {
	x := valueAfterKey(text, "LastUserConfirmedResolutionSizeX")
	y := valueAfterKey(text, "LastUserConfirmedResolutionSizeY")
	if x != "" && y != "" {
		return x, y
	}
	return valueAfterKey(text, "ResolutionSizeX"), valueAfterKey(text, "ResolutionSizeY")
}

// aresFloatRE matches the slider lines in RiotUserSettings.ini.
// Riot writes a space after the colons: "EAresFloatSettingName:: MouseSensitivity =0.272".
var aresFloatRE = regexp.MustCompile(`(?i)EAres(?:Float|String|Int)?Setting(?:Name)?\s*::\s*([A-Za-z0-9_]+)\s*[:=]\s*([-+0-9.]+)`)

func aresValues(text string) map[string]string {
	out := map[string]string{}
	if strings.TrimSpace(text) == "" {
		return out
	}
	for _, match := range aresFloatRE.FindAllStringSubmatch(text, -1) {
		out[strings.ToLower(match[1])] = match[2]
	}
	return out
}

func valueAfterKey(text, key string) string {
	lower := strings.ToLower(text)
	needle := strings.ToLower(key)
	from := 0
	for {
		i := strings.Index(lower[from:], needle)
		if i < 0 {
			return ""
		}
		i += from
		end := i + len(needle)
		if i > 0 {
			c := lower[i-1]
			if (c >= 'a' && c <= 'z') || c == '.' {
				from = end
				continue
			}
		}
		if end < len(lower) {
			c := lower[end]
			if c >= 'a' && c <= 'z' {
				from = end
				continue
			}
		}
		rest := text[end:]
		if len(rest) > 96 {
			rest = rest[:96]
		}
		rest = strings.TrimLeft(rest, " \t=:\"'")
		cut := strings.IndexAny(rest, "\r\n,}\"'")
		if cut >= 0 {
			rest = rest[:cut]
		}
		rest = strings.Trim(strings.TrimSpace(rest), `"'`)
		if rest != "" {
			return rest
		}
		from = end
	}
}

func formatSetting(kind, raw string) string {
	switch kind {
	case "bool":
		if truthy(raw) {
			return "On"
		}
		return "Off"
	case "quality":
		n, err := strconv.Atoi(trimNum(raw))
		if err != nil {
			return raw
		}
		switch n {
		case 0:
			return "Low"
		case 1:
			return "Medium"
		case 2:
			return "High"
		case 3:
			return "Epic"
		default:
			return raw
		}
	case "screen":
		switch trimNum(raw) {
		case "0":
			return "Fullscreen"
		case "1":
			return "Borderless"
		case "2":
			return "Windowed"
		default:
			return raw
		}
	case "fps":
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return raw
		}
		if n <= 0 {
			return "Unlimited"
		}
		return strconv.Itoa(int(n))
	case "sens":
		n, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return ""
		}
		if n > 10 && n <= 10000 {
			n = n / 1000
		}
		if n < 0.001 || n > 10 {
			return ""
		}
		return trimSens(n)
	case "reflex":
		switch trimNum(raw) {
		case "0":
			return "Off"
		case "1":
			return "On"
		case "2":
			return "On + Boost"
		default:
			return raw
		}
	default:
		return trimFloatText(raw)
	}
}

func trimSens(n float64) string {
	s := strconv.FormatFloat(n, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	if s == "" || s == "-0" {
		return "0"
	}
	return s
}

func truthy(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}

func trimNum(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.IndexByte(raw, '.'); i >= 0 {
		whole := raw[:i]
		if _, err := strconv.Atoi(whole); err == nil {
			return whole
		}
	}
	return raw
}

func trimFloatText(raw string) string {
	raw = strings.TrimSpace(raw)
	n, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return raw
	}
	return trimFloat(n)
}

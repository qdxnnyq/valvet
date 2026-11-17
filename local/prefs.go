package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
)

var lookupPause atomic.Bool

func init() {
	bag := readPrefs()
	lookupPause.Store(bag["valvet-pause-lookups"] == "on" || bag["velvet-pause-lookups"] == "on")
}

func lookupsPaused() bool {
	return lookupPause.Load()
}

func setLookupPause(on bool) {
	lookupPause.Store(on)
	bag := readPrefs()
	if on {
		bag["valvet-pause-lookups"] = "on"
	} else {
		delete(bag, "valvet-pause-lookups")
	}
	delete(bag, "velvet-pause-lookups")
	if len(bag) == 0 {
		bag["valvet-pause-lookups"] = "off"
	}
	_ = writePrefs(bag)
}

func prefsPath() string {
	base := os.Getenv("APPDATA")
	if base == "" {
		if home, err := os.UserHomeDir(); err == nil {
			base = home
		} else {
			base = os.TempDir()
		}
	}
	migrateAppDirs()
	next := filepath.Join(base, "Valvet", "settings.json")
	prev := filepath.Join(base, "Velvet", "settings.json")
	if _, err := os.Stat(next); err == nil {
		return next
	}
	if _, err := os.Stat(prev); err == nil {
		return prev
	}
	_ = os.MkdirAll(filepath.Dir(next), 0o700)
	return next
}

func canonPref(key string) (string, bool) {
	switch {
	case strings.HasPrefix(key, "valvet-") && len(key) < 80:
		return key, true
	case strings.HasPrefix(key, "velvet-") && len(key) < 80:
		return "valvet-" + strings.TrimPrefix(key, "velvet-"), true
	default:
		return "", false
	}
}

func readPrefs() map[string]string {
	out := map[string]string{}
	raw, err := os.ReadFile(prefsPath())
	if err != nil {
		return out
	}
	var bag map[string]string
	if json.Unmarshal(raw, &bag) != nil {
		return out
	}
	renamed := false
	for key, value := range bag {
		if len(value) >= 8000 {
			continue
		}
		next, ok := canonPref(key)
		if !ok {
			continue
		}
		if _, exists := out[next]; !exists || strings.HasPrefix(key, "valvet-") {
			out[next] = value
		}
		if next != key {
			renamed = true
		}
	}
	if renamed {
		_ = writePrefs(out)
	}
	return out
}

func writePrefs(bag map[string]string) error {
	clean := map[string]string{}
	for key, value := range bag {
		next, ok := canonPref(key)
		if !ok || len(value) >= 8000 {
			continue
		}
		clean[next] = value
		if len(clean) >= 200 {
			break
		}
	}
	if len(clean) == 0 {
		return nil
	}
	dir := filepath.Dir(prefsPath())
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(prefsPath(), raw, 0o600)
}

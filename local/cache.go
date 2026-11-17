package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func cacheDir() string {
	return filepath.Join(velvetBase(), "cache")
}

func cacheKey(raw string) string {
	sum := sha1.Sum([]byte(raw))
	return hex.EncodeToString(sum[:8])
}

func readCache(name string, dest any) (time.Duration, bool) {
	path := filepath.Join(cacheDir(), name+".json")
	info, err := os.Stat(path)
	if err != nil {
		return 0, false
	}
	raw, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(raw, dest) != nil {
		return 0, false
	}
	return time.Since(info.ModTime()), true
}

func writeCache(name string, v any) {
	raw, err := json.Marshal(v)
	if err != nil || len(raw) < 3 {
		return
	}
	dir := cacheDir()
	if os.MkdirAll(dir, 0o700) != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(dir, name+".json"), raw, 0o600)
}

var (
	cacheMu   sync.Mutex
	cacheBusy = map[string]bool{}
)

func cacheOnce(name string, fn func()) {
	cacheMu.Lock()
	if cacheBusy[name] {
		cacheMu.Unlock()
		return
	}
	cacheBusy[name] = true
	cacheMu.Unlock()
	go func() {
		defer func() {
			cacheMu.Lock()
			delete(cacheBusy, name)
			cacheMu.Unlock()
		}()
		fn()
	}()
}

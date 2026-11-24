package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type storageView struct {
	OK      bool   `json:"ok"`
	Label   string `json:"label"`
	Detail  string `json:"detail"`
	Cleared string `json:"cleared,omitempty"`
	Videos  string `json:"videos,omitempty"`
}

func formatBytes(n int64) string {
	if n < 0 {
		n = 0
	}
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	case n < 1024*1024*1024:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	default:
		return fmt.Sprintf("%.2f GB", float64(n)/(1024*1024*1024))
	}
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

func dirSize(root string) int64 {
	var n int64
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		n += info.Size()
		return nil
	})
	return n
}

func sameDir(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	aa, errA := filepath.Abs(a)
	bb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
	}
	return strings.EqualFold(aa, bb)
}

func storagePaths() (base, web, pics string) {
	base = velvetBase()
	if app := os.Getenv("APPDATA"); app != "" {
		web = filepath.Join(app, "Valvet")
		if sameDir(web, base) {
			web = ""
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		pics = filepath.Join(home, "Pictures", "Valvet")
	}
	return base, web, pics
}

func storageNow() storageView {
	base, web, pics := storagePaths()
	games := fileSize(filepath.Join(base, "records.json"))
	cache := dirSize(cacheDir())
	all := dirSize(base)
	webN := dirSize(web)
	picN := dirSize(pics)
	vidN := int64(0)
	vidCount := 0
	if videos, err := roundVideoDir(); err == nil {
		vidN = dirSize(videos)
		entries, _ := os.ReadDir(videos)
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(entry.Name()))
			if ext == ".mp4" || ext == ".webm" {
				vidCount++
			}
		}
	}
	total := all + webN + picN + vidN
	parts := []string{}
	if games > 0 {
		parts = append(parts, "Games "+formatBytes(games))
	}
	if cache > 0 {
		parts = append(parts, "Cache "+formatBytes(cache))
	}
	if webN > 0 {
		parts = append(parts, "App "+formatBytes(webN))
	}
	if picN > 0 {
		parts = append(parts, "Pictures "+formatBytes(picN))
	}
	if vidN > 0 {
		parts = append(parts, "Round videos "+formatBytes(vidN))
	}
	detail := "No round videos yet."
	if vidCount == 1 {
		detail = "1 round video in Videos\\Valvet."
	} else if vidCount > 1 {
		detail = fmt.Sprintf("%d round videos in Videos\\Valvet.", vidCount)
	} else if len(parts) > 0 {
		detail = strings.Join(parts, " · ")
	}
	return storageView{OK: true, Label: "Using " + formatBytes(total) + ".", Detail: detail, Videos: formatBytes(vidN)}
}

func clearOldStorage() storageView {
	base, web, pics := storagePaths()
	before := dirSize(cacheDir()) + dirSize(pics)
	_ = os.RemoveAll(cacheDir())
	if pics != "" {
		_ = os.RemoveAll(pics)
	}
	before += dropCacheDirs(web)
	before += dropCacheDirs(filepath.Join(base, "webview"))
	view := storageNow()
	if before > 0 {
		view.Cleared = "Cleared " + formatBytes(before) + "."
	} else {
		view.Cleared = "Nothing old to delete."
	}
	return view
}

func dropCacheDirs(root string) int64 {
	if root == "" {
		return 0
	}
	if _, err := os.Stat(root); err != nil {
		return 0
	}
	var hit []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || !info.IsDir() {
			return nil
		}
		switch strings.ToLower(info.Name()) {
		case "cache", "code cache", "gpucache", "dawngraphitecache", "dawnwebgpucache", "shadercache", "grshadercache", "cachestorage":
			hit = append(hit, path)
			return filepath.SkipDir
		}
		return nil
	})
	var n int64
	for _, path := range hit {
		n += dirSize(path)
		_ = os.RemoveAll(path)
	}
	return n
}

package main

import (
	"crypto/sha1"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func artHandler(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("u"))
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || !artHost(parsed.Host) {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	sum := sha1.Sum([]byte(parsed.String()))
	ext := strings.ToLower(filepath.Ext(parsed.Path))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" && ext != ".webp" {
		ext = ".png"
	}
	dir := filepath.Join(velvetBase(), "art")
	path := filepath.Join(dir, hex.EncodeToString(sum[:])+ext)
	if raw, err := os.ReadFile(path); err == nil && len(raw) > 32 {
		writeArt(w, ext, raw)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, parsed.String(), nil)
	if err != nil {
		http.Error(w, "bad", http.StatusBadRequest)
		return
	}
	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil || res.StatusCode >= 400 {
		if res != nil {
			res.Body.Close()
		}
		http.Error(w, "miss", http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil || len(body) < 32 {
		http.Error(w, "miss", http.StatusBadGateway)
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(path, body, 0o600)
	writeArt(w, ext, body)
}

func artHost(host string) bool {
	host = strings.ToLower(host)
	return host == "media.valorant-api.com" || host == "valorant-api.com" || strings.HasSuffix(host, ".valorant-api.com")
}

func writeArt(w http.ResponseWriter, ext string, body []byte) {
	switch ext {
	case ".jpg", ".jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
	case ".webp":
		w.Header().Set("Content-Type", "image/webp")
	default:
		w.Header().Set("Content-Type", "image/png")
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(body)
}

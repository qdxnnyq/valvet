package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type savedReplay struct {
	Name  string `json:"name"`
	Patch string `json:"patch"`
	Bytes int64  `json:"bytes"`
	Saved string `json:"saved"`
}

var quietLocal = &http.Client{Timeout: 1200 * time.Millisecond, Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}

func valorantSessions() map[string]any {
	port, password, err := readLockfile()
	if err != nil || port == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://127.0.0.1:"+port+"/product-session/v1/external-sessions", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte("riot:"+password)))
	res, err := quietLocal.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil
	}
	var body map[string]any
	if json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body) != nil {
		return nil
	}
	return body
}

func valorantOpen() bool {
	for _, item := range valorantSessions() {
		row := asMap(item)
		if strings.EqualFold(str(pick(row, "productId", "ProductId")), "valorant") {
			return true
		}
	}
	return false
}

func askValorantReplay(id string) bool {
	port, password, err := readLockfile()
	if err != nil || port == "" || id == "" {
		return false
	}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("riot:"+password))
	body, _ := json.Marshal(map[string]any{
		"productId":   "valorant",
		"patchlineId": "live",
		"arguments":   []string{"-replay=" + id},
	})
	paths := []string{
		"/product-session/v1/external-sessions",
		"/riotclient/launch-product?product=valorant&patchline=live&args=-replay%3D" + id,
	}
	for _, path := range paths {
		ctx, cancel := context.WithTimeout(context.Background(), 900*time.Millisecond)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://127.0.0.1:"+port+path, bytes.NewReader(body))
		if err != nil {
			cancel()
			continue
		}
		req.Header.Set("Authorization", auth)
		req.Header.Set("Content-Type", "application/json")
		res, err := quietLocal.Do(req)
		cancel()
		if err != nil || res == nil {
			continue
		}
		io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		res.Body.Close()
		if res.StatusCode < 400 {
			return true
		}
	}
	return false
}

var replayScan struct {
	mu sync.Mutex
	at time.Time
}

func maybeArchiveReplays() {
	replayScan.mu.Lock()
	due := replayScan.at.IsZero() || time.Since(replayScan.at) >= 90*time.Second
	replayScan.mu.Unlock()
	if due {
		go archiveReplays()
	}
}

func replayDir() string {
	return filepath.Join(velvetBase(), "replays")
}

func valorantDemos() []string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return nil
	}
	return []string{
		filepath.Join(base, "VALORANT", "Saved", "Demos"),
		filepath.Join(base, "VALORANT", "Saved", "Replays"),
	}
}

func valorantPatchLabel() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return ""
	}
	text := readTail(filepath.Join(base, "VALORANT", "Saved", "Logs", "ShooterGame.log"), 400_000)
	return parseClientVersion(text)
}

func archiveReplays() []savedReplay {
	replayScan.mu.Lock()
	if time.Since(replayScan.at) < 90*time.Second {
		rows := loadReplayIndex()
		replayScan.mu.Unlock()
		return rows
	}
	replayScan.at = time.Now()
	replayScan.mu.Unlock()
	dir := replayDir()
	_ = os.MkdirAll(dir, 0o755)
	patch := valorantPatchLabel()
	have := map[string]savedReplay{}
	for _, row := range loadReplayIndex() {
		have[strings.ToLower(row.Name)] = row
	}
	for _, root := range valorantDemos() {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".vrf") {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Size() < 1024 {
				continue
			}
			key := strings.ToLower(entry.Name())
			dest := filepath.Join(dir, entry.Name())
			if prev, ok := have[key]; ok && prev.Bytes == info.Size() {
				if _, err := os.Stat(dest); err == nil {
					continue
				}
			}
			if copyFile(filepath.Join(root, entry.Name()), dest) != nil {
				continue
			}
			label := patch
			if label == "" {
				if prev, ok := have[key]; ok {
					label = prev.Patch
				}
			}
			have[key] = savedReplay{Name: entry.Name(), Patch: label, Bytes: info.Size(), Saved: time.Now().Format(time.RFC3339)}
		}
	}
	rows := make([]savedReplay, 0, len(have))
	for _, row := range have {
		if _, err := os.Stat(filepath.Join(dir, row.Name)); err != nil {
			continue
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Saved > rows[j].Saved })
	raw, err := json.Marshal(rows)
	if err == nil {
		_ = os.WriteFile(filepath.Join(dir, "index.json"), raw, 0o600)
	}
	return rows
}

func loadReplayIndex() []savedReplay {
	raw, err := os.ReadFile(filepath.Join(replayDir(), "index.json"))
	if err != nil {
		return nil
	}
	var rows []savedReplay
	if json.Unmarshal(raw, &rows) != nil {
		return nil
	}
	return rows
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func replayPath(name string) (string, bool) {
	name = filepath.Base(strings.TrimSpace(name))
	if name == "" || name == "." || !strings.EqualFold(filepath.Ext(name), ".vrf") {
		return "", false
	}
	path := filepath.Join(replayDir(), name)
	if _, err := os.Stat(path); err != nil {
		return "", false
	}
	return path, true
}

func placeDemo(path string) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" || path == "" {
		return
	}
	destDir := filepath.Join(base, "VALORANT", "Saved", "Demos")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return
	}
	dest := filepath.Join(destDir, filepath.Base(path))
	src, err := os.Stat(path)
	if err != nil {
		return
	}
	if dst, err := os.Stat(dest); err == nil && dst.Size() == src.Size() {
		return
	}
	_ = copyFile(path, dest)
}

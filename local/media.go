package main

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func mediaDir() string {
	dir := filepath.Join(velvetBase(), "media")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func mediaKind(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "notify", "win", "loss", "background", "friend", "watch", "played":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return ""
	}
}

func mediaPath(kind string) string {
	if mediaKind(kind) == "" {
		return ""
	}
	return filepath.Join(mediaDir(), kind)
}

func saveMedia(kind string, r io.Reader, limit int64) error {
	path := mediaPath(kind)
	if path == "" {
		return os.ErrInvalid
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, io.LimitReader(r, limit))
	return err
}

func writeBytes(w http.ResponseWriter, r *http.Request, name, contentType string, body []byte) {
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(body))
}

func writeMedia(w http.ResponseWriter, r *http.Request, kind string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if kind == "gg" {
		writeBytes(w, r, "win.mp3", "audio/mpeg", winClip)
		return true
	}
	if kind == "sad" {
		writeBytes(w, r, "loss.mp3", "audio/mpeg", lossClip)
		return true
	}
	if kind == "meow" {
		writeBytes(w, r, "meow.mp3", "audio/mpeg", meowClip)
		return true
	}
	if kind == "knock" || kind == "bloom" || kind == "pulse" || kind == "clutch" {
		clip := clutchClip
		switch kind {
		case "knock":
			clip = knockClip
		case "bloom":
			clip = bloomClip
		case "pulse":
			clip = pulseClip
		}
		writeBytes(w, r, kind+".wav", "audio/wav", clip)
		return true
	}
	path := mediaPath(kind)
	if path == "" {
		return false
	}
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) < 32 {
		return false
	}
	kindType := "application/octet-stream"
	if kind == "background" {
		kindType = "image/jpeg"
	}
	writeBytes(w, r, kind, kindType, raw)
	return true
}

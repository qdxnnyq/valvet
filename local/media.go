package main

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

func writeMedia(w http.ResponseWriter, r *http.Request, kind string) bool {
	if r.Method == http.MethodPost {
		return false
	}
	if kind == "gg" {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(winClip)
		return true
	}
	if kind == "sad" {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(lossClip)
		return true
	}
	if kind == "meow" {
		w.Header().Set("Content-Type", "audio/mpeg")
		_, _ = w.Write(meowClip)
		return true
	}
	if kind == "knock" || kind == "bloom" || kind == "pulse" || kind == "clutch" {
		w.Header().Set("Content-Type", "audio/wav")
		switch kind {
		case "knock":
			_, _ = w.Write(knockClip)
		case "bloom":
			_, _ = w.Write(bloomClip)
		case "pulse":
			_, _ = w.Write(pulseClip)
		default:
			_, _ = w.Write(clutchClip)
		}
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
	if kind == "background" {
		w.Header().Set("Content-Type", "image/jpeg")
	} else {
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	_, _ = w.Write(raw)
	return true
}

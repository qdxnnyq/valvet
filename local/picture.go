package main

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func saveUserFile(name, dataURL string, video bool) (string, error) {
	raw := dataURL
	if i := strings.Index(dataURL, ","); i >= 0 {
		raw = dataURL[i+1:]
	}
	buf, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return "", err
	}
	var dir string
	if video {
		dir, err = roundVideoDir()
	} else {
		dir, err = userPictureDir()
	}
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	clean := strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || r == '.' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, name)
	if clean == "" || clean == "." {
		if video {
			clean = "valvet-round.mp4"
		} else {
			clean = "valvet.png"
		}
	}
	if video && isMP4(buf) && !strings.HasSuffix(strings.ToLower(clean), ".mp4") {
		clean = strings.TrimSuffix(clean, filepath.Ext(clean)) + ".mp4"
	}
	path := filepath.Join(dir, clean)
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		return "", err
	}
	if video && isWebM(buf) {
		if next, convErr := transcodeToMP4(path); convErr == nil && next != "" {
			path = next
		} else if strings.HasSuffix(strings.ToLower(path), ".mp4") {
			alt := strings.TrimSuffix(path, filepath.Ext(path)) + ".webm"
			if renameErr := os.Rename(path, alt); renameErr == nil {
				path = alt
			}
		}
	}
	if !video {
		revealFile(path)
	}
	return path, nil
}

func userPictureDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Pictures", "Valvet"), nil
}

func roundVideoDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Videos", "Valvet")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	collectRoundVideos(dir, home)
	return dir, nil
}

func collectRoundVideos(dst, home string) {
	roots := []string{
		filepath.Join(home, "Videos", "Velvet"),
		filepath.Join(home, "Pictures", "Valvet"),
		filepath.Join(home, "Pictures", "Velvet"),
	}
	if app := os.Getenv("APPDATA"); app != "" {
		roots = append(roots, filepath.Join(app, "Valvet"), filepath.Join(app, "Velvet"))
	}
	if base := velvetBase(); base != "" {
		roots = append(roots, base)
	}
	for _, root := range roots {
		if root == "" || sameDir(root, dst) {
			continue
		}
		_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil {
				return nil
			}
			if info.IsDir() {
				switch strings.ToLower(info.Name()) {
				case "webview", "cache", "code cache", "gpucache", "shadercache":
					return filepath.SkipDir
				}
				return nil
			}
			ext := strings.ToLower(filepath.Ext(info.Name()))
			if ext != ".mp4" && ext != ".webm" {
				return nil
			}
			name := strings.ToLower(info.Name())
			if !strings.Contains(name, "valvet") && !strings.Contains(name, "velvet") && !strings.Contains(name, "round") {
				return nil
			}
			target := filepath.Join(dst, info.Name())
			if _, statErr := os.Stat(target); statErr == nil {
				target = filepath.Join(dst, strings.TrimSuffix(info.Name(), ext)+"-"+strconv.FormatInt(info.ModTime().Unix(), 10)+ext)
			}
			if err := os.Rename(path, target); err != nil {
				raw, readErr := os.ReadFile(path)
				if readErr != nil {
					return nil
				}
				if writeErr := os.WriteFile(target, raw, 0o644); writeErr == nil {
					_ = os.Remove(path)
				}
			}
			return nil
		})
	}
}

func isWebM(buf []byte) bool {
	return len(buf) >= 4 && buf[0] == 0x1A && buf[1] == 0x45 && buf[2] == 0xDF && buf[3] == 0xA3
}

func isMP4(buf []byte) bool {
	return len(buf) >= 8 && string(buf[4:8]) == "ftyp"
}

func transcodeToMP4(src string) (string, error) {
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		return "", err
	}
	dst := strings.TrimSuffix(src, filepath.Ext(src)) + ".mp4"
	tmp := dst + ".part.mp4"
	cmd := exec.Command(ffmpeg, "-y", "-i", src, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", "-an", tmp)
	if err := cmd.Run(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if !strings.EqualFold(src, dst) {
		_ = os.Remove(dst)
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if !strings.EqualFold(src, dst) {
		_ = os.Remove(src)
	}
	return dst, nil
}

func revealFile(path string) {
	if runtime.GOOS != "windows" {
		return
	}
	cmd := exec.Command("explorer", "/select,"+path)
	_ = cmd.Start()
}

func revealFolder(path string) {
	if runtime.GOOS != "windows" || path == "" {
		return
	}
	_ = os.MkdirAll(path, 0o700)
	cmd := exec.Command("explorer", path)
	_ = cmd.Start()
}

//go:build windows

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

type launchPref struct {
	Mode string `json:"mode"`
}

func velvetDir() string {
	return velvetBase()
}

func launchMode() string {
	raw, err := os.ReadFile(filepath.Join(velvetDir(), "launch.json"))
	if err != nil {
		return "off"
	}
	var pref launchPref
	if json.Unmarshal(raw, &pref) != nil {
		return "off"
	}
	switch pref.Mode {
	case "windows", "riot":
		return pref.Mode
	default:
		return "off"
	}
}

func setLaunchMode(mode string) string {
	switch mode {
	case "windows", "riot":
	default:
		mode = "off"
	}
	raw, _ := json.Marshal(launchPref{Mode: mode})
	_ = os.WriteFile(filepath.Join(velvetDir(), "launch.json"), raw, 0o600)
	exe, err := os.Executable()
	if err != nil {
		return mode
	}
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	if mode == "off" {
		runHidden("reg", "delete", key, "/v", "Valvet", "/f")
		runHidden("reg", "delete", key, "/v", "Velvet", "/f")
		return mode
	}
	runHidden("reg", "delete", key, "/v", "Velvet", "/f")
	runHidden("reg", "add", key, "/v", "Valvet", "/t", "REG_SZ", "/d", exe+" --autostart", "/f")
	return mode
}

func runHidden(name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = cmd.Run()
}

func maybeWaitForRiot() {
	auto := false
	for _, arg := range os.Args[1:] {
		if arg == "--autostart" {
			auto = true
		}
	}
	if !auto || launchMode() != "riot" {
		return
	}
	for i := 0; i < 240; i++ {
		if _, _, err := readLockfile(); err == nil {
			return
		}
		time.Sleep(2 * time.Second)
	}
}

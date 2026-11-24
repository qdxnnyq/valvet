//go:build windows

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func openReplayFile(path string) error {
	placeDemo(path)
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if exe := riotClientExe(); exe != "" {
		cmd := exec.Command(exe, "--launch-product=valorant", "--launch-patchline=live", "--launch-params=-replay="+id)
		if err := cmd.Start(); err == nil {
			return nil
		}
	}
	target, _ := syscall.UTF16PtrFromString("riotclient://launch/valorant")
	verb, _ := syscall.UTF16PtrFromString("open")
	r, _, _ := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(target)), 0, 0, 1)
	if r <= 32 {
		return errors.New("Valorant could not be opened. Start the Riot Client, then try again.")
	}
	return nil
}

func riotClientExe() string {
	keys := []string{
		`HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\Riot Game riot_client.live`,
		`HKLM\SOFTWARE\WOW6432Node\Microsoft\Windows\CurrentVersion\Uninstall\Riot Game riot_client.live`,
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Riot Game riot_client.live`,
	}
	for _, key := range keys {
		out, err := exec.Command("reg", "query", key, "/v", "InstallLocation").Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(out), "\n") {
			if !strings.Contains(line, "REG_SZ") {
				continue
			}
			parts := strings.SplitN(line, "REG_SZ", 2)
			if len(parts) < 2 {
				continue
			}
			dir := strings.TrimSpace(parts[1])
			exe := filepath.Join(dir, "RiotClientServices.exe")
			if fileExists(exe) {
				return exe
			}
		}
	}
	for _, dir := range []string{
		`C:\Riot Games\Riot Client`,
		`D:\Riot Games\Riot Client`,
		`E:\Riot Games\Riot Client`,
		filepath.Join(os.Getenv("PROGRAMFILES"), "Riot Games", "Riot Client"),
		filepath.Join(os.Getenv("PROGRAMFILES(X86)"), "Riot Games", "Riot Client"),
	} {
		exe := filepath.Join(dir, "RiotClientServices.exe")
		if fileExists(exe) {
			return exe
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

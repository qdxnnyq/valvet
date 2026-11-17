//go:build windows

package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func installUpdate(raw string) error {
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		return err
	}
	if !allowedUpdateHost(req.URL.Hostname()) {
		return errors.New("update host not allowed")
	}
	req.Header.Set("User-Agent", "Valvet")
	res, err := updateClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("download %d", res.StatusCode)
	}
	if res.Request != nil && res.Request.URL != nil && !allowedUpdateHost(res.Request.URL.Hostname()) {
		return errors.New("update host not allowed")
	}
	next := filepath.Join(os.TempDir(), "Valvet-next.exe")
	file, err := os.Create(next)
	if err != nil {
		return err
	}
	n, copyErr := io.Copy(file, io.LimitReader(res.Body, 40<<20))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if n < 100000 {
		return errors.New("update file looked empty")
	}
	head := make([]byte, 2)
	check, err := os.Open(next)
	if err != nil {
		return err
	}
	_, _ = check.Read(head)
	_ = check.Close()
	if string(head) != "MZ" {
		return errors.New("update file was not an app")
	}
	current, err := os.Executable()
	if err != nil {
		return err
	}
	current, _ = filepath.Abs(current)
	script := filepath.Join(os.TempDir(), "velvet-update.cmd")
	body := strings.Join([]string{
		"@echo off",
		"ping 127.0.0.1 -n 3 >nul",
		":retry",
		fmt.Sprintf("move /Y \"%s\" \"%s\" >nul 2>&1", next, current),
		"if errorlevel 1 (",
		"  ping 127.0.0.1 -n 2 >nul",
		"  goto retry",
		")",
		fmt.Sprintf("start \"\" \"%s\"", current),
		"del /f /q \"%~f0\"",
		"",
	}, "\r\n")
	if err := os.WriteFile(script, []byte(body), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd.Start()
}

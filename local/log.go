package main

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	logMu   sync.Mutex
	logLast = map[string]time.Time{}
)

func appLog(msg string) {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return
	}
	logMu.Lock()
	if time.Since(logLast[msg]) < 10*time.Minute {
		logMu.Unlock()
		return
	}
	logLast[msg] = time.Now()
	logMu.Unlock()
	dir := velvetBase()
	if dir == "" {
		return
	}
	file, err := os.OpenFile(filepath.Join(dir, "valvet.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(time.Now().Format(time.RFC3339) + " " + msg + "\n")
}

func readAppLog() string {
	raw, err := os.ReadFile(filepath.Join(velvetBase(), "valvet.log"))
	if err != nil || len(raw) == 0 {
		return ""
	}
	if len(raw) > 12000 {
		raw = raw[len(raw)-12000:]
	}
	return string(raw)
}

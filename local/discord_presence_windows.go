//go:build windows

package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	presenceMu     sync.Mutex
	presencePipe   syscall.Handle
	presenceReady  bool
	presenceClient string
)

func closePresence() {
	if presencePipe != 0 {
		_ = syscall.CloseHandle(presencePipe)
		presencePipe = 0
	}
	presenceReady = false
}

func writeFrame(h syscall.Handle, op uint32, payload []byte) error {
	buf := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(buf[0:4], op)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(len(payload)))
	copy(buf[8:], payload)
	var wrote uint32
	return syscall.WriteFile(h, buf, &wrote, nil)
}

func readFrame(h syscall.Handle) error {
	head := make([]byte, 8)
	var got uint32
	if err := syscall.ReadFile(h, head, &got, nil); err != nil {
		return err
	}
	if got < 8 {
		return errors.New("short frame")
	}
	n := binary.LittleEndian.Uint32(head[4:8])
	if n > 1<<20 {
		return errors.New("huge frame")
	}
	body := make([]byte, n)
	var filled uint32
	for filled < n {
		var chunk uint32
		if err := syscall.ReadFile(h, body[filled:], &chunk, nil); err != nil {
			return err
		}
		if chunk == 0 {
			return errors.New("short frame")
		}
		filled += chunk
	}
	return nil
}

func dialDiscord() (syscall.Handle, error) {
	for i := 0; i < 10; i++ {
		path := fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i)
		p, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			continue
		}
		h, err := syscall.CreateFile(p, syscall.GENERIC_READ|syscall.GENERIC_WRITE, 0, nil, syscall.OPEN_EXISTING, 0, 0)
		if err == nil {
			return h, nil
		}
	}
	return 0, errors.New("Discord is not open")
}

func validClientID(id string) bool {
	if len(id) < 17 || len(id) > 22 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func ensurePresence(clientID string) error {
	if presenceReady && presenceClient == clientID && presencePipe != 0 {
		return nil
	}
	closePresence()
	h, err := dialDiscord()
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(map[string]any{"v": 1, "client_id": clientID})
	if err := writeFrame(h, 0, raw); err != nil {
		_ = syscall.CloseHandle(h)
		return err
	}
	if err := readFrame(h); err != nil {
		_ = syscall.CloseHandle(h)
		return err
	}
	presencePipe = h
	presenceClient = clientID
	presenceReady = true
	return nil
}

func setPresence(clientID, details, state string) error {
	clientID = strings.TrimSpace(clientID)
	if !validClientID(clientID) {
		return errors.New("Paste the Application ID from the Discord developer portal. It is the long number, not the bot token.")
	}
	details = strings.TrimSpace(details)
	state = strings.TrimSpace(state)
	if details == "" {
		details = "Valvet"
	}
	if len(details) > 120 {
		details = details[:120]
	}
	if len(state) > 120 {
		state = state[:120]
	}
	presenceMu.Lock()
	defer presenceMu.Unlock()
	if err := ensurePresence(clientID); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{
		"cmd":   "SET_ACTIVITY",
		"nonce": fmt.Sprint(time.Now().UnixNano()),
		"args": map[string]any{
			"pid": os.Getpid(),
			"activity": map[string]any{
				"details":  details,
				"state":    state,
				"instance": false,
			},
		},
	})
	if err := writeFrame(presencePipe, 1, payload); err != nil {
		closePresence()
		return errors.New("Discord closed the status connection")
	}
	if err := readFrame(presencePipe); err != nil {
		closePresence()
		return errors.New("Discord closed the status connection")
	}
	return nil
}

//go:build windows

package main

import (
	"errors"
	"syscall"
	"unsafe"
)

func openReplayFile(path string) error {
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(path)
	r, _, _ := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteW").Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
	if r <= 32 {
		return errors.New("Valorant could not open that replay.")
	}
	return nil
}

//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var singleMutex uintptr

func claimSingleInstance() bool {
	name, err := syscall.UTF16PtrFromString("Local\\Valvet.SingleInstance")
	if err != nil {
		return true
	}
	handle, _, callErr := syscall.NewLazyDLL("kernel32.dll").NewProc("CreateMutexW").Call(0, 0, uintptr(unsafe.Pointer(name)))
	if handle == 0 {
		return true
	}
	singleMutex = handle
	if errno, ok := callErr.(syscall.Errno); ok && errno == 183 {
		focusExisting()
		return false
	}
	return true
}

func focusExisting() {
	hwnd := findValvet()
	if hwnd == 0 {
		return
	}
	user32.NewProc("ShowWindow").Call(hwnd, 9)
	user32.NewProc("SetForegroundWindow").Call(hwnd)
}

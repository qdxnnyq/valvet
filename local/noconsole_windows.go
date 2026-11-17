//go:build windows

package main

import "syscall"

func init() {
	// Drop a console if Windows attached one (double-click, or a build
	// that forgot the GUI subsystem). Harmless when none is attached.
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	_, _, _ = kernel32.NewProc("FreeConsole").Call()
}

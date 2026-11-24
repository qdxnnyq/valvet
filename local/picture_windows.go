//go:build windows

package main

import (
	"os/exec"
	"syscall"
	"unsafe"
)

func knownPictures() string {
	// FOLDERID_Pictures
	id := syscall.GUID{
		Data1: 0x33E28130,
		Data2: 0x4E1E,
		Data3: 0x4676,
		Data4: [8]byte{0x83, 0x5A, 0x98, 0x39, 0x5C, 0x3B, 0xC3, 0xBB},
	}
	var ptr uintptr
	shell := syscall.NewLazyDLL("shell32.dll")
	ole := syscall.NewLazyDLL("ole32.dll")
	r, _, _ := shell.NewProc("SHGetKnownFolderPath").Call(
		uintptr(unsafe.Pointer(&id)),
		0,
		0,
		uintptr(unsafe.Pointer(&ptr)),
	)
	if r != 0 || ptr == 0 {
		return ""
	}
	defer ole.NewProc("CoTaskMemFree").Call(ptr)
	chars := make([]uint16, 0, 64)
	for i := uintptr(0); i < 512; i++ {
		c := *(*uint16)(unsafe.Pointer(ptr + i*2))
		if c == 0 {
			break
		}
		chars = append(chars, c)
	}
	return syscall.UTF16ToString(chars)
}

func revealFile(path string) {
	if path == "" {
		return
	}
	cmd := exec.Command("explorer", "/select,"+path)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = cmd.Start()
}

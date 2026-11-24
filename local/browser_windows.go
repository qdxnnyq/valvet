//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/jchv/go-webview2"
)

func openBrowser(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	_ = cmd.Start()
}

func openApp(url string) {
	setAppUserModelID()
	_ = os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--autoplay-policy=no-user-gesture-required")
	data := ""
	if base := os.Getenv("APPDATA"); base != "" {
		migrateAppDirs()
		data = filepath.Join(base, "Valvet", "webview")
		if _, err := os.Stat(data); os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(base, "Velvet", "webview")); err == nil {
				data = filepath.Join(base, "Velvet", "webview")
			}
		}
		_ = os.MkdirAll(data, 0o755)
	}
	width := uint(1280)
	height := uint(780)
	center := true
	if rect, ok := loadValvetWindow(); ok {
		width = uint(rect.W)
		height = uint(rect.H)
		center = false
		beginPlaceHook(rect)
	}
	window := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		DataPath:  data,
		WindowOptions: webview2.WindowOptions{
			Title:  "Valvet",
			Width:  width,
			Height: height,
			Center: center,
			IconId: 1,
		},
	})
	endPlaceHook()
	if window == nil {
		openBrowser(url)
		select {}
	}
	setTaskbarIcon(uintptr(window.Window()))
	defer window.Destroy()
	window.SetTitle("Valvet")
	_ = window.Bind("valvetResize", func(name string) {
		width, height := windowSize(name)
		window.Dispatch(func() {
			window.SetSize(width, height, webview2.HintNone)
		})
	})
	_ = window.Bind("velvetResize", func(name string) {
		width, height := windowSize(name)
		window.Dispatch(func() {
			window.SetSize(width, height, webview2.HintNone)
		})
	})
	window.SetSize(1000, 640, webview2.HintMin)
	hwnd := uintptr(window.Window())
	ownFrame(hwnd)
	paintThemeFrame(hwnd, "#10080d", "#ff5fa2")
	_ = window.Bind("valvetWin", func(cmd string) {
		window.Dispatch(func() { windowCommand(uintptr(window.Window()), cmd) })
	})
	_ = window.Bind("velvetWin", func(cmd string) {
		window.Dispatch(func() { windowCommand(uintptr(window.Window()), cmd) })
	})
	_ = window.Bind("valvetDrag", func() {
		window.Dispatch(func() { dragWindow(uintptr(window.Window())) })
	})
	_ = window.Bind("velvetDrag", func() {
		window.Dispatch(func() { dragWindow(uintptr(window.Window())) })
	})
	_ = window.Bind("valvetChrome", func(bg, accent string) {
		window.Dispatch(func() { paintThemeFrame(uintptr(window.Window()), bg, accent) })
	})
	_ = window.Bind("velvetChrome", func(bg, accent string) {
		window.Dispatch(func() { paintThemeFrame(uintptr(window.Window()), bg, accent) })
	})
	window.Navigate(url)
	trackWindow()
	window.Run()
	os.Exit(0)
}

func windowSize(name string) (int, int) {
	switch name {
	case "compact":
		return 1100, 700
	case "wide":
		return 1500, 900
	case "large":
		return 1760, 1000
	default:
		return 1280, 780
	}
}

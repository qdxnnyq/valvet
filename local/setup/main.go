//go:build setup

package main

import (
	_ "embed"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2"
)

//go:embed Valvet.exe
var payload []byte

//go:embed ui.html
var setupPage string

func runWindow() bool {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = os.TempDir()
	}
	home := filepath.Join(base, "Valvet")
	window := webview2.NewWithOptions(webview2.WebViewOptions{
		AutoFocus: true,
		WindowOptions: webview2.WindowOptions{
			Title:  "Valvet Setup",
			Width:  680,
			Height: 640,
			Center: true,
			IconId: 1,
		},
	})
	if window == nil {
		return false
	}
	defer window.Destroy()
	darkFrame(uintptr(window.Window()))
	_ = window.Bind("setupHome", func() string { return home })
	_ = window.Bind("setupBrowse", func(start string) string {
		if strings.TrimSpace(start) == "" {
			start = home
		}
		dir, ok := pickFolder("Choose a folder for Valvet", start)
		if !ok {
			return ""
		}
		return dir
	})
	_ = window.Bind("setupInstall", func(dir string, desktop, start, windows bool) string {
		return installTo(dir, desktop, start, windows)
	})
	_ = window.Bind("setupOpen", func() {
		_ = launchInstalled()
		window.Dispatch(func() { window.Terminate() })
	})
	_ = window.Bind("setupClose", func() {
		window.Dispatch(func() { window.Terminate() })
	})
	window.SetSize(640, 560, webview2.HintMin)
	window.SetHtml(setupPage)
	window.Run()
	return true
}

func darkFrame(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	use := int32(1)
	dwm := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	dwm.Call(hwnd, 20, uintptr(unsafe.Pointer(&use)), unsafe.Sizeof(use))
	dwm.Call(hwnd, 19, uintptr(unsafe.Pointer(&use)), unsafe.Sizeof(use))
	border := uint32(0x000D0810)
	text := uint32(0x00F4F7FF)
	dwm.Call(hwnd, 34, uintptr(unsafe.Pointer(&border)), 4)
	dwm.Call(hwnd, 35, uintptr(unsafe.Pointer(&border)), 4)
	dwm.Call(hwnd, 36, uintptr(unsafe.Pointer(&text)), 4)
	user32 := syscall.NewLazyDLL("user32.dll")
	kernel := syscall.NewLazyDLL("kernel32.dll")
	mod, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	load := user32.NewProc("LoadImageW")
	big, _, _ := load.Call(mod, 1, 1, 32, 32, 0)
	small, _, _ := load.Call(mod, 1, 1, 16, 16, 0)
	if big != 0 {
		if small == 0 {
			small = big
		}
		user32.NewProc("SendMessageW").Call(hwnd, 0x0080, 1, big)
		user32.NewProc("SendMessageW").Call(hwnd, 0x0080, 0, small)
		user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(13), big)
		user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(33), small)
	}
	brush, _, _ := syscall.NewLazyDLL("gdi32.dll").NewProc("CreateSolidBrush").Call(0x000D0810)
	if brush != 0 {
		user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(9), brush)
	}
}

func main() {
	if runWindow() {
		return
	}
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		box("Valvet", "Couldn't find a place to install.")
		return
	}
	fallback := filepath.Join(base, "Valvet")
	dir, ok := pickFolder("Choose a folder for Valvet", fallback)
	if !ok {
		return
	}
	if msg := installTo(dir, true, true, true); msg != "ok" {
		box("Valvet", msg)
		return
	}
	_ = launchInstalled()
}

func installTo(dir string, desktop, start, windows bool) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return "Choose a folder first."
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "Couldn't create that folder."
	}
	dest := filepath.Join(dir, "Valvet.exe")
	if err := os.WriteFile(dest, payload, 0o755); err != nil {
		return "Close Valvet if it is open, then run the installer again."
	}
	registerSetup(dir, dest)
	link := func(path string) {
		script := `$s=(New-Object -ComObject WScript.Shell).CreateShortcut('` + psQuote(path) + `'); $s.TargetPath='` + psQuote(dest) + `'; $s.WorkingDirectory='` + psQuote(dir) + `'; $s.IconLocation='` + psQuote(dest) + `,0'; $s.Description='Valvet'; $s.Save()`
		cmd := exec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		_ = cmd.Run()
	}
	if start {
		if app := os.Getenv("APPDATA"); app != "" {
			link(filepath.Join(app, "Microsoft", "Windows", "Start Menu", "Programs", "Valvet.lnk"))
		}
	}
	if desktop {
		if home, err := os.UserHomeDir(); err == nil {
			link(filepath.Join(home, "Desktop", "Valvet.lnk"))
		}
	}
	if windows {
		base := os.Getenv("LOCALAPPDATA")
		if base != "" {
			folder := filepath.Join(base, "Valvet")
			_ = os.MkdirAll(folder, 0o755)
			raw, _ := json.Marshal(map[string]string{"mode": "windows"})
			_ = os.WriteFile(filepath.Join(folder, "launch.json"), raw, 0o600)
		}
		add := exec.Command("reg", "add", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Valvet", "/t", "REG_SZ", "/d", dest+" --autostart", "/f")
		add.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		_ = add.Run()
		del := exec.Command("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Velvet", "/f")
		del.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		_ = del.Run()
	}
	installedPath = dest
	return "ok"
}

func launchInstalled() error {
	if installedPath == "" {
		return os.ErrNotExist
	}
	cmd := exec.Command(installedPath)
	cmd.Dir = filepath.Dir(installedPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000008 | 0x00000200}
	return cmd.Start()
}

var installedPath string

func registerSetup(dir, dest string) {
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Valvet`
	run := func(args ...string) {
		cmd := exec.Command("reg", args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		_ = cmd.Run()
	}
	run("delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Velvet`, "/f")
	run("add", key, "/f")
	run("add", key, "/v", "DisplayName", "/t", "REG_SZ", "/d", "Valvet", "/f")
	run("add", key, "/v", "DisplayVersion", "/t", "REG_SZ", "/d", "__VALVET__", "/f")
	run("add", key, "/v", "Publisher", "/t", "REG_SZ", "/d", "Danny", "/f")
	run("add", key, "/v", "InstallLocation", "/t", "REG_SZ", "/d", dir, "/f")
	run("add", key, "/v", "DisplayIcon", "/t", "REG_SZ", "/d", dest, "/f")
	run("add", key, "/v", "UninstallString", "/t", "REG_SZ", "/d", `"`+dest+`" -uninstall`, "/f")
	run("add", key, "/v", "NoModify", "/t", "REG_DWORD", "/d", "1", "/f")
	run("add", key, "/v", "NoRepair", "/t", "REG_DWORD", "/d", "1", "/f")
}

func box(title, text string) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), 0x40)
}

func psQuote(s string) string {
	return strings.ReplaceAll(s, `'`, `''`)
}

var browseStart *uint16

func browseProc(hwnd, msg, _, _ uintptr) uintptr {
	if msg == 1 && browseStart != nil {
		syscall.NewLazyDLL("user32.dll").NewProc("SendMessageW").Call(hwnd, 0x0467, 1, uintptr(unsafe.Pointer(browseStart)))
	}
	return 0
}

func pickFolder(title, start string) (string, bool) {
	ole := syscall.NewLazyDLL("ole32.dll")
	ole.NewProc("CoInitializeEx").Call(0, 2)
	shell := syscall.NewLazyDLL("shell32.dll")
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return start, true
	}
	startPtr, err := syscall.UTF16PtrFromString(start)
	if err != nil {
		return start, true
	}
	browseStart = startPtr
	var name [520]uint16
	info := struct {
		Owner       uintptr
		Root        uintptr
		DisplayName *uint16
		Title       *uint16
		Flags       uint32
		Callback    uintptr
		LParam      uintptr
		Image       int32
	}{
		DisplayName: &name[0],
		Title:       titlePtr,
		Flags:       0x0001 | 0x0040 | 0x0010,
		Callback:    syscall.NewCallback(browseProc),
	}
	pidl, _, _ := shell.NewProc("SHBrowseForFolderW").Call(uintptr(unsafe.Pointer(&info)))
	if pidl == 0 {
		return "", false
	}
	defer ole.NewProc("CoTaskMemFree").Call(pidl)
	var path [520]uint16
	ok, _, _ := shell.NewProc("SHGetPathFromIDListW").Call(pidl, uintptr(unsafe.Pointer(&path[0])))
	if ok == 0 {
		return "", false
	}
	chosen := syscall.UTF16ToString(path[:])
	if chosen == "" {
		return "", false
	}
	return chosen, true
}

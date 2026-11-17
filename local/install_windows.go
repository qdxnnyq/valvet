//go:build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/jchv/go-webview2"
)

func wantsUninstall() bool {
	for _, arg := range os.Args[1:] {
		if arg == "-uninstall" || arg == "--uninstall" {
			return true
		}
	}
	return false
}

func registerInstalled() {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return
	}
	key := `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Valvet`
	runHidden("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Velvet`, "/f")
	runHidden("reg", "add", key, "/f")
	setReg(key, "DisplayName", "REG_SZ", "Valvet")
	setReg(key, "DisplayVersion", "REG_SZ", appLabel)
	setReg(key, "Publisher", "REG_SZ", "Danny")
	setReg(key, "InstallLocation", "REG_SZ", filepath.Dir(exe))
	setReg(key, "DisplayIcon", "REG_SZ", exe)
	setReg(key, "UninstallString", "REG_SZ", `"`+exe+`" -uninstall`)
	setReg(key, "NoModify", "REG_DWORD", "1")
	setReg(key, "NoRepair", "REG_DWORD", "1")
	if info, err := os.Stat(exe); err == nil && info.Size() > 0 {
		setReg(key, "EstimatedSize", "REG_DWORD", strconv.FormatInt(info.Size()/1024, 10))
	}
}

func setReg(key, name, kind, value string) {
	runHidden("reg", "add", key, "/v", name, "/t", kind, "/d", value, "/f")
}

func uninstallValvet() {
	window := webview2.NewWithOptions(webview2.WebViewOptions{
		WindowOptions: webview2.WindowOptions{
			Title:  "Uninstall Valvet",
			Width:  560,
			Height: 480,
			Center: true,
			IconId: 1,
		},
	})
	if window == nil {
		uninstallPlain()
		return
	}
	hwnd := uintptr(window.Window())
	paintDarkTitle(hwnd)
	paintClientGap(hwnd)
	setTaskbarIcon(hwnd)
	_ = window.Bind("valvetRemove", func(wipe bool) string {
		performUninstall(wipe)
		return "ok"
	})
	_ = window.Bind("valvetClose", func() {
		window.Dispatch(func() { window.Terminate() })
	})
	window.SetHtml(strings.ReplaceAll(uninstallPage, "__VALVET__", appLabel))
	window.Run()
}

func uninstallPlain() {
	if winAsk("Valvet", "Uninstall Valvet? If it is open, that window will close.", 0x24) != 6 {
		return
	}
	wipe := winAsk("Valvet", "Also delete all saved data? Games, settings, and sounds on this PC. This cannot be undone.", 0x124) == 6
	performUninstall(wipe)
	if wipe {
		winBox("Valvet", "Valvet was removed, including saved games and settings.")
	} else {
		winBox("Valvet", "Valvet was removed. Saved games and settings stay on this PC.")
	}
}

func performUninstall(wipe bool) {
	exe, _ := os.Executable()
	closeOtherValvet()
	time.Sleep(400 * time.Millisecond)
	runHidden("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Valvet`, "/f")
	runHidden("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Uninstall\Velvet`, "/f")
	runHidden("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Valvet", "/f")
	runHidden("reg", "delete", `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`, "/v", "Velvet", "/f")
	if app := os.Getenv("APPDATA"); app != "" {
		_ = os.Remove(filepath.Join(app, "Microsoft", "Windows", "Start Menu", "Programs", "Valvet.lnk"))
		_ = os.Remove(filepath.Join(app, "Microsoft", "Windows", "Start Menu", "Programs", "Velvet.lnk"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		_ = os.Remove(filepath.Join(home, "Desktop", "Valvet.lnk"))
		_ = os.Remove(filepath.Join(home, "Desktop", "Velvet.lnk"))
	}
	if wipe {
		wipeValvetData(exe)
	}
	scheduleRemove(exe, wipe)
}

func closeOtherValvet() {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	snap, _, _ := kernel.NewProc("CreateToolhelp32Snapshot").Call(2, 0)
	if snap == 0 || snap == ^uintptr(0) {
		return
	}
	defer kernel.NewProc("CloseHandle").Call(snap)
	var entry processEntry
	entry.Size = uint32(unsafe.Sizeof(entry))
	first := kernel.NewProc("Process32FirstW")
	next := kernel.NewProc("Process32NextW")
	ok, _, _ := first.Call(snap, uintptr(unsafe.Pointer(&entry)))
	self := uint32(os.Getpid())
	for ok != 0 {
		name := syscall.UTF16ToString(entry.ExeFile[:])
		if (strings.EqualFold(name, "Valvet.exe") || strings.EqualFold(name, "Velvet.exe")) && entry.ProcessID != 0 && entry.ProcessID != self {
			killPID(kernel, entry.ProcessID)
		}
		entry.Size = uint32(unsafe.Sizeof(entry))
		ok, _, _ = next.Call(snap, uintptr(unsafe.Pointer(&entry)))
	}
}

type processEntry struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExeFile         [260]uint16
}

func killPID(kernel *syscall.LazyDLL, pid uint32) {
	const processTerminate = 0x0001
	handle, _, _ := kernel.NewProc("OpenProcess").Call(processTerminate, 0, uintptr(pid))
	if handle == 0 {
		return
	}
	kernel.NewProc("TerminateProcess").Call(handle, 1)
	kernel.NewProc("CloseHandle").Call(handle)
}

func wipeValvetData(exe string) {
	bases := []string{os.Getenv("LOCALAPPDATA"), os.Getenv("APPDATA")}
	for _, base := range bases {
		if base == "" {
			continue
		}
		for _, name := range []string{"Velvet", "Valvet"} {
			dir := filepath.Join(base, name)
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, entry := range entries {
				path := filepath.Join(dir, entry.Name())
				if exe != "" && strings.EqualFold(path, exe) {
					continue
				}
				_ = os.RemoveAll(path)
			}
			_ = os.Remove(dir)
		}
	}
}

func scheduleRemove(exe string, wipe bool) {
	if exe == "" {
		return
	}
	q := `"` + strings.ReplaceAll(exe, `"`, "") + `"`
	dir := `"` + strings.ReplaceAll(filepath.Dir(exe), `"`, "") + `"`
	done := "exit"
	if wipe {
		done = "rmdir /s /q " + dir + " & exit"
	}
	script := "for /l %i in (1,1,180) do @(del /f /q " + q + " & if not exist " + q + " (" + done + ") & ping -n 2 127.0.0.1 >nul)"
	cmd := exec.Command("cmd", "/c", script)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
	_ = cmd.Start()
}

func winAsk(title, text string, flags uintptr) uintptr {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), flags)
	return r
}

func winBox(title, text string) {
	t, _ := syscall.UTF16PtrFromString(title)
	b, _ := syscall.UTF16PtrFromString(text)
	syscall.NewLazyDLL("user32.dll").NewProc("MessageBoxW").Call(0, uintptr(unsafe.Pointer(b)), uintptr(unsafe.Pointer(t)), 0x40)
}

const uninstallPage = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>Uninstall Valvet</title>
<style>
  :root { color-scheme: dark; --accent: #ff5fa2; --ink: #fff7fa; --muted: #b89aaa; --line: rgba(255,247,250,.1); }
  * { box-sizing: border-box; }
  html, body { height: 100%; margin: 0; background: #10080d; color: var(--ink); font: 14px/1.45 "Segoe UI Variable Text", "Segoe UI", sans-serif; }
  body { display: flex; flex-direction: column; padding: 28px 32px 22px; background:
    radial-gradient(520px 220px at 0% 0%, rgba(255,95,162,.28), transparent 70%), #10080d; }
  .brand { display: flex; align-items: center; gap: 12px; }
  .marksvg { width: 44px; height: 44px; flex: none; }
  .mark { font-size: 22px; font-weight: 780; letter-spacing: .22em; }
  .mark span { color: var(--accent); }
  h1 { margin: 18px 0 6px; font-size: 32px; letter-spacing: -0.045em; font-weight: 740; }
  p { margin: 0; color: var(--muted); max-width: 440px; }
  .checks { margin-top: 22px; }
  .checks label { display: flex; align-items: flex-start; gap: 10px; color: var(--ink); }
  .checks span { display: block; color: var(--muted); font-size: 12px; margin-top: 2px; }
  .bar { height: 4px; margin-top: 22px; border-radius: 99px; background: rgba(255,255,255,.08); overflow: hidden; }
  .bar i { display: block; height: 100%; width: 0; background: var(--accent); transition: width .25s ease; }
  .status { min-height: 20px; margin-top: 10px; color: var(--muted); }
  .foot { margin-top: auto; display: flex; justify-content: space-between; align-items: center; gap: 12px; padding-top: 18px; }
  .foot span { color: var(--muted); font-size: 12px; }
  .actions { display: flex; gap: 8px; }
  button { font: inherit; cursor: pointer; }
  .ghost, .solid { height: 42px; border-radius: 12px; padding: 0 16px; font-weight: 740; }
  .ghost { border: 1px solid var(--line); background: transparent; color: var(--ink); }
  .solid { border: 0; background: var(--accent); color: #1a0610; }
  .solid.danger { background: #ff5fa2; }
  .solid:disabled { opacity: .55; cursor: default; }
  #done { display: none; }
</style>
</head>
<body>
  <div class="brand"><svg class="marksvg" viewBox="0 0 64 64" aria-hidden="true"><rect width="64" height="64" rx="14" fill="#10080d"/><path d="M16 18 L32 48 L48 18" fill="none" stroke="#ff5fa2" stroke-width="6" stroke-linecap="round" stroke-linejoin="round"/></svg><div class="mark">VAL<span>VET</span></div></div>
  <div id="ask">
    <h1>Uninstall Valvet</h1>
    <p>Removes the app, shortcuts, and the start-up entry. An open Valvet window will close.</p>
    <div class="checks">
      <label><input id="wipe" type="checkbox" /> <span><b style="color:#fff7fa;font-weight:740">Also delete saved data</b><br>Games, settings, and sounds on this PC. This cannot be undone.</span></label>
    </div>
    <div class="bar"><i id="meter"></i></div>
    <div class="status" id="status">Nothing is removed until you confirm.</div>
  </div>
  <div id="done">
    <h1>Valvet was removed</h1>
    <p id="done-line">Saved games and settings stay on this PC.</p>
  </div>
  <div class="foot">
    <span>__VALVET__</span>
    <div class="actions">
      <button class="ghost" id="close" type="button">Cancel</button>
      <button class="solid" id="go" type="button">Uninstall</button>
    </div>
  </div>
  <script>
    const status = document.getElementById("status");
    const meter = document.getElementById("meter");
    const go = document.getElementById("go");
    const close = document.getElementById("close");
    close.onclick = () => valvetClose();
    go.onclick = async () => {
      if (go.dataset.done === "1") { valvetClose(); return; }
      const wipe = document.getElementById("wipe").checked;
      go.disabled = true;
      close.disabled = true;
      status.textContent = "Removing Valvet...";
      meter.style.width = "40%";
      await valvetRemove(wipe);
      meter.style.width = "100%";
      document.getElementById("ask").style.display = "none";
      document.getElementById("done").style.display = "block";
      document.getElementById("done-line").textContent = wipe ? "Saved games and settings were deleted too." : "Saved games and settings stay on this PC.";
      close.style.display = "none";
      go.disabled = false;
      go.dataset.done = "1";
      go.textContent = "Close";
    };
  </script>
</body>
</html>`

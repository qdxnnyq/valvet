//go:build windows

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                = syscall.NewLazyDLL("user32.dll")
	procFindWindowW       = user32.NewProc("FindWindowW")
	procGetWindowRect     = user32.NewProc("GetWindowRect")
	procSetWindowPos      = user32.NewProc("SetWindowPos")
	procMonitorFromWindow = user32.NewProc("MonitorFromWindow")
	procGetMonitorInfoW   = user32.NewProc("GetMonitorInfoW")
	procGetWindowLong     = user32.NewProc("GetWindowLongPtrW")
	procSetWindowLong     = user32.NewProc("SetWindowLongPtrW")
	procCallWindowProc    = user32.NewProc("CallWindowProcW")
	procScreenToClient    = user32.NewProc("ScreenToClient")
	prevWndProc           uintptr
	frameHitCB            uintptr
)

func frameHit(hwnd, msg, wp, lp uintptr) uintptr {
	if msg == 0x0084 && prevWndProc != 0 {
		x := int32(int16(lp & 0xffff))
		y := int32(int16((lp >> 16) & 0xffff))
		pt := struct{ X, Y int32 }{x, y}
		procScreenToClient.Call(hwnd, uintptr(unsafe.Pointer(&pt)))
		var rc rawRect
		procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rc)))
		width := rc.Right - rc.Left
		if pt.Y >= 0 && pt.Y < 34 && pt.X >= 0 && pt.X < width-128 {
			return 2
		}
	}
	ret, _, _ := procCallWindowProc.Call(prevWndProc, hwnd, msg, wp, lp)
	return ret
}

func installFrameDrag(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	if frameHitCB == 0 {
		frameHitCB = syscall.NewCallback(frameHit)
	}
	cur, _, _ := procGetWindowLong.Call(hwnd, ^uintptr(3))
	if cur == 0 || cur == frameHitCB {
		return
	}
	prevWndProc = cur
	procSetWindowLong.Call(hwnd, ^uintptr(3), frameHitCB)
}

type winRect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type winBook struct {
	Last string             `json:"last"`
	By   map[string]winRect `json:"by"`
}

type monitorInfoEx struct {
	cbSize    uint32
	rcMonitor rawRect
	rcWork    rawRect
	dwFlags   uint32
	szDevice  [32]uint16
}

type rawRect struct {
	Left, Top, Right, Bottom int32
}

func findValvet() uintptr {
	title, _ := syscall.UTF16PtrFromString("Valvet")
	hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
	return hwnd
}

func readValvet(hwnd uintptr) (winRect, bool) {
	var raw rawRect
	ok, _, _ := procGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&raw)))
	if ok == 0 || raw.Right-raw.Left < 400 || raw.Bottom-raw.Top < 300 {
		return winRect{}, false
	}
	return winRect{X: int(raw.Left), Y: int(raw.Top), W: int(raw.Right - raw.Left), H: int(raw.Bottom - raw.Top)}, true
}

func moveValvet(hwnd uintptr, rect winRect) {
	if rect.W < 400 || rect.H < 300 {
		return
	}
	const flags = 0x0004 | 0x0010
	_, _, _ = procSetWindowPos.Call(hwnd, 0, uintptr(rect.X), uintptr(rect.Y), uintptr(rect.W), uintptr(rect.H), flags)
}

func monitorOf(hwnd uintptr) string {
	const nearest = 2
	hmon, _, _ := procMonitorFromWindow.Call(hwnd, nearest)
	if hmon == 0 {
		return "primary"
	}
	info := monitorInfoEx{cbSize: uint32(unsafe.Sizeof(monitorInfoEx{}))}
	ok, _, _ := procGetMonitorInfoW.Call(hmon, uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return "primary"
	}
	name := syscall.UTF16ToString(info.szDevice[:])
	if name == "" {
		return "primary"
	}
	return name
}

func loadWinBook() winBook {
	raw, err := os.ReadFile(filepath.Join(velvetDir(), "window.json"))
	book := winBook{By: map[string]winRect{}}
	if err != nil {
		return book
	}
	if json.Unmarshal(raw, &book) == nil && len(book.By) > 0 {
		if book.By == nil {
			book.By = map[string]winRect{}
		}
		return book
	}
	var rect winRect
	if json.Unmarshal(raw, &rect) == nil && rect.W >= 400 && rect.H >= 300 {
		book.Last = "primary"
		book.By["primary"] = rect
	}
	return book
}

func loadValvetWindow() (winRect, bool) {
	book := loadWinBook()
	if book.Last != "" {
		if rect, ok := book.By[book.Last]; ok && rect.W >= 400 && rect.H >= 300 {
			return rect, true
		}
	}
	for _, rect := range book.By {
		if rect.W >= 400 && rect.H >= 300 {
			return rect, true
		}
	}
	return winRect{}, false
}

func saveValvetWindow(rect winRect, mon string) {
	if rect.W < 400 || rect.H < 300 {
		return
	}
	if mon == "" {
		mon = "primary"
	}
	book := loadWinBook()
	book.By[mon] = rect
	book.Last = mon
	raw, err := json.Marshal(book)
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(velvetDir(), "window.json"), raw, 0o600)
}

func trackWindow() {
	go func() {
		var hwnd uintptr
		for i := 0; i < 50 && hwnd == 0; i++ {
			time.Sleep(40 * time.Millisecond)
			hwnd = findValvet()
		}
		if hwnd == 0 {
			return
		}
		setTaskbarIcon(hwnd)
		installFrameDrag(hwnd)
		if want, ok := loadValvetWindow(); ok {
			if cur, good := readValvet(hwnd); !good || absInt(cur.X-want.X) > 8 || absInt(cur.Y-want.Y) > 8 {
				moveValvet(hwnd, want)
			}
		}
		paintWindowChrome(hwnd)
		lastMon := ""
		for {
			hwnd = findValvet()
			if hwnd == 0 {
				return
			}
			paintWindowChrome(hwnd)
			mon := monitorOf(hwnd)
			if mon != lastMon && lastMon != "" {
				if want, ok := loadWinBook().By[mon]; ok {
					moveValvet(hwnd, want)
					lastMon = mon
					time.Sleep(2 * time.Second)
					continue
				}
			}
			if rect, ok := readValvet(hwnd); ok {
				saveValvetWindow(rect, mon)
			}
			lastMon = mon
			time.Sleep(2 * time.Second)
		}
	}()
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

var (
	placeOn               bool
	placeHook             uintptr
	placeProc             uintptr
	placeRect             winRect
	procCallNextHookEx    = user32.NewProc("CallNextHookEx")
	procSetWindowsHookEx  = user32.NewProc("SetWindowsHookExW")
	procUnhookWindowsHook = user32.NewProc("UnhookWindowsHookEx")
)

type winCreate struct {
	CreateParams uintptr
	Instance     uintptr
	Menu         uintptr
	Parent       uintptr
	Cy           int32
	Cx           int32
	Y            int32
	X            int32
	Style        int32
	Name         *uint16
	Class        *uint16
	ExStyle      uint32
}

func beginPlaceHook(rect winRect) {
	placeRect = rect
	placeOn = true
	if placeProc == 0 {
		placeProc = syscall.NewCallback(placeHookProc)
	}
	tid, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetCurrentThreadId").Call()
	placeHook, _, _ = procSetWindowsHookEx.Call(5, placeProc, 0, tid)
}

func endPlaceHook() {
	placeOn = false
	if placeHook != 0 {
		procUnhookWindowsHook.Call(placeHook)
		placeHook = 0
	}
}

func isValvetTitle(p *uint16) bool {
	if p == nil {
		return false
	}
	want := [7]uint16{'V', 'a', 'l', 'v', 'e', 't', 0}
	for i := 0; i < len(want); i++ {
		c := *(*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(p)) + uintptr(i*2)))
		if c != want[i] {
			return false
		}
	}
	return true
}

func placeHookProc(code, wparam, lparam uintptr) uintptr {
	if placeOn && int32(code) == 3 && lparam != 0 {
		head := (*struct {
			Lpcs            uintptr
			HwndInsertAfter uintptr
		})(unsafe.Pointer(lparam))
		if head.Lpcs != 0 {
			cs := (*winCreate)(unsafe.Pointer(head.Lpcs))
			if cs.Parent == 0 && isValvetTitle(cs.Name) {
				cs.X = int32(placeRect.X)
				cs.Y = int32(placeRect.Y)
				cs.Cx = int32(placeRect.W)
				cs.Cy = int32(placeRect.H)
			}
		}
	}
	ret, _, _ := procCallNextHookEx.Call(placeHook, code, wparam, lparam)
	return ret
}

func paintWindowChrome(hwnd uintptr) {
	paintDarkTitle(hwnd)
	paintClientGap(hwnd)
	setTaskbarIcon(hwnd)
	ensureTray(hwnd)
}

func ensureTray(hwnd uintptr) {
	_ = hwnd
	trayOnce.Do(func() {
		go trayLoop()
	})
}

var trayOnce sync.Once

func trayLoop() {
	class, _ := syscall.UTF16PtrFromString("ValvetTray")
	title, _ := syscall.UTF16PtrFromString("Valvet")
	instance, _, _ := syscall.NewLazyDLL("kernel32.dll").NewProc("GetModuleHandleW").Call(0)
	cb := syscall.NewCallback(trayProc)
	trayProcKeep = cb
	wc := struct {
		Size       uint32
		Style      uint32
		WndProc    uintptr
		ClsExtra   int32
		WndExtra   int32
		Instance   uintptr
		Icon       uintptr
		Cursor     uintptr
		Background uintptr
		MenuName   *uint16
		ClassName  *uint16
		IconSm     uintptr
	}{
		WndProc:   cb,
		Instance:  instance,
		ClassName: class,
	}
	wc.Size = uint32(unsafe.Sizeof(wc))
	user32.NewProc("RegisterClassExW").Call(uintptr(unsafe.Pointer(&wc)))
	hwnd, _, _ := user32.NewProc("CreateWindowExW").Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 0, 0, ^uintptr(2), 0, instance, 0)
	if hwnd == 0 {
		return
	}
	trayHwnd = hwnd
	addTrayIcon(hwnd)
	var msg struct {
		Hwnd    uintptr
		Message uint32
		Wparam  uintptr
		Lparam  uintptr
		Time    uint32
		X       int32
		Y       int32
	}
	get := user32.NewProc("GetMessageW")
	translate := user32.NewProc("TranslateMessage")
	dispatch := user32.NewProc("DispatchMessageW")
	for {
		r, _, _ := get.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			return
		}
		translate.Call(uintptr(unsafe.Pointer(&msg)))
		dispatch.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

func addTrayIcon(hwnd uintptr) {
	if taskbarSmall == 0 && taskbarBig == 0 {
		taskbarBig, taskbarSmall = loadValvetIcons()
	}
	icon := taskbarSmall
	if icon == 0 {
		icon = taskbarBig
	}
	if icon == 0 || hwnd == 0 {
		return
	}
	var tip [128]uint16
	copy(tip[:], syscall.StringToUTF16("Valvet"))
	data := struct {
		Size      uint32
		Hwnd      uintptr
		ID        uint32
		Flags     uint32
		Callback  uint32
		Icon      uintptr
		Tip       [128]uint16
		State     uint32
		StateMask uint32
		Info      [256]uint16
		Version   uint32
		InfoTitle [64]uint16
		InfoFlags uint32
		Guid      [16]byte
		Balloon   uintptr
	}{
		Hwnd:     hwnd,
		ID:       1,
		Flags:    1 | 2 | 4,
		Callback: 0x0414,
		Icon:     icon,
		Tip:      tip,
	}
	data.Size = uint32(unsafe.Sizeof(data))
	ok, _, _ := syscall.NewLazyDLL("shell32.dll").NewProc("Shell_NotifyIconW").Call(0, uintptr(unsafe.Pointer(&data)))
	if ok != 0 {
		trayOn = true
	}
}

func trayProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if msg == 0x0414 {
		switch lparam {
		case 0x0205, 0x007B:
			switch trayMenu(hwnd) {
			case 1:
				showValvet(false)
			case 2:
				showValvet(true)
			case 4:
				setLookupPause(!lookupsPaused())
			case 3:
				closeValvet()
			}
		case 0x0203:
			showValvet(false)
		}
		return 0
	}
	r, _, _ := user32.NewProc("DefWindowProcW").Call(hwnd, msg, wparam, lparam)
	return r
}

func trayMenu(hwnd uintptr) uintptr {
	user := user32.NewProc
	menu, _, _ := user("CreatePopupMenu").Call()
	if menu == 0 {
		return 0
	}
	defer user("DestroyMenu").Call(menu)
	add := func(id uintptr, label string) {
		text, _ := syscall.UTF16PtrFromString(label)
		flags := uintptr(0)
		if id == 4 && lookupsPaused() {
			flags = 8
		}
		user("AppendMenuW").Call(menu, flags, id, uintptr(unsafe.Pointer(text)))
	}
	add(1, "Open")
	add(2, "Settings")
	add(4, "Pause lookups")
	add(3, "Close")
	var pt struct{ X, Y int32 }
	user("GetCursorPos").Call(uintptr(unsafe.Pointer(&pt)))
	user("SetForegroundWindow").Call(hwnd)
	cmd, _, _ := user("TrackPopupMenu").Call(menu, 0x0002|0x0100, uintptr(pt.X), uintptr(pt.Y), 0, hwnd, 0)
	user("PostMessageW").Call(hwnd, 0, 0, 0)
	return cmd
}

func showValvet(settings bool) {
	if settings {
		askTray("settings")
	}
	hwnd := findValvet()
	if hwnd == 0 {
		return
	}
	user32.NewProc("ShowWindow").Call(hwnd, 9)
	user32.NewProc("SetForegroundWindow").Call(hwnd)
}

func closeValvet() {
	hwnd := findValvet()
	if hwnd != 0 {
		user32.NewProc("PostMessageW").Call(hwnd, 0x0010, 0, 0)
		return
	}
	os.Exit(0)
}

func desktopNotify(title, text string) {
	if trayHwnd == 0 {
		return
	}
	clip := func(value string, n int) string {
		value = strings.TrimSpace(value)
		if len(value) > n {
			value = value[:n]
		}
		return value
	}
	var tip [128]uint16
	copy(tip[:], syscall.StringToUTF16("Valvet"))
	var info [256]uint16
	var infoTitle [64]uint16
	copy(info[:], syscall.StringToUTF16(clip(text, 200)))
	copy(infoTitle[:], syscall.StringToUTF16(clip(title, 48)))
	icon := taskbarSmall
	if icon == 0 {
		icon = taskbarBig
	}
	data := struct {
		Size      uint32
		Hwnd      uintptr
		ID        uint32
		Flags     uint32
		Callback  uint32
		Icon      uintptr
		Tip       [128]uint16
		State     uint32
		StateMask uint32
		Info      [256]uint16
		Version   uint32
		InfoTitle [64]uint16
		InfoFlags uint32
		Guid      [16]byte
		Balloon   uintptr
	}{
		Hwnd:      trayHwnd,
		ID:        1,
		Flags:     0x10,
		Icon:      icon,
		Tip:       tip,
		Info:      info,
		InfoTitle: infoTitle,
		InfoFlags: 1,
	}
	data.Size = uint32(unsafe.Sizeof(data))
	syscall.NewLazyDLL("shell32.dll").NewProc("Shell_NotifyIconW").Call(1, uintptr(unsafe.Pointer(&data)))
}

func setAppUserModelID() {
	id, _ := syscall.UTF16PtrFromString("Danny.Valvet")
	syscall.NewLazyDLL("shell32.dll").NewProc("SetCurrentProcessExplicitAppUserModelID").Call(uintptr(unsafe.Pointer(id)))
}

var (
	taskbarBig    uintptr
	taskbarSmall  uintptr
	taskbarTries  int
	taskbarPinned bool
	trayOn        bool
	trayHwnd      uintptr
	trayProcKeep  uintptr
)

func setTaskbarIcon(hwnd uintptr) {
	if hwnd == 0 || taskbarTries >= 8 {
		return
	}
	taskbarTries++
	if taskbarBig == 0 {
		taskbarBig, taskbarSmall = loadValvetIcons()
	}
	if taskbarBig == 0 {
		taskbarTries = 0
		return
	}
	const wmSetIcon = 0x0080
	user32.NewProc("SendMessageW").Call(hwnd, wmSetIcon, 1, taskbarBig)
	small := taskbarSmall
	if small == 0 {
		small = taskbarBig
	}
	user32.NewProc("SendMessageW").Call(hwnd, wmSetIcon, 0, small)
	setClass := user32.NewProc("SetClassLongPtrW")
	setClass.Call(hwnd, ^uintptr(13), taskbarBig) // GCLP_HICON = -14
	setClass.Call(hwnd, ^uintptr(33), small)      // GCLP_HICONSM = -34
	const exStyle = ^uintptr(19)                  // GWL_EXSTYLE
	const appWindow = 0x00040000
	const toolWindow = 0x00000080
	get := user32.NewProc("GetWindowLongPtrW")
	set := user32.NewProc("SetWindowLongPtrW")
	style, _, _ := get.Call(hwnd, exStyle)
	next := (style | appWindow) &^ toolWindow
	if next != style {
		set.Call(hwnd, exStyle, next)
	}
	procSetWindowPos.Call(hwnd, 0, 0, 0, 0, 0, 0x0001|0x0002|0x0004|0x0020)
	if !taskbarPinned {
		if exe, err := os.Executable(); err == nil && exe != "" {
			pinTaskbarIcon(hwnd, exe)
			taskbarPinned = true
		}
	}
}

func loadValvetIcons() (uintptr, uintptr) {
	exe, err := os.Executable()
	if err != nil || exe == "" {
		return 0, 0
	}
	kernel := syscall.NewLazyDLL("kernel32.dll")
	mod, _, _ := kernel.NewProc("GetModuleHandleW").Call(0)
	load := user32.NewProc("LoadImageW")
	big, _, _ := load.Call(mod, 1, 1, 32, 32, 0)
	small, _, _ := load.Call(mod, 1, 1, 16, 16, 0)
	if big != 0 {
		if small == 0 {
			small = big
		}
		return big, small
	}
	path, _ := syscall.UTF16PtrFromString(exe)
	var large, little uintptr
	n, _, _ := user32.NewProc("ExtractIconExW").Call(uintptr(unsafe.Pointer(path)), 0, uintptr(unsafe.Pointer(&large)), uintptr(unsafe.Pointer(&little)), 1)
	if n == 0 || large == 0 || large == 1 {
		return 0, 0
	}
	if little == 0 || little == 1 {
		little = large
	}
	return large, little
}

func pinTaskbarIcon(hwnd uintptr, exe string) {
	ole := syscall.NewLazyDLL("ole32.dll")
	ole.NewProc("CoInitializeEx").Call(0, 2)
	shell := syscall.NewLazyDLL("shell32.dll")
	iid := [16]byte{0xEB, 0x8E, 0x6D, 0x88, 0xF2, 0x8C, 0x46, 0x44, 0x8D, 0x02, 0xCD, 0xBA, 0x1D, 0xBD, 0xCF, 0x99}
	var store uintptr
	hr, _, _ := shell.NewProc("SHGetPropertyStoreForWindow").Call(hwnd, uintptr(unsafe.Pointer(&iid)), uintptr(unsafe.Pointer(&store)))
	if hr != 0 || store == 0 {
		return
	}
	vtbl := *(*uintptr)(unsafe.Pointer(store))
	if vtbl == 0 {
		return
	}
	procs := *(*[8]uintptr)(unsafe.Pointer(vtbl))
	put := func(pid uint32, value string) {
		key := struct {
			fmtid [16]byte
			pid   uint32
		}{
			fmtid: [16]byte{0x55, 0x28, 0x4C, 0x9F, 0x79, 0x9F, 0x39, 0x4B, 0xA8, 0xD0, 0xE1, 0xD4, 0x2D, 0xE1, 0xD5, 0xF3},
			pid:   pid,
		}
		text, _ := syscall.UTF16PtrFromString(value)
		var pv struct {
			vt  uint16
			r1  uint16
			r2  uint16
			r3  uint16
			val uintptr
			pad uintptr
		}
		pv.vt = 31
		pv.val = uintptr(unsafe.Pointer(text))
		syscall.SyscallN(procs[6], store, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&pv)))
	}
	put(5, "Danny.Valvet")
	put(3, exe+",0")
	put(2, `"`+exe+`"`)
	syscall.SyscallN(procs[7], store)
	syscall.SyscallN(procs[2], store)
}

func paintClientGap(hwnd uintptr) {
	if hwnd == 0 || gapBrush != 0 {
		if hwnd != 0 && gapBrush != 0 {
			user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(9), gapBrush)
		}
		return
	}
	gdi := syscall.NewLazyDLL("gdi32.dll")
	brush, _, _ := gdi.NewProc("CreateSolidBrush").Call(0x000D0810)
	if brush == 0 {
		return
	}
	gapBrush = brush
	// GCLP_HBRBACKGROUND = -10. Any pixel the page does not cover stays the same dark colour, not white.
	user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(9), brush)
	user32.NewProc("InvalidateRect").Call(hwnd, 0, 1)
}

var gapBrush uintptr

var frameColors struct {
	mu     sync.Mutex
	bg     string
	accent string
}

func paintDarkTitle(hwnd uintptr) {
	frameColors.mu.Lock()
	bg, accent := frameColors.bg, frameColors.accent
	frameColors.mu.Unlock()
	if bg == "" {
		bg = "#10080d"
	}
	if accent == "" {
		accent = "#ff5fa2"
	}
	applyFrame(hwnd, bg, accent)
}

func paintThemeFrame(hwnd uintptr, bg, accent string) {
	frameColors.mu.Lock()
	if bg != "" {
		frameColors.bg = bg
	}
	if accent != "" {
		frameColors.accent = accent
	}
	bg, accent = frameColors.bg, frameColors.accent
	frameColors.mu.Unlock()
	applyFrame(hwnd, bg, accent)
}

func applyFrame(hwnd uintptr, bg, accent string) {
	if hwnd == 0 {
		return
	}
	use := int32(1)
	proc := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmSetWindowAttribute")
	proc.Call(hwnd, 20, uintptr(unsafe.Pointer(&use)), unsafe.Sizeof(use))
	proc.Call(hwnd, 19, uintptr(unsafe.Pointer(&use)), unsafe.Sizeof(use))
	border, ok := colorRef(bg)
	if !ok {
		border = 0x000D0810
	}
	text, ok := colorRef(accent)
	if !ok {
		text = 0x00A25FFF
	}
	proc.Call(hwnd, 34, uintptr(unsafe.Pointer(&border)), 4)
	proc.Call(hwnd, 35, uintptr(unsafe.Pointer(&border)), 4)
	proc.Call(hwnd, 36, uintptr(unsafe.Pointer(&text)), 4)
	gdi := syscall.NewLazyDLL("gdi32.dll")
	brush, _, _ := gdi.NewProc("CreateSolidBrush").Call(uintptr(border))
	if brush != 0 {
		user32.NewProc("SetClassLongPtrW").Call(hwnd, ^uintptr(9), brush)
		if gapBrush != 0 {
			gdi.NewProc("DeleteObject").Call(gapBrush)
		}
		gapBrush = brush
		user32.NewProc("InvalidateRect").Call(hwnd, 0, 1)
	}
}

func ownFrame(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	get := user32.NewProc("GetWindowLongPtrW")
	set := user32.NewProc("SetWindowLongPtrW")
	style, _, _ := get.Call(hwnd, ^uintptr(15))
	const wsCaption = 0x00C00000
	if style&^wsCaption != style {
		style &^= wsCaption
		set.Call(hwnd, ^uintptr(15), style)
		const flags = 0x0001 | 0x0002 | 0x0004 | 0x0020
		user32.NewProc("SetWindowPos").Call(hwnd, 0, 0, 0, 0, 0, flags)
	}
	installFrameDrag(hwnd)
}

func windowCommand(hwnd uintptr, cmd string) {
	if hwnd == 0 {
		return
	}
	show := user32.NewProc("ShowWindow")
	switch cmd {
	case "min":
		show.Call(hwnd, 6)
	case "max":
		zoomed, _, _ := user32.NewProc("IsZoomed").Call(hwnd)
		if zoomed != 0 {
			show.Call(hwnd, 9)
		} else {
			show.Call(hwnd, 3)
		}
	case "close":
		user32.NewProc("PostMessageW").Call(hwnd, 0x0010, 0, 0)
	}
}

func dragWindow(hwnd uintptr) {
	if hwnd == 0 {
		return
	}
	user32.NewProc("ReleaseCapture").Call()
	user32.NewProc("SendMessageW").Call(hwnd, 0x00A1, 2, 0)
}

func colorRef(hex string) (uint32, bool) {
	hex = strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(hex) != 6 {
		return 0, false
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, false
	}
	r := uint32((n >> 16) & 255)
	g := uint32((n >> 8) & 255)
	b := uint32(n & 255)
	return b<<16 | g<<8 | r, true
}

func appsUseDark() bool {
	adv := syscall.NewLazyDLL("advapi32.dll")
	var h syscall.Handle
	key, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`)
	r, _, _ := adv.NewProc("RegOpenKeyExW").Call(
		uintptr(0x80000001),
		uintptr(unsafe.Pointer(key)),
		0,
		0x20019,
		uintptr(unsafe.Pointer(&h)),
	)
	if r != 0 {
		return false
	}
	defer adv.NewProc("RegCloseKey").Call(uintptr(h))
	name, _ := syscall.UTF16PtrFromString("AppsUseLightTheme")
	var val uint32
	var n uint32 = 4
	r, _, _ = adv.NewProc("RegQueryValueExW").Call(
		uintptr(h),
		uintptr(unsafe.Pointer(name)),
		0,
		0,
		uintptr(unsafe.Pointer(&val)),
		uintptr(unsafe.Pointer(&n)),
	)
	return r == 0 && val == 0
}

package main

import "sync/atomic"

var trayTab atomic.Value

func init() {
	trayTab.Store("")
}

func askTray(tab string) {
	if tab != "" {
		trayTab.Store(tab)
	}
}

func takeTrayTab() string {
	v := trayTab.Swap("")
	s, _ := v.(string)
	return s
}

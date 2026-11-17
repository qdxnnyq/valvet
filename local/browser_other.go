//go:build !windows

package main

func openBrowser(url string) {}

func openApp(string) {
	select {}
}

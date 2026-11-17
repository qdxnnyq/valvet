//go:build !windows

package main

func wantsUninstall() bool { return false }

func registerInstalled() {}

func uninstallValvet() {}

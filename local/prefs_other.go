//go:build !windows

package main

func maybeWaitForRiot() {}

func launchMode() string { return "off" }

func setLaunchMode(string) string { return "off" }

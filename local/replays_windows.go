//go:build windows

package main

import (
	"errors"
	"path/filepath"
	"strings"
)

func openReplayFile(path string) error {
	if !valorantOpen() {
		return errors.New("Valorant isn't open.")
	}
	placeDemo(path)
	id := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if askValorantReplay(id) {
		return nil
	}
	return errors.New("Valorant is open. The replay is in your Demos folder.")
}

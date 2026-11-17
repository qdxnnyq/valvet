//go:build !windows

package main

import "errors"

func openReplayFile(path string) error {
	return errors.New("Valorant replays open on Windows.")
}

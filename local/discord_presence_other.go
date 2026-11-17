//go:build !windows

package main

import "errors"

func setPresence(string, string, string) error {
	return errors.New("Discord profile status runs in the Windows app.")
}

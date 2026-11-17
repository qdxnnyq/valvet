//go:build !windows

package main

import "errors"

func installUpdate(raw string) error {
	return errors.New("updates install from the Windows app")
}

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFormatBytes(t *testing.T) {
	if formatBytes(0) != "0 B" {
		t.Fatal(formatBytes(0))
	}
	if !strings.Contains(formatBytes(3<<20), "MB") {
		t.Fatal(formatBytes(3 << 20))
	}
}

func TestDirSize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abcd"), 0o644); err != nil {
		t.Fatal(err)
	}
	if dirSize(dir) != 4 {
		t.Fatal(dirSize(dir))
	}
}

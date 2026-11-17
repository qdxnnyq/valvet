package main

import "testing"

func TestPatchPath(t *testing.T) {
	if patchPath("/en-us/news/game-updates/valorant-patch-notes-13-10") == "" {
		t.Fatal("allowed path")
	}
	if patchPath("https://evil.example/en-us/news/x") != "" || patchPath("/en-us/news/../secret") != "" {
		t.Fatal("blocked path")
	}
}

func TestEsportName(t *testing.T) {
	if !esportName("VCT 2026 Sentinels Capsule") || esportName("Reaver") {
		t.Fatal("filter")
	}
}

func TestHTMLText(t *testing.T) {
	got := htmlText("<h2>General</h2><ul><li>Bookmarks</li></ul>")
	if got != "General\n• Bookmarks" && got != "• General\n• Bookmarks" {
		if !contains(got, "General") || !contains(got, "Bookmarks") {
			t.Fatal(got)
		}
	}
}

func contains(s, part string) bool {
	return len(s) >= len(part) && (s == part || len(part) == 0 || (len(s) > 0 && (stringIndex(s, part) >= 0)))
}

func stringIndex(s, part string) int {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return i
		}
	}
	return -1
}

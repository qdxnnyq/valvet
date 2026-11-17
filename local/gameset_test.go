package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestExtractGameRows(t *testing.T) {
	riot := "EAresFloatSettingName:: MouseSensitivity =0.314\nEAresFloatSettingName:: ScopedSensitivityMultiplier =1.0\n"
	game := "ResolutionSizeX=1280\nResolutionSizeY=720\nLastUserConfirmedResolutionSizeX=1920\nLastUserConfirmedResolutionSizeY=1080\nLastConfirmedFullscreenMode=1\nFullscreenMode=0\nFrameRateLimit=0.000000\nbUseVSync=False\nsg.ShadowQuality=2\nMouseSensitivity=1.000000\n"
	rows := extractGameRowsFrom(riot, game, "")
	got := map[string]string{}
	for _, row := range rows {
		got[row.Label] = row.Value
	}
	if got["Sensitivity"] != "0.314" || got["Scoped sensitivity"] != "1" {
		t.Fatalf("sens %+v", got)
	}
	if got["Resolution"] != "1920 × 1080" || got["Display mode"] != "Borderless" || got["Frame limit"] != "Unlimited" || got["VSync"] != "Off" || got["Shadows"] != "High" {
		t.Fatalf("%+v", got)
	}
}

func TestSensitivityPrefersAccount(t *testing.T) {
	game := "MouseSensitivity=0.070000\nResolutionSizeX=1920\nResolutionSizeY=1080\n"
	account := "EAresFloatSettingName::MouseSensitivity =0.314\nEAresFloatSettingName::ScopedSensitivityMultiplier =1.000000\n"
	rows := extractGameRowsFrom("", game, account)
	got := map[string]string{}
	for _, row := range rows {
		got[row.Label] = row.Value
	}
	if got["Sensitivity"] != "0.314" || got["Scoped sensitivity"] != "1" || got["Resolution"] != "1920 × 1080" {
		t.Fatalf("%+v", got)
	}
}

func TestSensitivityUsesSliderNotEngineDefault(t *testing.T) {
	riot := "EAresFloatSettingName:: MouseSensitivity =0.272000\nEAresFloatSettingName:: MouseSensitivity =0.900\nEAresFloatSettingName:: ScopedSensitivityMultiplier =1.000000\n"
	game := "MouseSensitivity=1.000000\n"
	account := "EAresFloatSettingName::MouseSensitivity =0.111\n"
	rows := extractGameRowsFrom(riot, game, account)
	got := map[string]string{}
	for _, row := range rows {
		got[row.Label] = row.Value
	}
	if got["Sensitivity"] != "0.9" {
		t.Fatalf("slider %+v", got)
	}
	rows = extractGameRowsFrom("", "MouseSensitivity=1.000000\n", "")
	if len(rows) != 0 {
		t.Fatalf("engine default leaked %+v", rows)
	}
}

func TestNewestRiotProfile(t *testing.T) {
	root := t.TempDir()
	oldDir := filepath.Join(root, "old-na", "Windows")
	newDir := filepath.Join(root, "new-eu", "Windows")
	if err := os.MkdirAll(oldDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := filepath.Join(oldDir, "RiotUserSettings.ini")
	newer := filepath.Join(newDir, "RiotUserSettings.ini")
	if err := os.WriteFile(old, []byte("EAresFloatSettingName:: MouseSensitivity =0.100\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newer, []byte("EAresFloatSettingName:: MouseSensitivity =0.272\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(newDir, "GameUserSettings.ini"), []byte("MouseSensitivity=1\nResolutionSizeX=2560\nResolutionSizeY=1440\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(old, past, past); err != nil {
		t.Fatal(err)
	}
	riot, game := readValorantConfig(root)
	rows := extractGameRowsFrom(riot, game, "")
	got := map[string]string{}
	for _, row := range rows {
		got[row.Label] = row.Value
	}
	if got["Sensitivity"] != "0.272" || got["Resolution"] != "2560 × 1440" {
		t.Fatalf("%+v", got)
	}
}

func TestPrizeAmount(t *testing.T) {
	if prizeAmount("$2,250,000") != 2250000 || prizeAmount("Regional") != 0 {
		t.Fatal(prizeAmount("$2,250,000"))
	}
}

func TestLiveEventsOnly(t *testing.T) {
	rows := sortEvents([]sportEvent{
		{Name: "Old", Status: "completed", Prize: "$9,000,000"},
		{Name: "Next", Status: "upcoming", Prize: "$100,000"},
		{Name: "Now", Status: "ongoing", Prize: "$50,000"},
		{Name: "Soon", Status: "upcoming", Prize: "$2,000,000"},
	})
	if len(rows) != 3 || rows[0].Name != "Now" || rows[1].Name != "Soon" || rows[2].Name != "Next" {
		t.Fatalf("%+v", rows)
	}
}

func TestStatDayCutsAtSix(t *testing.T) {
	loc := time.Local
	early := time.Date(2026, 11, 14, 5, 30, 0, 0, loc)
	late := time.Date(2026, 11, 14, 6, 0, 0, 0, loc)
	if statDayStart(early).Day() != 13 {
		t.Fatal(statDayStart(early))
	}
	if statDayStart(late).Day() != 14 {
		t.Fatal(statDayStart(late))
	}
}

func TestFreePassDetection(t *testing.T) {
	event := map[string]any{"displayName": "Give Back", "content": map[string]any{"relationType": "Event", "premiumVPCost": 0}}
	paid := map[string]any{"displayName": "Episode Act", "content": map[string]any{"relationType": "Season", "premiumVPCost": 1000}}
	agent := map[string]any{"displayName": "Jett", "content": map[string]any{"relationType": "Agent"}}
	named := map[string]any{"displayName": "Free Pass", "content": map[string]any{"relationType": "Season", "premiumVPCost": 1000}}
	if !isPass(event) || !isFreePass(event) || !isPass(paid) || isFreePass(paid) || isPass(agent) || !isFreePass(named) {
		t.Fatal("pass kinds")
	}
}

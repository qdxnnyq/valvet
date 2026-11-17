package main

import (
	"strings"
	"testing"
)

func TestRoomTopic(t *testing.T) {
	if _, ok := roomTopic("ab"); ok {
		t.Fatal("short")
	}
	if _, ok := roomTopic("bad code"); ok {
		t.Fatal("space")
	}
	got, ok := roomTopic("ab12")
	if !ok || !strings.HasPrefix(got, "valvet-") || strings.Contains(strings.ToLower(got), "ab12") || strings.Contains(got, "velvet") {
		t.Fatal(got, ok)
	}
	other, ok := roomTopic("ab13")
	if !ok || other == got {
		t.Fatal(other)
	}
}

func TestBoardFields(t *testing.T) {
	got := cleanPerson(roomPerson{ID: "abc", Age: 15, Gender: "nope", Queue: "Competitive", Games: 12, Note: "  evenings  "})
	if got.Age != 0 || got.Gender != "" || got.Queue != "Competitive" || got.Games != 12 || got.Note != "evenings" {
		t.Fatalf("%+v", got)
	}
	got = cleanPerson(roomPerson{ID: "abc", Age: 21, Gender: "nonbinary", Queue: "ranked"})
	if got.Age != 21 || got.Gender != "Non-binary" || got.Queue != "" {
		t.Fatalf("%+v", got)
	}
}

func TestAggregateRates(t *testing.T) {
	rows := []boardRate{
		{From: "aa", Target: "nova#1", Stars: 5, At: 1},
		{From: "aa", Target: "nova#1", Stars: 4, At: 2},
		{From: "bb", Target: "nova#1", Stars: 2, At: 3},
		{From: "cc", Target: "bad", Stars: 5, At: 4},
	}
	if _, ok := cleanRate(boardRate{From: "aa", Riot: "nova#1", Target: "Nova#1", Stars: 5}); ok {
		t.Fatal("self rate")
	}
	clean := []boardRate{}
	for _, row := range rows {
		if row, ok := cleanRate(row); ok {
			clean = append(clean, row)
		}
	}
	got := aggregateRates(clean)
	sum := got["nova#1"]
	if sum.N != 2 || sum.Avg != 3 {
		t.Fatalf("%+v", got)
	}
}

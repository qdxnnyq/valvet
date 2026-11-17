package main

import "testing"

func TestGunLabel(t *testing.T) {
	if gunLabel("9C82E19D-4575-0200-1A81-3EACF00CF872") != "Vandal" {
		t.Fatal(gunLabel("9C82E19D-4575-0200-1A81-3EACF00CF872"))
	}
	if gunsText(map[string]int{"Phantom": 2, "Vandal": 4}) != "Vandal:4,Phantom:2" {
		t.Fatal(gunsText(map[string]int{"Phantom": 2, "Vandal": 4}))
	}
}

func TestRecordView(t *testing.T) {
	won, lost := true, false
	friend, random, rival := "friend", "random", "rival"
	rows := []recordGame{
		{ID: "a", Map: "Ascent", Won: &won, Kills: 20, ACS: 300, Guns: "Vandal:10", Faces: []recordFace{{ID: friend, Name: "Ada", Side: "ally"}, {ID: random, Name: "Pat", Side: "ally"}, {ID: rival, Name: "Rex", Side: "enemy"}}},
		{ID: "b", Map: "Ascent", Won: &lost, Kills: 10, ACS: 180, Guns: "Vandal:4,Phantom:2", Faces: []recordFace{{ID: friend, Name: "Ada", Side: "ally"}, {ID: random, Name: "Pat", Side: "ally"}, {ID: rival, Name: "Rex", Side: "enemy"}}},
		{ID: "c", Map: "Bind", Won: &won, Kills: 8, ACS: 140, Guns: "Phantom:1", Faces: []recordFace{{ID: friend, Name: "Ada", Side: "ally"}, {ID: rival, Name: "Rex", Side: "enemy"}}},
	}
	view := buildRecordView(rows, map[string]bool{friend: true})
	if view.Games != 3 || view.Wins != 2 || view.BestKills != 20 || view.BestGun != "Vandal" {
		t.Fatalf("%d %d %d %s", view.Games, view.Wins, view.BestKills, view.BestGun)
	}
}

func TestNiceGun(t *testing.T) {
	if niceGun("29A0CFAB-485B-F5D5-779A-B59F85E204A8") != "Classic" {
		t.Fatal(niceGun("29A0CFAB-485B-F5D5-779A-B59F85E204A8"))
	}
	if niceGun("GrenadeAbility") != "Grenade" {
		t.Fatal(niceGun("GrenadeAbility"))
	}
	if edgeLabel(player{You: true, Agent: "Jett"}) != "You" {
		t.Fatal("you")
	}
	if edgeLabel(player{Name: "Player", Agent: "Reyna"}) != "Reyna" {
		t.Fatal(edgeLabel(player{Name: "Player", Agent: "Reyna"}))
	}
}

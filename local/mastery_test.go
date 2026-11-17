package main

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestLeagueOnlyWhenPlaying(t *testing.T) {
	mobile := map[string]any{"product": "league_of_legends", "state": "mobile"}
	if bucket, state := presenceBucket(mobile); bucket != "riot" || state != "Mobile" {
		t.Fatalf("mobile %+v %s", bucket, state)
	}
	chat := map[string]any{"product": "league_of_legends", "state": "chat"}
	if bucket, state := presenceBucket(chat); bucket != "league" || state != "Online" {
		t.Fatalf("league client %+v %s", bucket, state)
	}
	dnd := map[string]any{"product": "league_of_legends", "state": "dnd"}
	if bucket, state := presenceBucket(dnd); bucket != "league" || state != "Online" {
		t.Fatalf("league dnd %+v %s", bucket, state)
	}
	priv := base64.StdEncoding.EncodeToString([]byte(`{"gameStatus":"inGame"}`))
	live := map[string]any{"product": "league_of_legends", "state": "dnd", "private": priv}
	if bucket, state := presenceBucket(live); bucket != "league" || state != "In game" {
		t.Fatalf("in game %+v %s", bucket, state)
	}
	queuePriv := base64.StdEncoding.EncodeToString([]byte(`{"gameStatus":"inQueue"}`))
	queue := map[string]any{"product": "lol", "state": "dnd", "private": queuePriv}
	if bucket, state := presenceBucket(queue); bucket != "league" || state != "In queue" {
		t.Fatalf("queue %+v %s", bucket, state)
	}
	outPriv := base64.StdEncoding.EncodeToString([]byte(`{"gameStatus":"outOfGame","championId":"1"}`))
	sitting := map[string]any{"product": "league_of_legends", "state": "chat", "private": outPriv}
	if bucket, state := presenceBucket(sitting); bucket != "league" || state != "Online" {
		t.Fatalf("league client %+v %s", bucket, state)
	}
	nestedRaw := base64.StdEncoding.EncodeToString([]byte(`{"dat":{"gameStatus":"inGame"}}`))
	nested := map[string]any{"product": "league_of_legends", "state": "dnd", "private": nestedRaw}
	if bucket, state := presenceBucket(nested); bucket != "league" || state != "In game" {
		t.Fatalf("nested %+v %s", bucket, state)
	}
	away := map[string]any{"product": "league_of_legends", "state": "away"}
	if bucket, state := presenceBucket(away); bucket != "league" || state != "AFK" {
		t.Fatalf("afk %+v %s", bucket, state)
	}
	phone := map[string]any{"product": "valorant", "state": "mobile"}
	if bucket, state := presenceBucket(phone); bucket != "riot" || state != "Mobile" {
		t.Fatalf("valorant mobile %+v %s", bucket, state)
	}
	android := map[string]any{"product": "keystone", "state": "chat", "platform": "android"}
	if bucket, state := presenceBucket(android); bucket != "riot" || state != "Mobile" {
		t.Fatalf("android %+v %s", bucket, state)
	}
	riot := map[string]any{"product": "riotclient", "state": "chat"}
	if bucket, state := presenceBucket(riot); bucket != "riot" || state != "In Riot" {
		t.Fatalf("riot %+v %s", bucket, state)
	}
	keystone := map[string]any{"product": "keystone", "state": "chat"}
	if bucket, state := presenceBucket(keystone); bucket != "riot" || state != "In Riot" {
		t.Fatalf("keystone %+v %s", bucket, state)
	}
	lion := map[string]any{"product": "lion", "state": "chat"}
	if bucket, state := presenceBucket(lion); bucket != "league" || state != "Online" {
		t.Fatalf("lion %+v %s", bucket, state)
	}
	hidden := map[string]any{"product": "keystone", "state": "dnd", "pid": "abc@lol.pvp.net"}
	if bucket, state := presenceBucket(hidden); bucket != "league" || state != "Online" {
		t.Fatalf("pid %+v %s", bucket, state)
	}
	riotGame := map[string]any{"product": "riotclient", "state": "dnd", "private": priv}
	if bucket, state := presenceBucket(riotGame); bucket != "league" || state != "In game" {
		t.Fatalf("riot private %+v %s", bucket, state)
	}
	bucket, state, _ := classifyPresences([]map[string]any{
		{"product": "riotclient", "state": "chat", "puuid": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
		{"product": "league_of_legends", "state": "dnd", "puuid": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"},
	})
	if bucket != "league" || state != "Online" {
		t.Fatalf("mixed %+v %s", bucket, state)
	}
	if partyIDFrom(map[string]any{"CurrentPartyID": "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"}) == "" {
		t.Fatal("party id")
	}
}

func TestMasteryParse(t *testing.T) {
	blob := map[string]any{
		"Agents": map[string]any{
			"add6443a-41bd-e414-f6ad-e58d267f4e95": map[string]any{
				"ActLevel":      4,
				"LifetimeLevel": 12,
				"MasteryPoints": 80000,
			},
		},
	}
	got := collectMastery([]any{blob}, nil)
	row := got["add6443a-41bd-e414-f6ad-e58d267f4e95"]
	if row == nil || row.Act != 4 || row.Lifetime != 12 || row.Points != 80000 {
		t.Fatalf("%+v", row)
	}
	nested := map[string]any{
		"AgentID":  "add6443a-41bd-e414-f6ad-e58d267f4e95",
		"Act":      map[string]any{"Level": 7},
		"Lifetime": map[string]any{"Level": 18},
	}
	gotNested := collectMastery([]any{nested}, nil)
	nestedRow := gotNested["add6443a-41bd-e414-f6ad-e58d267f4e95"]
	if nestedRow == nil || nestedRow.Act != 7 || nestedRow.Lifetime != 18 {
		t.Fatalf("nested %+v", nestedRow)
	}
	def := map[string]any{
		"Definitions": []any{
			map[string]any{
				"ID":   "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				"Name": "Jett Mastery",
				"Content": map[string]any{
					"RelationType": "AgentMastery",
					"RelationUuid": "add6443a-41bd-e414-f6ad-e58d267f4e95",
					"Levels": []any{
						map[string]any{
							"Level":  5,
							"Name":   "Jett Spray",
							"ItemID": "11111111-2222-3333-4444-555555555555",
							"Cost":   map[string]any{"85ca954a-41f2-ce94-9b45-8ca3dd39a00d": 4000},
						},
					},
				},
			},
			map[string]any{
				"ID":   "bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee",
				"Name": "Jett Gear",
				"Content": map[string]any{
					"RelationType": "Agent",
					"RelationUuid": "add6443a-41bd-e414-f6ad-e58d267f4e95",
				},
			},
		},
	}
	links, rewards := masteryCatalog(def)
	if links["aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"] != "add6443a-41bd-e414-f6ad-e58d267f4e95" {
		t.Fatalf("links %+v", links)
	}
	if links["bbbbbbbb-bbbb-cccc-dddd-eeeeeeeeeeee"] != "" {
		t.Fatal("old gear leaked")
	}
	if len(rewards["add6443a-41bd-e414-f6ad-e58d267f4e95"]) != 1 || rewards["add6443a-41bd-e414-f6ad-e58d267f4e95"][0].Cost != 4000 {
		t.Fatalf("rewards %+v", rewards)
	}
}

func TestBundleRows(t *testing.T) {
	body := map[string]any{
		"FeaturedBundle": map[string]any{
			"BundleRemainingDurationInSeconds": 3600,
			"Bundle": map[string]any{
				"DataAssetID": "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
				"Items": []any{
					map[string]any{
						"Item":            map[string]any{"ItemTypeID": "e7c63390-eda7-46e0-bb7a-a6abdacd2433", "ItemID": "ed8a1109-4e48-f077-636b-e98dd332bfcc"},
						"BasePrice":       1775,
						"DiscountedPrice": 1775,
					},
				},
				"TotalDiscountedCost": map[string]any{"85ad13f7-3d1b-5128-9eb2-7cd8ee0b5741": 8700},
			},
		},
	}
	rows := (&riotClient{}).featuredBundles(body)
	if len(rows) != 1 || rows[0].Price != 8700 || len(rows[0].Items) != 1 || rows[0].Items[0].Icon == "" {
		t.Fatalf("%+v", rows)
	}
	if !strings.Contains(rows[0].Icon, "bundles/aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee") {
		t.Fatalf("icon %s", rows[0].Icon)
	}
}

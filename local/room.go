package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
	"unicode"
)

func ntfyTopic(name string) string {
	sum := sha256.Sum256([]byte("valvet.board.v1\n" + name))
	return "valvet-" + hex.EncodeToString(sum[:10])
}

func roomTopic(code string) (string, bool) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if len(code) < 4 || len(code) > 12 {
		return "", false
	}
	for _, r := range code {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return "", false
		}
	}
	return ntfyTopic("room." + code), true
}

type roomPerson struct {
	ID      string   `json:"id"`
	Nick    string   `json:"nick"`
	Game    string   `json:"game"`
	Phase   string   `json:"phase"`
	Detail  string   `json:"detail"`
	Map     string   `json:"map"`
	Mode    string   `json:"mode"`
	Score   string   `json:"score"`
	Rank    string   `json:"rank"`
	Riot    string   `json:"riot"`
	Discord string   `json:"discord"`
	Agent   string   `json:"agent"`
	Party   string   `json:"party"`
	Full    bool     `json:"full"`
	Ready   bool     `json:"ready"`
	Looking bool     `json:"looking"`
	Board   bool     `json:"board,omitempty"`
	Room    string   `json:"room"`
	Low     string   `json:"low"`
	High    string   `json:"high"`
	Server  string   `json:"server"`
	Until   int64    `json:"until"`
	Roles   []string `json:"roles"`
	Modes   []string `json:"modes"`
	Wish    []string `json:"wish"`
	Watch   []string `json:"watch"`
	Owned   []string `json:"owned"`
	Ping    string   `json:"ping"`
	At      int64    `json:"at"`
	Age     int      `json:"age,omitempty"`
	Gender  string   `json:"gender,omitempty"`
	Note    string   `json:"note,omitempty"`
	Queue   string   `json:"queue,omitempty"`
	Games   int      `json:"games,omitempty"`
}

func cleanPerson(in roomPerson) roomPerson {
	trim := func(s string, n int) string {
		s = strings.TrimSpace(s)
		if len(s) > n {
			return s[:n]
		}
		return s
	}
	list := func(rows []string, n int) []string {
		out := []string{}
		seen := map[string]bool{}
		for _, row := range rows {
			row = trim(row, 40)
			key := strings.ToLower(row)
			if row == "" || seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, row)
			if len(out) >= n {
				break
			}
		}
		return out
	}
	in.ID = trim(in.ID, 24)
	in.Nick = trim(in.Nick, 24)
	in.Game = trim(in.Game, 16)
	in.Phase = trim(in.Phase, 32)
	in.Detail = trim(in.Detail, 80)
	in.Map = trim(in.Map, 32)
	in.Mode = trim(in.Mode, 32)
	in.Score = trim(in.Score, 16)
	in.Rank = trim(in.Rank, 32)
	in.Riot = trim(in.Riot, 32)
	in.Discord = trim(in.Discord, 32)
	in.Agent = trim(in.Agent, 24)
	in.Party = trim(in.Party, 8)
	in.Low = trim(in.Low, 16)
	in.High = trim(in.High, 16)
	in.Server = trim(in.Server, 8)
	in.Room = strings.ToUpper(trim(in.Room, 12))
	in.Ping = trim(in.Ping, 40)
	in.Note = trim(in.Note, 160)
	in.Queue = boardQueue(in.Queue)
	in.Gender = boardGender(in.Gender)
	if in.Age < 16 || in.Age > 99 {
		in.Age = 0
	}
	if in.Games < 0 || in.Games > 20000 {
		in.Games = 0
	}
	in.Roles = list(in.Roles, 5)
	in.Modes = list(in.Modes, 8)
	in.Wish = list(in.Wish, 24)
	in.Watch = list(in.Watch, 24)
	in.Owned = list(in.Owned, 24)
	if in.Nick == "" {
		in.Nick = "Player"
	}
	if in.At == 0 {
		in.At = time.Now().Unix()
	}
	return in
}

func publishRoom(ctx context.Context, topic string, person roomPerson) error {
	person = cleanPerson(person)
	raw, err := json.Marshal(person)
	if err != nil {
		return err
	}
	if len(raw) > 3500 {
		person.Wish = person.Wish[:min(8, len(person.Wish))]
		person.Watch = person.Watch[:min(8, len(person.Watch))]
		person.Owned = person.Owned[:min(8, len(person.Owned))]
		raw, _ = json.Marshal(person)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://ntfy.sh/"+topic, strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", "Valvet")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
	if res.StatusCode >= 300 {
		return errMissing
	}
	return nil
}

func boardGender(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "woman":
		return "Woman"
	case "man":
		return "Man"
	case "non-binary", "nonbinary":
		return "Non-binary"
	default:
		return ""
	}
}

func boardQueue(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "Competitive", "Unrated", "Swiftplay", "Spike Rush", "Deathmatch", "Premier", "Escalation", "Team Deathmatch", "Replication", "Custom":
		return s
	default:
		return ""
	}
}

func readRoom(ctx context.Context, topic string) []roomPerson {
	return readRoomWindow(ctx, topic, 8*time.Minute, "8m")
}

func readBoard(ctx context.Context) []roomPerson {
	rows := readRoomWindow(ctx, ntfyTopic("lfg"), 12*time.Hour, "12h")
	out := []roomPerson{}
	for _, person := range rows {
		if person.Looking {
			out = append(out, person)
		}
	}
	return out
}

func readRoomWindow(ctx context.Context, topic string, window time.Duration, since string) []roomPerson {
	payload := ntfyPoll(ctx, topic, since)
	latest := map[string]roomPerson{}
	order := []string{}
	for _, line := range strings.Split(string(payload), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) != nil {
			continue
		}
		if str(pick(row, "event")) != "message" {
			continue
		}
		var person roomPerson
		if json.Unmarshal([]byte(str(pick(row, "message"))), &person) != nil || person.ID == "" {
			continue
		}
		person = cleanPerson(person)
		if person.ID == "diag" || strings.EqualFold(person.Riot, "Diag#0000") || strings.EqualFold(person.Nick, "diag") {
			continue
		}
		if when := int64(num(pick(row, "time"))); when > 0 {
			person.At = when
		}
		if person.At > 0 && time.Since(time.Unix(person.At, 0)) > window {
			continue
		}
		if _, ok := latest[person.ID]; !ok {
			order = append(order, person.ID)
		}
		latest[person.ID] = person
	}
	out := []roomPerson{}
	for _, id := range order {
		out = append(out, latest[id])
	}
	return out
}

type boardRate struct {
	From   string `json:"from"`
	Riot   string `json:"riot,omitempty"`
	Target string `json:"target"`
	Stars  int    `json:"stars"`
	At     int64  `json:"at"`
}

type rateSummary struct {
	Avg float64 `json:"avg"`
	N   int     `json:"n"`
}

func cleanRate(in boardRate) (boardRate, bool) {
	in.From = strings.TrimSpace(in.From)
	in.Riot = strings.ToLower(strings.TrimSpace(in.Riot))
	in.Target = strings.ToLower(strings.TrimSpace(in.Target))
	if len(in.From) < 2 || len(in.From) > 32 || !strings.Contains(in.Target, "#") || len(in.Target) > 40 {
		return boardRate{}, false
	}
	if in.Riot != "" && in.Riot == in.Target {
		return boardRate{}, false
	}
	if in.Stars < 1 || in.Stars > 5 {
		return boardRate{}, false
	}
	if strings.EqualFold(in.From, "diag") {
		return boardRate{}, false
	}
	if in.At == 0 {
		in.At = time.Now().Unix()
	}
	return in, true
}

func aggregateRates(rows []boardRate) map[string]rateSummary {
	type pair struct{ stars, at int64 }
	latest := map[string]pair{}
	for _, row := range rows {
		key := row.From + "|" + row.Target
		prev, ok := latest[key]
		if ok && prev.at > row.At {
			continue
		}
		latest[key] = pair{stars: int64(row.Stars), at: row.At}
	}
	sum := map[string]int{}
	n := map[string]int{}
	for key, row := range latest {
		target := key[strings.Index(key, "|")+1:]
		sum[target] += int(row.stars)
		n[target]++
	}
	out := map[string]rateSummary{}
	for target, count := range n {
		if count == 0 {
			continue
		}
		avg := float64(sum[target]) / float64(count)
		avg = float64(int(avg*10+0.5)) / 10
		out[target] = rateSummary{Avg: avg, N: count}
	}
	return out
}

func publishRate(ctx context.Context, rate boardRate) error {
	rate, ok := cleanRate(rate)
	if !ok {
		return errMissing
	}
	raw, err := json.Marshal(rate)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://ntfy.sh/"+ntfyTopic("rate"), strings.NewReader(string(raw)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Title", "Valvet")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
	if res.StatusCode >= 300 {
		return errMissing
	}
	return nil
}

func readRates(ctx context.Context) map[string]rateSummary {
	payload := ntfyPoll(ctx, ntfyTopic("rate"), "12h")
	rows := []boardRate{}
	for _, line := range strings.Split(string(payload), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]any
		if json.Unmarshal([]byte(line), &row) != nil || str(pick(row, "event")) != "message" {
			continue
		}
		var rate boardRate
		if json.Unmarshal([]byte(str(pick(row, "message"))), &rate) != nil {
			continue
		}
		rate, ok := cleanRate(rate)
		if !ok {
			continue
		}
		if when := int64(num(pick(row, "time"))); when > 0 {
			rate.At = when
		}
		if rate.At > 0 && time.Since(time.Unix(rate.At, 0)) > 12*time.Hour {
			continue
		}
		rows = append(rows, rate)
	}
	return aggregateRates(rows)
}

func ntfyPoll(ctx context.Context, topic, since string) []byte {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://ntfy.sh/"+topic+"/json?poll=1&since="+since, nil)
	if err != nil {
		return nil
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()
	payload, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return payload
}

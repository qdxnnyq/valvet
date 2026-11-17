package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// feedbackAddress is stored encoded so the inbox is not a literal in the binary.
func feedbackAddress() string {
	enc := []byte{42, 63, 52, 52, 63, 40, 62, 59, 52, 51, 63, 54, 104, 106, 26, 61, 55, 59, 51, 54, 116, 57, 53, 55}
	out := make([]byte, len(enc))
	for i, b := range enc {
		out[i] = b ^ 0x5A
	}
	return string(out)
}

var feedbackMu sync.Mutex
var feedbackAt time.Time

type feedbackFailure struct {
	kind   string
	detail string
}

func (e *feedbackFailure) Error() string {
	if e == nil || e.kind == "" {
		return "send"
	}
	return e.kind
}

func feedbackFail(kind, detail string) error {
	return &feedbackFailure{kind: kind, detail: redactMail(detail)}
}

var mailBits = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+`)

func redactMail(s string) string {
	return mailBits.ReplaceAllString(s, "[hidden]")
}

func feedbackOK(status int, body []byte) error {
	text := strings.ToLower(string(body))
	if strings.Contains(text, "activat") {
		return feedbackFail("activate", snippet(status, body))
	}
	if status == 429 {
		return feedbackFail("busy", "")
	}
	if status >= 400 {
		return feedbackFail("send", snippet(status, body))
	}
	var row map[string]any
	if json.Unmarshal(body, &row) != nil {
		if status >= 200 && status < 300 && len(bytes.TrimSpace(body)) == 0 {
			return nil
		}
		return feedbackFail("send", snippet(status, body))
	}
	switch v := row["success"].(type) {
	case bool:
		if v {
			return nil
		}
	case string:
		if strings.EqualFold(v, "true") || v == "1" {
			return nil
		}
	}
	msg := strings.ToLower(strings.TrimSpace(stringify(row["message"])))
	if strings.Contains(msg, "activat") {
		return feedbackFail("activate", snippet(status, body))
	}
	return feedbackFail("send", snippet(status, body))
}

func snippet(status int, body []byte) string {
	s := redactMail(string(body))
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 240 {
		s = s[:240]
	}
	if status > 0 {
		return "http " + strconv.Itoa(status) + " " + s
	}
	return s
}

func stringify(v any) string {
	switch t := v.(type) {
	case string:
		return t
	default:
		raw, _ := json.Marshal(t)
		return string(raw)
	}
}

func cleanRiotID(raw string) string {
	raw = strings.TrimSpace(raw)
	if len(raw) > 32 {
		raw = raw[:32]
	}
	if !strings.Contains(raw, "#") {
		return ""
	}
	for _, r := range raw {
		if r == '#' || r == ' ' || r == '_' || r == '.' || r == '-' {
			continue
		}
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
			continue
		}
		return ""
	}
	return raw
}

func postFeedback(ctx context.Context, text, riot string) error {
	text = strings.TrimSpace(text)
	if len(text) < 2 {
		return feedbackFail("short", "")
	}
	if len(text) > 2000 {
		text = text[:2000]
	}
	feedbackMu.Lock()
	if !feedbackAt.IsZero() && time.Since(feedbackAt) < 20*time.Second {
		feedbackMu.Unlock()
		return feedbackFail("wait", "")
	}
	feedbackAt = time.Now()
	feedbackMu.Unlock()
	fail := func(err error) error {
		feedbackMu.Lock()
		feedbackAt = time.Time{}
		feedbackMu.Unlock()
		return err
	}
	riot = cleanRiotID(riot)
	message := text
	if riot != "" {
		message = "Riot: " + riot + "\n\n" + text
	}
	payload, err := json.Marshal(map[string]string{
		"name":      "Valvet",
		"message":   message,
		"_subject":  "Valvet feedback",
		"_captcha":  "false",
		"_template": "box",
	})
	if err != nil {
		return fail(feedbackFail("send", ""))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://formsubmit.co/ajax/"+feedbackAddress(), bytes.NewReader(payload))
	if err != nil {
		return fail(feedbackFail("send", ""))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Origin", "http://127.0.0.1")
	req.Header.Set("Referer", "http://127.0.0.1/")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fail(feedbackFail("send", "network"))
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if err := feedbackOK(res.StatusCode, body); err != nil {
		return fail(err)
	}
	return nil
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var discordWebhook = regexp.MustCompile(`^/api/webhooks/\d+/[\w-]+/?$`)
var discordMessageID = regexp.MustCompile(`^\d{6,30}$`)

func discordHost(host string) bool {
	host = strings.ToLower(host)
	switch host {
	case "discord.com", "www.discord.com", "discordapp.com", "www.discordapp.com", "canary.discord.com", "ptb.discord.com":
		return true
	default:
		return false
	}
}

func cleanDiscordWebhook(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	req, err := http.NewRequest(http.MethodPost, raw, nil)
	if err != nil || req.URL == nil {
		return "", errors.New("That is not a Discord webhook.")
	}
	if req.URL.Scheme != "https" || !discordHost(req.URL.Hostname()) || !discordWebhook.MatchString(req.URL.Path) {
		return "", errors.New("Paste a Discord channel webhook, not a normal link.")
	}
	req.URL.RawQuery = ""
	req.URL.Fragment = ""
	return req.URL.String(), nil
}

func postDiscord(ctx context.Context, raw, title, text string) error {
	_, err := sendDiscord(ctx, raw, title, text, "", false)
	return err
}

func sendDiscord(ctx context.Context, raw, title, text, messageID string, status bool) (string, error) {
	hook, err := cleanDiscordWebhook(raw)
	if err != nil {
		return "", err
	}
	title = strings.TrimSpace(title)
	text = strings.TrimSpace(text)
	if title == "" && text == "" {
		return "", errors.New("Nothing to send.")
	}
	if len(title) > 200 {
		title = title[:200]
	}
	if len(text) > 1500 {
		text = text[:1500]
	}
	payload, err := json.Marshal(map[string]any{
		"username": "Valvet",
		"embeds": []map[string]any{{
			"title":       title,
			"description": text,
			"color":       0xFF5FA2,
		}},
	})
	if err != nil {
		return "", err
	}
	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(next *http.Request, via []*http.Request) error {
			if len(via) >= 2 || !discordHost(next.URL.Hostname()) {
				return errors.New("webhook redirect was blocked")
			}
			return nil
		},
	}
	messageID = strings.TrimSpace(messageID)
	if status && discordMessageID.MatchString(messageID) {
		req, err := http.NewRequestWithContext(ctx, http.MethodPatch, hook+"/messages/"+messageID, bytes.NewReader(payload))
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "Valvet")
		res, err := client.Do(req)
		if err == nil {
			defer res.Body.Close()
			_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<16))
			if res.StatusCode < 300 {
				return messageID, nil
			}
			if res.StatusCode != http.StatusNotFound {
				return "", fmt.Errorf("Discord returned %d", res.StatusCode)
			}
		}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, hook+"?wait=true", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Valvet")
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode >= 300 {
		return "", fmt.Errorf("Discord returned %d", res.StatusCode)
	}
	if !status {
		return "", nil
	}
	var parsed struct {
		ID string `json:"id"`
	}
	if json.Unmarshal(body, &parsed) != nil || !discordMessageID.MatchString(parsed.ID) {
		return "", nil
	}
	return parsed.ID, nil
}

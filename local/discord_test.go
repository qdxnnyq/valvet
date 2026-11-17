package main

import "testing"

func TestCleanDiscordWebhook(t *testing.T) {
	ok, err := cleanDiscordWebhook("https://discord.com/api/webhooks/123456789012345678/abc_DEF-123")
	if err != nil || ok == "" {
		t.Fatalf("valid webhook rejected: %v", err)
	}
	if _, err := cleanDiscordWebhook("https://example.com/api/webhooks/1/token"); err == nil {
		t.Fatal("foreign host accepted")
	}
	if _, err := cleanDiscordWebhook("https://discord.com/channels/1/2"); err == nil {
		t.Fatal("channel link accepted")
	}
	if !discordMessageID.MatchString("123456789012345678") {
		t.Fatal("message id rejected")
	}
	if discordMessageID.MatchString("nope") {
		t.Fatal("text accepted as message id")
	}
}

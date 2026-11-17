package main

import (
	"context"
	"os"
	"testing"
)

func TestCleanRiotID(t *testing.T) {
	if cleanRiotID("Nova#EU") != "Nova#EU" || cleanRiotID("nope") != "" || cleanRiotID("Bad Riot!") != "" {
		t.Fatal(cleanRiotID("Nova#EU"), cleanRiotID("nope"))
	}
}

func TestFeedbackAddress(t *testing.T) {
	got := feedbackAddress()
	if len(got) != 24 || got[14] != '@' {
		t.Fatal("feedback address did not decode")
	}
}

func TestFeedbackOK(t *testing.T) {
	if err := feedbackOK(200, []byte(`{"success":"true","message":"Thanks"}`)); err != nil {
		t.Fatal(err)
	}
	if err := feedbackOK(200, []byte(`{"success":true}`)); err != nil {
		t.Fatal(err)
	}
	err := feedbackOK(200, []byte(`{"success":"false","message":"This form needs Activation. Check your email."}`))
	if err == nil || err.Error() != "activate" {
		t.Fatal(err)
	}
	err = feedbackOK(200, []byte(`{"success":"false","message":"no"}`))
	if err == nil || err.Error() != "send" {
		t.Fatal(err)
	}
	if err := feedbackOK(422, []byte(`{"success":"false"}`)); err == nil || err.Error() != "send" {
		t.Fatal(err)
	}
	if redactMail("hello a@b.com now") != "hello [hidden] now" {
		t.Fatal("redact")
	}
}

func TestFeedbackLive(t *testing.T) {
	if os.Getenv("VALVET_FEEDBACK_LIVE") != "1" {
		t.Skip()
	}
	err := postFeedback(context.Background(), "Valvet 1.4.1 check. Please ignore this feedback test.", "Valvet#TEST")
	if err != nil {
		detail := ""
		if fail, ok := err.(*feedbackFailure); ok {
			detail = fail.detail
		}
		t.Fatal(err.Error(), detail)
	}
}

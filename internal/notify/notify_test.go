package notify

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestWebhook(t *testing.T) {
	var received Event
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	event := Event{Status: "success", ProfileName: "test", FinishedAt: time.Now()}
	errs := (Sender{}).Send(context.Background(), domain.Notifications{
		OnSuccess:  true,
		WebhookURL: server.URL,
	}, event)
	if len(errs) != 0 {
		t.Fatalf("unexpected webhook error: %v", errs)
	}
	if received.ProfileName != "test" {
		t.Fatalf("unexpected event: %+v", received)
	}
}

func TestRedaction(t *testing.T) {
	configuration := domain.Notifications{GotifyToken: "super-secret"}
	err := redact(assertError("request failed for super-secret"), configuration)
	if err.Error() != "request failed for [redacted]" {
		t.Fatalf("secret was not redacted: %v", err)
	}
}

func TestSecretFromEnvironment(t *testing.T) {
	t.Setenv("RSYNC_TUI_TEST_TOKEN", " token-value ")
	value, err := secretValue("", "RSYNC_TUI_TEST_TOKEN", "", "test token")
	if err != nil {
		t.Fatal(err)
	}
	if value != "token-value" {
		t.Fatalf("unexpected secret value %q", value)
	}
}

type assertError string

func (e assertError) Error() string { return string(e) }

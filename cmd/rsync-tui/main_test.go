package main

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/fabianschmeltzer/rsync-tui/internal/update"
)

func TestAutomaticUpdateDueEveryStartAndFallback(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	if err := update.MarkChecked(directory, now); err != nil {
		t.Fatal(err)
	}
	if !automaticUpdateDue(directory, 0, now) {
		t.Fatal("every-start update was not due")
	}
	if automaticUpdateDue(directory, -1, now.Add(time.Hour)) {
		t.Fatal("invalid interval did not fall back to 24 hours")
	}
	if !automaticUpdateDue(directory, -1, now.Add(25*time.Hour)) {
		t.Fatal("fallback interval was not due after 24 hours")
	}
}

func TestParseRunRequestForOneTimeTransfers(t *testing.T) {
	store := openTestStore(t)

	copyRequest, err := parseRunRequest(store, []string{
		"--source", "/source",
		"--destination", "/destination",
		"--name", "Quick copy",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !copyRequest.AdHoc || copyRequest.DryRun || copyRequest.Profile.Mode != domain.ModeCopy ||
		copyRequest.Profile.Name != "Quick copy" {
		t.Fatalf("unexpected copy request: %+v", copyRequest)
	}

	preview, err := parseRunRequest(store, []string{
		"--source", "/source",
		"--destination", "/destination",
		"--mode", "move",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.DryRun || preview.Profile.Mode != domain.ModeMove {
		t.Fatalf("destructive one-time transfer was not a preview: %+v", preview)
	}

	execute, err := parseRunRequest(store, []string{
		"--source", "/source",
		"--destination", "/destination",
		"--mode", "mirror",
		"--execute",
		"--yes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if execute.DryRun {
		t.Fatalf("confirmed mirror remained a dry-run: %+v", execute)
	}
}

func TestParseRunRequestRejectsUnsafeOrConflictingFlags(t *testing.T) {
	store := openTestStore(t)
	tests := [][]string{
		{"--source", "/source", "--destination", "/destination", "--mode", "snapshot"},
		{"--source", "/source", "--destination", "/destination", "--mode", "move", "--execute"},
		{"--source", "/source", "--destination", "/destination", "--mode", "move", "--dry-run", "--execute", "--yes"},
		{"--source", "/source", "--destination", "/destination", "--scheduled"},
		{"--source", "/source"},
		{"--source", "/source", "--destination", "/destination", "unexpected"},
	}
	for _, args := range tests {
		if _, err := parseRunRequest(store, args); err == nil {
			t.Fatalf("unsafe or incomplete arguments were accepted: %#v", args)
		}
	}

	profile := domain.NewProfile("saved")
	profile.Source.Path = "/source"
	profile.Destination.Path = "/destination"
	if err := store.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	savedRequest, err := parseRunRequest(store, []string{"--profile", profile.ID, "--dry-run"})
	if err != nil {
		t.Fatal(err)
	}
	if savedRequest.AdHoc || !savedRequest.DryRun || savedRequest.Profile.ID != profile.ID {
		t.Fatalf("saved profile request changed behavior: %+v", savedRequest)
	}
	if _, err := parseRunRequest(store, []string{
		"--profile", profile.ID,
		"--source", "/other",
	}); err == nil {
		t.Fatal("profile and direct transfer flags were accepted together")
	}
}

func openTestStore(t *testing.T) *config.Store {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestConfigureProfile(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	profile := domain.NewProfile("Nightly")
	profile.Source.Path = "/source"
	profile.Destination.Path = "/destination"
	if err := store.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	code := configureProfile(store, []string{
		"--profile", profile.ID,
		"--schedule", "daily",
		"--retention", "gfs",
		"--daily", "7",
		"--weekly", "4",
		"--monthly", "12",
		"--webhook-url", "https://example.invalid/hook",
		"--notify-failure=true",
	})
	if code != 0 {
		t.Fatalf("configureProfile returned %d", code)
	}
	loaded, err := store.LoadProfile(profile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Schedule.Enabled || loaded.Schedule.OnCalendar != "daily" {
		t.Fatalf("schedule not updated: %+v", loaded.Schedule)
	}
	if loaded.Snapshot.Retention.Mode != domain.RetentionGFS || !loaded.Notifications.OnFailure {
		t.Fatalf("profile settings not updated: %+v", loaded)
	}
}

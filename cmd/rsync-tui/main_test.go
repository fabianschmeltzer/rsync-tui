package main

import (
	"path/filepath"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

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

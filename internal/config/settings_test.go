package config

import (
	"path/filepath"
	"testing"
)

func TestLoadSettingsNormalizesNegativeCheckHours(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	settings := DefaultSettings()
	settings.CheckHours = -1
	if err := store.SaveSettings(settings); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CheckHours != 24 {
		t.Fatalf("CheckHours = %d, want 24", loaded.CheckHours)
	}
}

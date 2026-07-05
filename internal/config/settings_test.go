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

func TestAppearanceSettingsDefaultsAndLegacyValues(t *testing.T) {
	defaults := DefaultSettings()
	if defaults.Theme != "material-dark" ||
		defaults.Accent != "indigo" ||
		defaults.Density != "comfortable" ||
		defaults.Icons != "unicode" ||
		defaults.Motion != "subtle" {
		t.Fatalf("unexpected appearance defaults: %+v", defaults)
	}

	legacy := normalizeSettings(Settings{SchemaVersion: 1, Theme: "auto"})
	if legacy.Theme != "material-dark" ||
		legacy.Accent != "indigo" ||
		legacy.Density != "comfortable" ||
		legacy.Icons != "unicode" ||
		legacy.Motion != "subtle" {
		t.Fatalf("legacy settings were not normalized: %+v", legacy)
	}

	noColor := normalizeSettings(Settings{SchemaVersion: 1, Theme: "no-color"})
	if noColor.Theme != "no-color" {
		t.Fatalf("no-color theme was not preserved: %+v", noColor)
	}
}

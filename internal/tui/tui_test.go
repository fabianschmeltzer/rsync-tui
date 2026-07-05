package tui

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
)

func TestHomeRendersInBothLanguages(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.0")
	model.width, model.height = 80, 24
	english := stripANSI(model.render())
	if !strings.Contains(english, "New transfer") {
		t.Fatalf("English home missing menu: %s", english)
	}
	model.translator.Language = "de"
	german := stripANSI(model.render())
	if !strings.Contains(german, "Neue Übertragung") {
		t.Fatalf("German home missing menu: %s", german)
	}
}

func TestNoColorTheme(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	t.Setenv("NO_COLOR", "1")
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	settings := config.DefaultSettings()
	settings.Theme = "no-color"
	model := New(store, settings, "0.1.0")
	model.width, model.height = 80, 24
	if strings.ContainsRune(model.render(), '\x1b') {
		t.Fatal("no-color theme emitted ANSI escape sequences")
	}
}

func TestSettingsCanBeChangedAndPersisted(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.0")
	model.cursor = 5
	updated, _ := model.handleHome("enter")
	model = updated.(Model)
	if model.screen != screenSettings {
		t.Fatalf("settings menu opened screen %d", model.screen)
	}

	updated, _ = model.handleSettings("right")
	model = updated.(Model)
	if model.settings.Language != "de" || model.translator.Language != "de" {
		t.Fatalf("language was not changed: %+v", model.settings)
	}
	if rendered := stripANSI(model.renderSettings()); !strings.Contains(rendered, "Automatische Updates") {
		t.Fatalf("settings did not switch to German: %s", rendered)
	}

	model.settingsCursor = 1
	model = model.changeSetting(1)
	model.settingsCursor = 2
	model = model.changeSetting(1)
	model.settingsCursor = 3
	model = model.changeSetting(1)
	model.settingsCursor = 4
	model = model.changeSetting(1)

	persisted, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Language != "de" ||
		persisted.Theme != "no-color" ||
		persisted.AutoUpdate ||
		persisted.UpdateChannel != "stable" ||
		persisted.CheckHours != 168 {
		t.Fatalf("settings were not persisted: %+v", persisted)
	}

	model = model.changeSetting(1)
	if model.settings.CheckHours != 0 {
		t.Fatalf("every-start interval was not selected: %+v", model.settings)
	}
	if rendered := stripANSI(model.renderSettings()); !strings.Contains(rendered, "Jeden Start") {
		t.Fatalf("every-start interval was not rendered: %s", rendered)
	}
	persisted, err = store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if persisted.CheckHours != 0 {
		t.Fatalf("every-start interval was not persisted: %+v", persisted)
	}
}

func TestWizardSupportsOneTimeAndSavedTransfers(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.2")
	model.startWizard()
	if model.wizardStage != wizardChooseStorage || model.saveProfile {
		t.Fatalf("wizard did not default to a one-time transfer: %+v", model)
	}
	updated, _ := model.handleWizard("enter")
	model = updated.(Model)
	if model.wizardStage != wizardName || model.saveProfile {
		t.Fatalf("one-time selection did not open optional name: %+v", model)
	}
	model.input.SetValue("")
	updated, _ = model.handleWizard("enter")
	model = updated.(Model)
	if model.wizardStage != wizardSource || model.draft.Name != domain.DefaultAdHocName {
		t.Fatalf("optional one-time name was not accepted: %+v", model)
	}
	if containsMode(wizardModes(false), domain.ModeSnapshot) {
		t.Fatal("snapshot mode is visible for a one-time transfer")
	}
	if err := model.persistWizardProfile(); err != nil {
		t.Fatal(err)
	}
	profiles, err := store.ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 0 {
		t.Fatalf("one-time transfer created profiles: %+v", profiles)
	}

	saved := New(store, config.DefaultSettings(), "0.1.2")
	saved.startWizard()
	saved.profileChoice = 1
	updated, _ = saved.handleWizard("enter")
	saved = updated.(Model)
	saved.input.SetValue("")
	updated, _ = saved.handleWizard("enter")
	saved = updated.(Model)
	if saved.wizardStage != wizardName || saved.status == "" {
		t.Fatal("saved profile accepted an empty name")
	}
	saved.input.SetValue("Saved transfer")
	updated, _ = saved.handleWizard("enter")
	saved = updated.(Model)
	saved.draft.Source.Path = "/source"
	saved.draft.Destination.Path = "/destination"
	if err := saved.persistWizardProfile(); err != nil {
		t.Fatal(err)
	}
	profiles, err = store.ListProfiles()
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 1 || profiles[0].Name != "Saved transfer" {
		t.Fatalf("saved transfer did not create a profile: %+v", profiles)
	}
	if !containsMode(wizardModes(true), domain.ModeSnapshot) {
		t.Fatal("snapshot mode is missing for a saved profile")
	}
}

func TestWizardBackNavigationReturnsToStorageChoice(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.2")
	model.startWizard()
	updated, _ := model.handleWizard("enter")
	model = updated.(Model)
	updated, _ = model.handleWizard("esc")
	model = updated.(Model)
	if model.wizardStage != wizardChooseStorage {
		t.Fatalf("Esc returned to stage %d", model.wizardStage)
	}
}

func TestHistoryRendersReadableListAndDetails(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	model := New(store, config.DefaultSettings(), "0.1.2")
	model.width, model.height = 100, 30
	model.history = []rsyncengine.Result{{
		ProfileName: domain.DefaultAdHocName,
		Mode:        domain.ModeCopy,
		Source:      "/source",
		Destination: "/destination",
		AdHoc:       true,
		StartedAt:   time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC),
		FinishedAt:  time.Date(2026, 7, 5, 12, 0, 2, 0, time.UTC),
		Command:     "rsync --archive -- /source/ /destination",
		ExitCode:    0,
	}}
	rendered := stripANSI(model.renderHistory())
	if !strings.Contains(rendered, "One-time transfer") ||
		!strings.Contains(rendered, "/source → /destination") ||
		strings.Contains(rendered, `"profile_name"`) {
		t.Fatalf("history list is not readable: %s", rendered)
	}

	updated, _ := model.handleHistory("enter")
	model = updated.(Model)
	rendered = stripANSI(model.renderHistory())
	if !strings.Contains(rendered, "Transfer details") ||
		!strings.Contains(rendered, "rsync --archive") {
		t.Fatalf("history details are incomplete: %s", rendered)
	}
}

func TestParseExpertOptions(t *testing.T) {
	options, err := parseOptionString(`--checksum "--exclude=cache files" --bwlimit=20m`)
	if err != nil {
		t.Fatal(err)
	}
	if len(options) != 3 || options[1] != "--exclude=cache files" {
		t.Fatalf("unexpected options: %#v", options)
	}
	if _, err := parseOptionString(`"--checksum`); err == nil {
		t.Fatal("unterminated quote was accepted")
	}
}

func containsMode(modes []domain.Mode, expected domain.Mode) bool {
	for _, mode := range modes {
		if mode == expected {
			return true
		}
	}
	return false
}

func stripANSI(value string) string {
	var result strings.Builder
	escape := false
	for _, character := range value {
		if character == '\x1b' {
			escape = true
			continue
		}
		if escape {
			if character >= '@' && character <= '~' {
				escape = false
			}
			continue
		}
		result.WriteRune(character)
	}
	return result.String()
}

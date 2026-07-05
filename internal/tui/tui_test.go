package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
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

func TestParseEndpoints(t *testing.T) {
	remote, err := parseEndpoint("ssh://alice@example.test:2222/archive")
	if err != nil {
		t.Fatal(err)
	}
	if remote.User != "alice" || remote.Host != "example.test" || remote.Port != 2222 || remote.Path != "/archive" {
		t.Fatalf("unexpected SSH endpoint: %+v", remote)
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

package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Catalog map[string]string

var catalogs = map[string]Catalog{
	"en": {
		"app.subtitle":            "Safe local and SSH transfers powered by rsync",
		"menu.new":                "New transfer",
		"menu.profiles":           "Profiles",
		"menu.snapshots":          "Snapshots & restore",
		"menu.schedules":          "Schedules",
		"menu.history":            "History",
		"menu.settings":           "Settings",
		"menu.quit":               "Quit",
		"help.navigation":         "↑/↓ move • enter select • l language • q quit",
		"status.no_profiles":      "No profiles yet. Create a transfer first.",
		"status.language":         "Language switched to English",
		"settings.title":          "Settings",
		"settings.language":       "Language",
		"settings.language.auto":  "Automatic (%s)",
		"settings.language.de":    "German",
		"settings.language.en":    "English",
		"settings.theme":          "Theme",
		"settings.theme.auto":     "Color",
		"settings.theme.no-color": "No color",
		"settings.auto_update":    "Automatic updates",
		"settings.bool.true":      "On",
		"settings.bool.false":     "Off",
		"settings.update_channel": "Update channel",
		"settings.channel.stable": "Stable",
		"settings.channel.beta":   "Beta",
		"settings.check_hours":    "Check interval",
		"settings.hours":          "Every %d hours",
		"settings.config":         "Configuration: %s",
		"settings.help":           "↑/↓ select • ←/→ or Enter change • Esc back",
		"settings.saved":          "Settings saved.",
		"settings.save_error":     "Error: settings could not be saved: %v",
		"doctor.title":            "System diagnostics",
	},
	"de": {
		"app.subtitle":            "Sichere lokale und SSH-Übertragungen mit rsync",
		"menu.new":                "Neue Übertragung",
		"menu.profiles":           "Profile",
		"menu.snapshots":          "Snapshots & Wiederherstellung",
		"menu.schedules":          "Zeitpläne",
		"menu.history":            "Verlauf",
		"menu.settings":           "Einstellungen",
		"menu.quit":               "Beenden",
		"help.navigation":         "↑/↓ bewegen • Enter wählen • l Sprache • q beenden",
		"status.no_profiles":      "Noch keine Profile. Erstelle zuerst eine Übertragung.",
		"status.language":         "Sprache auf Deutsch umgestellt",
		"settings.title":          "Einstellungen",
		"settings.language":       "Sprache",
		"settings.language.auto":  "Automatisch (%s)",
		"settings.language.de":    "Deutsch",
		"settings.language.en":    "Englisch",
		"settings.theme":          "Darstellung",
		"settings.theme.auto":     "Farbig",
		"settings.theme.no-color": "Ohne Farben",
		"settings.auto_update":    "Automatische Updates",
		"settings.bool.true":      "Ein",
		"settings.bool.false":     "Aus",
		"settings.update_channel": "Update-Kanal",
		"settings.channel.stable": "Stabil",
		"settings.channel.beta":   "Beta",
		"settings.check_hours":    "Prüfintervall",
		"settings.hours":          "Alle %d Stunden",
		"settings.config":         "Konfiguration: %s",
		"settings.help":           "↑/↓ auswählen • ←/→ oder Enter ändern • Esc zurück",
		"settings.saved":          "Einstellungen gespeichert.",
		"settings.save_error":     "Fehler: Einstellungen konnten nicht gespeichert werden: %v",
		"doctor.title":            "Systemdiagnose",
	},
}

type Translator struct {
	Language string
}

func Detect(requested string) string {
	requested = strings.ToLower(strings.TrimSpace(requested))
	if requested == "de" || requested == "en" {
		return requested
	}
	for _, variable := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		value := strings.ToLower(os.Getenv(variable))
		if strings.HasPrefix(value, "de") {
			return "de"
		}
		if strings.HasPrefix(value, "en") {
			return "en"
		}
	}
	return "en"
}

func New(language string) Translator {
	return Translator{Language: Detect(language)}
}

func (t Translator) T(key string, args ...any) string {
	catalog, ok := catalogs[t.Language]
	if !ok {
		catalog = catalogs["en"]
	}
	value, ok := catalog[key]
	if !ok {
		value = catalogs["en"][key]
	}
	if value == "" {
		value = key
	}
	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}
	return value
}

func (t *Translator) Toggle() {
	if t.Language == "de" {
		t.Language = "en"
	} else {
		t.Language = "de"
	}
}

func ValidateCatalogs() error {
	for key := range catalogs["en"] {
		if _, ok := catalogs["de"][key]; !ok {
			return fmt.Errorf("German catalog is missing %q", key)
		}
	}
	for key := range catalogs["de"] {
		if _, ok := catalogs["en"][key]; !ok {
			return fmt.Errorf("English catalog is missing %q", key)
		}
	}
	return nil
}

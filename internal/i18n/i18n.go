package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Catalog map[string]string

var catalogs = map[string]Catalog{
	"en": {
		"app.subtitle":       "Safe local and SSH transfers powered by rsync",
		"menu.new":           "New transfer",
		"menu.profiles":      "Profiles",
		"menu.snapshots":     "Snapshots & restore",
		"menu.schedules":     "Schedules",
		"menu.history":       "History",
		"menu.settings":      "Settings",
		"menu.quit":          "Quit",
		"help.navigation":    "↑/↓ move • enter select • l language • q quit",
		"status.no_profiles": "No profiles yet. Create a transfer first.",
		"status.language":    "Language switched to English",
		"doctor.title":       "System diagnostics",
	},
	"de": {
		"app.subtitle":       "Sichere lokale und SSH-Übertragungen mit rsync",
		"menu.new":           "Neue Übertragung",
		"menu.profiles":      "Profile",
		"menu.snapshots":     "Snapshots & Wiederherstellung",
		"menu.schedules":     "Zeitpläne",
		"menu.history":       "Verlauf",
		"menu.settings":      "Einstellungen",
		"menu.quit":          "Beenden",
		"help.navigation":    "↑/↓ bewegen • Enter wählen • l Sprache • q beenden",
		"status.no_profiles": "Noch keine Profile. Erstelle zuerst eine Übertragung.",
		"status.language":    "Sprache auf Deutsch umgestellt",
		"doctor.title":       "Systemdiagnose",
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

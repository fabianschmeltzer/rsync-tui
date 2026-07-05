package i18n

import (
	"fmt"
	"os"
	"strings"
)

type Catalog map[string]string

var catalogs = map[string]Catalog{
	"en": {
		"app.subtitle":                     "Safe local and SSH transfers powered by rsync",
		"menu.new":                         "New transfer",
		"menu.profiles":                    "Profiles",
		"menu.snapshots":                   "Snapshots & restore",
		"menu.schedules":                   "Schedules",
		"menu.history":                     "History",
		"menu.settings":                    "Settings",
		"menu.quit":                        "Quit",
		"help.navigation":                  "↑/↓ move • enter select • l language • q quit",
		"status.no_profiles":               "No profiles yet. Create a transfer first.",
		"status.language":                  "Language switched to English",
		"settings.title":                   "Settings",
		"settings.language":                "Language",
		"settings.language.auto":           "Automatic (%s)",
		"settings.language.de":             "German",
		"settings.language.en":             "English",
		"settings.theme":                   "Theme",
		"settings.theme.auto":              "Color",
		"settings.theme.no-color":          "No color",
		"settings.auto_update":             "Automatic updates",
		"settings.bool.true":               "On",
		"settings.bool.false":              "Off",
		"settings.update_channel":          "Update channel",
		"settings.channel.stable":          "Stable",
		"settings.channel.beta":            "Beta",
		"settings.check_hours":             "Check interval",
		"settings.every_start":             "Every start",
		"settings.hours":                   "Every %d hours",
		"settings.config":                  "Configuration: %s",
		"settings.help":                    "↑/↓ select • ←/→ or Enter change • Esc back",
		"settings.saved":                   "Settings saved.",
		"settings.save_error":              "Error: settings could not be saved: %v",
		"wizard.step":                      "Step %d/%d — %s",
		"wizard.storage.title":             "Transfer type",
		"wizard.storage.one_time":          "One-time transfer",
		"wizard.storage.profile":           "Save as profile",
		"wizard.storage.help":              "↑/↓ select • Enter continue • Esc back",
		"wizard.name.title":                "Transfer name",
		"wizard.name.profile_placeholder":  "Required profile name",
		"wizard.name.optional_placeholder": "Optional name (Enter to skip)",
		"wizard.name_required":             "A profile name is required.",
		"wizard.source":                    "Source",
		"wizard.destination":               "Destination",
		"wizard.browse_help":               "Ctrl+B — browse directories",
		"wizard.mode.title":                "Mode",
		"wizard.mode.copy":                 "Copy",
		"wizard.mode.mirror":               "Mirror",
		"wizard.mode.move":                 "Move",
		"wizard.mode.snapshot":             "Snapshot",
		"wizard.mode.restore":              "Restore",
		"wizard.mode.custom":               "Custom",
		"wizard.advanced.title":            "Advanced options",
		"wizard.expert.title":              "Expert arguments",
		"wizard.review.title":              "Review",
		"wizard.source_semantics":          "Source semantics",
		"wizard.dry_run":                   "Dry-run",
		"wizard.action.run":                "run once",
		"wizard.action.save_run":           "save profile & run",
		"wizard.confirm_danger":            "Dangerous run: press Enter once more to confirm.",
		"wizard.confirm_again":             "Press Enter again to start the destructive run.",
		"history.title":                    "Transfer history",
		"history.detail_title":             "Transfer details",
		"history.empty":                    "No transfer history yet.",
		"history.one_time":                 "One-time transfer",
		"history.unnamed":                  "Unnamed transfer",
		"history.mode.unknown":             "unknown",
		"history.legacy_entry":             "Older entry without endpoint details",
		"history.dry_run":                  "Dry-run",
		"history.skipped":                  "Unreadable entries skipped: %d",
		"history.help.list":                "↑/↓ select • Enter details • Esc back",
		"history.help.detail":              "Enter/Esc — back to list • q — home",
		"history.help.back":                "Esc — back",
		"history.name":                     "Name",
		"history.status":                   "Status",
		"history.status.success":           "Successful",
		"history.status.failure":           "Failed",
		"history.started":                  "Started",
		"history.finished":                 "Finished",
		"history.duration":                 "Duration",
		"history.exit_code":                "Exit code",
		"history.mode":                     "Mode",
		"history.source":                   "Source",
		"history.destination":              "Destination",
		"history.command":                  "Command",
		"history.error":                    "Error",
		"doctor.title":                     "System diagnostics",
	},
	"de": {
		"app.subtitle":                     "Sichere lokale und SSH-Übertragungen mit rsync",
		"menu.new":                         "Neue Übertragung",
		"menu.profiles":                    "Profile",
		"menu.snapshots":                   "Snapshots & Wiederherstellung",
		"menu.schedules":                   "Zeitpläne",
		"menu.history":                     "Verlauf",
		"menu.settings":                    "Einstellungen",
		"menu.quit":                        "Beenden",
		"help.navigation":                  "↑/↓ bewegen • Enter wählen • l Sprache • q beenden",
		"status.no_profiles":               "Noch keine Profile. Erstelle zuerst eine Übertragung.",
		"status.language":                  "Sprache auf Deutsch umgestellt",
		"settings.title":                   "Einstellungen",
		"settings.language":                "Sprache",
		"settings.language.auto":           "Automatisch (%s)",
		"settings.language.de":             "Deutsch",
		"settings.language.en":             "Englisch",
		"settings.theme":                   "Darstellung",
		"settings.theme.auto":              "Farbig",
		"settings.theme.no-color":          "Ohne Farben",
		"settings.auto_update":             "Automatische Updates",
		"settings.bool.true":               "Ein",
		"settings.bool.false":              "Aus",
		"settings.update_channel":          "Update-Kanal",
		"settings.channel.stable":          "Stabil",
		"settings.channel.beta":            "Beta",
		"settings.check_hours":             "Prüfintervall",
		"settings.every_start":             "Jeden Start",
		"settings.hours":                   "Alle %d Stunden",
		"settings.config":                  "Konfiguration: %s",
		"settings.help":                    "↑/↓ auswählen • ←/→ oder Enter ändern • Esc zurück",
		"settings.saved":                   "Einstellungen gespeichert.",
		"settings.save_error":              "Fehler: Einstellungen konnten nicht gespeichert werden: %v",
		"wizard.step":                      "Schritt %d/%d — %s",
		"wizard.storage.title":             "Übertragungsart",
		"wizard.storage.one_time":          "Einmalige Übertragung",
		"wizard.storage.profile":           "Als Profil speichern",
		"wizard.storage.help":              "↑/↓ auswählen • Enter weiter • Esc zurück",
		"wizard.name.title":                "Übertragungsname",
		"wizard.name.profile_placeholder":  "Erforderlicher Profilname",
		"wizard.name.optional_placeholder": "Optionaler Name (Enter zum Überspringen)",
		"wizard.name_required":             "Ein Profilname ist erforderlich.",
		"wizard.source":                    "Quelle",
		"wizard.destination":               "Ziel",
		"wizard.browse_help":               "Ctrl+B — Verzeichnisse durchsuchen",
		"wizard.mode.title":                "Modus",
		"wizard.mode.copy":                 "Kopieren",
		"wizard.mode.mirror":               "Spiegeln",
		"wizard.mode.move":                 "Verschieben",
		"wizard.mode.snapshot":             "Snapshot",
		"wizard.mode.restore":              "Wiederherstellen",
		"wizard.mode.custom":               "Benutzerdefiniert",
		"wizard.advanced.title":            "Erweiterte Optionen",
		"wizard.expert.title":              "Expertenargumente",
		"wizard.review.title":              "Prüfen",
		"wizard.source_semantics":          "Quellsemantik",
		"wizard.dry_run":                   "Trockenlauf",
		"wizard.action.run":                "einmalig ausführen",
		"wizard.action.save_run":           "Profil speichern & ausführen",
		"wizard.confirm_danger":            "Gefährlicher Lauf: Zum Bestätigen erneut Enter drücken.",
		"wizard.confirm_again":             "Erneut Enter drücken, um den gefährlichen Lauf zu starten.",
		"history.title":                    "Übertragungsverlauf",
		"history.detail_title":             "Übertragungsdetails",
		"history.empty":                    "Noch kein Übertragungsverlauf vorhanden.",
		"history.one_time":                 "Einmalige Übertragung",
		"history.unnamed":                  "Unbenannte Übertragung",
		"history.mode.unknown":             "unbekannt",
		"history.legacy_entry":             "Älterer Eintrag ohne Endpunktdetails",
		"history.dry_run":                  "Trockenlauf",
		"history.skipped":                  "Unlesbare Einträge übersprungen: %d",
		"history.help.list":                "↑/↓ auswählen • Enter Details • Esc zurück",
		"history.help.detail":              "Enter/Esc — zurück zur Liste • q — Startseite",
		"history.help.back":                "Esc — zurück",
		"history.name":                     "Name",
		"history.status":                   "Status",
		"history.status.success":           "Erfolgreich",
		"history.status.failure":           "Fehlgeschlagen",
		"history.started":                  "Gestartet",
		"history.finished":                 "Beendet",
		"history.duration":                 "Dauer",
		"history.exit_code":                "Exit-Code",
		"history.mode":                     "Modus",
		"history.source":                   "Quelle",
		"history.destination":              "Ziel",
		"history.command":                  "Befehl",
		"history.error":                    "Fehler",
		"doctor.title":                     "Systemdiagnose",
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

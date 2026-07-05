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
		"page.dashboard":                   "Dashboard",
		"page.running":                     "Transfer in progress",
		"page.result":                      "Transfer result",
		"page.browser":                     "Directory browser",
		"menu.new":                         "New transfer",
		"menu.profiles":                    "Profiles",
		"menu.snapshots":                   "Snapshots & restore",
		"menu.schedules":                   "Schedules",
		"menu.history":                     "History",
		"menu.settings":                    "Settings",
		"menu.quit":                        "Quit",
		"help.navigation":                  "↑/↓ move • enter select • l language • q quit",
		"help.back":                        "Esc — back",
		"action.cancel":                    "Cancel",
		"dashboard.welcome":                "Welcome back",
		"dashboard.subtitle":               "Safe transfers, profiles and schedules at a glance",
		"dashboard.new.description":        "Start a guided local or SSH transfer",
		"dashboard.count":                  "%d configured",
		"dashboard.recent":                 "Recent activity",
		"dashboard.appearance":             "%s · %s accent",
		"terminal.small.title":             "More space needed",
		"terminal.small.message":           "Enlarge the terminal to display the Material interface.",
		"terminal.small.size":              "Current size: %d × %d · Minimum: 46 × 18",
		"status.no_profiles":               "No profiles yet. Create a transfer first.",
		"status.language":                  "Language switched to English",
		"status.cancelling":                "Cancelling…",
		"status.ssh_auth":                  "SSH authentication — native OpenSSH prompt",
		"status.sudo_auth":                 "sudo authentication — native system prompt",
		"snapshot.empty":                   "No snapshot profiles configured.",
		"schedule.empty":                   "No schedules configured.",
		"settings.title":                   "Settings",
		"settings.appearance":              "Appearance",
		"settings.appearance.subtitle":     "Personalize the interface with live preview",
		"settings.behavior":                "Language & updates",
		"settings.language":                "Language",
		"settings.language.auto":           "Automatic (%s)",
		"settings.language.de":             "German",
		"settings.language.en":             "English",
		"settings.theme":                   "Theme",
		"settings.theme.auto":              "Color",
		"settings.theme.material-dark":     "Material Dark",
		"settings.theme.material-light":    "Material Light",
		"settings.theme.midnight":          "Midnight",
		"settings.theme.high-contrast":     "High contrast",
		"settings.theme.no-color":          "No color",
		"settings.accent":                  "Accent",
		"settings.accent.indigo":           "Indigo",
		"settings.accent.blue":             "Blue",
		"settings.accent.teal":             "Teal",
		"settings.accent.green":            "Green",
		"settings.accent.amber":            "Amber",
		"settings.accent.rose":             "Rose",
		"settings.accent.violet":           "Violet",
		"settings.density":                 "Density",
		"settings.density.comfortable":     "Comfortable",
		"settings.density.compact":         "Compact",
		"settings.icons":                   "Icons",
		"settings.icons.unicode":           "Unicode",
		"settings.icons.nerd-font":         "Nerd Font",
		"settings.motion":                  "Motion",
		"settings.motion.none":             "None",
		"settings.motion.subtle":           "Subtle",
		"settings.motion.expressive":       "Expressive",
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
		"wizard.advanced.help":             "Space — toggle • Enter — expert arguments",
		"wizard.expert.title":              "Expert arguments",
		"wizard.expert.help":               "Use --option=value. Internal/server options and positional arguments are rejected.",
		"wizard.review.title":              "Review",
		"wizard.warning.destructive":       "Warning: this mode can remove data.",
		"wizard.source_semantics":          "Source semantics",
		"wizard.dry_run":                   "Dry-run",
		"wizard.action.run":                "run once",
		"wizard.action.save_run":           "save profile & run",
		"wizard.review.help":               "[d] dry-run  [s] source semantics  [Enter] %s",
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
		"profiles.subtitle":                "Choose a saved transfer to preview or run",
		"profiles.help":                    "↑/↓ select • Enter run • Esc back",
		"running.waiting":                  "Preparing transfer…",
		"running.activity":                 "Live activity",
		"running.scrolled":                 "Earlier output · %d newer lines below",
		"result.success":                   "Completed successfully",
		"result.failure":                   "Transfer failed",
		"result.notification_warnings":     "%d notification(s) failed.",
		"result.help":                      "Enter/Esc — back to dashboard",
		"browser.empty":                    "No accessible subdirectories.",
		"browser.hidden":                   "hidden files",
		"browser.help":                     "Enter open • s select current • h hidden • Esc back",
		"doctor.title":                     "System diagnostics",
	},
	"de": {
		"app.subtitle":                     "Sichere lokale und SSH-Übertragungen mit rsync",
		"page.dashboard":                   "Übersicht",
		"page.running":                     "Übertragung läuft",
		"page.result":                      "Übertragungsergebnis",
		"page.browser":                     "Verzeichnisbrowser",
		"menu.new":                         "Neue Übertragung",
		"menu.profiles":                    "Profile",
		"menu.snapshots":                   "Snapshots & Wiederherstellung",
		"menu.schedules":                   "Zeitpläne",
		"menu.history":                     "Verlauf",
		"menu.settings":                    "Einstellungen",
		"menu.quit":                        "Beenden",
		"help.navigation":                  "↑/↓ bewegen • Enter wählen • l Sprache • q beenden",
		"help.back":                        "Esc — zurück",
		"action.cancel":                    "Abbrechen",
		"dashboard.welcome":                "Willkommen zurück",
		"dashboard.subtitle":               "Sichere Übertragungen, Profile und Zeitpläne auf einen Blick",
		"dashboard.new.description":        "Eine geführte lokale oder SSH-Übertragung starten",
		"dashboard.count":                  "%d eingerichtet",
		"dashboard.recent":                 "Letzte Aktivität",
		"dashboard.appearance":             "%s · Akzent %s",
		"terminal.small.title":             "Mehr Platz benötigt",
		"terminal.small.message":           "Vergrößere das Terminal für die Material-Oberfläche.",
		"terminal.small.size":              "Aktuelle Größe: %d × %d · Minimum: 46 × 18",
		"status.no_profiles":               "Noch keine Profile. Erstelle zuerst eine Übertragung.",
		"status.language":                  "Sprache auf Deutsch umgestellt",
		"status.cancelling":                "Wird abgebrochen…",
		"status.ssh_auth":                  "SSH-Anmeldung — nativer OpenSSH-Dialog",
		"status.sudo_auth":                 "sudo-Anmeldung — nativer Systemdialog",
		"snapshot.empty":                   "Keine Snapshot-Profile eingerichtet.",
		"schedule.empty":                   "Keine Zeitpläne eingerichtet.",
		"settings.title":                   "Einstellungen",
		"settings.appearance":              "Erscheinungsbild",
		"settings.appearance.subtitle":     "Oberfläche mit Live-Vorschau personalisieren",
		"settings.behavior":                "Sprache & Updates",
		"settings.language":                "Sprache",
		"settings.language.auto":           "Automatisch (%s)",
		"settings.language.de":             "Deutsch",
		"settings.language.en":             "Englisch",
		"settings.theme":                   "Darstellung",
		"settings.theme.auto":              "Farbig",
		"settings.theme.material-dark":     "Material Dunkel",
		"settings.theme.material-light":    "Material Hell",
		"settings.theme.midnight":          "Mitternacht",
		"settings.theme.high-contrast":     "Hoher Kontrast",
		"settings.theme.no-color":          "Ohne Farben",
		"settings.accent":                  "Akzent",
		"settings.accent.indigo":           "Indigo",
		"settings.accent.blue":             "Blau",
		"settings.accent.teal":             "Türkis",
		"settings.accent.green":            "Grün",
		"settings.accent.amber":            "Bernstein",
		"settings.accent.rose":             "Rosa",
		"settings.accent.violet":           "Violett",
		"settings.density":                 "Dichte",
		"settings.density.comfortable":     "Komfortabel",
		"settings.density.compact":         "Kompakt",
		"settings.icons":                   "Symbole",
		"settings.icons.unicode":           "Unicode",
		"settings.icons.nerd-font":         "Nerd Font",
		"settings.motion":                  "Bewegung",
		"settings.motion.none":             "Keine",
		"settings.motion.subtle":           "Dezent",
		"settings.motion.expressive":       "Ausdrucksstark",
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
		"wizard.advanced.help":             "Leertaste — umschalten • Enter — Expertenargumente",
		"wizard.expert.title":              "Expertenargumente",
		"wizard.expert.help":               "--option=wert verwenden. Interne/serverseitige Optionen und Positionsargumente werden abgelehnt.",
		"wizard.review.title":              "Prüfen",
		"wizard.warning.destructive":       "Warnung: Dieser Modus kann Daten entfernen.",
		"wizard.source_semantics":          "Quellsemantik",
		"wizard.dry_run":                   "Trockenlauf",
		"wizard.action.run":                "einmalig ausführen",
		"wizard.action.save_run":           "Profil speichern & ausführen",
		"wizard.review.help":               "[d] Trockenlauf  [s] Quellsemantik  [Enter] %s",
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
		"profiles.subtitle":                "Gespeicherte Übertragung zum Prüfen oder Ausführen wählen",
		"profiles.help":                    "↑/↓ auswählen • Enter ausführen • Esc zurück",
		"running.waiting":                  "Übertragung wird vorbereitet…",
		"running.activity":                 "Live-Aktivität",
		"running.scrolled":                 "Frühere Ausgabe · %d neuere Zeilen darunter",
		"result.success":                   "Erfolgreich abgeschlossen",
		"result.failure":                   "Übertragung fehlgeschlagen",
		"result.notification_warnings":     "%d Benachrichtigung(en) fehlgeschlagen.",
		"result.help":                      "Enter/Esc — zurück zur Übersicht",
		"browser.empty":                    "Keine zugänglichen Unterverzeichnisse.",
		"browser.hidden":                   "versteckte Dateien",
		"browser.help":                     "Enter öffnen • s aktuell wählen • h versteckte • Esc zurück",
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

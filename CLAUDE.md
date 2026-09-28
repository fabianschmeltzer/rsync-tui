# CLAUDE.md

Hinweise für Claude Code in diesem Repository.

## Projekt

`rsync-tui` ist eine zweisprachige (Englisch/Deutsch) Terminal-Oberfläche für
sichere lokale und SSH-Übertragungen mit `rsync`, geschrieben in Go 1.26 mit
Bubble Tea v2, Bubbles v2 und Lip Gloss v2 (`charm.land/...`). Zielplattform ist
Linux (amd64, arm64, armv7) als statisches Binary; rsync und OpenSSH werden
zur Laufzeit vorausgesetzt, nicht mitgeliefert.

## Befehle

```bash
gofmt -w cmd internal                 # formatieren (CI bricht bei Abweichungen ab)
go vet ./...
go test ./...                         # CI nutzt zusätzlich -race
go test ./internal/rsync -run TestName
go run ./cmd/rsync-tui self-test      # läuft auch in CI
go run ./cmd/rsync-tui                # TUI starten
shellcheck install.sh
go test -tags=ssh_integration ./internal/browser   # braucht SSH_TEST_HOST=user@host
```

Die CI (`.github/workflows/ci.yml`) prüft gofmt, vet, Tests mit Race-Detector
auf Go 1.26.x und 1.27.x, den Self-Test, shellcheck, Cross-Builds
(amd64/arm64/armv7, `CGO_ENABLED=0`), SSH-Integration und govulncheck.

## Aufbau

- `cmd/rsync-tui/main.go`: CLI-Einstieg und Unterbefehle (`run`, `doctor`,
  `update`, `schedule`, `profile`, `notify`, `snapshot`, `self-test`); ohne
  Argumente startet die TUI. `version` wird per `-ldflags` gesetzt.
- `cmd/release-helper`: Hilfswerkzeug für den Release-Workflow (Signatur, Manifest).
- `internal/domain`: Profile, Endpunkte, Modi (copy, mirror, move, snapshot,
  restore, custom) und deren Validierung.
- `internal/rsync`: Bau der rsync-Argumente (`command.go`), Preflight-Prüfungen,
  Ausführung mit Fortschritts-Events (`runner.go`) und Verlauf.
- `internal/job`: orchestriert einen Lauf (Preflight, rsync, Snapshot,
  Benachrichtigungen).
- `internal/snapshot`: `--link-dest`-Snapshots mit Last-N- oder GFS-Aufbewahrung.
- `internal/config`: XDG-konforme TOML-Profile, Einstellungen und Zustand.
- `internal/tui`: Bubble-Tea-Modell (`tui.go`), Designsystem und Themes
  (`design.go`), Layout (`shell.go`), Maus-Hit-Targets (`mouse.go`).
- `internal/i18n`: alle sichtbaren Texte als Schlüssel mit englischer und
  deutscher Übersetzung.
- `internal/browser`, `internal/sshclient`: lokaler und entfernter
  Verzeichnis-Browser über natives OpenSSH.
- `internal/scheduler`: systemd-User-/System-Timer.
- `internal/notify`: ntfy, Gotify, Webhook, Sendmail, SMTP.
- `internal/update`: signierte Selbstaktualisierung aus GitHub Releases mit
  atomarem Rollback.
- `legacy/`: altes Shell-Skript `rsync-srv-gui`, nicht mehr aktiv entwickelt.

## Konventionen

- rsync und ssh immer mit Argument-Arrays aufrufen, nie über `sh -c`.
- Neue UI-Texte in `internal/i18n` für Englisch und Deutsch ergänzen; die
  Tests prüfen die Vollständigkeit.
- Tests setzen `XDG_CONFIG_HOME`, `XDG_STATE_HOME` und `XDG_CACHE_HOME` per
  `t.Setenv` auf ein Temp-Verzeichnis und fassen nie echte Nutzerdaten an.
- Änderungen an Argumentbau, Pfadbehandlung oder destruktivem Verhalten
  brauchen Tests (siehe `CONTRIBUTING.md`).
- Dry-Run-Voreinstellungen, Pfadüberlappungsprüfungen, Snapshot-Invarianten,
  Update-Verifikation und Argument-Isolation nur mit ausdrücklicher
  Sicherheitsbegründung abschwächen.
- Keine Passwörter, Schlüssel, Tokens oder echten Serveradressen in Tests,
  Logs, Issues oder PRs.
- Exportierte Bezeichner bekommen Doc-Kommentare.
- Nutzerrelevante Änderungen unter `[Unreleased]` in `CHANGELOG.md`
  eintragen (Keep a Changelog). Releases laufen nur über
  `.github/workflows/release.yml`, siehe `docs/RELEASING.md`.
- PRs folgen `.github/pull_request_template.md`.

## Cloud-Sitzungen

`.claude/hooks/session-start.sh` lädt in Claude-Code-Web-Sitzungen die
Go-Module und installiert bei Bedarf rsync, OpenSSH-Client und shellcheck.

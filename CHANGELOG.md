# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/) and
Semantic Versioning.

## [Unreleased][]

## [0.1.3][] - 2026-07-05

### Added

- Added a responsive Material 3-inspired dashboard with app bar, navigation
  rail, cards, chips, steppers, snackbars and adaptive compact layouts
- Added configurable Material Dark, Material Light, Midnight, High Contrast
  and No Color themes with accent, density, icon and motion preferences
- Added optional Nerd Font icons with a portable Unicode fallback
- Added real mouse hit targets and hover states for cards, lists and controls

### Changed

- Redesigned the wizard, profiles, history, settings, directory browser,
  running transfer and result views around a shared per-model design system
- Replaced raw transfer-result JSON with readable Material summary cards
- Completed English and German localization of visible TUI status text

## [0.1.2][] - 2026-07-05

### Added

- Added one-time transfers in the TUI and CLI without creating a saved profile
- Added an every-start automatic update check interval
- Added a navigable, localized transfer history with readable summaries and
  detailed command and error views

### Changed

- Extended history records with mode and endpoint metadata while retaining
  compatibility with existing JSONL entries

## [0.1.1][] - 2026-07-05

### Changed

- Made the installer resolve the newest published GitHub release by default,
  while retaining `VERSION` as an explicit override

### Fixed

- Made language, theme and update preferences editable and persistent in the
  settings screen
- Removed empty source directories after successful Move transfers, including
  existing profiles and local, sudo and SSH sources

## [0.1.0][] - 2026-07-04

### Added

- Go-based Bubble Tea terminal interface in English and German
- Local and SSH endpoints with directory browsing
- Copy, Mirror, Move, Snapshot, Restore and Custom profiles
- Safe argument-based rsync command construction and preflight checks
- XDG/TOML profile storage, history and transfer locking
- Snapshot retention using Last-N or GFS policies
- systemd scheduling and unattended destructive-operation safeguards
- ntfy, Gotify, generic webhook, Sendmail and SMTP/TLS notifications
- Signed automatic updates, rollback, multi-architecture release workflow and SBOM

[Unreleased]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/fabianschmeltzer/rsync-tui/releases/tag/v0.1.0

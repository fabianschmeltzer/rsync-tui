# Changelog

All notable changes follow [Keep a Changelog](https://keepachangelog.com/) and
Semantic Versioning.

## [Unreleased][]

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

[Unreleased]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/fabianschmeltzer/rsync-tui/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/fabianschmeltzer/rsync-tui/releases/tag/v0.1.0

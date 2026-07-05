# rsync-tui

[![CI](https://github.com/fabianschmeltzer/rsync-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/fabianschmeltzer/rsync-tui/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

`rsync-tui` is a modern, bilingual terminal interface for safe local and SSH
file transfers with `rsync`. It is designed for Raspberry Pi,
OpenMediaVault, Debian/Ubuntu servers, removable storage and homelabs that are
often administered through SSH.

> `v0.1.3` is a beta release. Always inspect the command and use the preselected
> dry-run before destructive Mirror or Move operations.

[Deutsche Dokumentation](README.de.md)

![rsync-tui dashboard](docs/assets/rsync-tui-preview.svg)

## Highlights

- Responsive Bubble Tea interface with keyboard and mouse support
- Responsive Material 3-inspired dashboard, cards, steppers and status surfaces
- English and German UI with automatic locale detection
- Local directory browser for `/`, home, `/srv`, `/mnt`, `/media` and detected mounts
- SSH push/pull, native OpenSSH authentication and remote directory browser
- Copy, Mirror, Move, Snapshot, Restore and Custom modes
- One-time TUI and CLI transfers without creating a profile
- Complete command preview; rsync is executed with argument arrays, never `sh -c`
- Guided advanced options plus validated expert arguments
- XDG-compliant TOML profiles and a navigable transfer history
- `--link-dest` snapshots with Last-N or GFS retention
- systemd user/system timers and unattended safety limits
- ntfy, Gotify, webhook, Sendmail and SMTP/TLS notifications
- Signed, automatic GitHub Release updates with atomic rollback

## Requirements

- Linux on amd64, arm64 or armv7
- rsync 3.1.0 or newer
- OpenSSH client for remote transfers
- systemd only when schedules or the background update timer are used
- `sudo` only for profiles that explicitly request elevated access

The application itself is shipped as a static Go binary. rsync and OpenSSH are
not bundled.

## Installation

Download and inspect the installer, then run it:

```bash
curl -fLO https://raw.githubusercontent.com/fabianschmeltzer/rsync-tui/main/install.sh
less install.sh
sh install.sh
```

The default destination is `/usr/local/bin/rsync-tui` when the installer runs
as root and `~/.local/bin/rsync-tui` otherwise. If the user-local directory is
not already on `PATH`, the installer adds it to `~/.profile`; open a new shell
to activate it. Override the destination with `INSTALL_DIR`, select a release
with `VERSION`, or skip the user update timer with `NO_SYSTEMD=1`. By default,
the installer resolves the newest published release, including prereleases.

Manual installation:

```bash
VERSION="$(curl -fsSL 'https://api.github.com/repos/fabianschmeltzer/rsync-tui/releases?per_page=1' \
  | sed -n 's/.*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' \
  | head -n 1)"
curl -fLO "https://github.com/fabianschmeltzer/rsync-tui/releases/download/${VERSION}/rsync-tui_linux_amd64.tar.gz"
curl -fLO "https://github.com/fabianschmeltzer/rsync-tui/releases/download/${VERSION}/SHA256SUMS"
sha256sum --check --ignore-missing SHA256SUMS
tar -xzf rsync-tui_linux_amd64.tar.gz
install -m 0755 rsync-tui ~/.local/bin/rsync-tui
```

Run diagnostics after installation:

```bash
rsync-tui doctor
rsync-tui
```

## Usage

The interactive wizard accepts local paths and SSH endpoints:

```text
/srv/dev-disk-by-uuid-.../photos
backup@example.net:/srv/backups/photos
ssh://backup@example.net:2222/srv/backups/photos
```

Press `Ctrl+B` in a source or destination field to open the local or remote
directory browser. The source semantics are explicit:

- **contents** adds a trailing slash and copies the directory contents;
- **directory** copies the directory as a child of the destination.

Public commands:

```text
rsync-tui
rsync-tui run --profile <id|name> [--dry-run] [--scheduled]
rsync-tui run --source <path> --destination <path> [--mode copy|mirror|move]
rsync-tui profile list|show|configure
rsync-tui notify test --profile <id|name>
rsync-tui snapshot list --profile <id|name>
rsync-tui snapshot restore --profile <id|name> --snapshot <id>
rsync-tui doctor [--json]
rsync-tui schedule install --profile <id|name>
rsync-tui update [--check|--rollback]
rsync-tui version [--json]
```

For a one-time CLI transfer, `--name` is optional and
`--source-semantics contents|directory` controls the trailing-slash behavior.
One-time Mirror and Move commands are dry-runs unless both `--execute` and
`--yes` are supplied. Direct transfers cannot be scheduled.

Profiles are stored in `~/.config/rsync-tui/profiles/`; logs and history are
stored below `~/.local/state/rsync-tui/`. Secret-bearing files are created with
mode `0600` on Linux.

The update check interval in Settings includes **Every start**. It applies only
when launching the TUI and remains disabled when automatic updates are off.

### Appearance

The interface defaults to Material Dark with an Indigo accent. Settings offer
Material Dark, Material Light, Midnight, High Contrast and No Color themes,
seven accent palettes, comfortable or compact density, Unicode or optional
Nerd Font icons, and no/subtle/expressive motion. `NO_COLOR` always disables
ANSI styling without overwriting the saved preference.

Example schedule and notification configuration:

```bash
rsync-tui profile configure --profile Nightly \
  --schedule 'daily' \
  --retention gfs --daily 7 --weekly 4 --monthly 12 \
  --webhook-url 'https://example.invalid/hook' \
  --notify-success=true --notify-failure=true
rsync-tui schedule install --profile Nightly
rsync-tui notify test --profile Nightly
```

Notification tokens and SMTP passwords should be referenced through
`--ntfy-token-env`, `--ntfy-token-file`, `--gotify-token-env`,
`--gotify-token-file`, `--smtp-password-env` or `--smtp-password-file`; avoid
placing credentials directly in shell history.

## Safety model

- Identical and unsafe overlapping local paths are rejected.
- Mirror and Move default to dry-run. Bypassing it requires two confirmations.
- Scheduled destructive jobs require an explicit profile authorization,
  mandatory preview and numeric deletion/removal limits.
- Scheduled Move freezes a NUL-delimited local source manifest before preview.
- Scheduled SSH profiles require non-interactive key authentication.
- Passwords and private keys stay with OpenSSH and are never stored by this app.
- Snapshot pruning runs only after a successful new snapshot and always keeps
  at least the newest successful backup.
- rsync internal/server/daemon options and remote-to-remote copies are rejected.

Snapshot restores are dry-runs by default. A real restore additionally requires
`--execute --yes`.

## Snapshot layout

Managed snapshots are stored below:

```text
DESTINATION/.rsync-tui/PROFILE_ID/
├── latest -> snapshots/20260704T190000Z
└── snapshots/
    ├── 20260703T190000Z/
    └── 20260704T190000Z/
```

The default keeps the last ten successful snapshots. GFS retention can instead
keep seven daily, four weekly and twelve monthly snapshots.

## Building and testing

Go 1.25 or newer is required:

```bash
go test ./...
go vet ./...
go build -o bin/rsync-tui ./cmd/rsync-tui
```

Release tags trigger static builds for Linux amd64, arm64 and armv7. Release
manifests are Ed25519-signed and accompanied by SHA-256 sums, an SBOM and
GitHub build provenance.

## Limitations of v0.1.3

- Linux only
- no rsync daemon configuration
- no bidirectional synchronization
- no remote-to-remote transfer
- scheduled remote Move requires an interactive review
- SSH passwords are available only for interactive runs

The original Whiptail Bash script remains in [`legacy/`](legacy/) as an
unsupported reference.

## Contributing and security

See [CONTRIBUTING.md](CONTRIBUTING.md). Report vulnerabilities privately as
described in [SECURITY.md](SECURITY.md); do not include credentials or
unredacted endpoint URLs in issues.

## License

MIT — see [LICENSE](LICENSE).

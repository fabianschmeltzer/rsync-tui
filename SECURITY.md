# Security policy

## Supported versions

During the beta, only the newest published `0.x` release receives security
updates.

## Reporting a vulnerability

Please use GitHub's private vulnerability reporting feature for this
repository. Do not open a public issue for command injection, credential
exposure, unsafe deletion, updater verification or privilege-escalation bugs.

Include the affected version, platform, a minimal reproduction and the expected
impact. Remove credentials, private paths, keys, tokens and unredacted webhook
URLs.

## Security boundaries

`rsync-tui` executes the system-provided `rsync`, `ssh`, `sudo` and `systemctl`
binaries. It does not implement SSH, store SSH passwords or bundle rsync. Users
remain responsible for host-key verification, remote permissions and reviewing
destructive transfer previews.

Notification and SMTP credentials should be provided through environment
variables or owner-only (`0600`) secret files. URLs and tokens are redacted from
notification errors and JSON profile output.

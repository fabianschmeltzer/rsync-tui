# Releasing rsync-tui

Releases are built only by `.github/workflows/release.yml`.

## One-time signing setup

Generate a random 32-byte Ed25519 seed without committing or logging it:

```bash
openssl rand -base64 32
```

Add the resulting Base64 value as the repository Actions secret
`RSYNC_TUI_SIGNING_KEY`. The release workflow derives the public key, embeds it
in every binary, signs `manifest.json` and discards the private value when the
job ends.

Keep an offline backup of the secret. Losing it requires a manual trust-key
rotation release; publishing it compromises automatic updates.

## Release procedure

1. Ensure CI passes on `main`.
2. Update `CHANGELOG.md`.
3. Create and push an annotated tag such as `v0.1.2`.
4. Confirm that the release workflow publishes:
   - amd64, arm64 and armv7 archives;
   - `SHA256SUMS`;
   - signed `manifest.json`;
   - SPDX JSON SBOM;
   - GitHub artifact attestations.
5. Test the installer and `rsync-tui update --check` on an isolated Linux host.
6. For the beta, leave the GitHub release marked as a prerelease.

Never build an official release from an uncommitted worktree.

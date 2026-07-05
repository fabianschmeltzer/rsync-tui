# Contributing

Thank you for improving `rsync-tui`.

1. Open an issue for substantial behavior or safety changes.
2. Create a focused branch and keep commits small.
3. Run:

   ```bash
   gofmt -w cmd internal
   go test ./...
   go vet ./...
   ```

4. Include tests for command construction, path handling and every destructive
   behavior change.
5. Never include passwords, private keys, notification tokens or real server
   addresses in tests, logs, issues or pull requests.

Changes that weaken dry-run defaults, path overlap checks, snapshot retention
invariants, update verification or argument isolation require an explicit
security rationale.

package rsync

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestRunnerExecutesDryRunWithoutShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake rsync is exercised by Linux CI")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(bin, "rsync")
	script := `#!/bin/sh
if [ "${1:-}" = "--version" ]; then
  echo "rsync  version 3.2.7  protocol version 31"
  exit 0
fi
if [ "${1:-}" = "--help" ]; then
  echo "--archive --update --dry-run --itemize-changes"
  exit 0
fi
printf '%s\n' "$@" > "$RSYNC_ARGS_LOG"
echo "sent 10 bytes  received 5 bytes"
`
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	argsLog := filepath.Join(root, "args.log")
	t.Setenv("RSYNC_ARGS_LOG", argsLog)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))

	source := filepath.Join(root, "source")
	destination := filepath.Join(root, "destination")
	if err := os.Mkdir(source, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	store, err := config.Open()
	if err != nil {
		t.Fatal(err)
	}
	profile := domain.NewProfile("integration")
	profile.Source.Path = source
	profile.Destination.Path = destination
	result, err := (Runner{Store: store}).Run(context.Background(), profile, RunOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %+v", result)
	}
	data, err := os.ReadFile(argsLog)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "--dry-run") {
		t.Fatalf("dry-run argument missing: %s", data)
	}
}

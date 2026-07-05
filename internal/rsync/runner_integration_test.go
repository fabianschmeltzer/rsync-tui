package rsync

import (
	"context"
	"errors"
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

func TestRunnerMoveRemovesEmptySourceDirectories(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based fake rsync is exercised by Linux CI")
	}
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	fake := filepath.Join(bin, "rsync")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	source := filepath.Join(root, "source")
	emptyLeaf := filepath.Join(source, "first", "second")
	nonEmpty := filepath.Join(source, "keep")
	destination := filepath.Join(root, "destination")
	for _, directory := range []string{emptyLeaf, nonEmpty, destination} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(nonEmpty, "excluded.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	profile := domain.NewProfile("move")
	profile.Mode = domain.ModeMove
	profile.Source.Path = source
	profile.Destination.Path = destination
	profile.RemoveEmptyDirs = false // Legacy profiles stored the old, ineffective default.
	result, err := (Runner{}).Run(context.Background(), profile, RunOptions{SkipPreflight: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit code: %+v", result)
	}
	if _, err := os.Stat(emptyLeaf); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty source tree was not removed: %v", err)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source root must be preserved: %v", err)
	}
	if _, err := os.Stat(filepath.Join(nonEmpty, "excluded.txt")); err != nil {
		t.Fatalf("non-empty source directory was removed: %v", err)
	}
}

func TestRemoveEmptyDirectoriesPreservesRootAndNonEmptyDirectories(t *testing.T) {
	root := t.TempDir()
	emptyLeaf := filepath.Join(root, "empty", "nested")
	nonEmpty := filepath.Join(root, "keep")
	for _, directory := range []string{emptyLeaf, nonEmpty} {
		if err := os.MkdirAll(directory, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	keptFile := filepath.Join(nonEmpty, "file.txt")
	if err := os.WriteFile(keptFile, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := removeEmptyDirectories(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(emptyLeaf); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("empty directory was not removed: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		t.Fatalf("source root must be preserved: %v", err)
	}
	if _, err := os.Stat(keptFile); err != nil {
		t.Fatalf("non-empty directory was removed: %v", err)
	}
}

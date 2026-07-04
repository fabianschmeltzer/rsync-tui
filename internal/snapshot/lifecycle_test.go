package snapshot

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestLocalSnapshotLifecycle(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink lifecycle is exercised by Linux CI")
	}
	root := t.TempDir()
	profile := domain.NewProfile("snapshot")
	profile.Mode = domain.ModeSnapshot
	profile.Source.Path = filepath.Join(root, "source")
	profile.Destination.Path = filepath.Join(root, "repository")
	if err := os.Mkdir(profile.Source.Path, 0o700); err != nil {
		t.Fatal(err)
	}
	manager := Manager{}
	prepared, err := manager.Prepare(profile, time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(prepared.PartialPath, "data"), []byte("backup"), 0o600); err != nil {
		t.Fatal(err)
	}
	record, err := manager.Finalize(prepared, true)
	if err != nil {
		t.Fatal(err)
	}
	if !record.Successful {
		t.Fatal("snapshot was not marked successful")
	}
	records, err := manager.List(profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].ID != record.ID {
		t.Fatalf("unexpected snapshots: %+v", records)
	}
	latest := filepath.Join(profile.Destination.Path, ".rsync-tui", profile.ID, "latest")
	if _, err := os.Stat(latest); err != nil {
		t.Fatalf("latest link is invalid: %v", err)
	}
}

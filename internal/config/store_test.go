package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestStoreRoundTrip(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "config"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(root, "state"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(root, "cache"))

	store, err := Open()
	if err != nil {
		t.Fatal(err)
	}
	settings, err := store.LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if !settings.AutoUpdate || settings.UpdateChannel != "beta" {
		t.Fatalf("unexpected defaults: %+v", settings)
	}

	profile := domain.NewProfile("Nightly")
	profile.Source.Path = "/source"
	profile.Destination.Path = "/destination"
	if err := store.SaveProfile(profile); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.LoadProfile("nightly")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != profile.ID || loaded.Name != profile.Name {
		t.Fatalf("round trip mismatch: %+v", loaded)
	}
	info, err := os.Stat(filepath.Join(store.Paths.ProfilesDir, profile.ID+".toml"))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("profile permissions are too broad: %o", info.Mode().Perm())
	}
}

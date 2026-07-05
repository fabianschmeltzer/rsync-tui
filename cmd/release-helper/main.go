package main

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	updater "github.com/fabianschmeltzer/rsync-tui/internal/update"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: release-helper public-key|sign|manifest")
	}
	switch args[0] {
	case "public-key":
		private, err := signingKey()
		if err != nil {
			return err
		}
		fmt.Println(base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey)))
		return nil
	case "sign":
		if len(args) != 2 {
			return errors.New("usage: release-helper sign <manifest>")
		}
		private, err := signingKey()
		if err != nil {
			return err
		}
		data, err := os.ReadFile(args[1])
		if err != nil {
			return err
		}
		fmt.Println(base64.StdEncoding.EncodeToString(ed25519.Sign(private, data)))
		return nil
	case "manifest":
		if len(args) != 3 {
			return errors.New("usage: release-helper manifest <version> <dist-dir>")
		}
		return writeManifest(args[1], args[2])
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func signingKey() (ed25519.PrivateKey, error) {
	encoded := strings.TrimSpace(os.Getenv("RSYNC_TUI_SIGNING_KEY"))
	if encoded == "" {
		return nil, errors.New("RSYNC_TUI_SIGNING_KEY is not set")
	}
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}
	switch len(decoded) {
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(decoded), nil
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(decoded), nil
	default:
		return nil, fmt.Errorf("signing key must contain %d-byte seed or %d-byte private key", ed25519.SeedSize, ed25519.PrivateKeySize)
	}
}

func writeManifest(version, directory string) error {
	files, err := filepath.Glob(filepath.Join(directory, "rsync-tui_linux_*.tar.gz"))
	if err != nil {
		return err
	}
	sort.Strings(files)
	manifest := updater.Manifest{Version: version, Assets: make(map[string]updater.Asset)}
	tag := version
	if !strings.HasPrefix(tag, "v") {
		tag = "v" + tag
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		name := filepath.Base(file)
		key := strings.TrimSuffix(strings.TrimPrefix(name, "rsync-tui_"), ".tar.gz")
		sum := sha256.Sum256(data)
		manifest.Assets[key] = updater.Asset{
			URL:    "https://github.com/fabianschmeltzer/rsync-tui/releases/download/" + tag + "/" + name,
			SHA256: hex.EncodeToString(sum[:]),
		}
	}
	if len(manifest.Assets) == 0 {
		return errors.New("no release archives found")
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(manifest)
}

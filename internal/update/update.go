package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const releasesURL = "https://api.github.com/repos/fabianschmeltzer/rsync-tui/releases?per_page=20"

var EmbeddedPublicKey = ""

type Asset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type Manifest struct {
	Version string           `json:"version"`
	Assets  map[string]Asset `json:"assets"`
}

type release struct {
	TagName    string `json:"tag_name"`
	Prerelease bool   `json:"prerelease"`
	Draft      bool   `json:"draft"`
	Assets     []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
}

type Available struct {
	Version      string
	Manifest     Manifest
	ManifestRaw  []byte
	SignatureRaw []byte
}

type Client struct {
	HTTP      *http.Client
	PublicKey ed25519.PublicKey
	APIURL    string
}

func (c Client) Check(ctx context.Context, current, channel string) (*Available, error) {
	if current == "dev" || current == "" {
		return nil, nil
	}
	releases, err := c.list(ctx)
	if err != nil {
		return nil, err
	}
	sort.Slice(releases, func(i, j int) bool {
		return compareVersion(releases[i].TagName, releases[j].TagName) > 0
	})
	for _, candidate := range releases {
		if candidate.Draft || (channel == "stable" && candidate.Prerelease) {
			continue
		}
		if compareVersion(candidate.TagName, current) <= 0 {
			continue
		}
		manifestURL, signatureURL := releaseAssets(candidate)
		if manifestURL == "" || signatureURL == "" {
			continue
		}
		manifestRaw, err := c.download(ctx, manifestURL)
		if err != nil {
			return nil, err
		}
		signatureRaw, err := c.download(ctx, signatureURL)
		if err != nil {
			return nil, err
		}
		if err := c.verify(manifestRaw, signatureRaw); err != nil {
			return nil, err
		}
		var manifest Manifest
		if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
			return nil, err
		}
		if normalizeVersion(manifest.Version) != normalizeVersion(candidate.TagName) {
			return nil, errors.New("release manifest version does not match the release tag")
		}
		return &Available{
			Version:      candidate.TagName,
			Manifest:     manifest,
			ManifestRaw:  manifestRaw,
			SignatureRaw: signatureRaw,
		}, nil
	}
	return nil, nil
}

func (c Client) Install(ctx context.Context, available Available) error {
	if locks, _ := filepath.Glob(filepath.Join(stateDir(), "run-*.lock")); len(locks) > 0 {
		return errors.New("an rsync-tui transfer is active; update deferred")
	}
	key := runtime.GOOS + "_" + runtime.GOARCH
	if runtime.GOARCH == "arm" {
		key += "v7"
	}
	asset, ok := available.Manifest.Assets[key]
	if !ok {
		return fmt.Errorf("release does not contain an asset for %s", key)
	}
	archive, err := c.download(ctx, asset.URL)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(archive)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), asset.SHA256) {
		return errors.New("downloaded update checksum mismatch")
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	candidate := executable + ".new"
	if err := extractBinary(archive, candidate); err != nil {
		return err
	}
	defer os.Remove(candidate)
	if err := os.Chmod(candidate, 0o755); err != nil {
		return err
	}
	selfTest := exec.CommandContext(ctx, candidate, "self-test")
	if output, err := selfTest.CombinedOutput(); err != nil {
		return fmt.Errorf("new binary self-test failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	previous := executable + ".previous"
	_ = os.Remove(previous)
	if err := backupFile(executable, previous); err != nil {
		return err
	}
	if err := os.Rename(candidate, executable); err != nil {
		return err
	}
	return nil
}

func Rollback() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return err
	}
	previous := executable + ".previous"
	if _, err := os.Stat(previous); err != nil {
		return errors.New("no previous rsync-tui binary is available")
	}
	current := executable + ".rollback"
	_ = os.Remove(current)
	if err := backupFile(executable, current); err != nil {
		return err
	}
	if err := os.Rename(previous, executable); err != nil {
		return err
	}
	_ = os.Remove(previous)
	return os.Rename(current, previous)
}

func (c Client) list(ctx context.Context) ([]release, error) {
	endpoint := c.APIURL
	if endpoint == "" {
		endpoint = releasesURL
	}
	data, err := c.download(ctx, endpoint)
	if err != nil {
		return nil, err
	}
	var releases []release
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, err
	}
	return releases, nil
}

func (c Client) download(ctx context.Context, endpoint string) ([]byte, error) {
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "rsync-tui")
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("download returned %s", response.Status)
	}
	return io.ReadAll(io.LimitReader(response.Body, 256*1024*1024))
}

func (c Client) verify(manifest, encodedSignature []byte) error {
	key := c.PublicKey
	if len(key) == 0 && EmbeddedPublicKey != "" {
		decoded, err := base64.StdEncoding.DecodeString(EmbeddedPublicKey)
		if err != nil {
			return err
		}
		key = ed25519.PublicKey(decoded)
	}
	if len(key) != ed25519.PublicKeySize {
		return errors.New("update signing public key is not configured")
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(encodedSignature)))
	if err != nil {
		return err
	}
	if !ed25519.Verify(key, manifest, signature) {
		return errors.New("release manifest signature is invalid")
	}
	return nil
}

func releaseAssets(release release) (string, string) {
	var manifest, signature string
	for _, asset := range release.Assets {
		switch asset.Name {
		case "manifest.json":
			manifest = asset.URL
		case "manifest.json.sig":
			signature = asset.URL
		}
	}
	return manifest, signature
}

func extractBinary(archive []byte, destination string) error {
	gzipReader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(header.Name) != "rsync-tui" || header.Typeflag != tar.TypeReg {
			continue
		}
		file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(file, io.LimitReader(tarReader, 128*1024*1024))
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return errors.New("update archive does not contain rsync-tui")
}

func compareVersion(a, b string) int {
	av, apre := parseVersion(a)
	bv, bpre := parseVersion(b)
	for index := range av {
		if av[index] < bv[index] {
			return -1
		}
		if av[index] > bv[index] {
			return 1
		}
	}
	if apre == bpre {
		return 0
	}
	if apre == "" {
		return 1
	}
	if bpre == "" {
		return -1
	}
	if apre < bpre {
		return -1
	}
	return 1
}

func parseVersion(value string) ([3]int, string) {
	value = normalizeVersion(value)
	main, prerelease, _ := strings.Cut(value, "-")
	parts := strings.Split(main, ".")
	var result [3]int
	for index := 0; index < len(result) && index < len(parts); index++ {
		result[index], _ = strconv.Atoi(parts[index])
	}
	return result, prerelease
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}

func stateDir() string {
	home, _ := os.UserHomeDir()
	if value := os.Getenv("XDG_STATE_HOME"); value != "" {
		return filepath.Join(value, "rsync-tui")
	}
	return filepath.Join(home, ".local", "state", "rsync-tui")
}

func backupFile(source, destination string) error {
	if err := os.Link(source, destination); err == nil {
		return nil
	}
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	info, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	destinationFile, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(destinationFile, sourceFile)
	closeErr := destinationFile.Close()
	if copyErr != nil {
		_ = os.Remove(destination)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(destination)
		return closeErr
	}
	return nil
}

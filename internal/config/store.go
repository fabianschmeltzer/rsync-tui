package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/pelletier/go-toml/v2"
)

type Settings struct {
	SchemaVersion int    `toml:"schema_version"`
	Language      string `toml:"language"`
	Theme         string `toml:"theme"`
	AutoUpdate    bool   `toml:"auto_update"`
	UpdateChannel string `toml:"update_channel"`
	CheckHours    int    `toml:"check_hours"`
}

func DefaultSettings() Settings {
	return Settings{
		SchemaVersion: 1,
		Language:      "auto",
		Theme:         "auto",
		AutoUpdate:    true,
		UpdateChannel: "beta",
		CheckHours:    24,
	}
}

type Paths struct {
	ConfigDir   string
	ProfilesDir string
	StateDir    string
	LogDir      string
	CacheDir    string
}

func ResolvePaths() (Paths, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, err
	}
	configRoot := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	stateRoot := envOr("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))
	cacheRoot := envOr("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	configDir := filepath.Join(configRoot, "rsync-tui")
	stateDir := filepath.Join(stateRoot, "rsync-tui")
	return Paths{
		ConfigDir:   configDir,
		ProfilesDir: filepath.Join(configDir, "profiles"),
		StateDir:    stateDir,
		LogDir:      filepath.Join(stateDir, "logs"),
		CacheDir:    filepath.Join(cacheRoot, "rsync-tui"),
	}, nil
}

func (p Paths) Ensure() error {
	for _, dir := range []string{p.ConfigDir, p.ProfilesDir, p.StateDir, p.LogDir, p.CacheDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return nil
}

type Store struct {
	Paths Paths
}

func Open() (*Store, error) {
	paths, err := ResolvePaths()
	if err != nil {
		return nil, err
	}
	if err := paths.Ensure(); err != nil {
		return nil, err
	}
	return &Store{Paths: paths}, nil
}

func (s *Store) LoadSettings() (Settings, error) {
	path := filepath.Join(s.Paths.ConfigDir, "config.toml")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		settings := DefaultSettings()
		return settings, s.SaveSettings(settings)
	}
	if err != nil {
		return Settings{}, err
	}
	settings := DefaultSettings()
	if err := toml.Unmarshal(data, &settings); err != nil {
		return Settings{}, fmt.Errorf("decode settings: %w", err)
	}
	if settings.SchemaVersion != 1 {
		return Settings{}, fmt.Errorf("unsupported settings schema %d", settings.SchemaVersion)
	}
	if settings.CheckHours < 0 {
		settings.CheckHours = DefaultSettings().CheckHours
	}
	return settings, nil
}

func (s *Store) SaveSettings(settings Settings) error {
	settings.SchemaVersion = 1
	data, err := toml.Marshal(settings)
	if err != nil {
		return err
	}
	return atomicWrite(filepath.Join(s.Paths.ConfigDir, "config.toml"), data, 0o600)
}

func (s *Store) SaveProfile(profile domain.Profile) error {
	if profile.ID == "" {
		return errors.New("profile has no ID")
	}
	profile.SchemaVersion = 1
	profile.UpdatedAt = time.Now().UTC()
	if profile.CreatedAt.IsZero() {
		profile.CreatedAt = profile.UpdatedAt
	}
	if err := profile.Validate(); err != nil {
		return err
	}
	data, err := toml.Marshal(profile)
	if err != nil {
		return err
	}
	return atomicWrite(s.profilePath(profile.ID), data, 0o600)
}

func (s *Store) LoadProfile(identifier string) (domain.Profile, error) {
	if strings.TrimSpace(identifier) == "" {
		return domain.Profile{}, errors.New("profile identifier is empty")
	}
	direct := s.profilePath(identifier)
	if profile, err := readProfile(direct); err == nil {
		return profile, nil
	}
	profiles, err := s.ListProfiles()
	if err != nil {
		return domain.Profile{}, err
	}
	for _, profile := range profiles {
		if strings.EqualFold(profile.Name, identifier) {
			return profile, nil
		}
	}
	return domain.Profile{}, fmt.Errorf("profile %q not found", identifier)
}

func (s *Store) ListProfiles() ([]domain.Profile, error) {
	entries, err := os.ReadDir(s.Paths.ProfilesDir)
	if err != nil {
		return nil, err
	}
	profiles := make([]domain.Profile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		profile, err := readProfile(filepath.Join(s.Paths.ProfilesDir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		profiles = append(profiles, profile)
	}
	sort.Slice(profiles, func(i, j int) bool {
		return strings.ToLower(profiles[i].Name) < strings.ToLower(profiles[j].Name)
	})
	return profiles, nil
}

func (s *Store) DeleteProfile(id string) error {
	if strings.TrimSpace(id) == "" || strings.ContainsAny(id, `/\`) {
		return errors.New("invalid profile ID")
	}
	return os.Remove(s.profilePath(id))
}

func (s *Store) profilePath(id string) string {
	return filepath.Join(s.Paths.ProfilesDir, id+".toml")
}

func readProfile(path string) (domain.Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return domain.Profile{}, err
	}
	var profile domain.Profile
	if err := toml.Unmarshal(data, &profile); err != nil {
		return domain.Profile{}, err
	}
	if err := profile.Validate(); err != nil {
		return domain.Profile{}, err
	}
	return profile, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := file.Name()
	defer os.Remove(tempName)
	if err := file.Chmod(mode); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func envOr(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

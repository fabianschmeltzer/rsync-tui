package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Mode string

const (
	ModeCopy     Mode = "copy"
	ModeMirror   Mode = "mirror"
	ModeMove     Mode = "move"
	ModeSnapshot Mode = "snapshot"
	ModeRestore  Mode = "restore"
	ModeCustom   Mode = "custom"
)

type EndpointKind string

const (
	EndpointLocal EndpointKind = "local"
	EndpointSSH   EndpointKind = "ssh"
)

type SourceSemantics string

const (
	CopyContents  SourceSemantics = "contents"
	CopyDirectory SourceSemantics = "directory"
)

type RetentionMode string

const (
	RetentionLastN RetentionMode = "last_n"
	RetentionGFS   RetentionMode = "gfs"
)

type Endpoint struct {
	Kind EndpointKind `toml:"kind" json:"kind"`
	Path string       `toml:"path" json:"path"`
	Host string       `toml:"host,omitempty" json:"host,omitempty"`
	User string       `toml:"user,omitempty" json:"user,omitempty"`
	Port int          `toml:"port,omitempty" json:"port,omitempty"`
}

func (e Endpoint) IsRemote() bool {
	return e.Kind == EndpointSSH
}

func (e Endpoint) SSHHost() string {
	if e.User == "" {
		return e.Host
	}
	return e.User + "@" + e.Host
}

func (e Endpoint) Address(copyContents bool) string {
	path := e.Path
	if copyContents && path != "/" {
		path = strings.TrimRight(path, "/") + "/"
	}
	if e.Kind == EndpointSSH {
		return e.SSHHost() + ":" + path
	}
	return path
}

func (e Endpoint) Validate() error {
	if e.Kind != EndpointLocal && e.Kind != EndpointSSH {
		return fmt.Errorf("unsupported endpoint kind %q", e.Kind)
	}
	if strings.TrimSpace(e.Path) == "" {
		return errors.New("endpoint path is empty")
	}
	if strings.IndexByte(e.Path, 0) >= 0 {
		return errors.New("endpoint path contains a NUL byte")
	}
	if e.Kind == EndpointSSH {
		if strings.TrimSpace(e.Host) == "" {
			return errors.New("SSH host is empty")
		}
		if strings.HasPrefix(e.Host, "-") {
			return errors.New("SSH host must not begin with '-'")
		}
		if strings.ContainsAny(e.Host, "\r\n\t ;|&$`\\\"'()<>") {
			return errors.New("SSH host contains unsafe characters")
		}
		for _, character := range e.Host {
			if !isSSHHostCharacter(character) {
				return errors.New("SSH host contains unsupported characters")
			}
		}
		if strings.HasPrefix(e.User, "-") {
			return errors.New("SSH user must not begin with '-'")
		}
		if e.User != "" && strings.ContainsAny(e.User, "\r\n\t ;|&$`\\\"'()<>:@") {
			return errors.New("SSH user contains unsafe characters")
		}
		if e.Port < 0 || e.Port > 65535 {
			return errors.New("SSH port is outside 1-65535")
		}
	}
	return nil
}

func isSSHHostCharacter(character rune) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9' ||
		strings.ContainsRune("._:-", character)
}

type FilterSet struct {
	Include []string `toml:"include,omitempty" json:"include,omitempty"`
	Exclude []string `toml:"exclude,omitempty" json:"exclude,omitempty"`
}

type Safety struct {
	AllowUnattendedDestructive bool   `toml:"allow_unattended_destructive" json:"allow_unattended_destructive"`
	MaxDelete                  int    `toml:"max_delete,omitempty" json:"max_delete,omitempty"`
	MaxSourceRemovals          int    `toml:"max_source_removals,omitempty" json:"max_source_removals,omitempty"`
	ExpectedSourceDevice       string `toml:"expected_source_device,omitempty" json:"expected_source_device,omitempty"`
	ExpectedDestinationDevice  string `toml:"expected_destination_device,omitempty" json:"expected_destination_device,omitempty"`
}

type Retention struct {
	Mode    RetentionMode `toml:"mode" json:"mode"`
	LastN   int           `toml:"last_n" json:"last_n"`
	Daily   int           `toml:"daily" json:"daily"`
	Weekly  int           `toml:"weekly" json:"weekly"`
	Monthly int           `toml:"monthly" json:"monthly"`
}

func DefaultRetention() Retention {
	return Retention{
		Mode:    RetentionLastN,
		LastN:   10,
		Daily:   7,
		Weekly:  4,
		Monthly: 12,
	}
}

type SnapshotConfig struct {
	Enabled        bool      `toml:"enabled" json:"enabled"`
	Root           string    `toml:"root,omitempty" json:"root,omitempty"`
	Retention      Retention `toml:"retention" json:"retention"`
	VerifyAfterRun bool      `toml:"verify_after_run" json:"verify_after_run"`
}

type Schedule struct {
	Enabled    bool   `toml:"enabled" json:"enabled"`
	OnCalendar string `toml:"on_calendar,omitempty" json:"on_calendar,omitempty"`
	System     bool   `toml:"system" json:"system"`
}

type SMTPConfig struct {
	Address      string `toml:"address,omitempty" json:"address,omitempty"`
	Username     string `toml:"username,omitempty" json:"username,omitempty"`
	PasswordEnv  string `toml:"password_env,omitempty" json:"-"`
	PasswordFile string `toml:"password_file,omitempty" json:"-"`
	From         string `toml:"from,omitempty" json:"from,omitempty"`
	To           string `toml:"to,omitempty" json:"to,omitempty"`
}

type Notifications struct {
	OnSuccess       bool       `toml:"on_success" json:"on_success"`
	OnFailure       bool       `toml:"on_failure" json:"on_failure"`
	NtfyURL         string     `toml:"ntfy_url,omitempty" json:"-"`
	NtfyToken       string     `toml:"ntfy_token,omitempty" json:"-"`
	NtfyTokenEnv    string     `toml:"ntfy_token_env,omitempty" json:"-"`
	NtfyTokenFile   string     `toml:"ntfy_token_file,omitempty" json:"-"`
	GotifyURL       string     `toml:"gotify_url,omitempty" json:"-"`
	GotifyToken     string     `toml:"gotify_token,omitempty" json:"-"`
	GotifyTokenEnv  string     `toml:"gotify_token_env,omitempty" json:"-"`
	GotifyTokenFile string     `toml:"gotify_token_file,omitempty" json:"-"`
	WebhookURL      string     `toml:"webhook_url,omitempty" json:"-"`
	Sendmail        string     `toml:"sendmail,omitempty" json:"sendmail,omitempty"`
	SMTP            SMTPConfig `toml:"smtp,omitempty" json:"smtp,omitempty"`
}

type Profile struct {
	SchemaVersion   int             `toml:"schema_version" json:"schema_version"`
	ID              string          `toml:"id" json:"id"`
	Name            string          `toml:"name" json:"name"`
	Description     string          `toml:"description,omitempty" json:"description,omitempty"`
	Mode            Mode            `toml:"mode" json:"mode"`
	Source          Endpoint        `toml:"source" json:"source"`
	Destination     Endpoint        `toml:"destination" json:"destination"`
	SourceSemantics SourceSemantics `toml:"source_semantics" json:"source_semantics"`
	UseSudo         bool            `toml:"use_sudo" json:"use_sudo"`
	RemoveEmptyDirs bool            `toml:"remove_empty_dirs" json:"remove_empty_dirs"`
	DryRunByDefault bool            `toml:"dry_run_by_default" json:"dry_run_by_default"`
	Options         []string        `toml:"options,omitempty" json:"options,omitempty"`
	Filters         FilterSet       `toml:"filters,omitempty" json:"filters,omitempty"`
	Safety          Safety          `toml:"safety" json:"safety"`
	Snapshot        SnapshotConfig  `toml:"snapshot" json:"snapshot"`
	Schedule        Schedule        `toml:"schedule" json:"schedule"`
	Notifications   Notifications   `toml:"notifications" json:"notifications"`
	CreatedAt       time.Time       `toml:"created_at" json:"created_at"`
	UpdatedAt       time.Time       `toml:"updated_at" json:"updated_at"`
}

func NewProfile(name string) Profile {
	now := time.Now().UTC()
	return Profile{
		SchemaVersion:   1,
		ID:              newID(),
		Name:            name,
		Mode:            ModeCopy,
		Source:          Endpoint{Kind: EndpointLocal},
		Destination:     Endpoint{Kind: EndpointLocal},
		SourceSemantics: CopyContents,
		RemoveEmptyDirs: true,
		DryRunByDefault: true,
		Snapshot:        SnapshotConfig{Retention: DefaultRetention()},
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func (p Profile) Destructive() bool {
	return p.Mode == ModeMirror || p.Mode == ModeMove
}

func (p Profile) Validate() error {
	if p.SchemaVersion != 1 {
		return fmt.Errorf("unsupported profile schema %d", p.SchemaVersion)
	}
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.Name) == "" {
		return errors.New("profile ID and name are required")
	}
	switch p.Mode {
	case ModeCopy, ModeMirror, ModeMove, ModeSnapshot, ModeRestore, ModeCustom:
	default:
		return fmt.Errorf("unsupported mode %q", p.Mode)
	}
	if err := p.Source.Validate(); err != nil {
		return fmt.Errorf("source: %w", err)
	}
	if err := p.Destination.Validate(); err != nil {
		return fmt.Errorf("destination: %w", err)
	}
	if p.Source.IsRemote() && p.Destination.IsRemote() {
		return errors.New("remote-to-remote transfers are not supported")
	}
	if p.SourceSemantics != CopyContents && p.SourceSemantics != CopyDirectory {
		return errors.New("invalid source semantics")
	}
	if p.Schedule.Enabled && strings.TrimSpace(p.Schedule.OnCalendar) == "" {
		return errors.New("scheduled profile requires an OnCalendar expression")
	}
	if p.Schedule.Enabled && p.Destructive() {
		if !p.Safety.AllowUnattendedDestructive {
			return errors.New("scheduled destructive profile requires explicit authorization")
		}
		if p.Mode == ModeMirror && p.Safety.MaxDelete < 1 {
			return errors.New("scheduled mirror requires max_delete > 0")
		}
		if p.Mode == ModeMove && p.Safety.MaxSourceRemovals < 1 {
			return errors.New("scheduled move requires max_source_removals > 0")
		}
	}
	if p.Snapshot.Retention.Mode == RetentionLastN && p.Snapshot.Retention.LastN < 1 {
		return errors.New("snapshot retention must keep at least one backup")
	}
	if p.Snapshot.Retention.Mode == RetentionGFS &&
		p.Snapshot.Retention.Daily+p.Snapshot.Retention.Weekly+p.Snapshot.Retention.Monthly < 1 {
		return errors.New("GFS retention must keep at least one backup")
	}
	return validatePort(p)
}

func validatePort(p Profile) error {
	for _, endpoint := range []Endpoint{p.Source, p.Destination} {
		if endpoint.Port == 0 {
			continue
		}
		if net.ParseIP(endpoint.Host) != nil || endpoint.Host != "" {
			if _, err := strconv.Atoi(strconv.Itoa(endpoint.Port)); err != nil {
				return err
			}
		}
	}
	return nil
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}

func CanonicalLocalPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	return filepath.Clean(absolute), nil
}

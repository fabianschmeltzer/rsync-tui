package rsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// CheckLevel identifies the severity of a preflight check result.
type CheckLevel string

// Supported preflight check levels.
const (
	CheckOK      CheckLevel = "ok"
	CheckWarning CheckLevel = "warning"
	CheckError   CheckLevel = "error"
)

// Check describes one preflight validation result.
type Check struct {
	Name    string     `json:"name"`
	Level   CheckLevel `json:"level"`
	Message string     `json:"message"`
}

// Preflight validates whether a profile is ready to run.
func Preflight(ctx context.Context, profile domain.Profile, scheduled bool) []Check {
	checks := make([]Check, 0, 10)
	if err := profile.Validate(); err != nil {
		return append(checks, Check{"profile", CheckError, err.Error()})
	}
	checks = append(checks, Check{"profile", CheckOK, "profile schema and options are valid"})

	checks = append(checks, commandCheck(ctx, "rsync", "--version"))
	checks = append(checks, capabilityCheck(ctx, profile))
	if profile.Source.IsRemote() || profile.Destination.IsRemote() {
		checks = append(checks, commandCheck(ctx, "ssh", "-V"))
	}
	if profile.UseSudo {
		if _, err := exec.LookPath("sudo"); err != nil {
			checks = append(checks, Check{"sudo", CheckError, "sudo is not installed"})
		} else {
			checks = append(checks, Check{"sudo", CheckOK, "sudo is available"})
		}
	}

	if !profile.Source.IsRemote() {
		if info, err := os.Stat(profile.Source.Path); err != nil {
			checks = append(checks, Check{"source", CheckError, err.Error()})
		} else if !info.IsDir() {
			checks = append(checks, Check{"source", CheckWarning, "source is not a directory"})
		} else {
			checks = append(checks, Check{"source", CheckOK, "source exists"})
		}
		if profile.Safety.ExpectedSourceDevice != "" {
			checks = append(checks, deviceCheck(ctx, "source-device", profile.Source.Path, profile.Safety.ExpectedSourceDevice))
		}
	}
	if !profile.Destination.IsRemote() {
		checks = append(checks, localDestinationCheck(profile.Destination.Path))
		checks = append(checks, diskSpaceCheck(ctx, profile.Destination.Path))
		if profile.Safety.ExpectedDestinationDevice != "" {
			checks = append(checks, deviceCheck(ctx, "destination-device", profile.Destination.Path, profile.Safety.ExpectedDestinationDevice))
		}
	}
	if !profile.Source.IsRemote() && !profile.Destination.IsRemote() {
		checks = append(checks, overlapCheck(profile))
	}
	if scheduled && (profile.Source.IsRemote() || profile.Destination.IsRemote()) {
		checks = append(checks, Check{
			Name:    "ssh-auth",
			Level:   CheckWarning,
			Message: "scheduled SSH jobs require key-based BatchMode authentication",
		})
	}
	if scheduled && profile.Destructive() && !profile.Safety.AllowUnattendedDestructive {
		checks = append(checks, Check{"safety", CheckError, "unattended destructive execution is not authorized"})
	}
	return checks
}

// HasErrors reports whether any preflight check failed.
func HasErrors(checks []Check) bool {
	for _, check := range checks {
		if check.Level == CheckError {
			return true
		}
	}
	return false
}

func commandCheck(ctx context.Context, name string, arg string) Check {
	path, err := exec.LookPath(name)
	if err != nil {
		return Check{name, CheckError, name + " is not installed or not in PATH"}
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(checkCtx, path, arg).CombinedOutput()
	if err != nil && name != "ssh" {
		return Check{name, CheckError, strings.TrimSpace(string(output))}
	}
	message := strings.TrimSpace(firstLine(string(output)))
	if name == "rsync" {
		if version, ok := parseRsyncVersion(message); ok && versionLess(version, [3]int{3, 1, 0}) {
			return Check{name, CheckError, fmt.Sprintf("rsync %d.%d.%d is too old; 3.1.0 or newer is required", version[0], version[1], version[2])}
		}
	}
	return Check{name, CheckOK, fmt.Sprintf("%s: %s", filepath.Base(path), message)}
}

func capabilityCheck(ctx context.Context, profile domain.Profile) Check {
	path, err := exec.LookPath("rsync")
	if err != nil {
		return Check{"rsync-options", CheckError, "rsync is not installed"}
	}
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	output, err := exec.CommandContext(checkCtx, path, "--help").CombinedOutput()
	if err != nil {
		return Check{"rsync-options", CheckWarning, "could not inspect rsync --help"}
	}
	help := string(output)
	for _, option := range profile.Options {
		name, _, _ := strings.Cut(option, "=")
		if strings.HasPrefix(name, "--") && !strings.Contains(help, name) {
			return Check{"rsync-options", CheckError, "installed rsync does not advertise " + name}
		}
	}
	return Check{"rsync-options", CheckOK, "selected expert options are supported locally"}
}

func localDestinationCheck(path string) Check {
	info, err := os.Stat(path)
	if err == nil {
		if !info.IsDir() {
			return Check{"destination", CheckWarning, "destination exists and is not a directory"}
		}
		return writableCheck(path)
	}
	parent := filepath.Dir(path)
	for {
		if _, parentErr := os.Stat(parent); parentErr == nil {
			break
		}
		next := filepath.Dir(parent)
		if next == parent {
			return Check{"destination", CheckError, "no existing destination parent"}
		}
		parent = next
	}
	check := writableCheck(parent)
	if check.Level == CheckError {
		return check
	}
	return Check{"destination", CheckWarning, "destination will be created below " + parent}
}

func writableCheck(path string) Check {
	if runtime.GOOS == "linux" {
		if err := exec.Command("test", "-w", path).Run(); err != nil {
			return Check{"destination", CheckError, "destination or parent is not writable"}
		}
	}
	return Check{"destination", CheckOK, "destination or parent appears writable"}
}

func deviceCheck(ctx context.Context, name, path, expected string) Check {
	if runtime.GOOS != "linux" {
		return Check{name, CheckWarning, "mount identity checks are available on Linux only"}
	}
	findmnt, err := exec.LookPath("findmnt")
	if err != nil {
		return Check{name, CheckWarning, "findmnt is unavailable; mount identity was not checked"}
	}
	target := existingParent(path)
	output, err := exec.CommandContext(ctx, findmnt, "-n", "-o", "SOURCE", "--target", target).CombinedOutput()
	if err != nil {
		return Check{name, CheckError, "could not determine mount source"}
	}
	actual := strings.TrimSpace(string(output))
	if actual != expected {
		return Check{name, CheckError, fmt.Sprintf("expected %s but %s is mounted", expected, actual)}
	}
	return Check{name, CheckOK, "expected filesystem is mounted"}
}

func diskSpaceCheck(ctx context.Context, path string) Check {
	if runtime.GOOS != "linux" {
		return Check{"disk-space", CheckWarning, "free-space checks are available on Linux only"}
	}
	df, err := exec.LookPath("df")
	if err != nil {
		return Check{"disk-space", CheckWarning, "df is unavailable"}
	}
	output, err := exec.CommandContext(ctx, df, "-Pk", existingParent(path)).CombinedOutput()
	if err != nil {
		return Check{"disk-space", CheckWarning, "could not determine free destination space"}
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return Check{"disk-space", CheckWarning, "unexpected df output"}
	}
	fields := strings.Fields(lines[len(lines)-1])
	if len(fields) < 4 {
		return Check{"disk-space", CheckWarning, "unexpected df output"}
	}
	kilobytes, err := strconv.ParseInt(fields[3], 10, 64)
	if err != nil {
		return Check{"disk-space", CheckWarning, "could not parse free destination space"}
	}
	return Check{"disk-space", CheckOK, fmt.Sprintf("%.1f GiB available", float64(kilobytes)/(1024*1024))}
}

func existingParent(path string) string {
	path = filepath.Clean(path)
	for {
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(path)
		if parent == path {
			return path
		}
		path = parent
	}
}

func overlapCheck(profile domain.Profile) Check {
	source, err := domain.CanonicalLocalPath(profile.Source.Path)
	if err != nil {
		return Check{"path-overlap", CheckError, err.Error()}
	}
	destination, err := domain.CanonicalLocalPath(profile.Destination.Path)
	if err != nil {
		return Check{"path-overlap", CheckError, err.Error()}
	}
	if equalPath(source, destination) {
		return Check{"path-overlap", CheckError, "source and destination are identical"}
	}
	if isInside(destination, source) {
		return Check{"path-overlap", CheckError, "destination is inside source"}
	}
	if isInside(source, destination) {
		if profile.Destructive() || profile.Mode == domain.ModeSnapshot {
			return Check{"path-overlap", CheckError, "source is inside destination for a destructive or snapshot operation"}
		}
		return Check{"path-overlap", CheckWarning, "source is inside destination"}
	}
	return Check{"path-overlap", CheckOK, "source and destination do not overlap"}
}

func isInside(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	if err != nil {
		return false
	}
	return relative != "." && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func equalPath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

var rsyncVersionPattern = regexp.MustCompile(`(?i)rsync\s+version\s+(\d+)\.(\d+)\.(\d+)`)

func parseRsyncVersion(line string) ([3]int, bool) {
	match := rsyncVersionPattern.FindStringSubmatch(line)
	if len(match) != 4 {
		return [3]int{}, false
	}
	var version [3]int
	for index := range version {
		value, err := strconv.Atoi(match[index+1])
		if err != nil {
			return [3]int{}, false
		}
		version[index] = value
	}
	return version, true
}

func versionLess(a, b [3]int) bool {
	for index := range a {
		if a[index] != b[index] {
			return a[index] < b[index]
		}
	}
	return false
}

func firstLine(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	lines := strings.Split(value, "\n")
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

var errPreflight = errors.New("preflight failed")

// PreflightError combines failed preflight checks into a single error.
func PreflightError(checks []Check) error {
	if HasErrors(checks) {
		return errPreflight
	}
	return nil
}

package browser

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// Entry describes a directory shown in the path browser.
type Entry struct {
	Name string
	Path string
}

// LocalDirectories lists child directories of a local path.
func LocalDirectories(current string, showHidden bool) ([]Entry, error) {
	if current == "" {
		var err error
		current, err = os.UserHomeDir()
		if err != nil {
			current = "/"
		}
	}
	current, err := filepath.Abs(current)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(current)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(items)+1)
	parent := filepath.Dir(current)
	if parent != current {
		entries = append(entries, Entry{Name: "..", Path: parent})
	}
	for _, item := range items {
		if !item.IsDir() {
			continue
		}
		if !showHidden && strings.HasPrefix(item.Name(), ".") {
			continue
		}
		entries = append(entries, Entry{Name: item.Name(), Path: filepath.Join(current, item.Name())})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Name == ".." {
			return true
		}
		if entries[j].Name == ".." {
			return false
		}
		return strings.ToLower(entries[i].Name) < strings.ToLower(entries[j].Name)
	})
	return entries, nil
}

// Shortcuts returns common local-directory shortcuts for the current user.
func Shortcuts() []Entry {
	home, _ := os.UserHomeDir()
	candidates := []Entry{
		{Name: "Home", Path: home},
		{Name: "Root", Path: "/"},
		{Name: "srv", Path: "/srv"},
		{Name: "mnt", Path: "/mnt"},
		{Name: "media", Path: "/media"},
	}
	seen := make(map[string]bool)
	var result []Entry
	for _, item := range candidates {
		if item.Path == "" || seen[item.Path] {
			continue
		}
		if info, err := os.Stat(item.Path); err == nil && info.IsDir() {
			seen[item.Path] = true
			result = append(result, item)
		}
	}
	for _, mount := range linuxMounts() {
		if seen[mount.Path] {
			continue
		}
		seen[mount.Path] = true
		result = append(result, mount)
	}
	return result
}

// RemoteDirectories lists child directories on an SSH endpoint.
func RemoteDirectories(ctx context.Context, endpoint domain.Endpoint, controlPath, current string, showHidden bool) ([]Entry, error) {
	if !endpoint.IsRemote() {
		return nil, fmt.Errorf("endpoint is not remote")
	}
	if current == "" {
		current = endpoint.Path
	}
	script := "find -- " + shellQuote(current) + " -mindepth 1 -maxdepth 1 -type d -printf '%f\\0'"
	args := []string{"-o", "BatchMode=yes"}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), script)
	output, err := exec.CommandContext(ctx, "ssh", args...).Output()
	if err != nil {
		return nil, err
	}
	entries := []Entry{{Name: "..", Path: remoteParent(current)}}
	for _, raw := range bytes.Split(output, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		name := string(raw)
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		entries = append(entries, Entry{Name: name, Path: strings.TrimRight(current, "/") + "/" + name})
	}
	sort.SliceStable(entries[1:], func(i, j int) bool {
		return strings.ToLower(entries[i+1].Name) < strings.ToLower(entries[j+1].Name)
	})
	return entries, nil
}

// ValidateRemotePath checks that a remote path exists and is optionally writable.
func ValidateRemotePath(ctx context.Context, endpoint domain.Endpoint, controlPath string, writable bool) error {
	test := "test -d " + shellQuote(endpoint.Path)
	if writable {
		test = "p=" + shellQuote(endpoint.Path) + "; " +
			"if [ -d \"$p\" ]; then test -w \"$p\"; else " +
			"p=$(dirname -- \"$p\"); while [ ! -d \"$p\" ] && [ \"$p\" != / ]; do p=$(dirname -- \"$p\"); done; test -w \"$p\"; fi"
	}
	args := []string{"-o", "BatchMode=yes"}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), test)
	if output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("remote path validation failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func linuxMounts() []Entry {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return nil
	}
	var entries []Entry
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		path := strings.ReplaceAll(fields[4], `\040`, " ")
		if path == "/" || strings.HasPrefix(path, "/proc") || strings.HasPrefix(path, "/sys") || strings.HasPrefix(path, "/dev") {
			continue
		}
		entries = append(entries, Entry{Name: "Mount " + path, Path: path})
	}
	return entries
}

func remoteParent(path string) string {
	path = strings.TrimRight(path, "/")
	index := strings.LastIndex(path, "/")
	if index <= 0 {
		return "/"
	}
	return path[:index]
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

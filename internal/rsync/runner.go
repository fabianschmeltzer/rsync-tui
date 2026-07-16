package rsync

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// Event reports output or progress from an rsync run.
type Event struct {
	Time    time.Time `json:"time"`
	Stream  string    `json:"stream"`
	Message string    `json:"message"`
}

// Result summarizes a completed rsync run.
type Result struct {
	ProfileID   string      `json:"profile_id"`
	ProfileName string      `json:"profile_name"`
	Mode        domain.Mode `json:"mode,omitempty"`
	Source      string      `json:"source,omitempty"`
	Destination string      `json:"destination,omitempty"`
	AdHoc       bool        `json:"ad_hoc,omitempty"`
	StartedAt   time.Time   `json:"started_at"`
	FinishedAt  time.Time   `json:"finished_at"`
	DryRun      bool        `json:"dry_run"`
	Command     string      `json:"command"`
	ExitCode    int         `json:"exit_code"`
	Error       string      `json:"error,omitempty"`
}

// RunOptions controls execution and event handling for an rsync run.
type RunOptions struct {
	DryRun        bool
	Scheduled     bool
	Build         BuildOptions
	OnEvent       func(Event)
	SkipPreflight bool
	AdHoc         bool
}

// Runner executes validated rsync profiles.
type Runner struct {
	Store *config.Store
}

// Run executes a profile and returns its result.
func (r Runner) Run(ctx context.Context, profile domain.Profile, options RunOptions) (Result, error) {
	result := Result{
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
		Mode:        profile.Mode,
		Source:      profile.Source.Address(false),
		Destination: profile.Destination.Address(false),
		AdHoc:       options.AdHoc,
		StartedAt:   time.Now().UTC(),
		DryRun:      options.DryRun,
		ExitCode:    -1,
	}
	if !options.SkipPreflight {
		checks := Preflight(ctx, profile, options.Scheduled)
		if err := PreflightError(checks); err != nil {
			result.FinishedAt = time.Now().UTC()
			result.Error = summarizeChecks(checks)
			result.ExitCode = 65
			r.writeHistory(result)
			return result, fmt.Errorf("%w: %s", err, result.Error)
		}
	}
	lock, err := r.acquireLock(profile.ID)
	if err != nil {
		result.FinishedAt = time.Now().UTC()
		result.Error = err.Error()
		result.ExitCode = 75
		return result, err
	}
	defer lock.Release()

	options.Build.DryRun = options.DryRun
	command, err := Build(profile, options.Build)
	if err != nil {
		return result, err
	}
	if profile.UseSudo {
		command = withSudo(command, options.Scheduled)
	}
	result.Command = command.Display

	process := exec.CommandContext(ctx, command.Program, command.Args...)
	stdout, err := process.StdoutPipe()
	if err != nil {
		return result, err
	}
	stderr, err := process.StderrPipe()
	if err != nil {
		return result, err
	}
	if err := process.Start(); err != nil {
		result.FinishedAt = time.Now().UTC()
		result.Error = err.Error()
		result.ExitCode = exitCode(err)
		r.writeHistory(result)
		return result, err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go streamOutput(stdout, "stdout", options.OnEvent, &wg)
	go streamOutput(stderr, "stderr", options.OnEvent, &wg)
	waitErr := process.Wait()
	wg.Wait()

	if waitErr == nil && profile.Mode == domain.ModeMove && !options.DryRun {
		waitErr = removeEmptySourceDirectories(ctx, profile, options)
	}
	result.FinishedAt = time.Now().UTC()
	result.ExitCode = exitCode(waitErr)
	if waitErr != nil {
		result.Error = waitErr.Error()
	}
	r.writeHistory(result)
	return result, waitErr
}

func withSudo(command Command, nonInteractive bool) Command {
	args := make([]string, 0, len(command.Args)+2)
	if nonInteractive {
		args = append(args, "-n")
	}
	args = append(args, command.Program)
	args = append(args, command.Args...)
	command.Program = "sudo"
	command.Args = args
	command.Display = DisplayWithSudo(Command{Program: "rsync", Display: command.Display}, nonInteractive)
	return command
}

func streamOutput(reader io.Reader, stream string, callback func(Event), wg *sync.WaitGroup) {
	defer wg.Done()
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	scanner.Split(splitLinesAndCarriageReturns)
	for scanner.Scan() {
		if callback != nil {
			callback(Event{
				Time:    time.Now().UTC(),
				Stream:  stream,
				Message: scanner.Text(),
			})
		}
	}
}

func splitLinesAndCarriageReturns(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for index, b := range data {
		if b == '\n' || b == '\r' {
			advance := index + 1
			for advance < len(data) && (data[advance] == '\n' || data[advance] == '\r') {
				advance++
			}
			return advance, data[:index], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	if errors.Is(err, context.Canceled) {
		return 130
	}
	return 1
}

func summarizeChecks(checks []Check) string {
	var messages []string
	for _, check := range checks {
		if check.Level == CheckError {
			messages = append(messages, check.Name+": "+check.Message)
		}
	}
	return strings.Join(messages, "; ")
}

func (r Runner) writeHistory(result Result) {
	if r.Store == nil {
		return
	}
	path := filepath.Join(r.Store.Paths.StateDir, "history.jsonl")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer file.Close()
	_ = json.NewEncoder(file).Encode(result)
}

type fileLock struct {
	path string
}

func (r Runner) acquireLock(profileID string) (*fileLock, error) {
	if r.Store == nil {
		return &fileLock{}, nil
	}
	path := filepath.Join(r.Store.Paths.StateDir, "run-"+profileID+".lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("profile %s is already running", profileID)
		}
		return nil, err
	}
	fmt.Fprintf(file, "%d\n", os.Getpid())
	file.Close()
	return &fileLock{path: path}, nil
}

func (l *fileLock) Release() {
	if l.path != "" {
		_ = os.Remove(l.path)
	}
}

func removeEmptySourceDirectories(ctx context.Context, profile domain.Profile, options RunOptions) error {
	if profile.Source.IsRemote() {
		return removeRemoteEmptyDirectories(ctx, profile.Source, options.Build.SSHControlPath)
	}
	if profile.UseSudo {
		return removeEmptyDirectoriesWithSudo(ctx, profile.Source.Path, options.Scheduled)
	}
	return removeEmptyDirectories(profile.Source.Path)
}

func removeEmptyDirectories(root string) error {
	var directories []string
	if err := filepath.WalkDir(root, func(path string, item os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() && path != root {
			directories = append(directories, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("inspect source directories: %w", err)
	}
	for index := len(directories) - 1; index >= 0; index-- {
		path := directories[index]
		entries, err := os.ReadDir(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect source directory %s: %w", path, err)
		}
		if len(entries) != 0 {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove empty source directory %s: %w", path, err)
		}
	}
	return nil
}

func removeEmptyDirectoriesWithSudo(ctx context.Context, root string, nonInteractive bool) error {
	args := make([]string, 0, 10)
	if nonInteractive {
		args = append(args, "-n")
	}
	args = append(args, "find", "--", root, "-depth", "-mindepth", "1", "-type", "d", "-empty", "-delete")
	output, err := exec.CommandContext(ctx, "sudo", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove empty source directories with sudo: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func removeRemoteEmptyDirectories(ctx context.Context, source domain.Endpoint, controlPath string) error {
	args := []string{"-o", "BatchMode=yes"}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if source.Port > 0 {
		args = append(args, "-p", strconv.Itoa(source.Port))
	}
	command := "find -- " + shellQuote(source.Path) + " -depth -mindepth 1 -type d -empty -delete"
	args = append(args, source.SSHHost(), command)
	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remove empty remote source directories: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// RuntimePlatform returns the current operating system and architecture.
func RuntimePlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

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
	"strings"
	"sync"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

type Event struct {
	Time    time.Time `json:"time"`
	Stream  string    `json:"stream"`
	Message string    `json:"message"`
}

type Result struct {
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name"`
	StartedAt   time.Time `json:"started_at"`
	FinishedAt  time.Time `json:"finished_at"`
	DryRun      bool      `json:"dry_run"`
	Command     string    `json:"command"`
	ExitCode    int       `json:"exit_code"`
	Error       string    `json:"error,omitempty"`
}

type RunOptions struct {
	DryRun        bool
	Scheduled     bool
	Build         BuildOptions
	OnEvent       func(Event)
	SkipPreflight bool
}

type Runner struct {
	Store *config.Store
}

func (r Runner) Run(ctx context.Context, profile domain.Profile, options RunOptions) (Result, error) {
	result := Result{
		ProfileID:   profile.ID,
		ProfileName: profile.Name,
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

	result.FinishedAt = time.Now().UTC()
	result.ExitCode = exitCode(waitErr)
	if waitErr != nil {
		result.Error = waitErr.Error()
	}
	r.writeHistory(result)
	if waitErr == nil && profile.Mode == domain.ModeMove && profile.RemoveEmptyDirs && !profile.Source.IsRemote() && !options.DryRun {
		removeEmptyDirectories(profile.Source.Path)
	}
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

func removeEmptyDirectories(root string) {
	type entry struct {
		path  string
		depth int
	}
	var directories []entry
	_ = filepath.WalkDir(root, func(path string, item os.DirEntry, err error) error {
		if err == nil && item.IsDir() && path != root {
			directories = append(directories, entry{path: path, depth: strings.Count(filepath.Clean(path), string(filepath.Separator))})
		}
		return nil
	})
	for index := len(directories) - 1; index >= 0; index-- {
		_ = os.Remove(directories[index].path)
	}
}

func RuntimePlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

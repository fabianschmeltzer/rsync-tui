package job

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/browser"
	"github.com/fabianschmeltzer/rsync-tui/internal/config"
	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	"github.com/fabianschmeltzer/rsync-tui/internal/notify"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
	"github.com/fabianschmeltzer/rsync-tui/internal/snapshot"
	"github.com/fabianschmeltzer/rsync-tui/internal/sshclient"
)

type Options struct {
	DryRun         bool
	Scheduled      bool
	OnEvent        func(rsyncengine.Event)
	Version        string
	SSHControlPath string
}

type Outcome struct {
	Result               rsyncengine.Result
	Preview              *rsyncengine.Result
	Snapshot             *snapshot.Record
	PrunedSnapshots      []snapshot.Record
	NotificationWarnings []error
}

type Manager struct {
	Store         *config.Store
	Runner        rsyncengine.Runner
	Snapshots     snapshot.Manager
	Notifications notify.Sender
}

func New(store *config.Store) Manager {
	return Manager{
		Store:     store,
		Runner:    rsyncengine.Runner{Store: store},
		Snapshots: snapshot.Manager{},
	}
}

func (m Manager) Execute(ctx context.Context, profile domain.Profile, options Options) (Outcome, error) {
	var outcome Outcome
	if endpoint, remote := sshclient.RemoteEndpoint(profile); remote {
		if err := sshclient.BatchCheck(ctx, endpoint, options.SSHControlPath); err != nil {
			return outcome, err
		}
		if err := sshclient.CheckRsync(ctx, endpoint, options.SSHControlPath); err != nil {
			return outcome, err
		}
		if profile.Source.IsRemote() {
			if err := browser.ValidateRemotePath(ctx, profile.Source, options.SSHControlPath, false); err != nil {
				return outcome, err
			}
		}
		if profile.Destination.IsRemote() {
			if err := browser.ValidateRemotePath(ctx, profile.Destination, options.SSHControlPath, true); err != nil {
				return outcome, err
			}
		}
	}
	var deleteCount atomic.Int64
	var changeCount atomic.Int64
	onEvent := func(event rsyncengine.Event) {
		message := strings.TrimSpace(event.Message)
		if strings.HasPrefix(message, "*deleting") {
			deleteCount.Add(1)
		}
		if isChangedLine(message) {
			changeCount.Add(1)
		}
		if options.OnEvent != nil {
			options.OnEvent(event)
		}
	}

	build := rsyncengine.BuildOptions{SSHControlPath: options.SSHControlPath}
	var prepared snapshot.Prepared
	if profile.Mode == domain.ModeSnapshot {
		var err error
		if profile.Destination.IsRemote() {
			if options.DryRun {
				prepared, err = m.Snapshots.PlanRemote(ctx, profile, options.SSHControlPath, time.Now())
			} else {
				prepared, err = m.Snapshots.PrepareRemote(ctx, profile, options.SSHControlPath, time.Now())
			}
		} else {
			if options.DryRun {
				prepared, err = m.Snapshots.Plan(profile, time.Now())
			} else {
				prepared, err = m.Snapshots.Prepare(profile, time.Now())
			}
		}
		if err != nil {
			return outcome, err
		}
		build.DestinationOverride = prepared.PartialPath
		build.LinkDestination = prepared.LinkDestination
	}

	var manifest string
	if options.Scheduled && profile.Mode == domain.ModeMove {
		if profile.Source.IsRemote() {
			return outcome, errors.New("scheduled remote Move requires an interactive review")
		}
		var err error
		manifest, err = freezeLocalSource(m.Store.Paths.StateDir, profile)
		if err != nil {
			return outcome, err
		}
		defer os.Remove(manifest)
		build.FrozenFileList = manifest
	}

	if options.Scheduled && profile.Destructive() {
		preview, err := m.Runner.Run(ctx, profile, rsyncengine.RunOptions{
			DryRun:    true,
			Scheduled: true,
			Build:     build,
			OnEvent:   onEvent,
		})
		outcome.Preview = &preview
		if err != nil {
			outcome.Result = preview
			m.sendNotifications(profile, options, &outcome)
			return outcome, fmt.Errorf("mandatory destructive preview failed: %w", err)
		}
		if profile.Mode == domain.ModeMirror && deleteCount.Load() > int64(profile.Safety.MaxDelete) {
			return outcome, fmt.Errorf("preview contains %d deletions, above max_delete %d", deleteCount.Load(), profile.Safety.MaxDelete)
		}
		if profile.Mode == domain.ModeMove && changeCount.Load() > int64(profile.Safety.MaxSourceRemovals) {
			return outcome, fmt.Errorf("preview contains %d source removals, above max_source_removals %d", changeCount.Load(), profile.Safety.MaxSourceRemovals)
		}
		deleteCount.Store(0)
		changeCount.Store(0)
	}

	result, runErr := m.Runner.Run(ctx, profile, rsyncengine.RunOptions{
		DryRun:    options.DryRun,
		Scheduled: options.Scheduled,
		Build:     build,
		OnEvent:   onEvent,
	})
	outcome.Result = result

	if profile.Mode == domain.ModeSnapshot && !options.DryRun {
		var record snapshot.Record
		var finalizeErr error
		if profile.Destination.IsRemote() {
			record, finalizeErr = m.Snapshots.FinalizeRemote(ctx, profile.Destination, options.SSHControlPath, prepared, runErr == nil)
		} else {
			record, finalizeErr = m.Snapshots.Finalize(prepared, runErr == nil)
		}
		outcome.Snapshot = &record
		if runErr == nil && finalizeErr == nil && profile.Snapshot.VerifyAfterRun {
			finalizeErr = m.verifySnapshot(ctx, profile, options, prepared.FinalPath)
		}
		if runErr == nil && finalizeErr == nil {
			if profile.Destination.IsRemote() {
				outcome.PrunedSnapshots, finalizeErr = m.Snapshots.PruneRemote(ctx, profile, options.SSHControlPath)
			} else {
				outcome.PrunedSnapshots, finalizeErr = m.Snapshots.Prune(profile)
			}
		}
		if finalizeErr != nil {
			if runErr == nil {
				runErr = finalizeErr
			} else {
				runErr = errors.Join(runErr, finalizeErr)
			}
		}
	}
	m.sendNotifications(profile, options, &outcome)
	return outcome, runErr
}

func (m Manager) verifySnapshot(ctx context.Context, profile domain.Profile, options Options, snapshotPath string) error {
	verify := profile
	verify.Mode = domain.ModeCustom
	verify.Options = []string{"--archive", "--checksum", "--delete"}
	verify.Filters.Exclude = append(append([]string(nil), profile.Filters.Exclude...), "/.rsync-tui.json")
	var changed atomic.Int64
	result, err := m.Runner.Run(ctx, verify, rsyncengine.RunOptions{
		DryRun:    true,
		Scheduled: options.Scheduled,
		Build: rsyncengine.BuildOptions{
			DestinationOverride: snapshotPath,
			SSHControlPath:      options.SSHControlPath,
		},
		OnEvent: func(event rsyncengine.Event) {
			if isChangedLine(strings.TrimSpace(event.Message)) || strings.HasPrefix(strings.TrimSpace(event.Message), "*deleting") {
				changed.Add(1)
			}
			if options.OnEvent != nil {
				options.OnEvent(event)
			}
		},
	})
	if err != nil {
		return fmt.Errorf("snapshot verification failed (exit %d): %w", result.ExitCode, err)
	}
	if changed.Load() > 0 {
		return fmt.Errorf("snapshot verification found %d difference(s)", changed.Load())
	}
	return nil
}

func (m Manager) sendNotifications(profile domain.Profile, options Options, outcome *Outcome) {
	status := "success"
	if outcome.Result.ExitCode != 0 {
		status = "failure"
	}
	event := notify.Event{
		Event:           "transfer.completed",
		Status:          status,
		ProfileID:       profile.ID,
		ProfileName:     profile.Name,
		StartedAt:       outcome.Result.StartedAt,
		FinishedAt:      outcome.Result.FinishedAt,
		DurationSeconds: outcome.Result.FinishedAt.Sub(outcome.Result.StartedAt).Seconds(),
		ExitCode:        outcome.Result.ExitCode,
		Message:         outcome.Result.Error,
		Version:         options.Version,
	}
	outcome.NotificationWarnings = m.Notifications.Send(context.Background(), profile.Notifications, event)
}

func isChangedLine(line string) bool {
	if line == "" || strings.HasPrefix(line, "*deleting") {
		return false
	}
	if len(line) < 11 {
		return false
	}
	first := line[0]
	return first == '<' || first == '>' || first == 'c' || first == 'h' || first == '.'
}

func freezeLocalSource(stateDir string, profile domain.Profile) (string, error) {
	file, err := os.CreateTemp(stateDir, "move-"+profile.ID+"-*.files")
	if err != nil {
		return "", err
	}
	name := file.Name()
	ok := false
	defer func() {
		file.Close()
		if !ok {
			os.Remove(name)
		}
	}()
	root := filepath.Clean(profile.Source.Path)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		if entry.IsDir() {
			relative += "/"
		}
		if _, err := file.WriteString(relative); err != nil {
			return err
		}
		if _, err := file.Write([]byte{0}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if err := file.Sync(); err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(name, 0o600); err != nil {
		return "", err
	}
	ok = true
	return name, nil
}

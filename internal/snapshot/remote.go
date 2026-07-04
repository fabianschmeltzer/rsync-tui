package snapshot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

var snapshotIDPattern = regexp.MustCompile(`^\d{8}T\d{6}Z(?:-\d+)?$`)

func (Manager) PrepareRemote(ctx context.Context, profile domain.Profile, controlPath string, now time.Time) (Prepared, error) {
	if !profile.Destination.IsRemote() {
		return Prepared{}, errors.New("remote snapshot preparation requires an SSH destination")
	}
	prepared, err := planRemote(ctx, profile, controlPath, now)
	if err != nil {
		return Prepared{}, err
	}
	if _, err := remoteCommand(ctx, profile.Destination, controlPath, "mkdir -p -- "+shellQuoteRemote(prepared.PartialPath)); err != nil {
		return Prepared{}, err
	}
	return prepared, nil
}

func (Manager) PlanRemote(ctx context.Context, profile domain.Profile, controlPath string, now time.Time) (Prepared, error) {
	if !profile.Destination.IsRemote() {
		return Prepared{}, errors.New("remote snapshot planning requires an SSH destination")
	}
	return planRemote(ctx, profile, controlPath, now)
}

func planRemote(ctx context.Context, profile domain.Profile, controlPath string, now time.Time) (Prepared, error) {
	root := strings.TrimSpace(profile.Snapshot.Root)
	if root == "" {
		root = profile.Destination.Path
	}
	base := path.Join(root, ".rsync-tui", profile.ID)
	snapshotsDir := path.Join(base, "snapshots")
	id := now.UTC().Format("20060102T150405Z")
	finalPath := path.Join(snapshotsDir, id)
	partialPath := finalPath + ".partial"
	script := "if [ -L " + shellQuoteRemote(path.Join(base, "latest")) + " ]; then readlink -f -- " + shellQuoteRemote(path.Join(base, "latest")) + "; fi"
	output, err := remoteCommand(ctx, profile.Destination, controlPath, script)
	if err != nil {
		return Prepared{}, err
	}
	linkDestination := strings.TrimSpace(output)
	return Prepared{
		Record: Record{
			ID:          id,
			ProfileID:   profile.ID,
			ProfileName: profile.Name,
			Path:        finalPath,
			CreatedAt:   now.UTC(),
		},
		PartialPath:     partialPath,
		FinalPath:       finalPath,
		LinkDestination: linkDestination,
	}, nil
}

func (Manager) FinalizeRemote(ctx context.Context, endpoint domain.Endpoint, controlPath string, prepared Prepared, success bool) (Record, error) {
	record := prepared.Record
	record.Successful = success
	metadata, err := json.Marshal(record)
	if err != nil {
		return record, err
	}
	directory := prepared.PartialPath
	var script string
	if success {
		directory = prepared.FinalPath
		base := path.Dir(path.Dir(prepared.FinalPath))
		relative := path.Join("snapshots", prepared.Record.ID)
		script = "mv -- " + shellQuoteRemote(prepared.PartialPath) + " " + shellQuoteRemote(prepared.FinalPath) + "; " +
			"printf '%s\\n' " + shellQuoteRemote(string(metadata)) + " > " + shellQuoteRemote(path.Join(directory, ".rsync-tui.json")) + "; " +
			"ln -sfn -- " + shellQuoteRemote(relative) + " " + shellQuoteRemote(path.Join(base, "latest.new")) + "; " +
			"mv -Tf -- " + shellQuoteRemote(path.Join(base, "latest.new")) + " " + shellQuoteRemote(path.Join(base, "latest"))
	} else {
		script = "printf '%s\\n' " + shellQuoteRemote(string(metadata)) + " > " + shellQuoteRemote(path.Join(directory, ".rsync-tui.json"))
	}
	_, err = remoteCommand(ctx, endpoint, controlPath, script)
	return record, err
}

func (Manager) ListRemote(ctx context.Context, profile domain.Profile, controlPath string) ([]Record, error) {
	root := profile.Snapshot.Root
	if root == "" {
		root = profile.Destination.Path
	}
	snapshotsDir := path.Join(root, ".rsync-tui", profile.ID, "snapshots")
	script := "if [ -d " + shellQuoteRemote(snapshotsDir) + " ]; then find -- " + shellQuoteRemote(snapshotsDir) +
		" -mindepth 1 -maxdepth 1 -type d -printf '%f\\n'; fi"
	output, err := remoteCommand(ctx, profile.Destination, controlPath, script)
	if err != nil {
		return nil, err
	}
	var records []Record
	for _, name := range strings.Split(strings.TrimSpace(output), "\n") {
		if !snapshotIDPattern.MatchString(name) || strings.HasSuffix(name, ".partial") {
			continue
		}
		timestamp := strings.Split(name, "-")[0]
		created, err := time.Parse("20060102T150405Z", timestamp)
		if err != nil {
			continue
		}
		records = append(records, Record{
			ID:          name,
			ProfileID:   profile.ID,
			ProfileName: profile.Name,
			Path:        path.Join(snapshotsDir, name),
			CreatedAt:   created,
			Successful:  true,
		})
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	return records, nil
}

func (manager Manager) PruneRemote(ctx context.Context, profile domain.Profile, controlPath string) ([]Record, error) {
	records, err := manager.ListRemote(ctx, profile, controlPath)
	if err != nil {
		return nil, err
	}
	keep := SelectKeep(records, profile.Snapshot.Retention)
	var removed []Record
	for _, record := range records {
		if keep[record.ID] {
			continue
		}
		if !snapshotIDPattern.MatchString(record.ID) || path.Base(record.Path) != record.ID {
			return removed, fmt.Errorf("refusing to prune unmanaged remote snapshot %s", record.Path)
		}
		if _, err := remoteCommand(ctx, profile.Destination, controlPath, "rm -rf -- "+shellQuoteRemote(record.Path)); err != nil {
			return removed, err
		}
		removed = append(removed, record)
	}
	return removed, nil
}

func remoteCommand(ctx context.Context, endpoint domain.Endpoint, controlPath, script string) (string, error) {
	args := []string{"-o", "BatchMode=yes"}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), script)
	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("remote snapshot command failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func shellQuoteRemote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

package snapshot

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// Record describes a completed snapshot stored on an endpoint.
type Record struct {
	ID          string    `json:"id"`
	ProfileID   string    `json:"profile_id"`
	ProfileName string    `json:"profile_name"`
	Path        string    `json:"path"`
	CreatedAt   time.Time `json:"created_at"`
	Successful  bool      `json:"successful"`
}

// Prepared contains the paths and metadata for an in-progress snapshot.
type Prepared struct {
	Record          Record
	PartialPath     string
	FinalPath       string
	LinkDestination string
}

// Manager creates, lists, and prunes snapshots.
type Manager struct{}

// Prepare creates the local staging area for a snapshot.
func (Manager) Prepare(profile domain.Profile, now time.Time) (Prepared, error) {
	if profile.Destination.IsRemote() {
		return Prepared{}, errors.New("remote snapshot repositories require the SSH snapshot coordinator")
	}
	prepared, err := planLocal(profile, now)
	if err != nil {
		return Prepared{}, err
	}
	if err := os.MkdirAll(prepared.PartialPath, 0o700); err != nil {
		return Prepared{}, err
	}
	return prepared, nil
}

// Plan computes local snapshot paths without creating them.
func (Manager) Plan(profile domain.Profile, now time.Time) (Prepared, error) {
	if profile.Destination.IsRemote() {
		return Prepared{}, errors.New("remote snapshot repositories require the SSH snapshot coordinator")
	}
	return planLocal(profile, now)
}

func planLocal(profile domain.Profile, now time.Time) (Prepared, error) {
	root := strings.TrimSpace(profile.Snapshot.Root)
	if root == "" {
		root = profile.Destination.Path
	}
	if root == "" {
		return Prepared{}, errors.New("snapshot root is empty")
	}
	base := filepath.Join(root, ".rsync-tui", profile.ID)
	snapshotsDir := filepath.Join(base, "snapshots")
	id := now.UTC().Format("20060102T150405Z")
	finalPath := filepath.Join(snapshotsDir, id)
	partialPath := finalPath + ".partial"
	if _, err := os.Stat(finalPath); err == nil {
		id += fmt.Sprintf("-%d", now.UnixNano()%1_000_000)
		finalPath = filepath.Join(snapshotsDir, id)
		partialPath = finalPath + ".partial"
	}
	latestPath := filepath.Join(base, "latest")
	linkDestination := ""
	if target, err := os.Readlink(latestPath); err == nil {
		if !filepath.IsAbs(target) {
			target = filepath.Join(base, target)
		}
		if info, statErr := os.Stat(target); statErr == nil && info.IsDir() {
			linkDestination = target
		}
	}
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

// Finalize commits or removes a prepared local snapshot based on run success.
func (Manager) Finalize(prepared Prepared, success bool) (Record, error) {
	record := prepared.Record
	record.Successful = success
	if !success {
		if err := writeMetadata(prepared.PartialPath, record); err != nil {
			return record, err
		}
		return record, nil
	}
	if err := os.Rename(prepared.PartialPath, prepared.FinalPath); err != nil {
		return record, err
	}
	if err := writeMetadata(prepared.FinalPath, record); err != nil {
		return record, err
	}
	base := filepath.Dir(filepath.Dir(prepared.FinalPath))
	latest := filepath.Join(base, "latest")
	temporary := latest + ".new"
	_ = os.Remove(temporary)
	relative, err := filepath.Rel(base, prepared.FinalPath)
	if err != nil {
		return record, err
	}
	if err := os.Symlink(relative, temporary); err != nil {
		return record, err
	}
	_ = os.Remove(latest)
	if err := os.Rename(temporary, latest); err != nil {
		return record, err
	}
	return record, nil
}

// List returns snapshots stored for a local profile.
func (Manager) List(profile domain.Profile) ([]Record, error) {
	root := profile.Snapshot.Root
	if root == "" {
		root = profile.Destination.Path
	}
	snapshotsDir := filepath.Join(root, ".rsync-tui", profile.ID, "snapshots")
	entries, err := os.ReadDir(snapshotsDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var records []Record
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasSuffix(entry.Name(), ".partial") {
			continue
		}
		var record Record
		expectedPath := filepath.Join(snapshotsDir, entry.Name())
		data, readErr := os.ReadFile(filepath.Join(expectedPath, ".rsync-tui.json"))
		if readErr != nil || json.Unmarshal(data, &record) != nil || !record.Successful {
			continue
		}
		if record.ID != entry.Name() || record.ProfileID != profile.ID {
			continue
		}
		record.Path = expectedPath
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool {
		return records[i].CreatedAt.After(records[j].CreatedAt)
	})
	return records, nil
}

// SelectKeep returns the snapshot paths retained by a policy.
func SelectKeep(records []Record, retention domain.Retention) map[string]bool {
	keep := make(map[string]bool)
	if len(records) == 0 {
		return keep
	}
	sorted := append([]Record(nil), records...)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})
	keep[sorted[0].ID] = true
	if retention.Mode == domain.RetentionLastN {
		count := retention.LastN
		if count < 1 {
			count = 1
		}
		for index := 0; index < len(sorted) && index < count; index++ {
			keep[sorted[index].ID] = true
		}
		return keep
	}
	selectBuckets(sorted, keep, retention.Daily, func(t time.Time) string {
		return t.Format("2006-01-02")
	})
	selectBuckets(sorted, keep, retention.Weekly, func(t time.Time) string {
		year, week := t.ISOWeek()
		return fmt.Sprintf("%04d-W%02d", year, week)
	})
	selectBuckets(sorted, keep, retention.Monthly, func(t time.Time) string {
		return t.Format("2006-01")
	})
	return keep
}

// Prune removes local snapshots not selected by the retention policy.
func (manager Manager) Prune(profile domain.Profile) ([]Record, error) {
	records, err := manager.List(profile)
	if err != nil {
		return nil, err
	}
	keep := SelectKeep(records, profile.Snapshot.Retention)
	root := profile.Snapshot.Root
	if root == "" {
		root = profile.Destination.Path
	}
	snapshotsDir := filepath.Clean(filepath.Join(root, ".rsync-tui", profile.ID, "snapshots"))
	var removed []Record
	for _, record := range records {
		if keep[record.ID] {
			continue
		}
		if filepath.Base(record.Path) != record.ID || record.ProfileID != profile.ID || filepath.Dir(filepath.Clean(record.Path)) != snapshotsDir {
			return removed, fmt.Errorf("refusing to prune unmanaged snapshot %s", record.Path)
		}
		if err := os.RemoveAll(record.Path); err != nil {
			return removed, err
		}
		removed = append(removed, record)
	}
	return removed, nil
}

func selectBuckets(records []Record, keep map[string]bool, limit int, key func(time.Time) string) {
	if limit < 1 {
		return
	}
	seen := make(map[string]bool)
	count := 0
	for _, record := range records {
		bucket := key(record.CreatedAt)
		if seen[bucket] {
			continue
		}
		seen[bucket] = true
		keep[record.ID] = true
		count++
		if count >= limit {
			return
		}
	}
}

func writeMetadata(directory string, record Record) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filepath.Join(directory, ".rsync-tui.json"), data, 0o600)
}

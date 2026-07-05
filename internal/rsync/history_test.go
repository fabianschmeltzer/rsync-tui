package rsync

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadHistoryReturnsNewestEntriesAndSkipsMalformedLines(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "history.jsonl")
	first := Result{ProfileName: "first", StartedAt: time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)}
	second := Result{ProfileName: "second", StartedAt: time.Date(2026, 7, 5, 11, 0, 0, 0, time.UTC)}
	third := Result{ProfileName: "third", StartedAt: time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)}
	var data []byte
	for _, entry := range []Result{first, second} {
		encoded, err := json.Marshal(entry)
		if err != nil {
			t.Fatal(err)
		}
		data = append(data, encoded...)
		data = append(data, '\n')
	}
	data = append(data, []byte("{not-json}\n")...)
	encoded, err := json.Marshal(third)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, encoded...)
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	history, err := LoadHistory(directory, 2)
	if err != nil {
		t.Fatal(err)
	}
	if history.Skipped != 1 {
		t.Fatalf("skipped = %d, want 1", history.Skipped)
	}
	if len(history.Entries) != 2 ||
		history.Entries[0].ProfileName != "third" ||
		history.Entries[1].ProfileName != "second" {
		t.Fatalf("unexpected history: %+v", history.Entries)
	}
}

func TestLoadHistoryAcceptsMissingFile(t *testing.T) {
	history, err := LoadHistory(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Entries) != 0 || history.Skipped != 0 {
		t.Fatalf("unexpected history: %+v", history)
	}
}

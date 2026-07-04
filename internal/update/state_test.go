package update

import (
	"testing"
	"time"
)

func TestUpdateCheckInterval(t *testing.T) {
	directory := t.TempDir()
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	if !Due(directory, 24*time.Hour, now) {
		t.Fatal("missing update state should be due")
	}
	if err := MarkChecked(directory, now); err != nil {
		t.Fatal(err)
	}
	if Due(directory, 24*time.Hour, now.Add(time.Hour)) {
		t.Fatal("recent update check should not be due")
	}
	if !Due(directory, 24*time.Hour, now.Add(25*time.Hour)) {
		t.Fatal("old update check should be due")
	}
}

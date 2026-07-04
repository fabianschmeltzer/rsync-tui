package snapshot

import (
	"fmt"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestLastNAlwaysKeepsOne(t *testing.T) {
	records := records(5)
	keep := SelectKeep(records, domain.Retention{Mode: domain.RetentionLastN, LastN: 0})
	if len(keep) != 1 || !keep[records[0].ID] {
		t.Fatalf("expected newest snapshot only, got %#v", keep)
	}
}

func TestGFSKeepsBuckets(t *testing.T) {
	records := records(40)
	keep := SelectKeep(records, domain.Retention{Mode: domain.RetentionGFS, Daily: 2, Weekly: 2, Monthly: 2})
	if len(keep) < 2 {
		t.Fatalf("GFS retained too few snapshots: %d", len(keep))
	}
	if !keep[records[0].ID] {
		t.Fatal("GFS did not retain newest snapshot")
	}
}

func records(count int) []Record {
	now := time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC)
	result := make([]Record, 0, count)
	for index := 0; index < count; index++ {
		result = append(result, Record{
			ID:         fmt.Sprintf("%02d", index),
			CreatedAt:  now.AddDate(0, 0, -index),
			Successful: true,
		})
	}
	return result
}

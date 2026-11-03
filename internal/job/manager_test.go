package job

import (
	"errors"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
	rsyncengine "github.com/fabianschmeltzer/rsync-tui/internal/rsync"
)

func TestNotificationEventReportsLateStageFailure(t *testing.T) {
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := started.Add(2 * time.Second)
	outcome := Outcome{Result: rsyncengine.Result{StartedAt: started, FinishedAt: finished}}

	event := notificationEvent(domain.Profile{ID: "p", Name: "n"}, Options{Version: "v1"}, outcome,
		errors.New("snapshot verification found 3 difference(s)"), started, finished)

	if event.Status != "failure" {
		t.Fatalf("status = %q, want failure when rsync succeeded but a later stage failed", event.Status)
	}
	if event.Message != "snapshot verification found 3 difference(s)" {
		t.Fatalf("message = %q", event.Message)
	}
	if event.DurationSeconds != 2 {
		t.Fatalf("duration = %v, want 2", event.DurationSeconds)
	}
}

func TestNotificationEventSuccessAndRsyncFailure(t *testing.T) {
	now := time.Now()
	ok := notificationEvent(domain.Profile{}, Options{}, Outcome{}, nil, now, now)
	if ok.Status != "success" || ok.Message != "" {
		t.Fatalf("unexpected success event: %+v", ok)
	}
	failed := notificationEvent(domain.Profile{}, Options{},
		Outcome{Result: rsyncengine.Result{ExitCode: 23, Error: "partial transfer"}}, errors.New("exit status 23"), now, now)
	if failed.Status != "failure" || failed.Message != "partial transfer" || failed.ExitCode != 23 {
		t.Fatalf("unexpected failure event: %+v", failed)
	}
}

func TestNotificationEventFillsTimesForPreRunFailure(t *testing.T) {
	started := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	finished := started.Add(time.Second)
	event := notificationEvent(domain.Profile{}, Options{Scheduled: true}, Outcome{},
		errors.New("scheduled remote Move requires an interactive review"), started, finished)
	if event.Status != "failure" || !event.StartedAt.Equal(started) || !event.FinishedAt.Equal(finished) {
		t.Fatalf("unexpected pre-run failure event: %+v", event)
	}
}

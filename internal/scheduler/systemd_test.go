package scheduler

import (
	"strings"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestRenderSystemdUnits(t *testing.T) {
	profile := domain.NewProfile("Nightly % backup")
	profile.Source.Path = "/source"
	profile.Destination.Path = "/destination"
	profile.Schedule = domain.Schedule{Enabled: true, OnCalendar: "daily"}
	units, err := Render(profile, "/home/user/.local/bin/rsync-tui")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(units.Service, "run --profile "+profile.ID+" --scheduled") {
		t.Fatalf("service does not run the profile: %s", units.Service)
	}
	if !strings.Contains(units.Timer, "Persistent=true") {
		t.Fatal("timer is not persistent")
	}
	if !strings.Contains(units.Service, "Nightly %% backup") {
		t.Fatal("systemd percent was not escaped")
	}
}

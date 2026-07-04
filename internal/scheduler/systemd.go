package scheduler

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

type Units struct {
	ServiceName string
	TimerName   string
	Service     string
	Timer       string
}

func Render(profile domain.Profile, executable string) (Units, error) {
	if !profile.Schedule.Enabled {
		return Units{}, errors.New("profile schedule is disabled")
	}
	if strings.TrimSpace(profile.Schedule.OnCalendar) == "" {
		return Units{}, errors.New("OnCalendar is empty")
	}
	if strings.ContainsAny(profile.Schedule.OnCalendar, "\r\n") {
		return Units{}, errors.New("OnCalendar contains a newline")
	}
	name := "rsync-tui-" + profile.ID
	service := fmt.Sprintf(`[Unit]
Description=rsync-tui profile %s
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
ExecStart=%s run --profile %s --scheduled
Nice=10
IOSchedulingClass=best-effort
IOSchedulingPriority=7
`, unitEscape(profile.Name), execEscape(executable), execEscape(profile.ID))
	timer := fmt.Sprintf(`[Unit]
Description=Schedule rsync-tui profile %s

[Timer]
OnCalendar=%s
Persistent=true
RandomizedDelaySec=5m
Unit=%s.service

[Install]
WantedBy=timers.target
`, unitEscape(profile.Name), profile.Schedule.OnCalendar, name)
	return Units{
		ServiceName: name + ".service",
		TimerName:   name + ".timer",
		Service:     service,
		Timer:       timer,
	}, nil
}

func Install(profile domain.Profile, executable string) (Units, error) {
	units, err := Render(profile, executable)
	if err != nil {
		return Units{}, err
	}
	directory, systemctlArgs, err := unitDirectory(profile.Schedule.System)
	if err != nil {
		return Units{}, err
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return Units{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, units.ServiceName), []byte(units.Service), 0o644); err != nil {
		return Units{}, err
	}
	if err := os.WriteFile(filepath.Join(directory, units.TimerName), []byte(units.Timer), 0o644); err != nil {
		return Units{}, err
	}
	if err := exec.Command("systemctl", append(systemctlArgs, "daemon-reload")...).Run(); err != nil {
		return Units{}, err
	}
	if err := exec.Command("systemctl", append(systemctlArgs, "enable", "--now", units.TimerName)...).Run(); err != nil {
		return Units{}, err
	}
	return units, nil
}

func unitDirectory(system bool) (string, []string, error) {
	if system {
		if !isRoot() {
			return "", nil, errors.New("system timers must be installed while running as root")
		}
		return "/etc/systemd/system", nil, nil
	}
	configHome, err := os.UserConfigDir()
	if err != nil {
		return "", nil, err
	}
	return filepath.Join(configHome, "systemd", "user"), []string{"--user"}, nil
}

func unitEscape(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")
	return value
}

func execEscape(value string) string {
	value = strings.ReplaceAll(value, "%", "%%")
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `"`, `\"`)
	if strings.ContainsAny(value, " \t") {
		return `"` + value + `"`
	}
	return value
}

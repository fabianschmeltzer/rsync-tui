package update

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// State records when the application last checked for updates.
type State struct {
	LastCheck time.Time `json:"last_check"`
}

// Due reports whether the configured update-check interval has elapsed.
func Due(stateDirectory string, interval time.Duration, now time.Time) bool {
	state, err := LoadState(stateDirectory)
	if err != nil {
		return true
	}
	return state.LastCheck.IsZero() || now.Sub(state.LastCheck) >= interval
}

// LoadState reads update state or returns an empty state when absent.
func LoadState(stateDirectory string) (State, error) {
	data, err := os.ReadFile(filepath.Join(stateDirectory, "update-state.json"))
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

// MarkChecked records a successful update check.
func MarkChecked(stateDirectory string, now time.Time) error {
	if stateDirectory == "" {
		return errors.New("state directory is empty")
	}
	if err := os.MkdirAll(stateDirectory, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(State{LastCheck: now.UTC()})
	if err != nil {
		return err
	}
	data = append(data, '\n')
	path := filepath.Join(stateDirectory, "update-state.json")
	temp, err := os.CreateTemp(stateDirectory, ".update-state-*.tmp")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer os.Remove(name)
	if err := temp.Chmod(0o600); err != nil {
		temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

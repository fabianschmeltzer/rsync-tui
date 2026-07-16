package sshclient

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

// RemoteEndpoint returns the remote endpoint in a profile, if one exists.
func RemoteEndpoint(profile domain.Profile) (domain.Endpoint, bool) {
	if profile.Source.IsRemote() {
		return profile.Source, true
	}
	if profile.Destination.IsRemote() {
		return profile.Destination, true
	}
	return domain.Endpoint{}, false
}

// ControlPath returns a stable local control-socket path for an SSH endpoint.
func ControlPath(stateDir string, endpoint domain.Endpoint) (string, error) {
	if !endpoint.IsRemote() {
		return "", errors.New("endpoint is not remote")
	}
	directory := filepath.Join(stateDir, "ssh")
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(endpoint.SSHHost() + ":" + strconv.Itoa(endpoint.Port)))
	return filepath.Join(directory, "cm-"+hex.EncodeToString(sum[:8])), nil
}

// MasterCommand constructs the persistent SSH control-master command.
func MasterCommand(endpoint domain.Endpoint, controlPath string) *exec.Cmd {
	args := []string{
		"-o", "ControlMaster=auto",
		"-o", "ControlPersist=10m",
		"-o", "ControlPath=" + controlPath,
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), "true")
	return exec.Command("ssh", args...)
}

// BatchCheck verifies non-interactive SSH connectivity through a control socket.
func BatchCheck(ctx context.Context, endpoint domain.Endpoint, controlPath string) error {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "ConnectTimeout=10",
	}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), "true")
	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("SSH connection failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// CheckRsync verifies that rsync is available on the remote endpoint.
func CheckRsync(ctx context.Context, endpoint domain.Endpoint, controlPath string) error {
	args := []string{"-o", "BatchMode=yes"}
	if controlPath != "" {
		args = append(args, "-o", "ControlPath="+controlPath)
	}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost(), "rsync --version")
	output, err := exec.CommandContext(ctx, "ssh", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("remote rsync check failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if !strings.Contains(strings.ToLower(string(output)), "rsync") {
		return errors.New("remote rsync check returned an unexpected response")
	}
	return nil
}

// Close asks SSH to terminate the control-master connection.
func Close(endpoint domain.Endpoint, controlPath string) {
	if controlPath == "" {
		return
	}
	args := []string{"-o", "ControlPath=" + controlPath, "-O", "exit"}
	if endpoint.Port > 0 {
		args = append(args, "-p", strconv.Itoa(endpoint.Port))
	}
	args = append(args, endpoint.SSHHost())
	_ = exec.Command("ssh", args...).Run()
}

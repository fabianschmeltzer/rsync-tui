//go:build ssh_integration

package browser

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestRemoteDirectoriesAgainstOpenSSH(t *testing.T) {
	target := os.Getenv("SSH_TEST_HOST")
	if target == "" {
		t.Skip("SSH_TEST_HOST is not configured")
	}
	user, host, found := strings.Cut(target, "@")
	if !found {
		host, user = user, ""
	}
	endpoint := domain.Endpoint{Kind: domain.EndpointSSH, User: user, Host: host, Path: "/tmp"}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	entries, err := RemoteDirectories(ctx, endpoint, "", "/tmp", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 || entries[0].Name != ".." {
		t.Fatalf("unexpected remote entries: %+v", entries)
	}
	if err := ValidateRemotePath(ctx, endpoint, "", false); err != nil {
		t.Fatal(err)
	}
}

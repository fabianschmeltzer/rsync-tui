package rsync

import (
	"strings"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func testProfile() domain.Profile {
	profile := domain.NewProfile("test")
	profile.Source.Path = "/source with space"
	profile.Destination.Path = "/destination"
	return profile
}

func TestBuildUsesArgumentBoundaries(t *testing.T) {
	profile := testProfile()
	profile.Mode = domain.ModeMirror
	profile.Safety.MaxDelete = 25
	profile.Filters.Exclude = []string{"*.tmp"}
	command, err := Build(profile, BuildOptions{DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, command.Args, "--delete-delay")
	assertContains(t, command.Args, "--max-delete=25")
	assertContains(t, command.Args, "--exclude=*.tmp")
	assertContains(t, command.Args, "/source with space/")
	if !strings.Contains(command.Display, "'/source with space/'") {
		t.Fatalf("display is not shell-escaped: %s", command.Display)
	}
}

func TestBuildSSH(t *testing.T) {
	profile := testProfile()
	profile.Destination = domain.Endpoint{Kind: domain.EndpointSSH, User: "backup", Host: "nas", Port: 2222, Path: "/archive"}
	command, err := Build(profile, BuildOptions{SSHControlPath: "/run/user/1000/rsync socket"})
	if err != nil {
		t.Fatal(err)
	}
	assertContains(t, command.Args, "--secluded-args")
	assertContains(t, command.Args, "--rsh=ssh -o ControlPath='/run/user/1000/rsync socket' -p 2222")
	assertContains(t, command.Args, "backup@nas:/archive")
}

func TestBuildRejectsInternalOptions(t *testing.T) {
	profile := testProfile()
	profile.Options = []string{"--server"}
	if _, err := Build(profile, BuildOptions{}); err == nil {
		t.Fatal("internal rsync option was accepted")
	}
	profile.Options = []string{"/another/source"}
	if _, err := Build(profile, BuildOptions{}); err == nil {
		t.Fatal("positional expert argument was accepted")
	}
}

func TestBuildRejectsConflictingOptions(t *testing.T) {
	profile := testProfile()
	profile.Options = []string{"--whole-file", "--no-whole-file"}
	if _, err := Build(profile, BuildOptions{}); err == nil {
		t.Fatal("conflicting rsync options were accepted")
	}
}

func TestVersionParser(t *testing.T) {
	version, ok := parseRsyncVersion("rsync  version 3.2.7  protocol version 31")
	if !ok || version != [3]int{3, 2, 7} {
		t.Fatalf("unexpected parse result: %v %t", version, ok)
	}
	if !versionLess([3]int{3, 0, 9}, [3]int{3, 1, 0}) {
		t.Fatal("version comparison failed")
	}
}

func assertContains(t *testing.T, values []string, expected string) {
	t.Helper()
	for _, value := range values {
		if value == expected {
			return
		}
	}
	t.Fatalf("%q not found in %#v", expected, values)
}

package rsync

import (
	"path/filepath"
	"testing"

	"github.com/fabianschmeltzer/rsync-tui/internal/domain"
)

func TestOverlapCheck(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "source")
	destination := filepath.Join(source, "nested")
	profile := domain.NewProfile("overlap")
	profile.Source.Path = source
	profile.Destination.Path = destination
	check := overlapCheck(profile)
	if check.Level != CheckError {
		t.Fatalf("nested destination should fail: %+v", check)
	}
}

func TestSplitLinesAndCarriageReturns(t *testing.T) {
	advance, token, err := splitLinesAndCarriageReturns([]byte("50%\r51%\n"), false)
	if err != nil || advance != 4 || string(token) != "50%" {
		t.Fatalf("unexpected split: %d %q %v", advance, token, err)
	}
}

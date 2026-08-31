package cli

import (
	"bytes"
	"context"
	"testing"

	"github.com/gyrifi/gyrif-context-ledger/runtime/internal/buildinfo"
)

func TestVersionUsesBuildInfo(t *testing.T) {
	originalVersion, originalCommit, originalDate := buildinfo.Version, buildinfo.Commit, buildinfo.Date
	t.Cleanup(func() {
		buildinfo.Version, buildinfo.Commit, buildinfo.Date = originalVersion, originalCommit, originalDate
	})
	buildinfo.Version = "9.9.9"
	buildinfo.Commit = "deadbeef"
	buildinfo.Date = "2026-09-01T00:00:00Z"

	var output bytes.Buffer
	handled, err := Run(context.Background(), []string{"version"}, nil, &output)
	if err != nil {
		t.Fatal(err)
	}
	if !handled {
		t.Fatal("version command was not handled")
	}
	if got, want := output.String(), "gyrifi 9.9.9 (deadbeef, 2026-09-01T00:00:00Z)\n"; got != want {
		t.Fatalf("output = %q, want %q", got, want)
	}
}

func TestRunVersionDefersOtherCommands(t *testing.T) {
	var output bytes.Buffer
	for _, args := range [][]string{nil, {"doctor"}, {"unknown"}} {
		handled, err := RunVersion(args, &output)
		if err != nil {
			t.Fatal(err)
		}
		if handled {
			t.Fatalf("RunVersion(%q) handled non-version command", args)
		}
	}
	if output.Len() != 0 {
		t.Fatalf("output = %q", output.String())
	}
}

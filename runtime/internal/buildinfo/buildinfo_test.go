package buildinfo

import "testing"

func TestString(t *testing.T) {
	originalVersion, originalCommit, originalDate := Version, Commit, Date
	t.Cleanup(func() {
		Version, Commit, Date = originalVersion, originalCommit, originalDate
	})

	if got, want := String(), "gyrifi dev (unknown, unknown)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	Version = "0.2.0"
	Commit = "a1b2c3d"
	Date = "2026-08-12T10:00:00Z"
	if got, want := String(), "gyrifi 0.2.0 (a1b2c3d, 2026-08-12T10:00:00Z)"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

package store

import "testing"

func TestReleaseType(t *testing.T) {
	tests := map[int]string{
		0: "single",
		1: "single",
		2: "ep",
		6: "ep",
		7: "album",
		12: "album",
	}
	for count, want := range tests {
		if got := ReleaseType(count); got != want {
			t.Fatalf("ReleaseType(%d) = %q, want %q", count, got, want)
		}
	}
}

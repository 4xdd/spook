package store

import "testing"

func TestArtistListedInBrowse(t *testing.T) {
	tests := []struct {
		name     string
		primary  int
		releases int
		tracks   int
		want     bool
	}{
		{"one-off feature", 0, 1, 1, false},
		{"primary single", 1, 1, 1, true},
		{"two single features", 0, 2, 2, true},
		{"featured on ep", 0, 1, 4, true},
		{"library artist", 5, 7, 40, true},
		{"no releases", 0, 0, 0, false},
		{"inflated album only", 0, 0, 0, false},
	}
	for _, tc := range tests {
		got := ArtistListedInBrowse(tc.primary, tc.releases, tc.tracks)
		if got != tc.want {
			t.Fatalf("%s: listed = %v, want %v", tc.name, got, tc.want)
		}
	}
}

package lyrics

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePlainText(t *testing.T) {
	got := Parse("\n\nFirst line\r\nSecond line\n\nAfter a break\n\n")

	if got.Synced {
		t.Fatalf("plain lyrics reported as synced")
	}
	want := []Line{
		{TimeMs: -1, Text: "First line"},
		{TimeMs: -1, Text: "Second line"},
		{TimeMs: -1, Text: ""},
		{TimeMs: -1, Text: "After a break"},
	}
	assertLines(t, got.Lines, want)
}

func TestParseLRC(t *testing.T) {
	got := Parse(`[ar:Someone]
[ti:A Song]
[00:12.34]First line
[01:05]Second line
[00:04.5]Intro
[02:00.123]Last line`)

	if !got.Synced {
		t.Fatalf("timed lyrics reported as unsynced")
	}
	want := []Line{
		{TimeMs: 4500, Text: "Intro"},
		{TimeMs: 12340, Text: "First line"},
		{TimeMs: 65000, Text: "Second line"},
		{TimeMs: 120123, Text: "Last line"},
	}
	assertLines(t, got.Lines, want)
}

// A chorus is written once with a timestamp per repeat.
func TestParseRepeatedTimestamps(t *testing.T) {
	got := Parse("[00:10.00][01:30.00]Chorus\n[00:20:50]Verse")

	want := []Line{
		{TimeMs: 10000, Text: "Chorus"},
		{TimeMs: 20500, Text: "Verse"},
		{TimeMs: 90000, Text: "Chorus"},
	}
	assertLines(t, got.Lines, want)
}

func TestParseStripsWordTimings(t *testing.T) {
	got := Parse("[00:01.00]<00:01.00>Word <00:01.50> by word")

	assertLines(t, got.Lines, []Line{{TimeMs: 1000, Text: "Word by word"}})
}

// A bracketed section marker is lyric content, not an LRC header.
func TestParseKeepsSectionMarkers(t *testing.T) {
	got := Parse("[Chorus]\nSing along")

	assertLines(t, got.Lines, []Line{
		{TimeMs: -1, Text: "[Chorus]"},
		{TimeMs: -1, Text: "Sing along"},
	})
}

func TestParseEmpty(t *testing.T) {
	for _, text := range []string{"", "   \n\n  ", "[ar:Someone]\n[ti:A Song]"} {
		if got := Parse(text); !got.Empty() {
			t.Fatalf("Parse(%q) = %v, want no lines", text, got.Lines)
		}
	}
}

func TestFindPrefersSidecar(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(track, []byte("not really audio"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "song.lrc"), []byte("[00:01.00]From the sidecar"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := Find(track)
	if got.Source != SourceSidecar || !got.Synced {
		t.Fatalf("Find() source = %q synced = %v, want sidecar and synced", got.Source, got.Synced)
	}
	assertLines(t, got.Lines, []Line{{TimeMs: 1000, Text: "From the sidecar"}})
}

func TestFindWithoutLyrics(t *testing.T) {
	dir := t.TempDir()
	track := filepath.Join(dir, "song.mp3")
	if err := os.WriteFile(track, []byte("not really audio"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := Find(track); !got.Empty() {
		t.Fatalf("Find() = %v, want no lines", got.Lines)
	}
}

func assertLines(t *testing.T, got, want []Line) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d lines %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

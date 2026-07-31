package lyrics

import "testing"

func TestParseLRCLibSynced(t *testing.T) {
	got := ParseLRC("[00:12.34]First line\n[01:05.00]Second line\n[00:04.50]Intro\n")

	if !got.Synced {
		t.Fatal("expected synced lyrics")
	}
	want := []Line{
		{TimeMs: 12340, Text: "First line"},
		{TimeMs: 65000, Text: "Second line"},
		{TimeMs: 4500, Text: "Intro"},
	}
	assertLines(t, got.Lines, want)
}

func TestPlainLines(t *testing.T) {
	got := plainLines("Line one\n\nLine two")

	if got.Synced {
		t.Fatal("plain lyrics should not be synced")
	}
	assertLines(t, got.Lines, []Line{
		{TimeMs: -1, Text: "Line one"},
		{TimeMs: -1, Text: ""},
		{TimeMs: -1, Text: "Line two"},
	})
}

func TestParseLRCLibEmpty(t *testing.T) {
	if got := ParseLRC("[ar:Someone]\n"); !got.Empty() {
		t.Fatalf("ParseLRC() = %v, want empty", got.Lines)
	}
}

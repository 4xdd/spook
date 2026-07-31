package deezer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAlbumTitleFromFolder(t *testing.T) {
	title, ok := albumTitleFromFolder("Kanye West - Yeezus")
	if !ok || title != "Yeezus" {
		t.Fatalf("got (%q, %v)", title, ok)
	}

	title, ok = albumTitleFromFolder("Tyler, The Creator - IGOR")
	if !ok || title != "IGOR" {
		t.Fatalf("got (%q, %v)", title, ok)
	}

	if _, ok := albumTitleFromFolder("NoSeparator"); ok {
		t.Fatal("expected false for folder without separator")
	}
}

func TestOrganizeAlbumDownloads(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "albums", "Kanye West - Yeezus")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "01 - Kanye West Black Skinhead.mp3"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "02 - Kanye West New Slaves.mp3"), []byte("b"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-stagingQuietPeriod - time.Second)
	for _, name := range []string{staging, filepath.Join(staging, "01 - Kanye West Black Skinhead.mp3"), filepath.Join(staging, "02 - Kanye West New Slaves.mp3")} {
		if err := os.Chtimes(name, old, old); err != nil {
			t.Fatal(err)
		}
	}

	moved, err := OrganizeAlbumDownloads(root)
	if err != nil {
		t.Fatal(err)
	}
	if !moved {
		t.Fatal("expected moved=true")
	}

	target := filepath.Join(root, "Yeezus")
	if _, err := os.Stat(filepath.Join(target, "01 - Kanye West Black Skinhead.mp3")); err != nil {
		t.Fatalf("missing track 1: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "albums")); os.IsNotExist(err) {
		t.Fatal("albums staging root should remain")
	}
}

package deezer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStagingFolderActive(t *testing.T) {
	dir := t.TempDir()
	if stagingFolderActive(dir) {
		t.Fatal("empty folder should not be active")
	}

	path := filepath.Join(dir, "01 - track.mp3")
	if err := os.WriteFile(path, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !stagingFolderActive(dir) {
		t.Fatal("recently written folder should be active")
	}

	old := time.Now().Add(-stagingQuietPeriod - time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatal(err)
	}
	if stagingFolderActive(dir) {
		t.Fatal("old folder should not be active")
	}
}

func TestOrganizeSkipsActiveStagingFolder(t *testing.T) {
	root := t.TempDir()
	staging := filepath.Join(root, "albums", "Charli xcx - BRAT")
	if err := os.MkdirAll(staging, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staging, "01 - track.mp3"), []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}

	moved, err := OrganizeAlbumDownloads(root)
	if err != nil {
		t.Fatal(err)
	}
	if moved {
		t.Fatal("expected active staging folder to be skipped")
	}
	if _, err := os.Stat(staging); err != nil {
		t.Fatalf("staging folder should remain: %v", err)
	}
}

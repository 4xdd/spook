package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spook/server/internal/credits"
)

func TestBackfillTrackArtists(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	ctx := context.Background()
	if err := st.UpsertTracks(ctx, []Track{{
		ID: "t1", Path: "/music/a.mp3", Filename: "a.mp3", Title: "Song",
		Artist: "Frost Children · Purple Label Records", AlbumArtist: "Purple Label Records",
		ArtistID: credits.ArtistID("Purple Label Records"), AlbumID: "alb1", AlbumName: "Mixtape",
	}, {
		ID: "t2", Path: "/music/b.mp3", Filename: "b.mp3",
		Title: "Bangarang (feat. Sirah)", Artist: "Skrillex", AlbumArtist: "Skrillex",
		ArtistID: credits.ArtistID("Skrillex"), AlbumID: "alb2", AlbumName: "Bangarang",
	}}); err != nil {
		t.Fatal(err)
	}

	if err := st.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	var creditCount, artistCount int
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM track_artists`).Scan(&creditCount); err != nil {
		t.Fatal(err)
	}
	if err := st.db.QueryRow(`SELECT COUNT(*) FROM artists`).Scan(&artistCount); err != nil {
		t.Fatal(err)
	}
	if creditCount < 4 {
		t.Fatalf("expected credits for both tracks, got %d", creditCount)
	}
	if artistCount == 0 {
		t.Fatal("expected artists after rebuild")
	}

	var sirah int
	if err := st.db.QueryRow(`
		SELECT COUNT(*) FROM track_artists
		WHERE track_id = 't2' AND artist_name = 'Sirah'`).Scan(&sirah); err != nil {
		t.Fatal(err)
	}
	if sirah != 1 {
		t.Fatal("expected title feat. Sirah to be credited")
	}

	var display string
	if err := st.db.QueryRow(`SELECT artist FROM tracks WHERE id = 't2'`).Scan(&display); err != nil {
		t.Fatal(err)
	}
	if display != "Skrillex · Sirah" {
		t.Fatalf("display artist = %q, want Skrillex · Sirah", display)
	}
}

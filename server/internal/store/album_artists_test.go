package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/spook/server/internal/credits"
)

func TestAlbumHeadliners(t *testing.T) {
	jay := albumArtistRow{id: "jay", name: "JAŸ-Z", count: 15}
	kanye := albumArtistRow{id: "kanye", name: "Kanye West", count: 1}

	got := albumHeadliners([]albumArtistRow{jay, kanye})
	if len(got) != 2 {
		t.Fatalf("watch the throne: got %d headliners, want 2", len(got))
	}

	if albumHeadliners([]albumArtistRow{jay}) != nil {
		t.Fatal("expected nil for single-artist album")
	}

	// One-track primary on a long album is a guest, not a co-lead.
	twista := albumArtistRow{id: "twista", name: "Twista", count: 1}
	kanyeAlbum := albumArtistRow{id: "kanye", name: "Kanye West", count: 20}
	if albumHeadliners([]albumArtistRow{kanyeAlbum, twista}) != nil {
		t.Fatal("expected nil when junior is a one-track blip on a long album")
	}

	// Mid-length set with a single remix cut (not long enough for the 6% rule).
	porter := albumArtistRow{id: "porter", name: "Porter Robinson", count: 8}
	caribou := albumArtistRow{id: "caribou", name: "Caribou", count: 1}
	if albumHeadliners([]albumArtistRow{porter, caribou}) != nil {
		t.Fatal("expected nil for 8+1 remix cut")
	}

	// DJ set: many artists each primary on one cut — nobody shares billing.
	set := []albumArtistRow{
		{id: "porter", name: "Porter Robinson", count: 14},
		{id: "id", name: "ID", count: 3},
		{id: "caribou", name: "Caribou", count: 1},
		{id: "lilb", name: "Lil B", count: 1},
		{id: "1975", name: "The 1975", count: 1},
	}
	if albumHeadliners(set) != nil {
		t.Fatal("expected nil for DJ-set style primary spread")
	}

	// Short collab single/EP: any second primary shares billing.
	a := albumArtistRow{id: "a", name: "A", count: 1}
	b := albumArtistRow{id: "b", name: "B", count: 1}
	if len(albumHeadliners([]albumArtistRow{a, b})) != 2 {
		t.Fatal("expected both artists on short collab")
	}

	// Balanced long collab: both clear via share.
	c := albumArtistRow{id: "c", name: "C", count: 6}
	d := albumArtistRow{id: "d", name: "D", count: 5}
	if len(albumHeadliners([]albumArtistRow{c, d})) != 2 {
		t.Fatal("expected both artists on balanced collab")
	}
}

func TestAlbumArtistsCollabAlbum(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	jayID := credits.ArtistID("JAŸ-Z")
	kanyeID := credits.ArtistID("Kanye West")
	albumID := "wtt"

	tracks := []Track{
		{
			ID: "t1", Path: "/music/1.mp3", Filename: "1.mp3", Title: "Track One",
			Artist: "JAŸ-Z", AlbumArtist: "JAŸ-Z", ArtistID: jayID, AlbumID: albumID, AlbumName: "Watch The Throne",
			Credits: []ArtistCredit{{ID: jayID, Name: "JAŸ-Z", Position: 0}},
		},
		{
			ID: "t2", Path: "/music/2.mp3", Filename: "2.mp3", Title: "Track Two",
			Artist: "JAŸ-Z", AlbumArtist: "JAŸ-Z", ArtistID: jayID, AlbumID: albumID, AlbumName: "Watch The Throne",
			Credits: []ArtistCredit{{ID: jayID, Name: "JAŸ-Z", Position: 0}},
		},
		{
			ID: "t3", Path: "/music/3.mp3", Filename: "3.mp3", Title: "H•A•M",
			Artist: "Kanye West", AlbumArtist: "JAŸ-Z", ArtistID: jayID, AlbumID: albumID, AlbumName: "Watch The Throne",
			Credits: []ArtistCredit{
				{ID: kanyeID, Name: "Kanye West", Position: 0},
				{ID: jayID, Name: "JAŸ-Z", Position: 1},
			},
		},
	}

	ctx := context.Background()
	if err := st.UpsertTracks(ctx, tracks); err != nil {
		t.Fatal(err)
	}
	if err := st.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	albums, err := st.AlbumsByArtist(ctx, kanyeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(albums) != 1 {
		t.Fatalf("got %d albums for Kanye, want 1", len(albums))
	}
	if albums[0].ArtistName != "JAŸ-Z · Kanye West" {
		t.Fatalf("artist name = %q, want collab billing", albums[0].ArtistName)
	}

	kanyeTracks, err := st.TracksByArtist(ctx, kanyeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(kanyeTracks) != 3 {
		t.Fatalf("got %d tracks for Kanye, want full album (3)", len(kanyeTracks))
	}
}

func TestAlbumArtistsIgnoresDJSetGuests(t *testing.T) {
	dir := t.TempDir()
	st, err := Open(filepath.Join(dir, "library.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	porterID := credits.ArtistID("Porter Robinson")
	caribouID := credits.ArtistID("Caribou")
	albumID := "secret-sky"

	tracks := make([]Track, 0, 10)
	for i := 0; i < 8; i++ {
		id := "p" + string(rune('a'+i))
		tracks = append(tracks, Track{
			ID: id, Path: "/music/" + id + ".mp3", Filename: id + ".mp3", Title: "P" + id,
			Artist: "Porter Robinson", AlbumArtist: "Porter Robinson", ArtistID: porterID,
			AlbumID: albumID, AlbumName: "Secret Sky Set",
			Credits: []ArtistCredit{{ID: porterID, Name: "Porter Robinson", Position: 0}},
		})
	}
	tracks = append(tracks, Track{
		ID: "c1", Path: "/music/c1.mp3", Filename: "c1.mp3", Title: "Caribou Cut",
		Artist: "Caribou", AlbumArtist: "Porter Robinson", ArtistID: porterID,
		AlbumID: albumID, AlbumName: "Secret Sky Set",
		Credits: []ArtistCredit{{ID: caribouID, Name: "Caribou", Position: 0}},
	})

	ctx := context.Background()
	if err := st.UpsertTracks(ctx, tracks); err != nil {
		t.Fatal(err)
	}
	if err := st.Rebuild(ctx); err != nil {
		t.Fatal(err)
	}

	artists, err := st.Artists(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, artist := range artists {
		if artist.ID == caribouID {
			t.Fatalf("Caribou listed in browse after one remix cut")
		}
	}

	// Direct artist page still works via track credit.
	caribou, err := st.Artist(ctx, caribouID)
	if err != nil {
		t.Fatal(err)
	}
	if caribou.TrackCount != 1 {
		t.Fatalf("caribou track count = %d, want 1", caribou.TrackCount)
	}
}

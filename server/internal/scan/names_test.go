package scan

import "testing"

func TestAlbumContainerName(t *testing.T) {
	root := "/music"
	tests := []struct {
		dir  string
		want string
	}{
		{"/music/Yeezus", "Yeezus"},
		{"/music/Yeezus/CD1", "Yeezus"},
		{"/music/Yeezus/disc 2", "Yeezus"},
		{"/music/Jealous (2025)", "Jealous (2025)"},
		{"/music/Ennaria", "Ennaria"},
	}
	for _, tc := range tests {
		if got := albumContainerName(root, tc.dir); got != tc.want {
			t.Fatalf("albumContainerName(%q) = %q, want %q", tc.dir, got, tc.want)
		}
	}
}

func TestResolveAlbumName(t *testing.T) {
	tests := []struct {
		tag, container, want string
	}{
		{"Jealous", "Jealous (2025)", "Jealous (2025)"},
		{"Jealous", "Ennaria", "Jealous"},
		{"name", "name2", "name2"},
		{"name", "name3", "name3"},
		{"Graduation", "Graduation", "Graduation"},
		{"", "Skrillex - 2010 - Recess", "Recess"},
		{"", "100 gecs - 1000 gecs", "1000 gecs"},
	}
	for _, tc := range tests {
		if got := resolveAlbumName(tc.tag, tc.container); got != tc.want {
			t.Fatalf("resolveAlbumName(%q, %q) = %q, want %q", tc.tag, tc.container, got, tc.want)
		}
	}
}

func TestParseFolderArtistAlbum(t *testing.T) {
	artist, album, ok := parseFolderArtistAlbum("Skrillex - 2010 - Scary Monsters And Nice Sprites")
	if !ok || artist != "Skrillex" || album != "Scary Monsters And Nice Sprites" {
		t.Fatalf("got (%q, %q, %v)", artist, album, ok)
	}

	artist, album, ok = parseFolderArtistAlbum("100 gecs - 1000 gecs")
	if !ok || artist != "100 gecs" || album != "1000 gecs" {
		t.Fatalf("got (%q, %q, %v)", artist, album, ok)
	}
}

func TestAlbumIDSplitsVariants(t *testing.T) {
	a := albumID("Ennaria", "Jealous", "Jealous (2025)")
	b := albumID("Ennaria", "Jealous", "Ennaria")
	if a == b {
		t.Fatal("expected different album IDs for different containers")
	}

	c := albumID("Artist", "name", "name")
	d := albumID("Artist", "name2", "name2")
	if c == d {
		t.Fatal("expected numbered variants to produce different album IDs")
	}
}

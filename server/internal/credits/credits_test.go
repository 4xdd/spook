package credits

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		raw  string
		want []string
	}{
		{"Ninajirachi, MGNA Crrrta", []string{"Ninajirachi", "MGNA Crrrta"}},
		{"underscores feat. 8485", []string{"underscores", "8485"}},
		{"Porter Robinson & livetune", []string{"Porter Robinson", "livetune"}},
		{"Frost Children, Olswel", []string{"Frost Children", "Olswel"}},
		{"Jane Remover w/ 8485", []string{"Jane Remover", "8485"}},
		{"Artist w/ Someone", []string{"Artist", "Someone"}},
		// A previous Format pass must still round-trip into individuals.
		{"underscores · 8485", []string{"underscores", "8485"}},
	}
	for _, tc := range tests {
		got := Parse(tc.raw)
		if len(got) != len(tc.want) {
			t.Fatalf("Parse(%q) = %v, want %v", tc.raw, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("Parse(%q)[%d] = %q, want %q", tc.raw, i, got[i], tc.want[i])
			}
		}
	}
}

func TestFromTitle(t *testing.T) {
	tests := []struct {
		title string
		want  []string
	}{
		{"Bangarang (feat. Sirah)", []string{"Sirah"}},
		{"Unfold (feat. Totally Enormous Extinct Dinosaurs)", []string{"Totally Enormous Extinct Dinosaurs"}},
		{"Years Of War (Feat. Breanne Duren & Sean Caskey)", []string{"Breanne Duren", "Sean Caskey"}},
		{"voltage (see you again) (feat. Virtual Riot, Loam, Eurohead & swedm®)", []string{"Virtual Riot", "Loam", "Eurohead", "swedm®"}},
		{"Divinity (Feat. Amy Millan)", []string{"Amy Millan"}},
		{"hit me where it hurts x (feat. Caroline Polachek & Dylan Brady)", []string{"Caroline Polachek", "Dylan Brady"}},
		{"Cash Shit (feat. DaBaby) [Milkfish edit]", []string{"DaBaby"}},
		{"Shelter", nil},
		{"Locals (Girls like us)", nil},
	}
	for _, tc := range tests {
		got := FromTitle(tc.title)
		if len(got) != len(tc.want) {
			t.Fatalf("FromTitle(%q) = %v, want %v", tc.title, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Fatalf("FromTitle(%q)[%d] = %q, want %q", tc.title, i, got[i], tc.want[i])
			}
		}
	}
}

func TestAllAppendsTitleFeatures(t *testing.T) {
	got := All("Porter Robinson", "Porter Robinson", "Mona Lisa (feat. Frost Children)")
	want := []string{"Porter Robinson", "Frost Children"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestMergePrefersTrackArtist(t *testing.T) {
	got := Merge("Frost Children, DJH", "Purple Label Records")
	if len(got) < 2 || got[0] != "Frost Children" {
		t.Fatalf("got %v, want Frost Children first", got)
	}
}

func TestDedupe(t *testing.T) {
	got := Merge("Ninajirachi, MGNA Crrrta", "Ninajirachi")
	want := []string{"Ninajirachi", "MGNA Crrrta"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

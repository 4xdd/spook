package store

import "testing"

func TestPickPrimaryArtist(t *testing.T) {
	primary := pickPrimaryArtist([]primaryCandidate{
		{id: "1", name: "Purple Label Records", count: 8},
		{id: "2", name: "Frost Children", count: 8},
	})
	if primary.Name != "Frost Children" {
		t.Fatalf("got %q, want Frost Children", primary.Name)
	}
}

func TestPickPrimaryArtistByCount(t *testing.T) {
	primary := pickPrimaryArtist([]primaryCandidate{
		{id: "1", name: "Featured Artist", count: 1},
		{id: "2", name: "Frost Children", count: 7},
	})
	if primary.Name != "Frost Children" {
		t.Fatalf("got %q, want Frost Children", primary.Name)
	}
}

func TestIsLikelyLabel(t *testing.T) {
	if !isLikelyLabel("Purple Label Records") {
		t.Fatal("expected label detection")
	}
	if isLikelyLabel("Frost Children") {
		t.Fatal("did not expect artist to be label")
	}
}

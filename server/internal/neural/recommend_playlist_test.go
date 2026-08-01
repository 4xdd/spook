package neural

import "testing"

func TestRecommendPlaylistCentroid(t *testing.T) {
	idx := NewIndex(3)
	idx.Set("a", []float32{1, 0, 0})
	idx.Set("b", []float32{0.9, 0.1, 0})
	idx.Set("c", []float32{0, 1, 0})
	idx.Set("d", []float32{0, 0, 1})

	out := idx.RecommendPlaylist(RecommendOptions{
		SeedIDs: []string{"a", "b"},
		Exclude: map[string]struct{}{"a": {}, "b": {}},
		Limit:   2,
		Nonce:   1,
	})
	if len(out) == 0 {
		t.Fatal("expected recommendations")
	}
	for _, s := range out {
		if s.ID == "a" || s.ID == "b" {
			t.Fatalf("excluded track recommended: %s", s.ID)
		}
	}
}

func TestRecommendPlaylistRefreshChangesOrder(t *testing.T) {
	idx := NewIndex(4)
	for i := 0; i < 10; i++ {
		id := string(rune('a' + i))
		vec := make([]float32, 4)
		vec[0] = float32(i) / 10
		vec[1] = 1
		idx.Set(id, vec)
	}

	a := idx.RecommendPlaylist(RecommendOptions{
		SeedIDs: []string{"a"},
		Exclude: map[string]struct{}{"a": {}},
		Limit:   5,
		Nonce:   10,
	})
	b := idx.RecommendPlaylist(RecommendOptions{
		SeedIDs: []string{"a"},
		Exclude: map[string]struct{}{"a": {}},
		Limit:   5,
		Nonce:   99,
	})
	if len(a) != 5 || len(b) != 5 {
		t.Fatalf("len a=%d b=%d", len(a), len(b))
	}
	sameOrder := true
	for i := range a {
		if a[i].ID != b[i].ID {
			sameOrder = false
			break
		}
	}
	if sameOrder {
		t.Fatal("expected nonce to change recommendation order")
	}
}

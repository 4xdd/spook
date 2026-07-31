package neural

import "testing"

func TestRecommendIndex(t *testing.T) {
	idx := NewIndex(3)
	idx.Set("a", []float32{1, 0, 0})
	idx.Set("b", []float32{0.9, 0.1, 0})
	idx.Set("c", []float32{0, 1, 0})

	recs := idx.Recommend("a", 2)
	if len(recs) != 2 {
		t.Fatalf("got %d recs", len(recs))
	}
	if recs[0].ID != "b" {
		t.Fatalf("top rec %q want b", recs[0].ID)
	}
	if recs[1].ID != "c" {
		t.Fatalf("second rec %q want c", recs[1].ID)
	}
}

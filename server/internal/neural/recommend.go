package neural

import (
	"context"
	"sort"
)

// ScoredID is a track identifier ranked by embedding similarity.
type ScoredID struct {
	ID    string
	Score float32 // cosine similarity in [-1, 1]
}

// Index stores track embeddings for nearest-neighbor recommendations.
type Index struct {
	dim   int
	items map[string][]float32
}

// NewIndex creates an empty similarity index.
func NewIndex(dim int) *Index {
	return &Index{dim: dim, items: make(map[string][]float32)}
}

// Set stores or replaces an embedding for id.
func (idx *Index) Set(id string, embedding []float32) {
	if len(embedding) != idx.dim {
		return
	}
	cp := make([]float32, len(embedding))
	copy(cp, embedding)
	idx.items[id] = cp
}

// Delete removes id from the index.
func (idx *Index) Delete(id string) {
	delete(idx.items, id)
}

// Len returns the number of indexed tracks.
func (idx *Index) Len() int { return len(idx.items) }

// Recommend returns up to n tracks most similar to id (excluding id itself).
func (idx *Index) Recommend(id string, n int) []ScoredID {
	emb, ok := idx.items[id]
	if !ok {
		return nil
	}
	return idx.RecommendFromEmbedding(emb, n, id)
}

// RecommendFromEmbedding ranks indexed tracks by cosine similarity to emb.
// skipID is omitted from results when non-empty.
func (idx *Index) RecommendFromEmbedding(emb []float32, n int, skipID string) []ScoredID {
	if len(emb) != idx.dim || n <= 0 {
		return nil
	}
	scored := make([]ScoredID, 0, len(idx.items))
	for id, other := range idx.items {
		if id == skipID {
			continue
		}
		scored = append(scored, ScoredID{ID: id, Score: Similarity(emb, other)})
	}
	sort.Slice(scored, func(i, j int) bool {
		return scored[i].Score > scored[j].Score
	})
	if n > len(scored) {
		n = len(scored)
	}
	return scored[:n]
}

// BuildIndex embeds audio files and returns a populated Index.
func BuildIndex(ctx context.Context, emb *Embedder, ids []string, paths []string) (*Index, error) {
	if len(ids) != len(paths) {
		return nil, errIndexLenMismatch
	}
	idx := NewIndex(emb.model.cfg.HiddenSize)
	for i := range ids {
		vec, err := emb.EmbedFile(ctx, paths[i])
		if err != nil {
			return nil, err
		}
		idx.Set(ids[i], vec)
	}
	return idx, nil
}

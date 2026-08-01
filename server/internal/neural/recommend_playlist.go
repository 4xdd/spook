package neural

import (
	"math"
	"math/rand"
	"sort"
)

const defaultRecommendPool = 48

// RecommendOptions configures playlist-style retrieval with mild randomization.
type RecommendOptions struct {
	SeedIDs  []string
	Exclude  map[string]struct{}
	Limit    int
	Nonce    int64
	PoolSize int
	Jitter   float32
}

// RecommendPlaylist ranks tracks for a playlist context.
// With seeds, uses the normalized centroid of seed embeddings.
// Without seeds, returns a mildly randomized slice of the library.
func (idx *Index) RecommendPlaylist(opts RecommendOptions) []ScoredID {
	if idx == nil || idx.Len() == 0 || opts.Limit <= 0 {
		return nil
	}
	if opts.PoolSize <= 0 {
		opts.PoolSize = defaultRecommendPool
	}
	if opts.Jitter <= 0 {
		opts.Jitter = 0.06
	}
	if opts.Exclude == nil {
		opts.Exclude = map[string]struct{}{}
	}

	rng := rand.New(rand.NewSource(opts.Nonce))

	candidates := make([]ScoredID, 0, idx.Len())
	if len(opts.SeedIDs) > 0 {
		centroid := idx.centroid(opts.SeedIDs)
		if centroid == nil {
			return idx.randomSample(opts, rng)
		}
		for id, vec := range idx.items {
			if _, skip := opts.Exclude[id]; skip {
				continue
			}
			candidates = append(candidates, ScoredID{ID: id, Score: Similarity(centroid, vec)})
		}
	} else {
		return idx.randomSample(opts, rng)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})

	pool := candidates
	if len(pool) > opts.PoolSize {
		pool = pool[:opts.PoolSize]
	}

	for i := range pool {
		pool[i].Score += (rng.Float32()*2 - 1) * opts.Jitter
	}
	sort.Slice(pool, func(i, j int) bool {
		return pool[i].Score > pool[j].Score
	})

	if len(pool) > opts.Limit {
		pool = pool[:opts.Limit]
	}
	return pool
}

func (idx *Index) centroid(ids []string) []float32 {
	if len(ids) == 0 || idx.dim == 0 {
		return nil
	}
	sum := make([]float32, idx.dim)
	var n int
	for _, id := range ids {
		vec, ok := idx.items[id]
		if !ok || len(vec) != idx.dim {
			continue
		}
		for i, v := range vec {
			sum[i] += v
		}
		n++
	}
	if n == 0 {
		return nil
	}
	inv := float32(1.0 / float64(n))
	for i := range sum {
		sum[i] *= inv
	}
	return normalize(sum)
}

func normalize(v []float32) []float32 {
	var norm float64
	for _, x := range v {
		norm += float64(x) * float64(x)
	}
	if norm == 0 {
		return v
	}
	inv := float32(1.0 / math.Sqrt(norm))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x * inv
	}
	return out
}

func (idx *Index) randomSample(opts RecommendOptions, rng *rand.Rand) []ScoredID {
	ids := make([]string, 0, idx.Len())
	for id := range idx.items {
		if _, skip := opts.Exclude[id]; skip {
			continue
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil
	}
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	limit := opts.Limit
	if limit > len(ids) {
		limit = len(ids)
	}
	out := make([]ScoredID, limit)
	for i := 0; i < limit; i++ {
		out[i] = ScoredID{ID: ids[i], Score: 0}
	}
	return out
}

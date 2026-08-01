package recommend

import (
	"context"
	"sync"

	"github.com/spook/server/internal/neural"
	"github.com/spook/server/internal/store"
)

// Engine serves similarity-based track recommendations from stored embeddings.
type Engine struct {
	store *store.Store
	dim   int

	mu    sync.RWMutex
	index *neural.Index
}

func NewEngine(st *store.Store, dim int) *Engine {
	if dim <= 0 {
		dim = 768
	}
	return &Engine{store: st, dim: dim, index: neural.NewIndex(dim)}
}

func (e *Engine) Reload(ctx context.Context) error {
	rows, err := e.store.AllEmbeddings(ctx)
	if err != nil {
		return err
	}
	idx := neural.NewIndex(e.dim)
	for _, row := range rows {
		if len(row.Vector) > 0 {
			idx.Set(row.TrackID, row.Vector)
			if e.dim == 768 && row.Dim > 0 {
				e.dim = row.Dim
			}
		}
	}
	e.mu.Lock()
	e.index = idx
	e.mu.Unlock()
	return nil
}

func (e *Engine) Len() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	if e.index == nil {
		return 0
	}
	return e.index.Len()
}

// Recommend returns track IDs ranked for the given playlist context.
func (e *Engine) Recommend(seedIDs, excludeIDs []string, limit int, nonce int64) []string {
	e.mu.RLock()
	idx := e.index
	e.mu.RUnlock()
	if idx == nil || limit <= 0 {
		return nil
	}

	exclude := make(map[string]struct{}, len(excludeIDs))
	for _, id := range excludeIDs {
		exclude[id] = struct{}{}
	}

	scored := idx.RecommendPlaylist(neural.RecommendOptions{
		SeedIDs: seedIDs,
		Exclude: exclude,
		Limit:   limit,
		Nonce:   nonce,
	})
	out := make([]string, len(scored))
	for i, s := range scored {
		out[i] = s.ID
	}
	return out
}

// Tracks returns full track rows for recommended IDs.
func (e *Engine) Tracks(ctx context.Context, seedIDs, excludeIDs []string, limit int, nonce int64) ([]store.Track, error) {
	ids := e.Recommend(seedIDs, excludeIDs, limit, nonce)
	if len(ids) == 0 {
		return nil, nil
	}
	return e.store.TracksByIDs(ctx, ids)
}

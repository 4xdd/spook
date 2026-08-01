package neural

import (
	"context"
	"fmt"
	"sync"
)

// Pool runs parallel MERT inference with one shared weight file and N model instances.
type Pool struct {
	path   string
	models []*Model
	sem    chan *Model
}

// OpenPool loads weights once and creates worker parallel inference goroutines.
func OpenPool(path string, workers int) (*Pool, error) {
	if workers < 1 {
		workers = 1
	}
	w, err := loadWeightsCached(path)
	if err != nil {
		return nil, err
	}
	p := &Pool{
		path: path,
		sem:  make(chan *Model, workers),
	}
	for i := 0; i < workers; i++ {
		m, err := NewModel(w)
		if err != nil {
			return nil, fmt.Errorf("model slot %d: %w", i, err)
		}
		p.models = append(p.models, m)
		p.sem <- m
	}
	return p, nil
}

func (p *Pool) Workers() int { return len(p.models) }

func (p *Pool) Backend() string { return "native" }

func (p *Pool) Close() {}

var _ EmbedPool = (*Pool)(nil)

func (p *Pool) borrow() *Model { return <-p.sem }

func (p *Pool) release(m *Model) { p.sem <- m }

// EmbedFileClip decodes a clip and returns an embedding using an pooled model instance.
func (p *Pool) EmbedFileClip(ctx context.Context, audioPath string, maxSeconds float64) ([]float32, error) {
	m := p.borrow()
	defer p.release(m)

	wav, err := DecodeMonoClip(ctx, audioPath, m.cfg.SampleRate, maxSeconds)
	if err != nil {
		return nil, err
	}
	return m.Embed(wav)
}

// EmbedWaveform is safe for concurrent use via the pool.
func (p *Pool) EmbedWaveform(waveform []float32) ([]float32, error) {
	m := p.borrow()
	defer p.release(m)
	return m.Embed(waveform)
}

var (
	sharedWeightsMu sync.Mutex
	sharedWeights   = make(map[string]*Weights)
)

// loadWeightsCached avoids reloading the ~90–360 MiB checkpoint for each pool slot.
func loadWeightsCached(path string) (*Weights, error) {
	sharedWeightsMu.Lock()
	defer sharedWeightsMu.Unlock()
	if w, ok := sharedWeights[path]; ok {
		return w, nil
	}
	w, err := LoadWeights(path)
	if err != nil {
		return nil, err
	}
	sharedWeights[path] = w
	return w, nil
}

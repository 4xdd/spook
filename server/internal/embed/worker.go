package embed

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spook/server/internal/neural"
	"github.com/spook/server/internal/store"
)

const (
	// Shorter clips cut attention cost (~O(n²)) with minimal quality loss for similarity.
	clipSeconds   = 12
	trackTimeout  = 3 * time.Minute
	dbBatchSize   = 32
	reloadEvery   = 64
	stateIdle     = "idle"
	stateRunning  = "running"
	stateDisabled = "disabled"
)

// Progress reports background embedding work.
type Progress struct {
	State     string `json:"state"`
	Total     int    `json:"total"`
	Processed int    `json:"processed"`
	Active    int    `json:"active"`
	Embedded  int    `json:"embedded"`
	Error     string `json:"error,omitempty"`
}

// Worker computes MERT embeddings after library scans.
type Worker struct {
	store    *store.Store
	pool     neural.EmbedPool
	onReload func()

	mu       sync.Mutex
	running  bool
	rerun    bool
	progress Progress
}

func NewWorker(st *store.Store, pool neural.EmbedPool, onReload func()) *Worker {
	state := stateIdle
	if pool == nil {
		state = stateDisabled
	}
	return &Worker{
		store:    st,
		pool:     pool,
		onReload: onReload,
		progress: Progress{State: state},
	}
}

func (w *Worker) Progress() Progress {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.progress
}

func (w *Worker) Workers() int {
	if w.pool == nil {
		return 0
	}
	return w.pool.Workers()
}

func (w *Worker) Enabled() bool {
	return w.pool != nil
}

func (w *Worker) Backend() string {
	if w.pool == nil {
		return ""
	}
	return w.pool.Backend()
}

func (w *Worker) setProgress(processed, embedded, active int) {
	w.mu.Lock()
	w.progress.Processed = processed
	w.progress.Embedded = embedded
	w.progress.Active = active
	w.mu.Unlock()
}

// Schedule starts embedding pending tracks in the background.
// If a pass is already running, a follow-up pass is queued for when it finishes
// (e.g. a rescan added tracks mid-embedding).
func (w *Worker) Schedule(ctx context.Context) {
	if w.pool == nil {
		return
	}
	w.mu.Lock()
	if w.running {
		w.rerun = true
		w.mu.Unlock()
		return
	}
	w.running = true
	w.progress = Progress{State: stateRunning}
	w.mu.Unlock()

	go w.loop(ctx)
}

func (w *Worker) loop(ctx context.Context) {
	defer func() {
		w.mu.Lock()
		rerun := w.rerun
		w.running = false
		w.rerun = false
		if w.progress.State == stateRunning {
			w.progress.State = stateIdle
		}
		w.mu.Unlock()
		if rerun {
			w.Schedule(ctx)
		}
	}()

	for {
		if err := w.run(ctx); err != nil {
			w.mu.Lock()
			w.progress.State = stateIdle
			w.progress.Error = err.Error()
			w.mu.Unlock()
			log.Printf("embeddings: %v", err)
			return
		}

		pending, err := w.store.PendingEmbeddings(ctx, 1)
		if err != nil {
			log.Printf("embeddings: check pending: %v", err)
			return
		}
		if len(pending) == 0 {
			break
		}
		log.Printf("embeddings: more tracks pending, continuing pass")
	}

	if w.onReload != nil {
		go w.onReload()
	}
}

type embedOutcome struct {
	emb store.TrackEmbedding
}

func (w *Worker) run(ctx context.Context) error {
	pending, err := w.store.PendingEmbeddings(ctx, 0)
	if err != nil {
		return err
	}
	if len(pending) == 0 {
		return nil
	}

	total := len(pending)
	w.mu.Lock()
	w.progress = Progress{State: stateRunning, Total: total, Processed: 0, Embedded: 0, Error: ""}
	w.mu.Unlock()

	workers := w.pool.Workers()
	log.Printf("embeddings: indexing %d tracks (%s, %d workers)", total, w.pool.Backend(), workers)
	if w.pool.Backend() == "native" {
		log.Printf("embeddings: native backend is ~1-2 min/track — use ONNX for speed: make install-ort convert-mert-onnx build")
	}

	jobs := make(chan store.PendingEmbedding, workers*2)
	// Large buffer so workers never block waiting for DB writes.
	outcomes := make(chan embedOutcome, 512)

	var wg sync.WaitGroup
	var processed atomic.Int32
	var saved atomic.Int32
	var active atomic.Int32

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if err := ctx.Err(); err != nil {
					return
				}
				active.Add(1)
				w.setProgress(int(processed.Load()), int(saved.Load()), int(active.Load()))
				trackCtx, cancel := context.WithTimeout(ctx, trackTimeout)
				vec, err := w.pool.EmbedFileClip(trackCtx, item.Path, clipSeconds)
				cancel()
				active.Add(-1)
				n := processed.Add(1)
				w.setProgress(int(n), int(saved.Load()), int(active.Load()))
				if err != nil {
					log.Printf("embed %s: %v", item.Path, err)
					continue
				}
				select {
				case outcomes <- embedOutcome{emb: store.TrackEmbedding{
					TrackID: item.ID,
					ModTime: item.ModTime,
					Dim:     len(vec),
					Vector:  vec,
				}}:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		for _, item := range pending {
			select {
			case <-ctx.Done():
				close(jobs)
				return
			case jobs <- item:
			}
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(outcomes)
	}()

	batch := make([]store.TrackEmbedding, 0, dbBatchSize)
	lastReload := 0
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := w.store.PutEmbeddings(ctx, batch); err != nil {
			return err
		}
		n := int(saved.Add(int32(len(batch))))
		w.setProgress(int(processed.Load()), n, int(active.Load()))
		if w.onReload != nil && n/reloadEvery > lastReload {
			lastReload = n / reloadEvery
			go w.onReload()
		}
		batch = batch[:0]
		return nil
	}

	for o := range outcomes {
		if err := ctx.Err(); err != nil {
			_ = flush()
			return err
		}
		batch = append(batch, o.emb)
		if len(batch) >= dbBatchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

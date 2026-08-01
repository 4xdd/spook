//go:build cgo

package neural

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	ort "github.com/yalue/onnxruntime_go"
)

var (
	ortInitOnce sync.Once
	ortInitErr  error
)

func initORT(libPath string) error {
	ortInitOnce.Do(func() {
		ort.SetSharedLibraryPath(libPath)
		ortInitErr = ort.InitializeEnvironment()
	})
	return ortInitErr
}

type ortPool struct {
	path    string
	workers int
	sem     chan *ortSession
}

type ortSession struct {
	sess *ort.DynamicAdvancedSession
}

func openORTPool(onnxPath, libPath string, workers int) (*ortPool, error) {
	if workers < 1 {
		workers = 1
	}
	if err := initORT(libPath); err != nil {
		return nil, fmt.Errorf("init ORT: %w", err)
	}

	p := &ortPool{
		path:    onnxPath,
		workers: workers,
		sem:     make(chan *ortSession, workers),
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		return nil, err
	}
	defer opts.Destroy()
	threads := runtime.NumCPU()
	if threads > workers*2 {
		threads = workers * 2
	}
	if threads < 1 {
		threads = 1
	}
	_ = opts.SetIntraOpNumThreads(threads)
	_ = opts.SetInterOpNumThreads(workers)
	_ = opts.SetGraphOptimizationLevel(ort.GraphOptimizationLevelEnableAll)

	for i := 0; i < workers; i++ {
		sess, err := ort.NewDynamicAdvancedSession(onnxPath, []string{"input_values"}, []string{"embedding"}, opts)
		if err != nil {
			p.Close()
			return nil, fmt.Errorf("session %d: %w", i, err)
		}
		p.sem <- &ortSession{sess: sess}
	}
	return p, nil
}

func (p *ortPool) Workers() int { return p.workers }

func (p *ortPool) Backend() string { return "onnxruntime" }

func (p *ortPool) Close() {
	for {
		select {
		case s := <-p.sem:
			if s != nil && s.sess != nil {
				_ = s.sess.Destroy()
			}
		default:
			return
		}
	}
}

func (p *ortPool) borrow() *ortSession { return <-p.sem }

func (p *ortPool) release(s *ortSession) { p.sem <- s }

func (p *ortPool) EmbedFileClip(ctx context.Context, audioPath string, maxSeconds float64) ([]float32, error) {
	wav, err := DecodeMonoClip(ctx, audioPath, 24000, maxSeconds)
	if err != nil {
		return nil, err
	}
	return p.embedWaveform(wav)
}

func (p *ortPool) embedWaveform(wav []float32) ([]float32, error) {
	s := p.borrow()
	defer p.release(s)

	shape := ort.NewShape(1, int64(len(wav)))
	input, err := ort.NewTensor(shape, wav)
	if err != nil {
		return nil, err
	}
	defer input.Destroy()

	outputShape := ort.NewShape(1, 768)
	output, err := ort.NewEmptyTensor[float32](outputShape)
	if err != nil {
		return nil, err
	}
	defer output.Destroy()

	if err := s.sess.Run([]ort.Value{input}, []ort.Value{output}); err != nil {
		return nil, err
	}
	data := output.GetData()
	if len(data) != 768 {
		return nil, fmt.Errorf("unexpected embedding dim %d", len(data))
	}
	out := make([]float32, 768)
	copy(out, data)
	return out, nil
}

var _ EmbedPool = (*ortPool)(nil)

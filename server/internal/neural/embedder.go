package neural

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

// Embedder loads MERT once and produces fixed-size audio embeddings for similarity search.
type Embedder struct {
	model *Model
	path  string
}

// OpenEmbedder loads a .mert weights file from path.
func OpenEmbedder(path string) (*Embedder, error) {
	w, err := LoadWeights(path)
	if err != nil {
		return nil, err
	}
	model, err := NewModel(w)
	if err != nil {
		return nil, err
	}
	return &Embedder{model: model, path: path}, nil
}

func (e *Embedder) Model() *Model { return e.model }

// EmbedFile decodes audio to mono float32 at the model sample rate and returns an embedding.
func (e *Embedder) EmbedFile(ctx context.Context, audioPath string) ([]float32, error) {
	wav, err := DecodeMono(ctx, audioPath, e.model.cfg.SampleRate)
	if err != nil {
		return nil, err
	}
	return e.model.Embed(wav)
}

// EmbedWaveform embeds an already decoded mono PCM buffer.
func (e *Embedder) EmbedWaveform(waveform []float32) ([]float32, error) {
	return e.model.Embed(waveform)
}

// Similarity returns cosine similarity in [-1, 1].
func Similarity(a, b []float32) float32 {
	return cosineSimilarity(a, b)
}

// DefaultModelPath returns the conventional on-disk location under dataDir.
func DefaultModelPath(dataDir string) string {
	return filepath.Join(dataDir, "models", "mert-v1-95m.mert")
}

type ffmpegDecoder struct {
	lookup sync.Once
	path   string
}

func (d *ffmpegDecoder) available() bool {
	d.lookup.Do(func() {
		p, err := exec.LookPath("ffmpeg")
		if err == nil {
			d.path = p
		}
	})
	return d.path != ""
}

var defaultDecoder ffmpegDecoder

// DecodeMono reads audioPath and returns mono float32 samples at sampleRate Hz.
func DecodeMono(ctx context.Context, audioPath string, sampleRate int) ([]float32, error) {
	if !defaultDecoder.available() {
		return nil, fmt.Errorf("ffmpeg is required to decode audio for MERT")
	}
	if sampleRate <= 0 {
		sampleRate = 24000
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin",
		"-i", audioPath,
		"-ac", "1",
		"-ar", fmt.Sprintf("%d", sampleRate),
		"-f", "f32le",
		"pipe:1",
	}
	cmd := exec.CommandContext(ctx, defaultDecoder.path, args...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("ffmpeg decode: %w: %s", err, ee.Stderr)
		}
		return nil, fmt.Errorf("ffmpeg decode: %w", err)
	}
	if len(out)%4 != 0 {
		return nil, fmt.Errorf("ffmpeg decode: corrupt f32le output")
	}
	n := len(out) / 4
	samples := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(out[i*4:])
		samples[i] = math.Float32frombits(bits)
	}
	return samples, nil
}

// ModelExists reports whether path looks like a converted MERT checkpoint.
func ModelExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 1024
}

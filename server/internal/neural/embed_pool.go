package neural

import (
	"context"
	"os"
	"path/filepath"
)

// EmbedPool runs parallel MERT embedding inference.
type EmbedPool interface {
	Workers() int
	Backend() string
	EmbedFileClip(ctx context.Context, audioPath string, maxSeconds float64) ([]float32, error)
	Close()
}

// DefaultONNXPath returns the conventional ONNX model location under dataDir.
func DefaultONNXPath(dataDir string) string {
	return filepath.Join(dataDir, "models", "mert-v1-95m.onnx")
}

// ONNXExists reports whether an ONNX model is present (graph + optional .data).
func ONNXExists(path string) bool {
	st, err := os.Stat(path)
	if err != nil || st.Size() <= 1024 {
		return false
	}
	// External weights live beside the graph file.
	if _, err := os.Stat(path + ".data"); err == nil {
		return true
	}
	// Small self-contained graphs are also valid.
	return st.Size() > 1024*1024
}

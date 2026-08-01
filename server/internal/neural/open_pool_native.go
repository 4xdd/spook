//go:build !cgo

package neural

import (
	"fmt"
	"log"
)

// OpenEmbedPool loads the native .mert backend.
func OpenEmbedPool(dataDir string, workers int) (EmbedPool, error) {
	if ONNXExists(DefaultONNXPath(dataDir)) {
		log.Printf("recommendations: ONNX model found — rebuild with CGO for ~20× faster embeddings:")
		log.Printf("  make install-ort convert-mert-onnx build")
	}
	path := DefaultModelPath(dataDir)
	if !ModelExists(path) {
		return nil, fmt.Errorf("MERT model not found at %s (make convert-mert)", path)
	}
	pool, err := OpenPool(path, capNativeWorkers(workers))
	if err != nil {
		return nil, err
	}
	return pool, nil
}

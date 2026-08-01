//go:build cgo

package neural

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// OpenEmbedPool prefers ONNX Runtime when the .onnx model and runtime library
// are available, otherwise falls back to the native .mert backend.
func OpenEmbedPool(dataDir string, workers int) (EmbedPool, error) {
	onnxPath := DefaultONNXPath(dataDir)
	if ONNXExists(onnxPath) {
		if lib := findORTLibrary(dataDir); lib != "" {
			pool, err := openORTPool(onnxPath, lib, workers)
			if err == nil {
				log.Printf("recommendations: using ONNX Runtime (%s)", lib)
				return pool, nil
			}
			log.Printf("recommendations: ONNX Runtime unavailable: %v (falling back to native)", err)
		} else {
			log.Printf("recommendations: ONNX model found but libonnxruntime not found — run: make install-ort")
		}
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

func findORTLibrary(dataDir string) string {
	if p := os.Getenv("SPOOK_ORT_LIB"); p != "" {
		if fileExists(p) {
			return p
		}
	}
	candidates := []string{
		filepath.Join(dataDir, "lib", "libonnxruntime.so"),
		"/usr/lib/x86_64-linux-gnu/libonnxruntime.so",
		"/usr/lib/libonnxruntime.so",
		"/usr/local/lib/libonnxruntime.so",
	}
	for _, c := range candidates {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

func fileExists(path string) bool {
	st, err := os.Stat(path)
	return err == nil && !st.IsDir()
}

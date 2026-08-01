//go:build cgo

package neural

import (
	"os"
	"path/filepath"
	"testing"
)

func TestORTEmbed(t *testing.T) {
	lib := os.Getenv("SPOOK_ORT_LIB")
	if lib == "" {
		lib = filepath.Join(os.Getenv("HOME"), ".local/share/spook/lib/libonnxruntime.so")
	}
	onnx := os.Getenv("MERT_ONNX_PATH")
	if onnx == "" {
		onnx = filepath.Join(os.Getenv("HOME"), ".local/share/spook/models/mert-v1-95m.onnx")
	}
	if _, err := os.Stat(lib); err != nil {
		t.Skip("ONNX Runtime library not found; run make install-ort")
	}
	if !ONNXExists(onnx) {
		t.Skip("ONNX model not found; run make convert-mert-onnx")
	}

	pool, err := openORTPool(onnx, lib, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	goldenDir := os.Getenv("MERT_GOLDEN_DIR")
	if goldenDir == "" {
		goldenDir = "/tmp/mert-work"
	}
	if _, err := os.Stat(filepath.Join(goldenDir, "golden_wav.npy")); err != nil {
		t.Skip("golden wav not found")
	}
	wav := loadGoldenWave(t, goldenDir)

	emb, err := pool.embedWaveform(wav)
	if err != nil {
		t.Fatal(err)
	}
	if len(emb) != 768 {
		t.Fatalf("dim=%d", len(emb))
	}

	mertPath := os.Getenv("MERT_MODEL_PATH")
	if mertPath == "" {
		mertPath = filepath.Join(os.Getenv("HOME"), ".local/share/spook/models/mert-v1-95m.mert")
	}
	if !ModelExists(mertPath) {
		t.Log("native .mert not found; skipping cosine check")
		return
	}
	native, err := OpenPool(mertPath, 1)
	if err != nil {
		t.Fatal(err)
	}
	nativeEmb, err := native.EmbedWaveform(wav)
	if err != nil {
		t.Fatal(err)
	}
	cos := Similarity(emb, nativeEmb)
	t.Logf("ORT vs native cosine: %.4f", cos)
	if cos < 0.99 {
		t.Fatalf("ORT/native cosine %.4f below 0.99", cos)
	}
}

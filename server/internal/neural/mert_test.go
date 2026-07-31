package neural

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWeights(t *testing.T) {
	path := modelPath(t)
	w, err := LoadWeights(path)
	if err != nil {
		t.Skip(err)
	}
	if w.Config.HiddenSize != 768 {
		t.Fatalf("hidden_size=%d", w.Config.HiddenSize)
	}
	if len(w.Tensors) < 200 {
		t.Fatalf("expected ~209 tensors, got %d", len(w.Tensors))
	}
}

func TestForwardMatchesGolden(t *testing.T) {
	path := modelPath(t)
	goldenDir := goldenPath(t)

	w, err := LoadWeights(path)
	if err != nil {
		t.Skip(err)
	}
	model, err := NewModel(w)
	if err != nil {
		t.Fatal(err)
	}

	wav := loadGoldenWave(t, goldenDir)
	got, seqLen, err := model.Forward(wav)
	if err != nil {
		t.Fatal(err)
	}

	wantHidden := loadGoldenNpy(t, filepath.Join(goldenDir, "golden_hidden.npy"))
	wantSeq := wantHidden.shape[1]
	wantDim := wantHidden.shape[2]
	if seqLen != wantSeq || model.cfg.HiddenSize != wantDim {
		t.Fatalf("shape mismatch got seq=%d dim=%d want seq=%d dim=%d", seqLen, model.cfg.HiddenSize, wantSeq, wantDim)
	}
	if len(got) != seqLen*wantDim {
		t.Fatalf("output len %d", len(got))
	}

	// int8 weights accumulate error across layers; embeddings remain close to f32.
	const hiddenTol = 0.55
	maxDiff := 0.0
	for i := range got {
		d := math.Abs(float64(got[i] - wantHidden.flat[i]))
		if d > maxDiff {
			maxDiff = d
		}
	}
	if maxDiff > hiddenTol {
		t.Fatalf("hidden max diff %.4f exceeds tol %.4f", maxDiff, hiddenTol)
	}

	gotEmb, err := model.Embed(wav)
	if err != nil {
		t.Fatal(err)
	}
	wantEmb := loadGoldenNpy(t, filepath.Join(goldenDir, "golden_emb.npy")).flat
	cos := float64(Similarity(gotEmb, wantEmb))
	if cos < 0.995 {
		t.Fatalf("embedding cosine %.4f below 0.995", cos)
	}
}

func TestSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if Similarity(a, b) < 0.99 {
		t.Fatal("expected ~1")
	}
	c := []float32{0, 1, 0}
	if Similarity(a, c) > 0.01 {
		t.Fatal("expected ~0")
	}
}

func modelPath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("MERT_MODEL_PATH"); p != "" {
		return p
	}
	candidates := []string{
		"/tmp/mert-work/mert-v1-95m.mert",
		filepath.Join("testdata", "mert-v1-95m.mert"),
	}
	for _, c := range candidates {
		if ModelExists(c) {
			return c
		}
	}
	t.Skip("MERT model not found; run scripts/convert_mert.py and set MERT_MODEL_PATH")
	return ""
}

func goldenPath(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("MERT_GOLDEN_DIR"); p != "" {
		return p
	}
	p := "/tmp/mert-work"
	if _, err := os.Stat(filepath.Join(p, "golden_wav.npy")); err == nil {
		return p
	}
	t.Skip("golden vectors not found in /tmp/mert-work; run python export in mert_test export")
	return ""
}

type npyArray struct {
	shape []int
	flat  []float32
}

func loadGoldenWave(t *testing.T, dir string) []float32 {
	arr := loadGoldenNpy(t, filepath.Join(dir, "golden_wav.npy"))
	if len(arr.shape) != 2 || arr.shape[0] != 1 {
		t.Fatalf("wav shape %v", arr.shape)
	}
	return arr.flat
}

func loadGoldenNpy(t *testing.T, path string) *npyArray {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Minimal npy v1.0 parser for little-endian float32 arrays saved by numpy.
	if len(b) < 10 || b[0] != 0x93 || string(b[1:6]) != "NUMPY" {
		t.Fatalf("not npy: %s", path)
	}
	headerLen := int(b[8]) | int(b[9])<<8
	header := string(b[10 : 10+headerLen])
	shape := parseNpyShape(header)
	offset := 10 + headerLen
	padding := (16 - (10+headerLen)%16) % 16
	offset += padding
	raw := b[offset:]
	n := 1
	for _, d := range shape {
		n *= d
	}
	flat := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := uint32(raw[i*4]) | uint32(raw[i*4+1])<<8 | uint32(raw[i*4+2])<<16 | uint32(raw[i*4+3])<<24
		flat[i] = math.Float32frombits(bits)
	}
	return &npyArray{shape: shape, flat: flat}
}

func stringsIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func parseNpyShape(header string) []int {
	// parse tuple like (1, 74, 768) or (1, 24000)
	start := stringsIndex(header, "'shape'")
	if start < 0 {
		return nil
	}
	rest := header[start:]
	paren := stringsIndex(rest, "(")
	end := stringsIndex(rest, ")")
	if paren < 0 || end <= paren {
		return nil
	}
	inner := rest[paren+1 : end]
	if inner == "" {
		return nil
	}
	parts := splitComma(inner)
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		var v int
		for _, ch := range p {
			if ch >= '0' && ch <= '9' {
				v = v*10 + int(ch-'0')
			}
		}
		out = append(out, v)
	}
	return out
}

func splitComma(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	parts = append(parts, trimSpace(s[start:]))
	return parts
}

func trimSpace(s string) string {
	for len(s) > 0 && (s[0] == ' ' || s[0] == '\n') {
		s = s[1:]
	}
	for len(s) > 0 && (s[len(s)-1] == ' ' || s[len(s)-1] == '\n') {
		s = s[:len(s)-1]
	}
	return s
}

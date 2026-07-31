package audio

import (
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNativeMatchesFFprobe walks a real library and checks every native parser
// against ffprobe. Set SPOOK_TEST_MUSIC to a music directory to run it.
func TestNativeMatchesFFprobe(t *testing.T) {
	root := os.Getenv("SPOOK_TEST_MUSIC")
	if root == "" {
		t.Skip("set SPOOK_TEST_MUSIC to a music directory to run this test")
	}

	prober := NewProber(true)
	if _, ok := prober.ffprobe(os.DevNull); ok {
		t.Fatal("expected ffprobe to reject /dev/null")
	}

	seen := map[string]int{}
	const perFormat = 5

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".flac", ".mp3", ".m4a", ".ogg", ".opus", ".wav":
		default:
			return nil
		}
		if seen[ext] >= perFormat {
			return nil
		}
		seen[ext]++

		native, nativeErr := probeNative(path)
		reference, ok := prober.ffprobe(path)
		if !ok {
			t.Logf("%s: ffprobe unavailable, skipping", path)
			return nil
		}
		if nativeErr != nil {
			t.Errorf("%s: native parser failed: %v", path, nativeErr)
			return nil
		}

		delta := math.Abs(float64(native.DurationMS - reference.DurationMS))
		if delta > 1500 {
			t.Errorf("%s: duration %dms, ffprobe %dms", path, native.DurationMS, reference.DurationMS)
		}
		if native.SampleRateHz != 0 && reference.SampleRateHz != 0 &&
			native.SampleRateHz != reference.SampleRateHz {
			t.Errorf("%s: sample rate %d, ffprobe %d", path, native.SampleRateHz, reference.SampleRateHz)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("checked formats: %v", seen)
}

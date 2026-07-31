package audio

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestStreamerStreamMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}

	src := "/tmp/spook-probe/out.wav"
	if _, err := os.Stat(src); err != nil {
		t.Skip("test audio not found at " + src)
	}
	abs, err := filepath.Abs(src)
	if err != nil {
		t.Fatal(err)
	}

	streamer := NewStreamer()
	if !streamer.Available() {
		t.Fatal("expected ffmpeg streamer to be available")
	}

	req := httptest.NewRequest(http.MethodGet, "/stream/x?transcode=1", nil)
	rec := httptest.NewRecorder()
	streamer.StreamMP3(rec, req, abs, 32*1024)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("content-type = %q, want audio/mpeg", got)
	}
	if got := rec.Header().Get("X-Spook-Transcoded"); got != "1" {
		t.Fatal("missing transcode marker header")
	}
	body := rec.Body.Bytes()
	if len(body) < 1024 {
		t.Fatalf("expected streamed MP3 body, got %d bytes", len(body))
	}
}

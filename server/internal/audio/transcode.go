package audio

import (
	"context"
	"io"
	"net/http"
	"os/exec"
	"sync"
)

// Streamer pipes ffmpeg output straight to the HTTP response so the browser
// can start playback without writing a temporary MP3 to disk.
type Streamer struct {
	lookup sync.Once
	path   string
}

func NewStreamer() *Streamer {
	return &Streamer{}
}

func (s *Streamer) Available() bool {
	s.lookup.Do(func() {
		path, err := exec.LookPath("ffmpeg")
		if err == nil {
			s.path = path
		}
	})
	return s.path != ""
}

// StreamMP3 transcodes path to MP3 on the fly. Range seeking is not supported
// on transcoded responses; the client should only request this when native
// playback is unavailable.
func (s *Streamer) StreamMP3(w http.ResponseWriter, r *http.Request, path string, chunkSize int64) {
	if !s.Available() {
		http.Error(w, "ffmpeg is required for transcoding", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "audio/mpeg")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Spook-Transcoded", "1")

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}

	args := []string{
		"-hide_banner", "-loglevel", "error", "-nostdin",
		"-i", path,
		"-vn",
		"-f", "mp3",
		"-b:a", "192k",
		"-write_xing", "0",
		"pipe:1",
	}

	cmd := exec.CommandContext(r.Context(), s.path, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "failed to start transcode", http.StatusInternalServerError)
		return
	}
	if err := cmd.Start(); err != nil {
		http.Error(w, "failed to start transcode", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	pipeResponse(r.Context(), w, stdout, chunkSize)

	_ = cmd.Wait()
}

func pipeResponse(ctx context.Context, w http.ResponseWriter, r io.Reader, chunkSize int64) {
	if chunkSize <= 0 {
		chunkSize = 256 * 1024
	}

	flusher, canFlush := w.(http.Flusher)
	buf := make([]byte, chunkSize)

	for {
		if err := ctx.Err(); err != nil {
			return
		}

		n, readErr := r.Read(buf)
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			if canFlush {
				flusher.Flush()
			}
		}
		if readErr != nil {
			return
		}
	}
}

package httpx

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ServeAudio streams a file with byte-range support, copying in chunks so a
// seek in the browser never forces the whole file through memory.
func ServeAudio(w http.ResponseWriter, r *http.Request, path string, chunkSize int64) {
	file, err := os.Open(path)
	if err != nil {
		http.Error(w, "failed to open track", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.Error(w, "failed to stat track", http.StatusInternalServerError)
		return
	}
	size := info.Size()

	contentType := audioContentType(path)
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Spook-Chunk-Size", strconv.FormatInt(chunkSize, 10))

	if chunkSize <= 0 {
		chunkSize = 256 * 1024
	}

	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
		copyChunks(w, file, size, chunkSize)
		return
	}

	start, end, err := parseRange(rangeHeader, size)
	if err != nil {
		w.Header().Set("Content-Range", fmt.Sprintf("bytes */%d", size))
		http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
		return
	}

	length := end - start + 1
	w.Header().Set("Content-Length", strconv.FormatInt(length, 10))
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, size))

	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusPartialContent)
		return
	}
	if _, err := file.Seek(start, io.SeekStart); err != nil {
		http.Error(w, "seek failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusPartialContent)
	copyChunks(w, file, length, chunkSize)
}

func copyChunks(w http.ResponseWriter, r io.Reader, total, chunkSize int64) {
	if chunkSize <= 0 {
		chunkSize = 256 * 1024
	}

	buf := make([]byte, chunkSize)
	remaining := total

	for remaining > 0 {
		toRead := chunkSize
		if remaining < toRead {
			toRead = remaining
		}
		n, readErr := r.Read(buf[:toRead])
		if n > 0 {
			if _, writeErr := w.Write(buf[:n]); writeErr != nil {
				return
			}
			remaining -= int64(n)
		}
		if readErr != nil {
			return
		}
	}
}

func parseRange(header string, size int64) (int64, int64, error) {
	if !strings.HasPrefix(header, "bytes=") {
		return 0, 0, fmt.Errorf("invalid range")
	}

	spec := strings.TrimPrefix(header, "bytes=")
	// Multi-range requests are legal but no audio element issues them.
	if comma := strings.Index(spec, ","); comma >= 0 {
		spec = spec[:comma]
	}

	from, to, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, fmt.Errorf("invalid range")
	}

	if from == "" {
		suffix, err := strconv.ParseInt(to, 10, 64)
		if err != nil || suffix <= 0 {
			return 0, 0, fmt.Errorf("invalid range")
		}
		if suffix > size {
			suffix = size
		}
		return size - suffix, size - 1, nil
	}

	start, err := strconv.ParseInt(from, 10, 64)
	if err != nil || start < 0 || start >= size {
		return 0, 0, fmt.Errorf("invalid range")
	}

	end := size - 1
	if to != "" {
		end, err = strconv.ParseInt(to, 10, 64)
		if err != nil || end < start {
			return 0, 0, fmt.Errorf("invalid range")
		}
		if end >= size {
			end = size - 1
		}
	}

	return start, end, nil
}

func audioContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".mp3":
		return "audio/mpeg"
	case ".flac":
		return "audio/flac"
	case ".ogg", ".oga":
		return "audio/ogg"
	case ".opus":
		return "audio/ogg; codecs=opus"
	case ".m4a", ".mp4":
		return "audio/mp4"
	case ".aac":
		return "audio/aac"
	case ".wav":
		return "audio/wav"
	}
	if guess := mime.TypeByExtension(ext); guess != "" {
		return guess
	}
	return "application/octet-stream"
}

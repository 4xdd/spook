// Package audio derives stream properties (duration, bitrate, sample rate)
// that tag readers do not expose.
//
// Native parsers handle the common formats by reading a few headers, which
// keeps a full library scan cheap. ffprobe is only consulted for what is left.
package audio

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var errUnsupported = errors.New("audio: no native parser for this format")

type Info struct {
	DurationMS   int64
	BitrateKbps  int
	SampleRateHz int
	Channels     int
}

type Prober struct {
	allowFFprobe bool
	lookup       sync.Once
	ffprobePath  string
}

func NewProber(allowFFprobe bool) *Prober {
	return &Prober{allowFFprobe: allowFFprobe}
}

// Probe never fails: an unreadable header simply yields a zero duration.
func (p *Prober) Probe(path string, size int64) Info {
	info, err := probeNative(path)
	if err != nil || info.DurationMS <= 0 {
		if fallback, ok := p.ffprobe(path); ok {
			if info.DurationMS <= 0 {
				info.DurationMS = fallback.DurationMS
			}
			if info.SampleRateHz == 0 {
				info.SampleRateHz = fallback.SampleRateHz
			}
			if info.Channels == 0 {
				info.Channels = fallback.Channels
			}
			if info.BitrateKbps == 0 {
				info.BitrateKbps = fallback.BitrateKbps
			}
		}
	}

	if info.BitrateKbps == 0 && info.DurationMS > 0 && size > 0 {
		info.BitrateKbps = int(size * 8 / info.DurationMS)
	}
	return info
}

func probeNative(path string) (Info, error) {
	f, err := os.Open(path)
	if err != nil {
		return Info{}, err
	}
	defer f.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".flac":
		return probeFLAC(f)
	case ".wav":
		return probeWAV(f)
	case ".mp3":
		return probeMP3(f)
	case ".m4a", ".mp4", ".aac", ".m4b":
		return probeMP4(f)
	case ".ogg", ".oga", ".opus":
		return probeOgg(f)
	default:
		return Info{}, errUnsupported
	}
}

func (p *Prober) ffprobe(path string) (Info, bool) {
	if !p.allowFFprobe {
		return Info{}, false
	}
	p.lookup.Do(func() {
		if found, err := exec.LookPath("ffprobe"); err == nil {
			p.ffprobePath = found
		}
	})
	if p.ffprobePath == "" {
		return Info{}, false
	}

	out, err := exec.Command(p.ffprobePath,
		"-v", "quiet",
		"-select_streams", "a:0",
		"-show_entries", "format=duration,bit_rate:stream=sample_rate,channels,duration",
		"-of", "default=noprint_wrappers=1",
		path).Output()
	if err != nil {
		return Info{}, false
	}

	var info Info
	for _, line := range strings.Split(string(out), "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok || value == "" || value == "N/A" {
			continue
		}
		switch key {
		case "duration":
			if info.DurationMS == 0 {
				info.DurationMS = int64(parseFloat(value) * 1000)
			}
		case "bit_rate":
			info.BitrateKbps = parseInt(value) / 1000
		case "sample_rate":
			info.SampleRateHz = parseInt(value)
		case "channels":
			info.Channels = parseInt(value)
		}
	}
	return info, info.DurationMS > 0
}

func parseFloat(v string) float64 {
	var result float64
	var frac float64 = 1
	seenDot := false
	for _, r := range v {
		switch {
		case r == '.':
			seenDot = true
		case r >= '0' && r <= '9':
			if seenDot {
				frac /= 10
				result += float64(r-'0') * frac
			} else {
				result = result*10 + float64(r-'0')
			}
		default:
			return result
		}
	}
	return result
}

func parseInt(v string) int {
	result := 0
	for _, r := range v {
		if r < '0' || r > '9' {
			break
		}
		result = result*10 + int(r-'0')
	}
	return result
}

// skipID3 reports the offset at which audio data begins, stepping over an
// ID3v2 tag when one is present.
func skipID3(r io.ReadSeeker) int64 {
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0
	}
	header := make([]byte, 10)
	if _, err := io.ReadFull(r, header); err != nil {
		return 0
	}
	if string(header[0:3]) != "ID3" {
		return 0
	}
	// Syncsafe integer: 7 significant bits per byte.
	size := int64(header[6]&0x7f)<<21 | int64(header[7]&0x7f)<<14 |
		int64(header[8]&0x7f)<<7 | int64(header[9]&0x7f)
	return size + 10
}

func fileSize(r io.Seeker) int64 {
	size, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return 0
	}
	return size
}

func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
func be64(b []byte) uint64 { return binary.BigEndian.Uint64(b) }
func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func le32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func le64(b []byte) uint64 { return binary.LittleEndian.Uint64(b) }

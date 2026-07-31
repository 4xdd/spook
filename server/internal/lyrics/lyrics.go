// Package lyrics finds and parses the lyrics that ship with an audio file,
// either as a sidecar file next to it or embedded in its tags.
package lyrics

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/spook/server/internal/tags"
)

// Sources, in the order they are tried.
const (
	SourceSidecar  = "sidecar"
	SourceEmbedded = "embedded"
)

// maxSidecarBytes stops a stray large text file from being read into memory.
// Even a densely timed song lands well under this.
const maxSidecarBytes = 512 << 10

// Line is one lyric line. TimeMs is -1 when the lyrics carry no timing.
type Line struct {
	TimeMs int64
	Text   string
}

// Lyrics is the parsed result. Lines is empty when a track has no lyrics.
type Lyrics struct {
	Source string
	Synced bool
	Lines  []Line
}

func (l Lyrics) Empty() bool { return len(l.Lines) == 0 }

// sidecarExts are the companion files a track may sit next to. .lrc comes
// first because it is the one that usually carries timings.
var sidecarExts = []string{".lrc", ".txt"}

// Find returns the lyrics for an audio file, preferring a sidecar over the
// file's own tags: a sidecar is what the user deliberately put there.
func Find(path string) Lyrics {
	if text, ok := readSidecar(path); ok {
		if parsed := Parse(text); !parsed.Empty() {
			parsed.Source = SourceSidecar
			return parsed
		}
	}
	if text := tags.Lyrics(path); text != "" {
		parsed := Parse(text)
		parsed.Source = SourceEmbedded
		return parsed
	}
	return Lyrics{}
}

func readSidecar(path string) (string, bool) {
	stem := strings.TrimSuffix(path, filepath.Ext(path))
	for _, ext := range sidecarExts {
		candidate := stem + ext
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() || info.Size() == 0 || info.Size() > maxSidecarBytes {
			continue
		}
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		return string(data), true
	}
	return "", false
}

// timestamp matches an LRC time tag: [mm:ss], [mm:ss.xx] or the older
// [mm:ss:xx] hundredths form.
var timestamp = regexp.MustCompile(`^\[(\d{1,3}):([0-5]?\d)(?:[.:](\d{1,3}))?\]`)

// metaTag matches the [ar:...] style header lines an LRC file may open with.
var metaTag = regexp.MustCompile(`^\[[a-zA-Z#]+:[^\]]*\]`)

// wordTimings matches the <mm:ss.xx> marks enhanced LRC puts inside a line.
var wordTimings = regexp.MustCompile(`<\d{1,3}:[0-5]?\d(?:[.:]\d{1,3})?>`)

var extraSpace = regexp.MustCompile(` {2,}`)

// Parse reads plain or LRC-formatted lyrics. Timed lines are returned in time
// order; untimed lyrics keep the order they were written in, blank lines and
// all, so verse breaks survive.
func Parse(text string) Lyrics {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var timed []Line
	var plain []Line

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)

		var times []int64
		for {
			match := timestamp.FindStringSubmatch(line)
			if match == nil {
				break
			}
			times = append(times, toMillis(match[1], match[2], match[3]))
			line = line[len(match[0]):]
		}

		if len(times) == 0 {
			// A header tag carries no lyric of its own.
			if metaTag.MatchString(line) {
				continue
			}
			plain = append(plain, Line{TimeMs: -1, Text: clean(line)})
			continue
		}

		// One timestamp per repeat of a chorus line.
		body := clean(line)
		for _, at := range times {
			timed = append(timed, Line{TimeMs: at, Text: body})
		}
	}

	if len(timed) > 0 {
		sort.SliceStable(timed, func(i, j int) bool { return timed[i].TimeMs < timed[j].TimeMs })
		return Lyrics{Synced: true, Lines: timed}
	}

	return Lyrics{Lines: trimBlankEdges(plain)}
}

func clean(line string) string {
	if wordTimings.MatchString(line) {
		// Removing a mark that sat between two spaces would leave a gap behind.
		line = extraSpace.ReplaceAllString(wordTimings.ReplaceAllString(line, ""), " ")
	}
	return strings.TrimSpace(line)
}

func toMillis(minutes, seconds, fraction string) int64 {
	m, _ := strconv.ParseInt(minutes, 10, 64)
	s, _ := strconv.ParseInt(seconds, 10, 64)
	total := m*60_000 + s*1000

	if fraction == "" {
		return total
	}
	value, _ := strconv.ParseInt(fraction, 10, 64)
	// Two digits are hundredths, three are already milliseconds.
	switch len(fraction) {
	case 1:
		value *= 100
	case 2:
		value *= 10
	}
	return total + value
}

// trimBlankEdges drops the leading and trailing empty lines that tags and text
// files pick up, without touching the blank lines that separate verses.
func trimBlankEdges(lines []Line) []Line {
	start, end := 0, len(lines)
	for start < end && lines[start].Text == "" {
		start++
	}
	for end > start && lines[end-1].Text == "" {
		end--
	}
	if start == end {
		return nil
	}
	return lines[start:end]
}

package lyrics

import (
	"regexp"
	"strconv"
	"strings"
)

// lrcLineRe matches discify's synced timestamp format: [mm:ss.xx] text
var lrcLineRe = regexp.MustCompile(`^\[(\d{2}):(\d{2})\.(\d{2,3})\]\s?(.*)$`)

// ParseLRC reads LRCLIB syncedLyrics into timed lines for playback sync.
func ParseLRC(lrc string) Lyrics {
	var lines []Line
	for _, raw := range strings.Split(lrc, "\n") {
		raw = strings.TrimSpace(raw)
		m := lrcLineRe.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		min, _ := strconv.Atoi(m[1])
		sec, _ := strconv.Atoi(m[2])
		frac, _ := strconv.Atoi(m[3])
		if len(m[3]) == 2 {
			frac *= 10
		}
		ms := int64(min*60000 + sec*1000 + frac)
		lines = append(lines, Line{TimeMs: ms, Text: m[4]})
	}
	if len(lines) == 0 {
		return Lyrics{}
	}
	return Lyrics{Synced: true, Lines: lines}
}

func plainLines(text string) Lyrics {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var lines []Line
	for _, raw := range strings.Split(text, "\n") {
		lines = append(lines, Line{TimeMs: -1, Text: strings.TrimSpace(raw)})
	}
	return Lyrics{Lines: trimBlankEdges(lines)}
}

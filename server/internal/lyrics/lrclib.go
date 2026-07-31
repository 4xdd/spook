package lyrics

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const SourceLRCLIB = "lrclib"

const lrclibBase = "https://lrclib.net"

// Online fetches synced lyrics from LRCLIB, matching discify's /api/get lookup.
type Online struct {
	enabled bool
	client  *http.Client

	mu    sync.RWMutex
	cache map[string]Lyrics
}

func NewOnline(enabled bool, _ string) *Online {
	return &Online{
		enabled: enabled,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: make(map[string]Lyrics),
	}
}

func (o *Online) Enabled() bool { return o.enabled }

func (o *Online) Fetch(ctx context.Context, track Track) Lyrics {
	if !o.enabled || track.Title == "" || track.Artist == "" {
		return Lyrics{}
	}

	key := cacheKey(track)
	o.mu.RLock()
	if cached, ok := o.cache[key]; ok {
		o.mu.RUnlock()
		return cached
	}
	o.mu.RUnlock()

	found := o.fetch(ctx, track)

	o.mu.Lock()
	o.cache[key] = found
	o.mu.Unlock()

	return found
}

type lrclibResponse struct {
	SyncedLyrics string `json:"syncedLyrics"`
	PlainLyrics  string `json:"plainLyrics"`
}

func (o *Online) fetch(ctx context.Context, track Track) Lyrics {
	params := url.Values{
		"track_name":  {track.Title},
		"artist_name": {track.Artist},
	}
	if album := strings.TrimSpace(track.Album); album != "" {
		params.Set("album_name", album)
	}
	if track.DurationMS > 0 {
		params.Set("duration", strconv.Itoa(int(math.Round(float64(track.DurationMS)/1000))))
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, lrclibBase+"/api/get?"+params.Encode(), nil)
	if err != nil {
		return Lyrics{}
	}
	req.Header.Set("User-Agent", "discify/1.0")

	resp, err := o.client.Do(req)
	if err != nil {
		return Lyrics{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Lyrics{}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSidecarBytes))
	if err != nil {
		return Lyrics{}
	}

	var lrc lrclibResponse
	if err := json.Unmarshal(body, &lrc); err != nil {
		return Lyrics{}
	}

	if lrc.SyncedLyrics != "" {
		parsed := ParseLRC(lrc.SyncedLyrics)
		if !parsed.Empty() {
			parsed.Source = SourceLRCLIB
			return parsed
		}
	}

	if lrc.PlainLyrics != "" {
		parsed := plainLines(lrc.PlainLyrics)
		parsed.Source = SourceLRCLIB
		return parsed
	}

	return Lyrics{}
}

func cacheKey(track Track) string {
	return strings.ToLower(strings.Join([]string{
		track.Title,
		track.Artist,
		track.Album,
		strconv.FormatInt(track.DurationMS, 10),
	}, "|"))
}

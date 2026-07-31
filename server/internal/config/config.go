// Package config resolves runtime configuration from flags, environment
// variables and platform defaults.
package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	MusicDir       string
	Addr           string
	ChunkSize      int64
	DataDir        string
	ArtDir         string
	WebDir         string
	AccessKeysFile string
	AccessKeys     []string
	OpenBrowser    bool
	ScanOnStart    bool
	UseFFprobe     bool
	Deezer         DeezerConfig
	Lyrics         LyricsConfig
	LastFM         LastFMConfig
}

type LyricsConfig struct {
	Enabled bool
	BaseURL string
}

type LastFMConfig struct {
	APIKey string
	Secret string
}

type DeezerConfig struct {
	Enabled bool
	ARL     string
	URL     string
	Port    int
	Quality string
	Command string
}

func Load() Config {
	loadDotEnv()

	musicDir := envOr("MUSIC_DIR", "/Music")
	addr := envOr("SPOOK_ADDR", ":8080")
	dataDir := envOr("SPOOK_DATA_DIR", defaultDataDir())
	artDir := envOr("SPOOK_ART_DIR", defaultArtDir())
	webDir := envOr("SPOOK_WEB_DIR", "")

	chunkSize := int64(256 * 1024)
	if v := os.Getenv("SPOOK_CHUNK_SIZE"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 0 {
			chunkSize = n
		}
	}

	flag.StringVar(&musicDir, "music-dir", musicDir, "root directory containing music files")
	flag.StringVar(&addr, "addr", addr, "HTTP listen address")
	flag.StringVar(&dataDir, "data-dir", dataDir, "directory for the library database")
	flag.StringVar(&artDir, "art-dir", artDir, "directory for the cover art cache")
	flag.StringVar(&webDir, "web-dir", webDir, "serve the web UI from this directory instead of the embedded build")
	flag.Int64Var(&chunkSize, "chunk-size", chunkSize, "preferred streaming chunk size in bytes")
	openBrowser := false
	flag.BoolVar(&openBrowser, "open", openBrowser, "open the web UI in the default browser on startup")
	scanOnStart := true
	flag.BoolVar(&scanOnStart, "scan", scanOnStart, "scan the music directory on startup")
	useFFprobe := true
	flag.BoolVar(&useFFprobe, "ffprobe", useFFprobe, "fall back to ffprobe for formats without a native duration parser")

	deezerEnabled := envOr("SPOOK_DEEZER", "") != "0" && envOr("SPOOK_DEEZER", "") != "false"
	deezerARL := envOr("DEEZER_ARL", envOr("DEEZER_COOKIE_ARL", ""))
	deezerURL := envOr("SPOOK_DEEZER_URL", "")
	deezerPort := 5001
	if v := os.Getenv("SPOOK_DEEZER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			deezerPort = n
		}
	}
	deezerQuality := envOr("DEEZER_QUALITY", "mp3")
	deezerCommand := envOr("SPOOK_DEEZER_COMMAND", "")

	lyricsEnabled := envOr("SPOOK_LYRICS", "") != "0" && envOr("SPOOK_LYRICS", "") != "false"
	lyricsURL := envOr("SPOOK_LYRICS_URL", "")

	flag.BoolVar(&deezerEnabled, "deezer", deezerEnabled, "enable Deezer search and download subworker")
	flag.StringVar(&deezerARL, "deezer-arl", deezerARL, "Deezer ARL cookie (or set DEEZER_ARL)")
	flag.StringVar(&deezerURL, "deezer-url", deezerURL, "existing deezer-downloader base URL (skip spawning a child process)")
	flag.IntVar(&deezerPort, "deezer-port", deezerPort, "local deezer-downloader listen port when spawned by Spook")
	flag.StringVar(&deezerQuality, "deezer-quality", deezerQuality, "download quality: mp3 or flac")
	flag.StringVar(&deezerCommand, "deezer-command", deezerCommand, "path to deezer-downloader executable")

	flag.BoolVar(&lyricsEnabled, "lyrics", lyricsEnabled, "fetch lyrics from LRCLIB when not embedded in the file")
	flag.StringVar(&lyricsURL, "lyrics-url", lyricsURL, "LRCLIB API base URL (default https://lrclib.net)")

	lastfmKey := envOr("LASTFM_API_KEY", "")
	lastfmSecret := envOr("LASTFM_API_SECRET", "")
	flag.StringVar(&lastfmKey, "lastfm-key", lastfmKey, "Last.fm API key (or set LASTFM_API_KEY)")
	flag.StringVar(&lastfmSecret, "lastfm-secret", lastfmSecret, "Last.fm API secret (or set LASTFM_API_SECRET)")

	accessKeysFile := envOr("SPOOK_ACCESS_KEYS_FILE", "")
	accessKeysEnv := envOr("SPOOK_ACCESS_KEYS", "")
	flag.StringVar(&accessKeysFile, "access-keys-file", accessKeysFile, "file of access keys, one per line (default: <data-dir>/access_keys)")
	flag.StringVar(&accessKeysEnv, "access-keys", accessKeysEnv, "comma-separated access keys (or set SPOOK_ACCESS_KEYS)")

	flag.Parse()

	if abs, err := filepath.Abs(musicDir); err == nil {
		musicDir = abs
	}
	if accessKeysFile == "" {
		accessKeysFile = filepath.Join(dataDir, "access_keys")
	}

	return Config{
		MusicDir:       musicDir,
		Addr:           addr,
		ChunkSize:      chunkSize,
		DataDir:        dataDir,
		ArtDir:         artDir,
		WebDir:         webDir,
		AccessKeysFile: accessKeysFile,
		AccessKeys:     splitCSV(accessKeysEnv),
		OpenBrowser:    openBrowser,
		ScanOnStart:    scanOnStart,
		UseFFprobe:     useFFprobe,
		Deezer: DeezerConfig{
			Enabled: deezerEnabled,
			ARL:     deezerARL,
			URL:     deezerURL,
			Port:    deezerPort,
			Quality: deezerQuality,
			Command: deezerCommand,
		},
		Lyrics: LyricsConfig{
			Enabled: lyricsEnabled,
			BaseURL: lyricsURL,
		},
		LastFM: LastFMConfig{
			APIKey: lastfmKey,
			Secret: lastfmSecret,
		},
	}
}

func (c Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "library.db")
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if key := strings.TrimSpace(part); key != "" {
			out = append(out, key)
		}
	}
	return out
}

func defaultDataDir() string {
	if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
		return filepath.Join(dir, "spook")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ".spook"
	}
	return filepath.Join(home, ".local", "share", "spook")
}

func defaultArtDir() string {
	if dir := os.Getenv("XDG_CACHE_HOME"); dir != "" {
		return filepath.Join(dir, "spook", "art")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".spook", "art")
	}
	return filepath.Join(home, ".cache", "spook", "art")
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

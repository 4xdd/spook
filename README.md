# Spook

A local music streaming system: a **Go server** indexes your music folder, extracts album art, and serves an **Apple Music-style web player** built with React and Tailwind.

## Architecture

```
/Music  ──scan──▶  Go server (server/)
                     • SQLite library index + FTS5 search
                     • embedded cover art + folder image fallback
                     • HTTP range/chunk streaming
                     • serve web UI (embedded React SPA)
                           │
                           ▼
                     Browser
```

The browser uses native `<audio>` streaming — no buffering the whole file.

## Quick start

### Pre-built binary (no Go or Node required)

Download the archive for your platform from [GitHub Releases](https://github.com/4xdd/spook/releases), extract it, and run:

```bash
chmod +x spook          # Linux / macOS only
./spook -music-dir ~/Music -open
```

On Windows: `spook.exe -music-dir %USERPROFILE%\Music -open`

Copy `.env.example` to `.env` beside the binary to set `MUSIC_DIR`, Deezer, Last.fm, and other options. Spook also reads `.env` from the current working directory.

See `INSTALL.txt` in the release archive for a short reference.

### From source

```bash
make run MUSIC_DIR=~/Music
```

Then open **http://localhost:8080/** (or use `-open` to launch your browser automatically).

On first launch Spook creates an access key in `$SPOOK_DATA_DIR/access_keys` and prints it in the server log. Enter that key in the browser gate; it is stored only in this browser’s `localStorage`. Every `/api` request (including artwork and audio streams) requires a valid key.

## Web UI

The player includes:

- Library sidebar with search
- Recently Added, Artists, Albums, and Songs views
- Album detail with real cover art, Play/Shuffle, and track list
- Bottom player bar with seek, shuffle, repeat, and queue
- Full-screen Now Playing with ambient artwork tint
- Settings pane with theme switch, six-band equalizer, and Last.fm scrobbling
- Keyboard shortcuts (`Space`, arrows, `/` for search, `,` for settings, `m` mute, `s` shuffle, `r` repeat)

Frontend source lives in [`web/`](web/). The production build is embedded into the Go binary via `go:embed`.

### Equalizer

Open **Settings** in the sidebar (or press `,`) for a six-band graphic equalizer, built on Web Audio biquad filters wired between the `<audio>` element and the output.

- Bands at 60, 150, 400 Hz and 1K, 2.4K, 15K Hz, each adjustable ±12 dB
- Drag a point on the curve to shape it, double-click a point to zero it, or use arrow keys
- Spotify's preset list (Bass booster, Vocal booster, Rock, Loudness, …); editing a preset switches it to *Custom*
- Settings are per-browser: they persist in `localStorage` under `spook.eq.v1`, so nothing is sent to the server

The filter chain is created lazily on first use and stays bypassed at 0 dB — which is mathematically transparent — while the equalizer is switched off.

### Last.fm scrobbling

1. Create an API account at [last.fm/api/account/create](https://www.last.fm/api/account/create)
2. Put the key and secret in `.env`:

```bash
LASTFM_API_KEY=…
LASTFM_API_SECRET=…
```

3. Restart Spook, open **Settings → Last.fm → Connect Last.fm**, and authorize in the browser

The API secret stays on the server. Your session (`username` + session key) is stored only in this browser under `spook.lastfm.v1`. Spook sends Now Playing when a track starts, and scrobbles once you’ve heard half the track or 4 minutes (tracks under 30s are skipped), matching Last.fm’s rules. Failed scrobbles queue in `localStorage` and retry later.

## Server (Go)

### Requirements

- Go 1.24+ (auto-downloaded via `GOTOOLCHAIN=auto` if needed)
- Node.js 18+ (for building and running the web UI dev server)

### Build

```bash
make build          # build UI + server binary → bin/spook
make run            # build and run
make release        # cross-compile binaries → dist/releases/
make package-release # release binaries + .tar.gz / .zip archives
make dev            # Vite dev server + Go API (UI served from disk)
make test           # run Go tests
```

### Run directly

```bash
cd server
go run ./cmd/spook -music-dir ~/Music -addr :8080 -open
```

### Configuration

| Flag / env | Default | Description |
|---|---|---|
| `-music-dir` / `MUSIC_DIR` | `/Music` | Root folder to scan |
| `-addr` / `SPOOK_ADDR` | `:8080` | HTTP listen address |
| `-data-dir` / `SPOOK_DATA_DIR` | `~/.local/share/spook` | SQLite database directory |
| `-art-dir` / `SPOOK_ART_DIR` | `~/.cache/spook/art` | Cover art cache |
| `-web-dir` / `SPOOK_WEB_DIR` | _(empty)_ | Serve UI from disk instead of embedded build |
| `-chunk-size` / `SPOOK_CHUNK_SIZE` | `262144` | Preferred streaming chunk size (bytes) |
| `-scan` | `true` | Scan music directory on startup |
| `-ffprobe` | `true` | Fall back to ffprobe for duration probing |
| `-open` | `false` | Open web UI in default browser on startup |
| `-deezer` / `SPOOK_DEEZER` | `true` | Enable Deezer search and download subworker |
| `-deezer-arl` / `DEEZER_ARL` | _(empty)_ | Deezer ARL cookie (also loaded from `.env`) |
| `-deezer-url` / `SPOOK_DEEZER_URL` | _(empty)_ | Use an existing [deezer-downloader](https://github.com/kmille/deezer-downloader) instance instead of spawning one |
| `-deezer-port` / `SPOOK_DEEZER_PORT` | `5001` | Local port when Spook spawns deezer-downloader |
| `-deezer-quality` / `DEEZER_QUALITY` | `mp3` | Download quality: `mp3` or `flac` (flac needs Premium) |
| `-deezer-command` / `SPOOK_DEEZER_COMMAND` | _(PATH)_ | Path to `deezer-downloader` executable |
| `-lastfm-key` / `LASTFM_API_KEY` | _(empty)_ | Last.fm API key for scrobbling |
| `-lastfm-secret` / `LASTFM_API_SECRET` | _(empty)_ | Last.fm API secret (server-side signing) |
| `-access-keys-file` / `SPOOK_ACCESS_KEYS_FILE` | `$SPOOK_DATA_DIR/access_keys` | Local file of access keys (one per line) |
| `-access-keys` / `SPOOK_ACCESS_KEYS` | _(empty)_ | Extra comma-separated access keys |

Data paths:

- Library database: `$SPOOK_DATA_DIR/library.db`
- Art cache: `$SPOOK_ART_DIR/<hash>-{64,300,1000}.jpg`
- Access keys: `$SPOOK_DATA_DIR/access_keys` (mode `0600`; created with a random key if missing)

### Access keys

All `/api/v1/*` routes require a key via:

- `Authorization: Bearer <key>`, or
- `X-Spook-Access-Key: <key>`, or
- `?key=<key>` (used by `<audio>` / `<img>` URLs)

`/health` and the web UI assets stay open so the unlock screen can load. Add more keys by editing the access-keys file (one per line) or setting `SPOOK_ACCESS_KEYS`. Restart after changing the file. In the player, **Settings → Access → Lock this browser** clears the stored key.

### Deezer downloads

Spook can search Deezer and download tracks/albums into your music folder using [kmille/deezer-downloader](https://github.com/kmille/deezer-downloader) as a local subworker.

**Requirements**

```bash
pip install deezer-downloader   # also installs yt-dlp dependency path check
# yt-dlp must exist (e.g. apt install yt-dlp)
```

**Setup**

1. Log into [deezer.com](https://www.deezer.com) in your browser.
2. Copy your `arl` cookie value (192-character hex string).
3. Copy `.env.example` to `.env` and paste your `arl` cookie (or edit the sample `.env` already in the repo root):

```bash
cp .env.example .env   # if needed
# edit .env → set DEEZER_ARL=...
make run
```

Spook loads `.env` automatically from the project root. Existing shell environment variables take precedence.

Or pass it inline:

```bash
DEEZER_ARL='your-arl-cookie' make run MUSIC_DIR=~/Music
```

Downloads land under your music directory: albums are organized into `Album Name/track files` (for example `Yeezus/01 - ….mp3`), and single tracks go in `songs/`. Spook rescans the library automatically when a download batch finishes.

Use **Add from Deezer** in the sidebar, or the API:

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/deezer/status` | Subworker status |
| `GET` | `/api/v1/deezer/search?q=&type=track\|album\|artist` | Search Deezer |
| `POST` | `/api/v1/deezer/download` | Queue download (`{"type":"track\|album","musicId":123}`) |
| `GET` | `/api/v1/deezer/jobs` | Download queue |
| `GET` | `/api/v1/lastfm/status` | Whether Last.fm API credentials are configured |
| `GET` | `/api/v1/lastfm/auth-url?callback=` | Last.fm authorize URL |
| `POST` | `/api/v1/lastfm/session` | Exchange auth token for session (`{"token"}`) |
| `POST` | `/api/v1/lastfm/now-playing` | Update Now Playing (session key + track) |
| `POST` | `/api/v1/lastfm/scrobble` | Submit a scrobble (session key + track + timestamp) |

To attach an already-running deezer-downloader instead of spawning one, set `SPOOK_DEEZER_URL=http://127.0.0.1:5000`.

### API (v1)

| Method | Path | Description |
|---|---|---|
| `GET` | `/` | Web player UI |
| `GET` | `/health` | Health check |
| `GET` | `/api/v1/stats` | Library stats + scan progress |
| `GET` | `/api/v1/albums` | Album list (`?sort=title\|artist\|year\|recent`) |
| `GET` | `/api/v1/albums/{id}` | Album with tracks |
| `GET` | `/api/v1/artists` | Artist list |
| `GET` | `/api/v1/artists/{id}` | Artist with albums and tracks |
| `GET` | `/api/v1/tracks` | Track list (`?sort=title\|artist\|album\|recent`) |
| `GET` | `/api/v1/search?q=` | Full-text search |
| `GET` | `/api/v1/art/{id}?size=64\|300\|1000` | Cover art (cached, immutable) |
| `GET\|HEAD` | `/api/v1/stream/{id}` | Stream audio (`Accept-Ranges: bytes`) |
| `POST` | `/api/v1/scan` | Start background rescan |
| `GET` | `/api/v1/scan` | Scan progress |

Supported formats: MP3, FLAC, OGG, M4A, WAV, AAC, Opus (browser playback depends on codec support).

Track IDs are the first 16 hex characters of `sha256(absolute path)`.

### Artwork

Cover art is resolved per album:

1. Embedded picture from audio tags (preferred)
2. Folder image: `cover.jpg`, `folder.png`, etc. (also checks parent folder for multi-disc albums)

Images are deduplicated by content hash, resized to 64/300/1000px, and cached on disk. Dominant color is computed for UI tinting.

## Development

```bash
# Terminal 1: Go API (serves built UI from disk)
make dev-server MUSIC_DIR=~/Music

# Terminal 2: Vite with hot reload (proxies /api to :8080)
make dev-ui
```

Or run both together:

```bash
make dev MUSIC_DIR=~/Music
```

After changing the frontend, rebuild the embedded UI:

```bash
make build-ui
```

### Troubleshooting

**Port already in use**

```bash
# Option 1: free the port
fuser -k 8080/tcp

# Option 2: use another port (Vite proxy follows ADDR automatically)
make dev ADDR=:8081
```

**Vite fails with `styleText` / Node version errors**

You need Node 18 or newer. Check with `node -v`. The project uses Vite 6, which does not run on Node 16 or below.

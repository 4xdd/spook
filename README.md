# Spook

Stream your own music library. Go server + React player that looks a bit like Apple Music.

Point it at a folder, it indexes everything into SQLite, caches cover art, and serves the UI from the same binary.

## Run it

**Release binary** — grab one from [Releases](https://github.com/4xdd/spook/releases):

```bash
./spook -music-dir ~/Music -open
```

**From source:**

```bash
make run MUSIC_DIR=~/Music
```

First launch prints an access key. Paste it into the unlock screen. Key lives in `$SPOOK_DATA_DIR/access_keys` (default `~/.local/share/spook`).

Copy `.env.example` → `.env` for Deezer / Last.fm / etc. Flags and env vars both work; see `./spook -h`.

## Dev

```bash
make dev MUSIC_DIR=~/Music   # Vite + Go
make build                   # → bin/spook
make test
```

UI is in `web/`. Server is in `server/`.

## Bits worth knowing

- Formats: MP3, FLAC, OGG, M4A, WAV, AAC, Opus
- Lyrics from sidecar `.lrc`, tags, or LRCLIB
- Optional Deezer downloads via [deezer-downloader](https://github.com/kmille/deezer-downloader) (`DEEZER_ARL` in `.env`)
- Optional Last.fm scrobbling (`LASTFM_API_KEY` / `LASTFM_API_SECRET`)
- Similarity recs need a MERT model: `make convert-mert` (or `convert-mert-onnx` + `install-ort` for the fast path)
- EQ, theme, Last.fm session — all browser `localStorage`, nothing sent to the server for those

Port busy? `make run ADDR=:8081`

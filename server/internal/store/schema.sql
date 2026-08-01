CREATE TABLE IF NOT EXISTS meta (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS artwork (
  id      TEXT PRIMARY KEY,
  mime    TEXT NOT NULL,
  width   INTEGER NOT NULL,
  height  INTEGER NOT NULL,
  color   TEXT NOT NULL,
  is_dark INTEGER NOT NULL,
  source  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tracks (
  id             TEXT PRIMARY KEY,
  path           TEXT NOT NULL UNIQUE,
  filename       TEXT NOT NULL,
  title          TEXT NOT NULL,
  sort_title     TEXT NOT NULL,
  artist         TEXT NOT NULL,
  album_artist   TEXT NOT NULL,
  artist_id      TEXT NOT NULL,
  album_id       TEXT NOT NULL,
  album_name     TEXT NOT NULL,
  genre          TEXT NOT NULL DEFAULT '',
  year           INTEGER NOT NULL DEFAULT 0,
  track_no       INTEGER NOT NULL DEFAULT 0,
  disc_no        INTEGER NOT NULL DEFAULT 0,
  duration_ms    INTEGER NOT NULL DEFAULT 0,
  bitrate_kbps   INTEGER NOT NULL DEFAULT 0,
  sample_rate_hz INTEGER NOT NULL DEFAULT 0,
  channels       INTEGER NOT NULL DEFAULT 0,
  format         TEXT NOT NULL,
  size_bytes     INTEGER NOT NULL,
  mod_time       INTEGER NOT NULL,
  artwork_id     TEXT,
  added_at       INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS tracks_album_idx  ON tracks(album_id, disc_no, track_no);
CREATE INDEX IF NOT EXISTS tracks_artist_idx ON tracks(artist_id);
CREATE INDEX IF NOT EXISTS tracks_sort_idx   ON tracks(sort_title);

CREATE TABLE IF NOT EXISTS track_artists (
  track_id    TEXT NOT NULL REFERENCES tracks(id) ON DELETE CASCADE,
  artist_id   TEXT NOT NULL,
  artist_name TEXT NOT NULL,
  position    INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (track_id, artist_id)
);

CREATE INDEX IF NOT EXISTS track_artists_artist_idx ON track_artists(artist_id);

CREATE TABLE IF NOT EXISTS album_artists (
  album_id    TEXT NOT NULL,
  artist_id   TEXT NOT NULL,
  artist_name TEXT NOT NULL,
  position    INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (album_id, artist_id)
);

CREATE INDEX IF NOT EXISTS album_artists_artist_idx ON album_artists(artist_id);

-- artists and albums are derived from tracks and rebuilt at the end of a scan.
CREATE TABLE IF NOT EXISTS artists (
  id        TEXT PRIMARY KEY,
  name      TEXT NOT NULL,
  sort_name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS albums (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  sort_name   TEXT NOT NULL,
  artist_id   TEXT NOT NULL,
  artist_name TEXT NOT NULL,
  year        INTEGER NOT NULL DEFAULT 0,
  genre       TEXT NOT NULL DEFAULT '',
  artwork_id  TEXT,
  release_type TEXT NOT NULL DEFAULT 'album'
);

CREATE INDEX IF NOT EXISTS albums_artist_idx ON albums(artist_id);
CREATE INDEX IF NOT EXISTS albums_sort_idx   ON albums(sort_name);

CREATE VIRTUAL TABLE IF NOT EXISTS search_fts USING fts5(
  kind UNINDEXED,
  ref_id UNINDEXED,
  title,
  subtitle,
  tokenize = 'unicode61 remove_diacritics 2'
);

CREATE TABLE IF NOT EXISTS track_embeddings (
  track_id  TEXT PRIMARY KEY REFERENCES tracks(id) ON DELETE CASCADE,
  mod_time  INTEGER NOT NULL,
  dim       INTEGER NOT NULL,
  vector    BLOB NOT NULL
);

CREATE INDEX IF NOT EXISTS track_embeddings_mod_idx ON track_embeddings(mod_time);

import type { Track } from "./api";

export const LIKED_PLAYLIST_ID = "liked";
const STORAGE_KEY = "spook.playlists.v1";

export interface PlaylistEntry {
  trackId: string;
  title: string;
  artist: string;
  album: string;
  albumId: string;
  artistId: string;
  artworkId?: string;
  color?: string;
  durationMs: number;
  format: string;
  streamUrl: string;
  addedAt: number;
}

export interface Playlist {
  id: string;
  name: string;
  entries: PlaylistEntry[];
  createdAt: number;
  updatedAt: number;
  /** Built-in playlists (Liked) cannot be deleted or renamed. */
  system?: true;
}

interface Store {
  playlists: Playlist[];
}

function defaultStore(): Store {
  const now = Date.now();
  return {
    playlists: [
      {
        id: LIKED_PLAYLIST_ID,
        name: "Liked",
        entries: [],
        createdAt: now,
        updatedAt: now,
        system: true,
      },
    ],
  };
}

export function trackToEntry(track: Track): PlaylistEntry {
  return {
    trackId: track.id,
    title: track.title,
    artist: track.artist,
    album: track.album,
    albumId: track.albumId,
    artistId: track.artistId,
    artworkId: track.artworkId,
    color: track.color,
    durationMs: track.durationMs,
    format: track.format,
    streamUrl: track.streamUrl,
    addedAt: Date.now(),
  };
}

export function entryToTrack(entry: PlaylistEntry): Track {
  return {
    id: entry.trackId,
    title: entry.title,
    artist: entry.artist,
    album: entry.album,
    albumId: entry.albumId,
    artistId: entry.artistId,
    albumArtist: entry.artist,
    artworkId: entry.artworkId,
    color: entry.color,
    durationMs: entry.durationMs,
    format: entry.format || "mp3",
    streamUrl: entry.streamUrl || `/api/v1/stream/${entry.trackId}`,
    sizeBytes: 0,
    addedAt: entry.addedAt,
  };
}

export function readPlaylists(): Playlist[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return defaultStore().playlists;
    const parsed = JSON.parse(raw) as Store;
    if (!parsed.playlists?.length) return defaultStore().playlists;
    const hasLiked = parsed.playlists.some((p) => p.id === LIKED_PLAYLIST_ID);
    if (!hasLiked) {
      parsed.playlists.unshift(defaultStore().playlists[0]);
    }
    return parsed.playlists;
  } catch {
    return defaultStore().playlists;
  }
}

export function writePlaylists(playlists: Playlist[]): void {
  localStorage.setItem(STORAGE_KEY, JSON.stringify({ playlists }));
}

export function createPlaylistId(): string {
  return `pl_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`;
}

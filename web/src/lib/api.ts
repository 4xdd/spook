import { accessKeyHeaders, withAccessKey } from "./accessKey";

export interface Track {
  id: string;
  title: string;
  artist: string;
  artistId: string;
  album: string;
  albumId: string;
  albumArtist: string;
  genre?: string;
  year?: number;
  trackNo?: number;
  discNo?: number;
  durationMs: number;
  format: string;
  bitrateKbps?: number;
  sampleRateHz?: number;
  sizeBytes: number;
  artworkId?: string;
  color?: string;
  streamUrl: string;
  addedAt: number;
}

export interface Album {
  id: string;
  name: string;
  artist: string;
  artistId: string;
  genre?: string;
  year?: number;
  artworkId?: string;
  color?: string;
  isDark: boolean;
  releaseType: "album" | "single" | "ep";
  trackCount: number;
  discCount: number;
  durationMs: number;
  addedAt: number;
}

export interface Artist {
  id: string;
  name: string;
  artworkId?: string;
  color?: string;
  isDark: boolean;
  albumCount: number;
  trackCount: number;
  durationMs: number;
}

export interface AlbumDetail {
  album: Album;
  tracks: Track[];
}

export interface ArtistDetail {
  artist: Artist;
  albums: Album[];
  tracks: Track[];
}

export interface SearchResults {
  query: string;
  artists: Artist[];
  albums: Album[];
  tracks: Track[];
}

/** `timeMs` is -1 when the lyrics carry no timing. */
export interface LyricLine {
  timeMs: number;
  text: string;
}

export interface Lyrics {
  trackId: string;
  source?: "sidecar" | "embedded" | "lrclib";
  synced: boolean;
  lines: LyricLine[];
}

export interface ScanStatus {
  state: "idle" | "scanning" | "done" | "error";
  total: number;
  processed: number;
  indexed: number;
  removed: number;
  startedAt?: number;
  finishedAt?: number;
  error?: string;
}

export interface Stats {
  root: string;
  tracks: number;
  albums: number;
  artists: number;
  durationMs: number;
  lastScan?: number;
  scan: ScanStatus;
}

export type AlbumSort = "title" | "artist" | "recent" | "year";
export type TrackSort = "title" | "artist" | "album" | "recent";

export type DeezerSearchType = "track" | "album" | "artist";

export interface DeezerResult {
  id: string;
  type: DeezerSearchType;
  title?: string;
  album?: string;
  albumId?: string;
  artist?: string;
  artistId?: string;
  imageUrl?: string;
  previewUrl?: string;
}

export interface DeezerSearchResponse {
  query: string;
  type: DeezerSearchType;
  results: DeezerResult[];
}

export interface DeezerJob {
  id: number;
  description: string;
  state: string;
  progress: number;
  progressMax: number;
  error?: string;
}

export interface DeezerStatus {
  enabled: boolean;
  running: boolean;
  configured: boolean;
  baseUrl?: string;
  musicDir?: string;
  quality?: string;
  error?: string;
}

export interface LastfmStatus {
  enabled: boolean;
  configured: boolean;
}

export interface LastfmSession {
  name: string;
  key: string;
}

export interface LastfmTrackPayload {
  sessionKey: string;
  artist: string;
  title: string;
  album?: string;
  albumArtist?: string;
  trackNumber?: number;
  durationSec?: number;
  timestamp?: number;
}

const base = "/api/v1";

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers);
  const auth = accessKeyHeaders();
  for (const [name, value] of Object.entries(auth)) {
    if (!headers.has(name)) headers.set(name, value);
  }

  const response = await fetch(`${base}${path}`, { ...init, headers });
  if (!response.ok) {
    let message = `Request failed (${response.status})`;
    try {
      const body = (await response.json()) as { error?: string };
      if (body.error) message = body.error;
    } catch {
      // A non-JSON body leaves the status-code message in place.
    }
    throw new ApiError(response.status, message);
  }
  return (await response.json()) as T;
}

/** Probe whether a candidate key is accepted by the server. */
export async function verifyAccessKey(key: string): Promise<void> {
  const response = await fetch(`${base}/stats`, {
    headers: { Authorization: `Bearer ${key.trim()}` },
  });
  if (response.ok) return;
  if (response.status === 401) {
    throw new ApiError(401, "That access key is not valid");
  }
  let message = `Request failed (${response.status})`;
  try {
    const body = (await response.json()) as { error?: string };
    if (body.error) message = body.error;
  } catch {
    // keep status message
  }
  throw new ApiError(response.status, message);
}

export const api = {
  stats: () => request<Stats>("/stats"),
  albums: (sort: AlbumSort = "title") => request<Album[]>(`/albums?sort=${sort}`),
  album: (id: string) => request<AlbumDetail>(`/albums/${id}`),
  artists: () => request<Artist[]>("/artists"),
  artist: (id: string) => request<ArtistDetail>(`/artists/${id}`),
  tracks: (sort: TrackSort = "title") => request<Track[]>(`/tracks?sort=${sort}`),
  lyrics: (id: string) =>
    request<Lyrics>(`/tracks/${id}/lyrics`, {
      signal: AbortSignal.timeout(12_000),
    }),
  search: (query: string) => request<SearchResults>(`/search?q=${encodeURIComponent(query)}`),
  scanStatus: () => request<ScanStatus>("/scan"),
  startScan: () => request<ScanStatus>("/scan", { method: "POST" }),
  deezerStatus: () => request<DeezerStatus>("/deezer/status"),
  deezerSearch: (query: string, type: DeezerSearchType = "track") =>
    request<DeezerSearchResponse>(`/deezer/search?q=${encodeURIComponent(query)}&type=${type}`),
  deezerDownload: (type: "track" | "album", musicId: number) =>
    request<{ taskId: number }>("/deezer/download", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ type, musicId }),
    }),
  deezerJobs: () => request<{ jobs: DeezerJob[] }>("/deezer/jobs"),
  lastfmStatus: () => request<LastfmStatus>("/lastfm/status"),
  lastfmAuthURL: (callback: string) =>
    request<{ url: string }>(`/lastfm/auth-url?callback=${encodeURIComponent(callback)}`),
  lastfmSession: (token: string) =>
    request<LastfmSession>("/lastfm/session", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    }),
  lastfmNowPlaying: (payload: LastfmTrackPayload) =>
    request<{ ok: boolean }>("/lastfm/now-playing", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),
  lastfmScrobble: (payload: LastfmTrackPayload & { timestamp: number }) =>
    request<{ ok: boolean }>("/lastfm/scrobble", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    }),
};

/** Artwork URLs are content-addressed, so they can be cached forever. */
export function artworkUrl(artworkId: string | undefined, size: 64 | 300 | 1000): string | undefined {
  if (!artworkId) return undefined;
  return withAccessKey(`${base}/art/${artworkId}?size=${size}`);
}

/** Stream URLs need the key in the query string for `<audio>` elements. */
export function streamUrl(url: string): string {
  return withAccessKey(url);
}

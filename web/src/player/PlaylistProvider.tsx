import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { Track } from "@/lib/api";
import {
  LIKED_PLAYLIST_ID,
  createPlaylistId,
  readPlaylists,
  trackToEntry,
  writePlaylists,
  type Playlist,
  type PlaylistEntry,
} from "@/lib/playlists";

interface PlaylistValue {
  playlists: Playlist[];
  liked: Playlist;
  addTracks(playlistId: string, tracks: Track[]): void;
  addToLiked(tracks: Track | Track[]): void;
  removeTrack(playlistId: string, trackId: string): void;
  createPlaylist(name: string): Playlist;
  isInPlaylist(playlistId: string, trackId: string): boolean;
  playlistById(id: string): Playlist | undefined;
}

const PlaylistContext = createContext<PlaylistValue | null>(null);

function mergeEntries(existing: PlaylistEntry[], tracks: Track[]): PlaylistEntry[] {
  const seen = new Set(existing.map((e) => e.trackId));
  const added = tracks.filter((t) => !seen.has(t.id)).map(trackToEntry);
  if (added.length === 0) return existing;
  return [...existing, ...added];
}

export function PlaylistProvider({ children }: { children: ReactNode }) {
  const [playlists, setPlaylists] = useState<Playlist[]>(() => readPlaylists());

  const persist = useCallback((next: Playlist[]) => {
    setPlaylists(next);
    writePlaylists(next);
  }, []);

  const updatePlaylist = useCallback(
    (playlistId: string, updater: (playlist: Playlist) => Playlist) => {
      persist(
        playlists.map((p) => (p.id === playlistId ? updater(p) : p)),
      );
    },
    [persist, playlists],
  );

  const addTracks = useCallback(
    (playlistId: string, tracks: Track[]) => {
      if (tracks.length === 0) return;
      updatePlaylist(playlistId, (p) => ({
        ...p,
        entries: mergeEntries(p.entries, tracks),
        updatedAt: Date.now(),
      }));
    },
    [updatePlaylist],
  );

  const addToLiked = useCallback(
    (tracks: Track | Track[]) => {
      const list = Array.isArray(tracks) ? tracks : [tracks];
      addTracks(LIKED_PLAYLIST_ID, list);
    },
    [addTracks],
  );

  const removeTrack = useCallback(
    (playlistId: string, trackId: string) => {
      updatePlaylist(playlistId, (p) => ({
        ...p,
        entries: p.entries.filter((e) => e.trackId !== trackId),
        updatedAt: Date.now(),
      }));
    },
    [updatePlaylist],
  );

  const createPlaylist = useCallback(
    (name: string) => {
      const trimmed = name.trim();
      if (!trimmed) throw new Error("playlist name required");
      const now = Date.now();
      const playlist: Playlist = {
        id: createPlaylistId(),
        name: trimmed,
        entries: [],
        createdAt: now,
        updatedAt: now,
      };
      persist([...playlists, playlist]);
      return playlist;
    },
    [persist, playlists],
  );

  const isInPlaylist = useCallback(
    (playlistId: string, trackId: string) => {
      const playlist = playlists.find((p) => p.id === playlistId);
      return playlist?.entries.some((e) => e.trackId === trackId) ?? false;
    },
    [playlists],
  );

  const playlistById = useCallback(
    (id: string) => playlists.find((p) => p.id === id),
    [playlists],
  );

  const liked = useMemo(
    () => playlists.find((p) => p.id === LIKED_PLAYLIST_ID) ?? readPlaylists()[0],
    [playlists],
  );

  const value = useMemo<PlaylistValue>(
    () => ({
      playlists,
      liked,
      addTracks,
      addToLiked,
      removeTrack,
      createPlaylist,
      isInPlaylist,
      playlistById,
    }),
    [playlists, liked, addTracks, addToLiked, removeTrack, createPlaylist, isInPlaylist, playlistById],
  );

  return <PlaylistContext.Provider value={value}>{children}</PlaylistContext.Provider>;
}

export function usePlaylists() {
  const value = useContext(PlaylistContext);
  if (!value) throw new Error("usePlaylists requires PlaylistProvider");
  return value;
}

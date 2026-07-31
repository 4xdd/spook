import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api, type AlbumSort, type DeezerSearchType, type TrackSort } from "./api";

export const keys = {
  stats: ["stats"] as const,
  albums: (sort: AlbumSort) => ["albums", sort] as const,
  album: (id: string) => ["album", id] as const,
  artists: ["artists"] as const,
  artist: (id: string) => ["artist", id] as const,
  tracks: (sort: TrackSort) => ["tracks", sort] as const,
  lyrics: (id: string) => ["lyrics", id] as const,
  search: (query: string) => ["search", query] as const,
  deezerStatus: ["deezer", "status"] as const,
  deezerSearch: (query: string, type: DeezerSearchType) => ["deezer", "search", type, query] as const,
  deezerJobs: ["deezer", "jobs"] as const,
};

export function useStats() {
  return useQuery({
    queryKey: keys.stats,
    queryFn: api.stats,
    // Poll while a scan is running so progress and new content appear live.
    refetchInterval: (query) => (query.state.data?.scan.state === "scanning" ? 1000 : false),
  });
}

export function useAlbums(sort: AlbumSort = "title") {
  return useQuery({ queryKey: keys.albums(sort), queryFn: () => api.albums(sort) });
}

export function useAlbum(id: string) {
  return useQuery({ queryKey: keys.album(id), queryFn: () => api.album(id), enabled: Boolean(id) });
}

export function useArtists() {
  return useQuery({ queryKey: keys.artists, queryFn: api.artists });
}

export function useArtist(id: string) {
  return useQuery({ queryKey: keys.artist(id), queryFn: () => api.artist(id), enabled: Boolean(id) });
}

export function useTracks(sort: TrackSort = "title") {
  return useQuery({ queryKey: keys.tracks(sort), queryFn: () => api.tracks(sort) });
}

/**
 * Fetched as soon as a track starts, so the Lyrics button already knows
 * whether it has anything to show by the time it is looked at.
 */
export function useLyrics(trackId: string | undefined) {
  return useQuery({
    queryKey: keys.lyrics(trackId ?? ""),
    queryFn: () => api.lyrics(trackId as string),
    enabled: Boolean(trackId),
    staleTime: 24 * 60 * 60 * 1000,
    gcTime: 30 * 60 * 1000,
    retry: 1,
  });
}

/** Whether a track has lyrics to show. False until the fetch settles. */
export function useLyricsAvailable(trackId: string | undefined) {
  const { data } = useLyrics(trackId);
  return (data?.lines.length ?? 0) > 0;
}

/** Whether those lyrics carry timings, which is what following along needs. */
export function useSyncedLyrics(trackId: string | undefined) {
  const { data } = useLyrics(trackId);
  return Boolean(data?.synced) && (data?.lines.length ?? 0) > 0;
}

export function useSearch(query: string) {
  const trimmed = query.trim();
  return useQuery({
    queryKey: keys.search(trimmed),
    queryFn: () => api.search(trimmed),
    enabled: trimmed.length > 0,
    placeholderData: (previous) => previous,
  });
}

export function useStartScan() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: api.startScan,
    onSuccess: () => client.invalidateQueries({ queryKey: keys.stats }),
  });
}

/** Refreshes every library view once a scan finishes. */
export function useRefreshLibrary() {
  const client = useQueryClient();
  return () => client.invalidateQueries();
}

export function useDeezerStatus() {
  return useQuery({
    queryKey: keys.deezerStatus,
    queryFn: api.deezerStatus,
    refetchInterval: (query) => (query.state.data?.running ? 3000 : false),
  });
}

export function useDeezerSearch(query: string, type: DeezerSearchType) {
  const trimmed = query.trim();
  return useQuery({
    queryKey: keys.deezerSearch(trimmed, type),
    queryFn: () => api.deezerSearch(trimmed, type),
    enabled: trimmed.length > 0,
    placeholderData: (previous) => previous,
  });
}

export function useDeezerJobs(enabled: boolean) {
  return useQuery({
    queryKey: keys.deezerJobs,
    queryFn: api.deezerJobs,
    enabled,
    refetchInterval: enabled ? 2000 : false,
  });
}

export function useDeezerDownload() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ type, musicId }: { type: "track" | "album"; musicId: number }) =>
      api.deezerDownload(type, musicId),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: keys.deezerJobs });
    },
  });
}

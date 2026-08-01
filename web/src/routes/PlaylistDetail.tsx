import { Play, Shuffle } from "lucide-react";
import { useMemo, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { PageShell } from "@/components/PageShell";
import { PlaylistSuggestions } from "@/components/PlaylistSuggestions";
import { EmptyState, ErrorState } from "@/components/States";
import { TrackRow } from "@/components/TrackRow";
import { entryToTrack } from "@/lib/playlists";
import { formatRuntime, plural } from "@/lib/format";
import { usePlaylists } from "@/player/PlaylistProvider";
import { usePlayer } from "@/player/PlayerProvider";
import type { Track } from "@/lib/api";

export function PlaylistDetail() {
  const { id = "" } = useParams();
  const { playlistById } = usePlaylists();
  const { play, playShuffled } = usePlayer();
  const navigate = useNavigate();
  const [nonce, setNonce] = useState(() => Date.now());

  const playlist = playlistById(id);
  const tracks = useMemo(
    () => (playlist ? playlist.entries.map(entryToTrack) : []),
    [playlist],
  );
  const seedIds = useMemo(() => tracks.map((t) => t.id), [tracks]);

  if (!playlist) {
    return (
      <PageShell title="Playlist">
        <ErrorState message="Playlist not found." />
      </PageShell>
    );
  }

  const context = { label: playlist.name, id: playlist.id };
  const durationMs = tracks.reduce((sum, t) => sum + t.durationMs, 0);
  const excludeIds = seedIds;

  function playSuggested(track: Track, suggested: Track[]) {
    const idx = suggested.findIndex((t) => t.id === track.id);
    play(suggested, idx >= 0 ? idx : 0, { label: "Recommended", id: `rec:${id}` });
  }

  return (
    <PageShell
      title={playlist.name}
      subtitle={plural(tracks.length, "song")}
      actions={
        tracks.length > 0 ? (
          <div className="flex items-center gap-1">
            <button
              type="button"
              onClick={() => play(tracks, 0, context)}
              aria-label={`Play ${playlist.name}`}
              className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
            >
              <Play className="h-3.5 w-3.5" fill="currentColor" strokeWidth={0} aria-hidden />
            </button>
            <button
              type="button"
              onClick={() => playShuffled(tracks, context)}
              aria-label={`Shuffle ${playlist.name}`}
              className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
            >
              <Shuffle className="h-3.5 w-3.5" aria-hidden />
            </button>
          </div>
        ) : undefined
      }
    >
      {tracks.length === 0 ? (
        <EmptyState
          title={playlist.system ? "No liked songs yet" : "This playlist is empty"}
          description={
            playlist.system
              ? "Tap + in Now Playing or use the menu on any song to add tracks here."
              : "Add songs from the track menu or right-click any album."
          }
        />
      ) : (
        <>
          <p className="mb-4 text-[13px] text-secondary">{formatRuntime(durationMs)}</p>
          <div className="-mx-2.5 divide-y divide-separator">
            {tracks.map((track, position) => (
              <TrackRow
                key={track.id}
                track={track}
                variant="artwork"
                position={position + 1}
                showAlbum
                onPlay={() => play(tracks, position, context)}
              />
            ))}
          </div>
        </>
      )}

      <PlaylistSuggestions
        seedIds={seedIds}
        excludeIds={excludeIds}
        nonce={nonce}
        onRefresh={() => setNonce(Date.now())}
        onPlay={playSuggested}
      />

      {playlist.system && (
        <div className="mt-8 flex justify-center">
          <button
            type="button"
            onClick={() => navigate("/")}
            className="text-[13px] text-accent hover:underline"
          >
            Browse your library
          </button>
        </div>
      )}
    </PageShell>
  );
}

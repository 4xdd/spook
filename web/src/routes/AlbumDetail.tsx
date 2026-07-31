import { Play, Shuffle } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { Artwork } from "@/components/Artwork";
import { PageShell } from "@/components/PageShell";
import { ErrorState, LoadingState } from "@/components/States";
import { TrackRow } from "@/components/TrackRow";
import { formatReleaseType, formatRuntime, plural } from "@/lib/format";
import { useAlbum } from "@/lib/queries";
import { usePlayer } from "@/player/PlayerProvider";
import type { Track } from "@/lib/api";

export function AlbumDetail() {
  const { id = "" } = useParams();
  const { data, isPending, error } = useAlbum(id);
  const { play, playShuffled } = usePlayer();

  if (isPending) {
    return (
      <PageShell title="Album">
        <LoadingState />
      </PageShell>
    );
  }
  if (error) {
    return (
      <PageShell title="Album">
        <ErrorState message={error.message} />
      </PageShell>
    );
  }

  const { album, tracks } = data;
  const context = { label: album.name, id: album.id };
  const discs = groupByDisc(tracks, album.discCount);

  const hero = (
    <div className="flex flex-col gap-6 px-6 pt-16 pb-6 sm:flex-row sm:items-end">
      <Artwork
        artworkId={album.artworkId}
        size={1000}
        alt={`${album.name} by ${album.artist}`}
        color={album.color}
        rounded="lg"
        eager
        className="aspect-square w-44 shadow-art sm:w-56"
      />

      <div className="flex min-w-0 flex-1 flex-col gap-3">
        <div className="min-w-0">
          <h1 className="text-[28px] leading-tight font-bold sm:text-[34px]">{album.name}</h1>
          <Link
            to={`/artists/${album.artistId}`}
            className="text-[18px] font-medium text-accent hover:underline sm:text-[20px]"
          >
            {album.artist}
          </Link>
          <p className="mt-1 text-[12px] tracking-[0.02em] text-secondary uppercase">
            {[formatReleaseType(album.releaseType), album.genre, album.year || null]
              .filter(Boolean)
              .join(" · ")}
          </p>
          <p className="text-[12px] text-tertiary">
            {plural(album.trackCount, "song")} · {formatRuntime(album.durationMs)}
          </p>
        </div>

        <div className="flex gap-2.5">
          <button
            type="button"
            onClick={() => play(tracks, 0, context)}
            aria-label={`Play ${album.name}`}
            className="flex items-center gap-1.5 rounded-lg bg-accent px-5 py-2 text-[14px] font-medium text-accent-content transition-transform duration-100 hover:brightness-110 active:scale-[0.97]"
          >
            <Play className="h-3.5 w-3.5" fill="currentColor" strokeWidth={0} aria-hidden />
            Play
          </button>
          <button
            type="button"
            onClick={() => playShuffled(tracks, context)}
            className="flex items-center gap-1.5 rounded-lg bg-fill px-5 py-2 text-[14px] font-medium transition-[transform,background-color] duration-100 hover:bg-fill-strong active:scale-[0.97]"
          >
            <Shuffle className="h-3.5 w-3.5" aria-hidden />
            Shuffle
          </button>
        </div>
      </div>
    </div>
  );

  return (
    <PageShell title={album.name} hero={hero} tint={album.color}>
      <div className="-mx-2.5">
        {discs.map((disc) => (
          <section key={disc.number}>
            {discs.length > 1 && (
              <h2 className="px-2.5 pt-5 pb-1.5 text-[11px] font-semibold tracking-[0.04em] text-tertiary uppercase">
                Disc {disc.number}
              </h2>
            )}
            <div className="divide-y divide-separator">
              {disc.tracks.map((track) => (
                <TrackRow
                  key={track.id}
                  track={track}
                  position={tracks.indexOf(track) + 1}
                  onPlay={() => play(tracks, tracks.indexOf(track), context)}
                />
              ))}
            </div>
          </section>
        ))}
      </div>
    </PageShell>
  );
}

function groupByDisc(tracks: Track[], discCount: number) {
  if (discCount <= 1) return [{ number: 1, tracks }];

  const discs = new Map<number, Track[]>();
  for (const track of tracks) {
    const number = track.discNo || 1;
    const existing = discs.get(number);
    if (existing) existing.push(track);
    else discs.set(number, [track]);
  }
  return [...discs.entries()]
    .sort(([a], [b]) => a - b)
    .map(([number, discTracks]) => ({ number, tracks: discTracks }));
}

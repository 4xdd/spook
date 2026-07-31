import { Play, Shuffle } from "lucide-react";
import { useParams } from "react-router-dom";
import { AlbumCard } from "@/components/AlbumCard";
import { Artwork } from "@/components/Artwork";
import { PageShell } from "@/components/PageShell";
import { AlbumGrid, ErrorState, LoadingState } from "@/components/States";
import { TrackRow } from "@/components/TrackRow";
import { formatRuntime, plural } from "@/lib/format";
import { useArtist } from "@/lib/queries";
import { usePlayer } from "@/player/PlayerProvider";

export function ArtistDetail() {
  const { id = "" } = useParams();
  const { data, isPending, error } = useArtist(id);
  const { play, playShuffled } = usePlayer();

  if (isPending) {
    return (
      <PageShell title="Artist">
        <LoadingState />
      </PageShell>
    );
  }
  if (error) {
    return (
      <PageShell title="Artist">
        <ErrorState message={error.message} />
      </PageShell>
    );
  }

  const { artist, albums, tracks } = data;
  const context = { label: artist.name, id: artist.id };

  const hero = (
    <div className="flex flex-col items-center gap-4 px-4 pt-14 pb-6 text-center sm:px-6 sm:pt-16">
      <Artwork
        artworkId={artist.artworkId}
        size={300}
        alt={artist.name}
        color={artist.color}
        eager
        className="aspect-square w-32 rounded-full shadow-art"
      />
      <div>
        <h1 className="text-[30px] font-bold">{artist.name}</h1>
        <p className="mt-0.5 text-[13px] text-secondary">
          {plural(artist.albumCount, "album")} · {plural(artist.trackCount, "song")} ·{" "}
          {formatRuntime(artist.durationMs)}
        </p>
      </div>
      <div className="flex gap-2.5">
        <button
          type="button"
          onClick={() => play(tracks, 0, context)}
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
  );

  return (
    <PageShell title={artist.name} hero={hero} tint={artist.color}>
      <section>
        <h2 className="pb-3 text-[19px] font-bold">Releases</h2>
        <AlbumGrid>
          {albums.map((album) => (
            <AlbumCard key={album.id} album={album} showArtist={false} />
          ))}
        </AlbumGrid>
      </section>

      <section className="pt-9">
        <h2 className="pb-1 text-[19px] font-bold">Songs</h2>
        <div className="-mx-2.5 divide-y divide-separator">
          {tracks.map((track, position) => (
            <TrackRow
              key={track.id}
              track={track}
              variant="artwork"
              position={position + 1}
              onPlay={() => play(tracks, position, context)}
            />
          ))}
        </div>
      </section>
    </PageShell>
  );
}

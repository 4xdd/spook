import { Link } from "react-router-dom";
import { Artwork } from "@/components/Artwork";
import { PageShell } from "@/components/PageShell";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import { plural } from "@/lib/format";
import { useArtists } from "@/lib/queries";

export function Artists() {
  const { data: artists, isPending, error } = useArtists();

  return (
    <PageShell title="Artists" subtitle={artists ? plural(artists.length, "artist") : undefined}>
      {isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : artists.length === 0 ? (
        <EmptyState title="No artists yet" description="Scan a music folder to fill your library." />
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] gap-x-4 gap-y-6 sm:grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] sm:gap-x-5 sm:gap-y-7">
          {artists.map((artist) => (
            <Link
              key={artist.id}
              to={`/artists/${artist.id}`}
              className="group flex flex-col items-center gap-2.5 text-center"
            >
              <Artwork
                artworkId={artist.artworkId}
                size={300}
                alt={artist.name}
                color={artist.color}
                className="aspect-square w-full rounded-full shadow-art transition-transform duration-200 ease-out group-hover:-translate-y-0.5 group-active:scale-[0.985]"
              />
              <div className="min-w-0">
                <div className="truncate text-[13px] group-hover:underline">{artist.name}</div>
                <div className="truncate text-[12px] text-secondary">{plural(artist.albumCount, "album")}</div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </PageShell>
  );
}

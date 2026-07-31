import { Link, useSearchParams } from "react-router-dom";
import { AlbumCard } from "@/components/AlbumCard";
import { Artwork } from "@/components/Artwork";
import { PageShell } from "@/components/PageShell";
import { AlbumGrid, EmptyState, ErrorState, LoadingState } from "@/components/States";
import { TrackRow } from "@/components/TrackRow";
import { plural } from "@/lib/format";
import { useSearch } from "@/lib/queries";
import { usePlayer } from "@/player/PlayerProvider";

export function SearchResults() {
  const [searchParams] = useSearchParams();
  const query = searchParams.get("q") ?? "";
  const { data, isPending, error } = useSearch(query);
  const { play } = usePlayer();

  const empty = data && data.artists.length === 0 && data.albums.length === 0 && data.tracks.length === 0;

  return (
    <PageShell title="Search" subtitle={query ? `Results for “${query}”` : undefined}>
      {!query.trim() ? (
        <EmptyState title="Search your library" description="Find artists, albums and songs by name." />
      ) : isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : empty ? (
        <EmptyState title="No results" description={`Nothing in your library matches “${query}”.`} />
      ) : (
        <div className="flex flex-col gap-9">
          {data.artists.length > 0 && (
            <section>
              <h2 className="pb-3 text-[19px] font-bold">Artists</h2>
              <div className="grid grid-cols-[repeat(auto-fill,minmax(7rem,1fr))] gap-x-4 gap-y-6 sm:grid-cols-[repeat(auto-fill,minmax(8rem,1fr))] sm:gap-x-5 sm:gap-y-6">
                {data.artists.map((artist) => (
                  <Link
                    key={artist.id}
                    to={`/artists/${artist.id}`}
                    className="group flex flex-col items-center gap-2 text-center"
                  >
                    <Artwork
                      artworkId={artist.artworkId}
                      size={300}
                      alt={artist.name}
                      color={artist.color}
                      className="aspect-square w-full rounded-full shadow-art transition-transform duration-200 group-hover:-translate-y-0.5"
                    />
                    <div className="truncate text-[13px] group-hover:underline">{artist.name}</div>
                    <div className="-mt-1.5 text-[12px] text-secondary">{plural(artist.albumCount, "album")}</div>
                  </Link>
                ))}
              </div>
            </section>
          )}

          {data.albums.length > 0 && (
            <section>
              <h2 className="pb-3 text-[19px] font-bold">Albums</h2>
              <AlbumGrid>
                {data.albums.map((album) => (
                  <AlbumCard key={album.id} album={album} />
                ))}
              </AlbumGrid>
            </section>
          )}

          {data.tracks.length > 0 && (
            <section>
              <h2 className="pb-1 text-[19px] font-bold">Songs</h2>
              <div className="-mx-2.5 divide-y divide-separator">
                {data.tracks.map((track, position) => (
                  <TrackRow
                    key={track.id}
                    track={track}
                    variant="artwork"
                    showAlbum
                    position={position + 1}
                    onPlay={() => play(data.tracks, position, { label: `“${query}”`, id: `search:${query}` })}
                  />
                ))}
              </div>
            </section>
          )}
        </div>
      )}
    </PageShell>
  );
}

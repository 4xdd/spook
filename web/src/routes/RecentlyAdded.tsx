import { useAlbums, useStats } from "@/lib/queries";
import { AlbumCard } from "@/components/AlbumCard";
import { PageShell } from "@/components/PageShell";
import { AlbumGrid, EmptyState, ErrorState, LoadingState } from "@/components/States";
import { plural } from "@/lib/format";

export function RecentlyAdded() {
  const { data: albums, isPending, error } = useAlbums("recent");
  const { data: stats } = useStats();

  return (
    <PageShell
      title="Recently Added"
      subtitle={stats ? `${plural(stats.albums, "album")} · ${plural(stats.tracks, "song")}` : undefined}
    >
      {isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : albums.length === 0 ? (
        <EmptyState
          title="Your library is empty"
          description={
            stats?.root
              ? `Nothing was found in ${stats.root}. Point Spook at a music folder with -music-dir, then rescan.`
              : "Add music to your library folder and rescan."
          }
        />
      ) : (
        <AlbumGrid>
          {albums.map((album) => (
            <AlbumCard key={album.id} album={album} />
          ))}
        </AlbumGrid>
      )}
    </PageShell>
  );
}

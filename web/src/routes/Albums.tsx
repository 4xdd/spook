import { useState } from "react";
import { AlbumCard } from "@/components/AlbumCard";
import { PageShell } from "@/components/PageShell";
import { AlbumGrid, EmptyState, ErrorState, LoadingState } from "@/components/States";
import { cn } from "@/lib/cn";
import { plural } from "@/lib/format";
import { useAlbums } from "@/lib/queries";
import type { AlbumSort } from "@/lib/api";

const sorts: { value: AlbumSort; label: string }[] = [
  { value: "title", label: "Title" },
  { value: "artist", label: "Artist" },
  { value: "year", label: "Year" },
  { value: "recent", label: "Recent" },
];

export function Albums() {
  const [sort, setSort] = useState<AlbumSort>("title");
  const { data: albums, isPending, error } = useAlbums(sort);

  return (
    <PageShell
      title="Albums"
      subtitle={albums ? plural(albums.length, "album") : undefined}
      actions={
        <div className="flex items-center gap-0.5 rounded-lg bg-fill p-0.5" role="group" aria-label="Sort albums">
          {sorts.map((option) => (
            <button
              key={option.value}
              type="button"
              onClick={() => setSort(option.value)}
              aria-pressed={sort === option.value}
              className={cn(
                "rounded-[6px] px-2.5 py-1 text-[12px] transition-colors duration-100 active:scale-[0.97]",
                sort === option.value ? "bg-raised text-content shadow-pop" : "text-secondary hover:text-content",
              )}
            >
              {option.label}
            </button>
          ))}
        </div>
      }
    >
      {isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : albums.length === 0 ? (
        <EmptyState title="No albums yet" description="Scan a music folder to fill your library." />
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

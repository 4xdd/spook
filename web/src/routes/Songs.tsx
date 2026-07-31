import { Play, Shuffle } from "lucide-react";
import { useState } from "react";
import { PageShell } from "@/components/PageShell";
import { EmptyState, ErrorState, LoadingState } from "@/components/States";
import { TrackRow } from "@/components/TrackRow";
import { cn } from "@/lib/cn";
import { plural } from "@/lib/format";
import { useTracks } from "@/lib/queries";
import { usePlayer } from "@/player/PlayerProvider";
import type { TrackSort } from "@/lib/api";

const sorts: { value: TrackSort; label: string }[] = [
  { value: "title", label: "Title" },
  { value: "artist", label: "Artist" },
  { value: "album", label: "Album" },
  { value: "recent", label: "Recent" },
];

export function Songs() {
  const [sort, setSort] = useState<TrackSort>("title");
  const { data: tracks, isPending, error } = useTracks(sort);
  const { play, playShuffled } = usePlayer();

  const context = { label: "Songs", id: `songs:${sort}` };

  return (
    <PageShell
      title="Songs"
      subtitle={tracks ? plural(tracks.length, "song") : undefined}
      actions={
        <div className="flex items-center gap-2">
          {tracks && tracks.length > 0 && (
            <div className="flex items-center gap-1">
              <button
                type="button"
                onClick={() => play(tracks, 0, context)}
                aria-label="Play all songs"
                className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
              >
                <Play className="h-3.5 w-3.5" fill="currentColor" strokeWidth={0} aria-hidden />
              </button>
              <button
                type="button"
                onClick={() => playShuffled(tracks, context)}
                aria-label="Shuffle all songs"
                className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
              >
                <Shuffle className="h-3.5 w-3.5" aria-hidden />
              </button>
            </div>
          )}
          <div className="flex items-center gap-0.5 rounded-lg bg-fill p-0.5" role="group" aria-label="Sort songs">
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
        </div>
      }
    >
      {isPending ? (
        <LoadingState />
      ) : error ? (
        <ErrorState message={error.message} />
      ) : tracks.length === 0 ? (
        <EmptyState title="No songs yet" description="Scan a music folder to fill your library." />
      ) : (
        <div className="-mx-2.5 divide-y divide-separator">
          {tracks.map((track, position) => (
            <TrackRow
              key={track.id}
              track={track}
              variant="artwork"
              showAlbum
              position={position + 1}
              onPlay={() => play(tracks, position, context)}
            />
          ))}
        </div>
      )}
    </PageShell>
  );
}

import { RefreshCw } from "lucide-react";
import { IconButton } from "@/components/IconButton";
import { TrackRow } from "@/components/TrackRow";
import { useRecommendations } from "@/lib/queries";
import type { Track } from "@/lib/api";

interface Props {
  seedIds: string[];
  excludeIds: string[];
  nonce: number;
  onRefresh(): void;
  onPlay(track: Track, tracks: Track[]): void;
}

export function PlaylistSuggestions({ seedIds, excludeIds, nonce, onRefresh, onPlay }: Props) {
  const { data, isPending, error } = useRecommendations(seedIds, excludeIds, nonce);

  const tracks = data?.tracks ?? [];

  return (
    <section className="mt-10 border-t border-separator pt-8">
      <div className="mb-3 flex items-center justify-between gap-3">
        <h2 className="text-[19px] font-bold">Recommended</h2>
        <IconButton label="Refresh recommendations" onClick={onRefresh} size="sm">
          <RefreshCw className="h-4 w-4" aria-hidden />
        </IconButton>
      </div>

      {isPending && tracks.length === 0 ? (
        <p className="text-[13px] text-secondary">Finding similar songs…</p>
      ) : error ? (
        <p className="text-[13px] text-secondary">Recommendations unavailable.</p>
      ) : tracks.length === 0 ? (
        <p className="text-[13px] text-secondary">
          Recommendations appear after the library is scanned and embeddings are built.
        </p>
      ) : (
        <div className="-mx-2.5 divide-y divide-separator">
          {tracks.map((track, position) => (
            <TrackRow
              key={track.id}
              track={track}
              variant="artwork"
              showAlbum
              position={position + 1}
              onPlay={() => onPlay(track, tracks)}
            />
          ))}
        </div>
      )}
    </section>
  );
}

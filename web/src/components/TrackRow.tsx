import { ListEnd, ListStart, MoreHorizontal, Play } from "lucide-react";
import { useNavigate } from "react-router-dom";
import type { Track } from "@/lib/api";
import { cn } from "@/lib/cn";
import { formatDurationMs } from "@/lib/format";
import { usePlayer } from "@/player/PlayerProvider";
import { Artwork } from "./Artwork";
import { Menu } from "./Menu";
import { NowPlayingBars } from "./NowPlayingBars";

interface Props {
  track: Track;
  onPlay(): void;
  /** Numbered rows suit an album; artwork rows suit mixed lists. */
  variant?: "numbered" | "artwork";
  position?: number;
  showAlbum?: boolean;
}

export function TrackRow({ track, onPlay, variant = "numbered", position, showAlbum }: Props) {
  const { current, isPlaying, playNext, playLater } = usePlayer();
  const navigate = useNavigate();
  const isCurrent = current?.id === track.id;

  return (
    <div
      role="button"
      tabIndex={0}
      onDoubleClick={onPlay}
      onClick={onPlay}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onPlay();
        }
      }}
      className={cn(
        "group grid cursor-default select-none items-center gap-3 rounded-lg px-2.5 transition-colors duration-100",
        "hover:bg-fill active:bg-fill-strong",
        variant === "numbered" ? "h-12 grid-cols-[2rem_1fr_auto_2rem]" : "h-14 grid-cols-[2.5rem_1fr_auto_2rem]",
        showAlbum && "sm:grid-cols-[2.5rem_1fr_1fr_auto_2rem]",
      )}
    >
      {variant === "numbered" ? (
        <div className="grid h-8 w-8 place-items-center">
          {isCurrent ? (
            <NowPlayingBars playing={isPlaying} />
          ) : (
            <>
              <span className="text-[13px] tabular-nums text-tertiary group-hover:hidden">
                {track.trackNo || position || "-"}
              </span>
              <Play
                className="hidden h-3.5 w-3.5 translate-x-[1px] text-content group-hover:block"
                fill="currentColor"
                strokeWidth={0}
                aria-hidden
              />
            </>
          )}
        </div>
      ) : (
        <div className="relative h-10 w-10">
          <Artwork artworkId={track.artworkId} size={64} color={track.color} rounded="sm" className="h-10 w-10" />
          <div className="absolute inset-0 grid place-items-center rounded-[4px] bg-black/45 opacity-0 transition-opacity duration-150 group-hover:opacity-100">
            {isCurrent ? (
              <NowPlayingBars playing={isPlaying} />
            ) : (
              <Play className="h-3.5 w-3.5 translate-x-[1px] text-white" fill="currentColor" strokeWidth={0} aria-hidden />
            )}
          </div>
          {isCurrent && (
            <div className="absolute inset-0 grid place-items-center rounded-[4px] bg-black/45 group-hover:opacity-0">
              <NowPlayingBars playing={isPlaying} />
            </div>
          )}
        </div>
      )}

      <div className="min-w-0">
        <div className={cn("truncate text-[14px] leading-tight", isCurrent ? "text-accent" : "text-content")}>
          {track.title}
        </div>
        <div className="truncate text-[12px] leading-tight text-secondary">{track.artist}</div>
      </div>

      {showAlbum && <div className="hidden min-w-0 truncate text-[13px] text-secondary sm:block">{track.album}</div>}

      <div className="text-[13px] tabular-nums text-tertiary">{formatDurationMs(track.durationMs)}</div>

      <div className="opacity-100 transition-opacity duration-150 sm:opacity-0 sm:group-hover:opacity-100 sm:focus-within:opacity-100">
        <Menu
          label={`More options for ${track.title}`}
          items={[
            { label: "Play Next", icon: <ListStart className="h-4 w-4" />, onSelect: () => playNext([track]) },
            { label: "Play Later", icon: <ListEnd className="h-4 w-4" />, onSelect: () => playLater([track]) },
            { label: "Go to Album", onSelect: () => navigate(`/albums/${track.albumId}`) },
            { label: "Go to Artist", onSelect: () => navigate(`/artists/${track.artistId}`) },
          ]}
        >
          <MoreHorizontal className="h-4 w-4" aria-hidden />
        </Menu>
      </div>
    </div>
  );
}

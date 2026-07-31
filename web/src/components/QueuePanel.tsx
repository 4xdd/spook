import { X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { cn } from "@/lib/cn";
import { formatDurationMs } from "@/lib/format";
import { usePlayer } from "@/player/PlayerProvider";
import { Artwork } from "./Artwork";
import { NowPlayingBars } from "./NowPlayingBars";

export function QueuePanel({ open, onClose }: { open: boolean; onClose(): void }) {
  const { queue, index, current, isPlaying, jumpTo, removeAt, clearQueue, context } = usePlayer();
  const upcoming = queue.slice(index + 1);

  return (
    <AnimatePresence>
      {open && (
        <>
          <motion.button
            key="queue-scrim"
            type="button"
            aria-label="Close queue"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.18 }}
            onClick={onClose}
            className="fixed inset-0 z-30 bg-black/45 sm:hidden"
          />

          <motion.aside
            key="queue"
            aria-label="Playing next"
            initial={{ x: "100%" }}
            animate={{ x: 0 }}
            exit={{ x: "100%" }}
            transition={{ type: "spring", bounce: 0, duration: 0.35 }}
            className={cn(
              "material z-40 flex h-full flex-col border-separator",
              "fixed inset-y-0 right-0 w-full max-w-sm sm:relative sm:z-20 sm:w-80 sm:max-w-none sm:shrink-0 sm:border-l",
            )}
          >
          <header className="flex items-center justify-between gap-2 px-4 py-3">
            <div className="min-w-0">
              <h2 className="text-[13px] font-semibold">Playing Next</h2>
              {context && <p className="truncate text-[11px] text-tertiary">From {context.label}</p>}
            </div>
            <div className="flex items-center gap-1">
              {upcoming.length > 0 && (
                <button
                  type="button"
                  onClick={clearQueue}
                  className="rounded-md px-1.5 py-1 text-[11px] text-secondary transition-colors hover:text-content"
                >
                  Clear
                </button>
              )}
              <button
                type="button"
                onClick={onClose}
                aria-label="Close queue"
                className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
              >
                <X className="h-4 w-4" aria-hidden />
              </button>
            </div>
          </header>

          <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
            {current && (
              <>
                <div className="px-2 pt-1 pb-1.5 text-[11px] font-semibold tracking-[0.04em] text-tertiary uppercase">
                  Now Playing
                </div>
                <QueueRow
                  title={current.title}
                  artist={current.artist}
                  artworkId={current.artworkId}
                  color={current.color}
                  durationMs={current.durationMs}
                  active
                  playing={isPlaying}
                  onSelect={() => jumpTo(index)}
                />
              </>
            )}

            {upcoming.length > 0 ? (
              <>
                <div className="px-2 pt-3 pb-1.5 text-[11px] font-semibold tracking-[0.04em] text-tertiary uppercase">
                  Next Up
                </div>
                {upcoming.map((track, offset) => {
                  const position = index + 1 + offset;
                  return (
                    <QueueRow
                      key={`${track.id}-${position}`}
                      title={track.title}
                      artist={track.artist}
                      artworkId={track.artworkId}
                      color={track.color}
                      durationMs={track.durationMs}
                      onSelect={() => jumpTo(position)}
                      onRemove={() => removeAt(position)}
                    />
                  );
                })}
              </>
            ) : (
              <p className="px-2 pt-6 text-[12px] text-tertiary">
                {current ? "Nothing queued after this." : "The queue is empty."}
              </p>
            )}
          </div>
          </motion.aside>
        </>
      )}
    </AnimatePresence>
  );
}

interface RowProps {
  title: string;
  artist: string;
  artworkId?: string;
  color?: string;
  durationMs: number;
  active?: boolean;
  playing?: boolean;
  onSelect(): void;
  onRemove?(): void;
}

function QueueRow({ title, artist, artworkId, color, durationMs, active, playing, onSelect, onRemove }: RowProps) {
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onSelect}
      onKeyDown={(event) => {
        if (event.key === "Enter") onSelect();
      }}
      className={cn(
        "group flex cursor-default items-center gap-2.5 rounded-lg px-2 py-1.5 transition-colors",
        active ? "bg-fill" : "hover:bg-fill",
      )}
    >
      <div className="relative h-9 w-9 shrink-0">
        <Artwork artworkId={artworkId} size={64} color={color} rounded="sm" className="h-9 w-9" />
        {active && (
          <div className="absolute inset-0 grid place-items-center rounded-[4px] bg-black/50">
            <NowPlayingBars playing={Boolean(playing)} />
          </div>
        )}
      </div>

      <div className="min-w-0 flex-1">
        <div className={cn("truncate text-[12.5px] leading-tight", active && "text-accent")}>{title}</div>
        <div className="truncate text-[11.5px] leading-tight text-secondary">{artist}</div>
      </div>

      {onRemove ? (
        <button
          type="button"
          aria-label={`Remove ${title} from queue`}
          onClick={(event) => {
            event.stopPropagation();
            onRemove();
          }}
          className="grid h-6 w-6 shrink-0 place-items-center rounded-full text-tertiary hover:bg-fill-strong hover:text-content max-sm:grid sm:hidden sm:group-hover:grid"
        >
          <X className="h-3.5 w-3.5" aria-hidden />
        </button>
      ) : null}

      <span className="shrink-0 text-[11px] tabular-nums text-tertiary max-sm:hidden sm:group-hover:hidden">
        {formatDurationMs(durationMs)}
      </span>
    </div>
  );
}

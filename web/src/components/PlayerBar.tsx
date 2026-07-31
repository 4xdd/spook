import { ChevronUp, ListMusic, Repeat, Repeat1, Shuffle, SkipBack, SkipForward } from "lucide-react";
import { cn } from "@/lib/cn";
import { usePlayer } from "@/player/PlayerProvider";
import { Artwork } from "./Artwork";
import { IconButton } from "./IconButton";
import { LyricsButton } from "./LyricsButton";
import { PlayButton } from "./PlayButton";
import { Scrubber } from "./Scrubber";
import { VolumeControl } from "./VolumeControl";

interface Props {
  onExpand(): void;
  onToggleLyrics(): void;
  onToggleQueue(): void;
  lyricsOpen: boolean;
  queueOpen: boolean;
}

export function PlayerBar({ onExpand, onToggleLyrics, onToggleQueue, lyricsOpen, queueOpen }: Props) {
  const { current, isPlaying, toggle, next, previous, shuffle, repeat, toggleShuffle, cycleRepeat, error } =
    usePlayer();

  const RepeatIcon = repeat === "one" ? Repeat1 : Repeat;

  return (
    <footer className="material relative z-30 border-t border-separator">
      <div className="mx-auto flex h-[72px] max-w-[1600px] items-center gap-2 px-3 sm:gap-3 sm:px-4">
        {/* Left: transport + volume */}
        <div className="flex shrink-0 items-center gap-2">
          <div className="flex items-center gap-0.5">
            <IconButton label="Previous" onClick={previous} disabled={!current}>
              <SkipBack className="h-[18px] w-[18px]" fill="currentColor" strokeWidth={0} aria-hidden />
            </IconButton>
            <PlayButton playing={isPlaying} onClick={toggle} size="md" />
            <IconButton label="Next" onClick={next} disabled={!current}>
              <SkipForward className="h-[18px] w-[18px]" fill="currentColor" strokeWidth={0} aria-hidden />
            </IconButton>
          </div>
          <VolumeControl className="hidden xl:flex" />
        </div>

        {/* Centre: Apple Music-style LCD */}
        <div
          className={cn(
            "group relative flex h-14 min-w-0 flex-1 items-center gap-3 rounded-lg px-2.5",
            "bg-fill transition-colors duration-150 hover:bg-fill-strong",
          )}
        >
          {current ? (
            <>
              <button
                type="button"
                onClick={onExpand}
                aria-label="Open Now Playing"
                className="shrink-0 transition-transform duration-100 active:scale-95"
              >
                <Artwork
                  artworkId={current.artworkId}
                  size={64}
                  color={current.color}
                  rounded="sm"
                  className="h-10 w-10 shadow-pop"
                />
              </button>

              <div className="flex min-w-0 flex-1 flex-col justify-center gap-1">
                <button
                  type="button"
                  onClick={onExpand}
                  className="min-w-0 text-left"
                  aria-label={`${current.title} by ${current.artist}. Open Now Playing`}
                >
                  <div className="truncate text-[13px] leading-tight">{current.title}</div>
                  <div className="truncate text-[12px] leading-tight text-secondary">
                    {current.artist} — {current.album}
                  </div>
                </button>
                <Scrubber layout="inline" />
              </div>

              <button
                type="button"
                onClick={onExpand}
                aria-label="Open Now Playing"
                className="absolute top-1 right-1.5 text-tertiary opacity-0 transition-opacity duration-150 group-hover:opacity-100"
              >
                <ChevronUp className="h-4 w-4" aria-hidden />
              </button>
            </>
          ) : (
            <div className="flex w-full items-center justify-center text-[13px] text-tertiary">
              {error ?? "Nothing playing"}
            </div>
          )}
        </div>

        {/* Right: lyrics / shuffle / repeat / queue */}
        <div className="flex shrink-0 items-center gap-0.5">
          <LyricsButton
            trackId={current?.id}
            active={lyricsOpen}
            onClick={() => {
              if (!current) return;
              onToggleLyrics();
              if (!lyricsOpen) onExpand();
            }}
          />
          <IconButton label="Shuffle" active={shuffle} onClick={toggleShuffle} disabled={!current}>
            <Shuffle className="h-4 w-4" aria-hidden />
          </IconButton>
          <IconButton
            label={repeat === "off" ? "Repeat off" : repeat === "all" ? "Repeat all" : "Repeat one"}
            active={repeat !== "off"}
            onClick={cycleRepeat}
            disabled={!current}
          >
            <RepeatIcon className="h-4 w-4" aria-hidden />
          </IconButton>
          <IconButton label="Queue" active={queueOpen} onClick={onToggleQueue}>
            <ListMusic className="h-4 w-4" aria-hidden />
          </IconButton>
        </div>
      </div>
    </footer>
  );
}

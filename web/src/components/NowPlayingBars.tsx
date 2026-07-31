import { cn } from "@/lib/cn";

/**
 * The animated level meter next to the playing row. It pauses with the audio,
 * so the row always reflects the real transport state.
 */
export function NowPlayingBars({ playing, className }: { playing: boolean; className?: string }) {
  return (
    <span className={cn("flex h-3 items-end gap-[2px]", className)} aria-hidden>
      {[0, 1, 2].map((bar) => (
        <span
          key={bar}
          className="w-[2px] rounded-full bg-accent"
          style={{
            height: playing ? undefined : "40%",
            animation: playing ? `spook-bar 900ms ease-in-out ${bar * 160}ms infinite` : undefined,
          }}
        />
      ))}
      <style>{`
        @keyframes spook-bar {
          0%, 100% { height: 25%; }
          50% { height: 100%; }
        }
        @media (prefers-reduced-motion: reduce) {
          @keyframes spook-bar {
            0%, 100% { height: 60%; }
          }
        }
      `}</style>
    </span>
  );
}

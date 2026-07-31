import { Volume1, Volume2, VolumeX } from "lucide-react";
import { cn } from "@/lib/cn";
import { usePlayer } from "@/player/PlayerProvider";
import { useDragValue } from "@/player/useDragValue";

export function VolumeControl({ className }: { className?: string }) {
  const { volume, muted, setVolume, toggleMute } = usePlayer();
  const effective = muted ? 0 : volume;

  const drag = useDragValue({
    value: effective,
    max: 1,
    onChange: setVolume,
    knobRadius: 6,
  });

  const Icon = effective === 0 ? VolumeX : effective < 0.5 ? Volume1 : Volume2;

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <button
        type="button"
        onClick={toggleMute}
        aria-label={muted ? "Unmute" : "Mute"}
        className="text-tertiary transition-colors hover:text-secondary active:scale-90"
      >
        <Icon className="h-4 w-4" aria-hidden />
      </button>

      <div
        ref={drag.ref}
        role="slider"
        tabIndex={0}
        aria-label="Volume"
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={Math.round(drag.displayValue * 100)}
        onPointerDown={drag.onPointerDown}
        onKeyDown={drag.onKeyDown}
        className="group relative flex h-4 w-24 cursor-pointer touch-none items-center"
      >
        <div className="h-1 w-full overflow-hidden rounded-full bg-fill-strong">
          <div className="h-full rounded-full bg-content/60" style={{ width: `${drag.displayValue * 100}%` }} />
        </div>
        <div
          className={cn(
            "pointer-events-none absolute top-1/2 h-2.5 w-2.5 -translate-x-1/2 -translate-y-1/2 rounded-full bg-content shadow-pop",
            "opacity-0 transition-opacity duration-150 group-hover:opacity-100",
            drag.isDragging && "opacity-100",
          )}
          style={{ left: `${drag.displayValue * 100}%` }}
          aria-hidden
        />
      </div>
    </div>
  );
}

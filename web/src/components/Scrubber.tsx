import { useState } from "react";
import { cn } from "@/lib/cn";
import { formatTime } from "@/lib/format";
import { usePlayer, useProgress } from "@/player/PlayerProvider";
import { useDragValue } from "@/player/useDragValue";

interface Props {
  className?: string;
  /** The expanded player uses a taller track and larger labels. */
  size?: "sm" | "lg";
  /** Inline puts the times beside the track, which fits the compact player bar. */
  layout?: "stacked" | "inline";
}

export function Scrubber({ className, size = "sm", layout = "stacked" }: Props) {
  const { current, seek } = usePlayer();
  const { currentTime, duration } = useProgress();
  const [hovered, setHovered] = useState(false);

  const total = duration || (current?.durationMs ?? 0) / 1000;
  const disabled = !current || total <= 0;

  const drag = useDragValue({
    value: Math.min(currentTime, total),
    max: total,
    onChange: seek,
    disabled,
  });

  const ratio = total > 0 ? Math.min(drag.displayValue / total, 1) : 0;
  const expanded = hovered || drag.isDragging;

  const elapsed = formatTime(drag.displayValue);
  const remaining = total > 0 ? `-${formatTime(total - drag.displayValue)}` : "--:--";

  const timeClass = cn(
    "shrink-0 font-medium tabular-nums text-tertiary",
    size === "lg" ? "text-xs" : "text-[11px]",
  );

  const slider = (
    <div
      ref={drag.ref}
      role="slider"
      tabIndex={disabled ? -1 : 0}
      aria-label="Seek"
      aria-valuemin={0}
      aria-valuemax={Math.round(total)}
      aria-valuenow={Math.round(drag.displayValue)}
      aria-valuetext={`${elapsed} of ${formatTime(total)}`}
      aria-disabled={disabled}
      onPointerDown={drag.onPointerDown}
      onKeyDown={drag.onKeyDown}
      onPointerEnter={() => setHovered(true)}
      onPointerLeave={() => setHovered(false)}
      className={cn(
        "group relative flex min-w-0 flex-1 touch-none items-center",
        size === "lg" ? "h-5" : "h-4",
        disabled ? "cursor-default opacity-50" : "cursor-pointer",
      )}
    >
      <div
        className={cn(
          "relative w-full overflow-hidden rounded-full bg-fill-strong transition-[height] duration-150 ease-out",
          expanded ? (size === "lg" ? "h-2" : "h-1.5") : size === "lg" ? "h-1.5" : "h-1",
        )}
      >
        <div className="h-full rounded-full bg-content/80" style={{ width: `${ratio * 100}%` }} />
      </div>
      <div
        className={cn(
          "pointer-events-none absolute top-1/2 -translate-x-1/2 -translate-y-1/2 rounded-full bg-content shadow-pop",
          "transition-[opacity,transform] duration-150 ease-out",
          expanded ? "opacity-100" : "scale-50 opacity-0",
          size === "lg" ? "h-3.5 w-3.5" : "h-3 w-3",
        )}
        style={{ left: `${ratio * 100}%` }}
        aria-hidden
      />
    </div>
  );

  if (layout === "inline") {
    return (
      <div className={cn("flex min-w-0 items-center gap-2", className)}>
        <span className={timeClass}>{elapsed}</span>
        {slider}
        <span className={timeClass}>{remaining}</span>
      </div>
    );
  }

  return (
    <div className={cn("flex w-full flex-col gap-1.5", className)}>
      {slider}
      <div className={cn("flex justify-between", timeClass)}>
        <span>{elapsed}</span>
        <span>{remaining}</span>
      </div>
    </div>
  );
}

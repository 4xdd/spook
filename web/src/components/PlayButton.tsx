import { Pause, Play } from "lucide-react";
import { cn } from "@/lib/cn";

interface Props {
  playing?: boolean;
  onClick(event: React.MouseEvent): void;
  size?: "sm" | "md" | "lg";
  label?: string;
  className?: string;
}

const sizes = {
  sm: "h-8 w-8",
  md: "h-11 w-11",
  lg: "h-14 w-14",
};

const iconSizes = {
  sm: "h-3.5 w-3.5",
  md: "h-5 w-5",
  lg: "h-6 w-6",
};

/**
 * A circular transport button. Press feedback fires on pointer-down via
 * :active so it never waits for the click to land.
 */
export function PlayButton({ playing, onClick, size = "md", label, className }: Props) {
  const Icon = playing ? Pause : Play;
  return (
    <button
      type="button"
      onClick={onClick}
      aria-label={label ?? (playing ? "Pause" : "Play")}
      className={cn(
        "group/play grid place-items-center rounded-full bg-accent text-accent-content shadow-pop",
        "transition-transform duration-100 ease-out active:scale-[0.92] hover:brightness-110",
        sizes[size],
        className,
      )}
    >
      <Icon
        className={cn(iconSizes[size], !playing && "translate-x-[1px]")}
        fill="currentColor"
        strokeWidth={0}
        aria-hidden
      />
    </button>
  );
}

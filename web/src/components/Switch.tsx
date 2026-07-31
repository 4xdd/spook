import { cn } from "@/lib/cn";

interface Props {
  checked: boolean;
  onChange(checked: boolean): void;
  /** Names the control for assistive tech; never rendered. */
  label: string;
  disabled?: boolean;
  className?: string;
}

export function Switch({ checked, onChange, label, disabled, className }: Props) {
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      aria-label={label}
      disabled={disabled}
      onClick={() => onChange(!checked)}
      className={cn(
        "relative h-[26px] w-[44px] shrink-0 rounded-full transition-colors duration-200 ease-out",
        "disabled:cursor-not-allowed disabled:opacity-40",
        checked ? "bg-accent" : "bg-fill-strong",
        className,
      )}
    >
      <span
        className={cn(
          "absolute top-[3px] left-[3px] h-5 w-5 rounded-full bg-white shadow-pop",
          "transition-transform duration-200 ease-out",
          checked && "translate-x-[18px]",
        )}
        aria-hidden
      />
    </button>
  );
}

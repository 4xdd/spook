import { forwardRef } from "react";
import { cn } from "@/lib/cn";

interface Props extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  active?: boolean;
  size?: "sm" | "md";
}

const sizes = {
  sm: "h-8 w-8",
  md: "h-9 w-9",
};

export const IconButton = forwardRef<HTMLButtonElement, Props>(function IconButton(
  { label, active, size = "md", className, children, ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      type="button"
      aria-label={label}
      title={label}
      aria-pressed={active}
      className={cn(
        "grid shrink-0 place-items-center rounded-full transition-[transform,color,background-color] duration-100 ease-out",
        "hover:bg-fill active:scale-[0.92] disabled:cursor-not-allowed disabled:hover:bg-transparent",
        rest.disabled ? "disabled:opacity-40" : "",
        active ? "text-accent" : "text-secondary hover:text-content",
        sizes[size],
        className,
      )}
      {...rest}
    >
      {children}
    </button>
  );
});

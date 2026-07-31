import { Check, ChevronDown } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/cn";
import { CUSTOM_PRESET, EQ_PRESETS } from "@/lib/eq";

interface Props {
  value: string;
  disabled?: boolean;
  onChange(name: string): void;
}

/**
 * Custom listbox for EQ presets. A native <select> is clipped by the settings
 * dialog's overflow:hidden (especially in Firefox), so this renders the menu
 * into a portal and positions it against the trigger.
 */
export function PresetSelect({ value, disabled, onChange }: Props) {
  const [open, setOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);
  const listId = useId();
  const [coords, setCoords] = useState<{ top: number; left: number; width: number } | null>(null);

  const options =
    value === CUSTOM_PRESET
      ? [{ name: CUSTOM_PRESET }, ...EQ_PRESETS]
      : EQ_PRESETS;

  useLayoutEffect(() => {
    if (!open) return;

    const place = () => {
      const trigger = triggerRef.current;
      if (!trigger) return;
      const rect = trigger.getBoundingClientRect();
      const menuHeight = Math.min(280, options.length * 36 + 8);
      const spaceBelow = window.innerHeight - rect.bottom - 12;
      const openUp = spaceBelow < menuHeight && rect.top > spaceBelow;
      setCoords({
        top: openUp ? rect.top - menuHeight - 6 : rect.bottom + 6,
        left: rect.left,
        width: rect.width,
      });
    };

    place();
    window.addEventListener("resize", place);
    window.addEventListener("scroll", place, true);
    return () => {
      window.removeEventListener("resize", place);
      window.removeEventListener("scroll", place, true);
    };
  }, [open, options.length]);

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node;
      if (triggerRef.current?.contains(target) || menuRef.current?.contains(target)) return;
      setOpen(false);
    };
    // Capture so Escape closes the menu before the settings dialog sees it.
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.stopPropagation();
      event.preventDefault();
      setOpen(false);
      triggerRef.current?.focus();
    };

    document.addEventListener("pointerdown", onPointerDown);
    window.addEventListener("keydown", onKeyDown, true);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      window.removeEventListener("keydown", onKeyDown, true);
    };
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const selected = menuRef.current?.querySelector<HTMLElement>('[aria-selected="true"]');
    selected?.scrollIntoView({ block: "nearest" });
  }, [open]);

  return (
    <div className="relative min-w-0 flex-1">
      <button
        ref={triggerRef}
        type="button"
        id={listId}
        disabled={disabled}
        aria-label="Equalizer preset"
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={open ? `${listId}-list` : undefined}
        onClick={() => setOpen((value) => !value)}
        className={cn(
          "flex w-full items-center justify-between gap-2 rounded-lg bg-fill py-1.5 pr-2.5 pl-2.5 text-left text-[13px] text-content",
          "transition-colors outline-none hover:bg-fill-strong focus:bg-fill-strong",
          "disabled:cursor-not-allowed",
        )}
      >
        <span className="truncate">{value}</span>
        <ChevronDown
          className={cn("h-3.5 w-3.5 shrink-0 text-tertiary transition-transform", open && "rotate-180")}
          aria-hidden
        />
      </button>

      {createPortal(
        <AnimatePresence>
          {open && coords && (
            <motion.div
              ref={menuRef}
              id={`${listId}-list`}
              role="listbox"
              aria-labelledby={listId}
              initial={{ opacity: 0, y: -4, scale: 0.98 }}
              animate={{ opacity: 1, y: 0, scale: 1 }}
              exit={{ opacity: 0, y: -2, scale: 0.98 }}
              transition={{ type: "spring", bounce: 0, duration: 0.2 }}
              style={{
                position: "fixed",
                top: coords.top,
                left: coords.left,
                width: coords.width,
                transformOrigin: "top center",
              }}
              className="z-[60] max-h-[280px] overflow-y-auto rounded-xl border border-separator bg-canvas p-1 shadow-pop"
            >
              {options.map((option) => {
                const selected = option.name === value;
                return (
                  <button
                    key={option.name}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => {
                      if (option.name !== CUSTOM_PRESET) onChange(option.name);
                      setOpen(false);
                      triggerRef.current?.focus();
                    }}
                    className={cn(
                      "flex w-full items-center justify-between gap-2 rounded-lg px-2.5 py-1.5 text-left text-[13px] transition-colors",
                      selected
                        ? "bg-fill-strong text-content"
                        : "text-content hover:bg-accent hover:text-accent-content",
                      option.name === CUSTOM_PRESET && "text-secondary",
                    )}
                  >
                    <span className="truncate">{option.name}</span>
                    {selected && <Check className="h-3.5 w-3.5 shrink-0 text-accent" aria-hidden />}
                  </button>
                );
              })}
            </motion.div>
          )}
        </AnimatePresence>,
        document.body,
      )}
    </div>
  );
}

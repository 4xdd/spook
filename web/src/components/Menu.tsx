import { AnimatePresence, motion } from "motion/react";
import { useEffect, useId, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import type { MenuItem } from "./MenuPanel";
import { MenuPanel } from "./MenuPanel";

export type { MenuItem } from "./MenuPanel";

interface Props {
  items: MenuItem[];
  label: string;
  children: React.ReactNode;
  align?: "left" | "right";
  className?: string;
  onBeforeOpen?(): void | Promise<void>;
}

/**
 * A small dropdown anchored to its trigger, so the panel visibly grows out of
 * the control that opened it rather than appearing from nowhere.
 */
export function Menu({ items, label, children, align = "right", className, onBeforeOpen }: Props) {
  const [open, setOpen] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);
  const menuId = useId();

  useEffect(() => {
    if (!open) return;

    const onPointerDown = (event: PointerEvent) => {
      if (!containerRef.current?.contains(event.target as Node)) setOpen(false);
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  const close = () => setOpen(false);

  return (
    <div ref={containerRef} className={cn("relative", className)}>
      <button
        type="button"
        aria-label={label}
        aria-haspopup="menu"
        aria-expanded={open}
        aria-controls={open ? menuId : undefined}
        onClick={async (event) => {
          event.stopPropagation();
          await onBeforeOpen?.();
          setOpen((value) => !value);
        }}
        className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform duration-100 hover:bg-fill hover:text-content active:scale-90"
      >
        {children}
      </button>

      <AnimatePresence>
        {open && (
          <motion.div
            id={menuId}
            role="menu"
            aria-label={label}
            initial={{ opacity: 0, scale: 0.94, y: -4 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.96, y: -2 }}
            transition={{ type: "spring", bounce: 0, duration: 0.22 }}
            style={{ transformOrigin: align === "right" ? "top right" : "top left" }}
            className={cn(
              "material absolute z-50 mt-1 min-w-48 overflow-visible rounded-xl border border-separator p-1 shadow-pop",
              align === "right" ? "right-0" : "left-0",
            )}
          >
            <MenuPanel items={items} onClose={close} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}

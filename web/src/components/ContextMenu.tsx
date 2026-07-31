import { AnimatePresence, motion } from "motion/react";
import { useEffect, useId, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import type { MenuItem } from "./MenuPanel";
import { MenuPanel } from "./MenuPanel";

interface Props {
  open: boolean;
  x: number;
  y: number;
  items: MenuItem[];
  label: string;
  onClose(): void;
}

/** Right-click (or long-press) menu anchored to viewport coordinates. */
export function ContextMenu({ open, x, y, items, label, onClose }: Props) {
  const menuId = useId();
  const panelRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ left: x, top: y });

  useLayoutEffect(() => {
    if (!open || !panelRef.current) return;
    const rect = panelRef.current.getBoundingClientRect();
    const margin = 8;
    let left = x;
    let top = y;
    if (left + rect.width > window.innerWidth - margin) {
      left = Math.max(margin, window.innerWidth - rect.width - margin);
    }
    if (top + rect.height > window.innerHeight - margin) {
      top = Math.max(margin, window.innerHeight - rect.height - margin);
    }
    setPosition({ left, top });
  }, [open, x, y, items]);

  useEffect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      if (!panelRef.current?.contains(event.target as Node)) onClose();
    };
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") onClose();
    };
    const onScroll = () => onClose();

    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    window.addEventListener("scroll", onScroll, true);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("scroll", onScroll, true);
    };
  }, [open, onClose]);

  return createPortal(
    <AnimatePresence>
      {open && (
        <motion.div
          ref={panelRef}
          id={menuId}
          role="menu"
          aria-label={label}
          initial={{ opacity: 0, scale: 0.96 }}
          animate={{ opacity: 1, scale: 1 }}
          exit={{ opacity: 0, scale: 0.98 }}
          transition={{ type: "spring", bounce: 0, duration: 0.18 }}
          style={{ left: position.left, top: position.top, transformOrigin: "top left" }}
          className="material fixed z-[100] min-w-48 overflow-visible rounded-xl border border-separator p-1 shadow-pop"
        >
          <MenuPanel items={items} onClose={onClose} />
        </motion.div>
      )}
    </AnimatePresence>,
    document.body,
  );
}

export function useContextMenu() {
  const [state, setState] = useState<{ x: number; y: number } | null>(null);

  const openAt = (event: React.MouseEvent) => {
    event.preventDefault();
    event.stopPropagation();
    setState({ x: event.clientX, y: event.clientY });
  };

  const close = () => setState(null);

  return { position: state, openAt, close, isOpen: state !== null };
}

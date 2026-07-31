import { Menu, Search, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { useLocation } from "react-router-dom";
import { Sidebar } from "./Sidebar";

interface Props {
  onOpenSettings(): void;
}

export interface MobileNavHandle {
  focusSearch(): void;
}

export const MobileNav = forwardRef<MobileNavHandle, Props>(function MobileNav({ onOpenSettings }, ref) {
  const [open, setOpen] = useState(false);
  const searchRef = useRef<HTMLInputElement>(null);
  const location = useLocation();

  useImperativeHandle(ref, () => ({
    focusSearch() {
      setOpen(true);
      requestAnimationFrame(() => searchRef.current?.focus());
    },
  }));

  // Route changes close the drawer so the new page is immediately visible.
  useEffect(() => {
    setOpen(false);
  }, [location.pathname, location.search]);

  useEffect(() => {
    if (!open) return;
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") setOpen(false);
    };
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [open]);

  return (
    <>
      <header className="material flex h-12 shrink-0 items-center gap-2 border-b border-separator px-3 pt-safe sm:hidden">
        <button
          type="button"
          onClick={() => setOpen(true)}
          aria-label="Open menu"
          className="grid h-9 w-9 shrink-0 place-items-center rounded-full text-secondary transition-colors hover:bg-fill hover:text-content active:scale-90"
        >
          <Menu className="h-5 w-5" aria-hidden />
        </button>

        <span className="min-w-0 flex-1 truncate text-[15px] font-semibold tracking-[-0.01em]">Spook</span>

        <button
          type="button"
          onClick={() => {
            setOpen(true);
            requestAnimationFrame(() => searchRef.current?.focus());
          }}
          aria-label="Search library"
          className="grid h-9 w-9 shrink-0 place-items-center rounded-full text-secondary transition-colors hover:bg-fill hover:text-content active:scale-90"
        >
          <Search className="h-4.5 w-4.5" aria-hidden />
        </button>
      </header>

      <AnimatePresence>
        {open && (
          <>
            <motion.button
              key="mobile-nav-scrim"
              type="button"
              aria-label="Close menu"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              exit={{ opacity: 0 }}
              transition={{ duration: 0.18 }}
              onClick={() => setOpen(false)}
              className="fixed inset-0 z-40 bg-black/45 sm:hidden"
            />

            <motion.aside
              key="mobile-nav-drawer"
              aria-label="Navigation"
              initial={{ x: "-100%" }}
              animate={{ x: 0 }}
              exit={{ x: "-100%" }}
              transition={{ type: "spring", bounce: 0, duration: 0.35 }}
              className="fixed inset-y-0 left-0 z-50 w-[min(18rem,88vw)] bg-canvas pt-safe sm:hidden"
            >
              <div className="relative flex h-full flex-col">
                <button
                  type="button"
                  onClick={() => setOpen(false)}
                  aria-label="Close menu"
                  className="absolute top-3 right-2 z-10 grid h-8 w-8 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
                >
                  <X className="h-4 w-4" aria-hidden />
                </button>
                <Sidebar
                  ref={searchRef}
                  onOpenSettings={() => {
                    setOpen(false);
                    onOpenSettings();
                  }}
                />
              </div>
            </motion.aside>
          </>
        )}
      </AnimatePresence>
    </>
  );
});

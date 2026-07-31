import { Monitor, Moon, Sun, X } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";
import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import { applyTheme, persistTheme, readTheme, type Theme } from "@/lib/theme";
import { AccessSettings } from "./AccessSettings";
import { Equalizer } from "./Equalizer";
import { LastfmSettings } from "./LastfmSettings";

const themes: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: "system", label: "System", icon: Monitor },
  { value: "light", label: "Light", icon: Sun },
  { value: "dark", label: "Dark", icon: Moon },
];

const shortcuts: [string, string][] = [
  ["Space", "Play or pause"],
  ["← / →", "Seek 10s"],
  ["↑ / ↓", "Volume"],
  ["N / P", "Next / previous"],
  ["S / R", "Shuffle / repeat"],
  ["M", "Mute"],
  ["/", "Search"],
  [",", "Settings"],
];

export function SettingsPanel({
  open,
  onClose,
  lastfmError,
  lastfmRefresh,
  onLock,
}: {
  open: boolean;
  onClose(): void;
  lastfmError?: string | null;
  lastfmRefresh?: number;
  onLock(): void;
}) {
  const [theme, setTheme] = useState<Theme>(readTheme);
  const panelRef = useRef<HTMLDivElement>(null);
  const restoreFocusRef = useRef<HTMLElement | null>(null);

  function chooseTheme(next: Theme) {
    setTheme(next);
    applyTheme(next);
    persistTheme(next);
  }

  useEffect(() => {
    if (!open) return;

    restoreFocusRef.current = document.activeElement as HTMLElement | null;
    panelRef.current?.focus();
    // The panel owns Escape while it is up, so it does not also reach the
    // player shortcuts underneath.
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.stopPropagation();
      onClose();
    };

    // Bubble phase so nested overlays (EQ preset list) can swallow Escape first.
    window.addEventListener("keydown", onKeyDown);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      restoreFocusRef.current?.focus();
    };
  }, [onClose, open]);

  return (
    <AnimatePresence>
      {open && (
        <div className="fixed inset-0 z-40 grid place-items-center p-4">
          <motion.div
            key="settings-scrim"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.18 }}
            onClick={onClose}
            className="absolute inset-0 bg-black/45"
            aria-hidden
          />

          <motion.div
            key="settings-panel"
            ref={panelRef}
            role="dialog"
            aria-modal="true"
            aria-label="Settings"
            tabIndex={-1}
            initial={{ opacity: 0, scale: 0.96, y: 8 }}
            animate={{ opacity: 1, scale: 1, y: 0 }}
            exit={{ opacity: 0, scale: 0.97, y: 4 }}
            transition={{ type: "spring", bounce: 0, duration: 0.28 }}
            className={cn(
              "material relative flex max-h-[min(88vh,720px)] w-full max-w-lg flex-col",
              "overflow-hidden rounded-2xl border border-separator shadow-pop outline-none",
            )}
          >
            <header className="flex shrink-0 items-center justify-between gap-2 border-b border-separator px-5 py-3.5">
              <h2 className="text-[15px] font-semibold">Settings</h2>
              <button
                type="button"
                onClick={onClose}
                aria-label="Close settings"
                className="grid h-7 w-7 place-items-center rounded-full text-secondary transition-transform hover:bg-fill hover:text-content active:scale-90"
              >
                <X className="h-4 w-4" aria-hidden />
              </button>
            </header>

            <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-5 py-5">
              <section className="flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <h3 className="text-[13px] font-semibold">Appearance</h3>
                  <p className="text-[11.5px] text-tertiary">System follows your device setting.</p>
                </div>
                <div className="flex shrink-0 gap-0.5 rounded-lg bg-fill p-0.5" role="group" aria-label="Theme">
                  {themes.map(({ value, label, icon: Icon }) => (
                    <button
                      key={value}
                      type="button"
                      aria-pressed={theme === value}
                      onClick={() => chooseTheme(value)}
                      className={cn(
                        "flex items-center gap-1.5 rounded-[7px] px-2.5 py-1 text-[12px] transition-colors",
                        theme === value
                          ? "bg-raised text-content shadow-pop"
                          : "text-secondary hover:text-content",
                      )}
                    >
                      <Icon className="h-3.5 w-3.5" aria-hidden />
                      {label}
                    </button>
                  ))}
                </div>
              </section>

              <div className="h-px bg-separator" aria-hidden />

              <Equalizer />

              <div className="h-px bg-separator" aria-hidden />

              <LastfmSettings authError={lastfmError} refreshKey={lastfmRefresh} />

              <div className="h-px bg-separator" aria-hidden />

              <AccessSettings onLock={onLock} />

              <div className="h-px bg-separator" aria-hidden />

              <section>
                <h3 className="text-[13px] font-semibold">Keyboard shortcuts</h3>
                <dl className="mt-2 grid grid-cols-2 gap-x-6 gap-y-1.5">
                  {shortcuts.map(([keys, action]) => (
                    <div key={keys} className="flex items-baseline justify-between gap-2">
                      <dt className="text-[12px] text-secondary">{action}</dt>
                      <dd className="rounded-md bg-fill px-1.5 py-0.5 text-[11px] text-tertiary tabular-nums">
                        {keys}
                      </dd>
                    </div>
                  ))}
                </dl>
              </section>
            </div>
          </motion.div>
        </div>
      )}
    </AnimatePresence>
  );
}

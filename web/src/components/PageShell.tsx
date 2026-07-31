import { useEffect, useRef, useState } from "react";
import { cn } from "@/lib/cn";

interface Props {
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
  children: React.ReactNode;
  /** Detail pages render their own hero instead of the standard big title. */
  hero?: React.ReactNode;
  tint?: string;
}

/**
 * The scrolling content area. Content slides under a translucent header, whose
 * compact title fades in only once the large one has scrolled away.
 */
export function PageShell({ title, subtitle, actions, children, hero, tint }: Props) {
  const scrollRef = useRef<HTMLDivElement>(null);
  const [condensed, setCondensed] = useState(false);

  useEffect(() => {
    const element = scrollRef.current;
    if (!element) return;

    let frame = 0;
    const onScroll = () => {
      cancelAnimationFrame(frame);
      frame = requestAnimationFrame(() => setCondensed(element.scrollTop > (hero ? 220 : 52)));
    };

    element.addEventListener("scroll", onScroll, { passive: true });
    return () => {
      cancelAnimationFrame(frame);
      element.removeEventListener("scroll", onScroll);
    };
  }, [hero]);

  return (
    <div className="relative flex h-full min-h-0 flex-col">
      <header
        className={cn(
          "absolute inset-x-0 top-0 z-20 flex h-12 items-center gap-3 px-6 transition-colors duration-200",
          condensed ? "material border-b border-separator" : "border-b border-transparent",
        )}
      >
        <h2
          className={cn(
            "truncate text-[15px] font-semibold transition-[opacity,transform] duration-200 ease-out",
            condensed ? "translate-y-0 opacity-100" : "translate-y-1 opacity-0",
          )}
          aria-hidden={!condensed}
        >
          {title}
        </h2>
        <div className="ml-auto flex items-center gap-2">{actions}</div>
      </header>

      <div ref={scrollRef} className="scroll-edge min-h-0 flex-1 overflow-y-auto overflow-x-hidden">
        {tint && (
          <div
            className="pointer-events-none absolute inset-x-0 top-0 -z-10 h-96"
            style={{
              background: `linear-gradient(to bottom, color-mix(in oklab, ${tint} 45%, transparent), transparent)`,
            }}
            aria-hidden
          />
        )}

        {hero ?? (
          <div className="px-6 pt-16 pb-5">
            <h1 className="text-[30px] font-bold">{title}</h1>
            {subtitle && <p className="mt-0.5 text-[14px] text-secondary">{subtitle}</p>}
          </div>
        )}

        <div className="px-6 pb-10">{children}</div>
      </div>
    </div>
  );
}

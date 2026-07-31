import { ChevronRight } from "lucide-react";
import { useState } from "react";
import { cn } from "@/lib/cn";

export interface MenuItem {
  label: string;
  onSelect?(): void;
  icon?: React.ReactNode;
  destructive?: boolean;
  disabled?: boolean;
  checked?: boolean;
  children?: MenuItem[];
}

interface Props {
  items: MenuItem[];
  onClose(): void;
}

export function MenuPanel({ items, onClose }: Props) {
  return (
    <>
      {items.map((item) => (
        <MenuPanelRow key={item.label} item={item} onClose={onClose} />
      ))}
    </>
  );
}

function MenuPanelRow({ item, onClose }: { item: MenuItem; onClose(): void }) {
  const [subOpen, setSubOpen] = useState(false);
  const hasChildren = item.children && item.children.length > 0;

  if (hasChildren) {
    return (
      <div
        className="relative"
        onMouseEnter={() => setSubOpen(true)}
        onMouseLeave={() => setSubOpen(false)}
      >
        <button
          type="button"
          role="menuitem"
          aria-haspopup="menu"
          aria-expanded={subOpen}
          className={cn(
            "flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left text-[13px] transition-colors",
            "hover:bg-accent hover:text-accent-content text-content",
          )}
        >
          {item.icon}
          <span className="min-w-0 flex-1 truncate">{item.label}</span>
          <ChevronRight className="h-3.5 w-3.5 shrink-0 opacity-60" aria-hidden />
        </button>
        {subOpen && (
          <div
            role="menu"
            className="material absolute top-0 left-full z-10 ml-1 min-w-44 overflow-hidden rounded-xl border border-separator p-1 shadow-pop"
          >
            <MenuPanel items={item.children!} onClose={onClose} />
          </div>
        )}
      </div>
    );
  }

  return (
    <button
      type="button"
      role="menuitem"
      disabled={item.disabled}
      onClick={(event) => {
        event.stopPropagation();
        if (item.disabled) return;
        onClose();
        item.onSelect?.();
      }}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-lg px-2.5 py-1.5 text-left text-[13px] transition-colors",
        "hover:bg-accent hover:text-accent-content disabled:cursor-not-allowed disabled:opacity-40",
        item.destructive ? "text-red-400" : "text-content",
      )}
    >
      {item.icon}
      <span className="min-w-0 flex-1 truncate">{item.label}</span>
      {item.checked && <span className="text-[11px] text-tertiary">✓</span>}
    </button>
  );
}

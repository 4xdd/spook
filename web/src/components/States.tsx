import { Loader2, Music4 } from "lucide-react";
import { cn } from "@/lib/cn";

export function LoadingState({ className }: { className?: string }) {
  return (
    <div className={cn("flex items-center justify-center gap-2 py-20 text-secondary", className)}>
      <Loader2 className="h-4 w-4 animate-spin" aria-hidden />
      <span className="text-[13px]">Loading…</span>
    </div>
  );
}

export function EmptyState({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex flex-col items-center justify-center gap-3 py-24 text-center">
      <span className="grid h-12 w-12 place-items-center rounded-full bg-fill text-tertiary">
        <Music4 className="h-5 w-5" aria-hidden />
      </span>
      <div>
        <p className="text-[15px] font-medium">{title}</p>
        {description && <p className="mt-1 max-w-sm text-[13px] text-secondary">{description}</p>}
      </div>
      {action}
    </div>
  );
}

export function ErrorState({ message }: { message: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-24 text-center">
      <p className="text-[15px] font-medium">Something went wrong</p>
      <p className="max-w-sm text-[13px] text-secondary">{message}</p>
    </div>
  );
}

/** The shared responsive album grid. */
export function AlbumGrid({ children }: { children: React.ReactNode }) {
  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(9.5rem,1fr))] gap-x-5 gap-y-7">{children}</div>
  );
}

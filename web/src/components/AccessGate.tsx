import { KeyRound, Loader2, Music2 } from "lucide-react";
import { useState, type FormEvent } from "react";
import { ApiError, verifyAccessKey } from "@/lib/api";
import { persistAccessKey } from "@/lib/accessKey";
import { cn } from "@/lib/cn";

export function AccessGate({ onUnlocked }: { onUnlocked(): void }) {
  const [key, setKey] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function onSubmit(event: FormEvent) {
    event.preventDefault();
    const candidate = key.trim();
    if (!candidate) {
      setError("Enter an access key");
      return;
    }

    setBusy(true);
    setError(null);
    try {
      await verifyAccessKey(candidate);
      persistAccessKey(candidate);
      onUnlocked();
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        setError("That access key is not valid");
      } else {
        setError(err instanceof Error ? err.message : "Unable to check access key");
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid h-full place-items-center bg-canvas px-4">
      <form
        onSubmit={onSubmit}
        className="flex w-full max-w-sm flex-col gap-5"
        aria-labelledby="access-gate-title"
      >
        <div className="flex flex-col items-center gap-3 text-center">
          <span className="grid h-12 w-12 place-items-center rounded-2xl bg-accent text-accent-content shadow-pop">
            <Music2 className="h-6 w-6" aria-hidden />
          </span>
          <div>
            <h1 id="access-gate-title" className="text-[22px] font-semibold tracking-[-0.02em]">
              Spook
            </h1>
            <p className="mt-1 text-[13px] text-secondary">Enter an access key to open your library.</p>
          </div>
        </div>

        <div className="flex flex-col gap-1.5">
          <label htmlFor="access-key" className="text-[12px] font-medium text-secondary">
            Access key
          </label>
          <div className="relative">
            <KeyRound
              className="pointer-events-none absolute top-1/2 left-3 h-3.5 w-3.5 -translate-y-1/2 text-tertiary"
              aria-hidden
            />
            <input
              id="access-key"
              name="access-key"
              type="password"
              autoComplete="current-password"
              autoFocus
              value={key}
              onChange={(event) => {
                setKey(event.target.value);
                if (error) setError(null);
              }}
              aria-invalid={error ? true : undefined}
              aria-describedby={error ? "access-key-error" : undefined}
              className={cn(
                "w-full rounded-xl bg-fill py-2.5 pr-3 pl-9 text-[14px] text-content placeholder:text-tertiary",
                "outline-none transition-[background-color,box-shadow] focus:bg-fill-strong",
                error && "ring-1 ring-accent",
              )}
              placeholder="Paste your key"
            />
          </div>
          {error && (
            <p id="access-key-error" role="alert" className="text-[12px] text-accent">
              {error}
            </p>
          )}
        </div>

        <button
          type="submit"
          disabled={busy}
          className={cn(
            "inline-flex items-center justify-center gap-2 rounded-xl bg-accent px-4 py-2.5",
            "text-[14px] font-medium text-accent-content transition-[transform,opacity]",
            "hover:opacity-95 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-60",
          )}
        >
          {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : null}
          {busy ? "Checking" : "Unlock"}
        </button>
      </form>
    </div>
  );
}

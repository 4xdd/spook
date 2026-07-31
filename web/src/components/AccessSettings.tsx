import { persistAccessKey, readAccessKey } from "@/lib/accessKey";

export function AccessSettings({ onLock }: { onLock(): void }) {
  const key = readAccessKey();
  const preview = key
    ? key.length > 10
      ? `${key.slice(0, 4)}…${key.slice(-4)}`
      : "••••"
    : null;

  function lock() {
    persistAccessKey(null);
    onLock();
  }

  return (
    <section className="flex flex-col gap-3">
      <div className="min-w-0">
        <h3 className="text-[13px] font-semibold">Access</h3>
        <p className="text-[11.5px] text-tertiary">
          {preview
            ? `This browser is unlocked with key ${preview}. Keys stay on this device.`
            : "No access key is stored in this browser."}
        </p>
      </div>
      {key && (
        <button
          type="button"
          onClick={lock}
          className="self-start rounded-lg bg-fill px-2.5 py-1.5 text-[12.5px] text-content transition-colors hover:bg-fill-strong"
        >
          Lock this browser
        </button>
      )}
    </section>
  );
}

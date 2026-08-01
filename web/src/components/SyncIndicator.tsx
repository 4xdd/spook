import { Check, Loader2 } from "lucide-react";
import { cn } from "@/lib/cn";
import type { EmbeddingStatus, ScanStatus } from "@/lib/api";
import { formatEtaRemaining } from "@/lib/format";
import { useEmbedEta } from "@/hooks/useEmbedEta";

type StepState = "done" | "active" | "waiting" | "error" | "off";

interface Props {
  scan: ScanStatus;
  embeddings: EmbeddingStatus;
}

function pct(processed: number, total: number) {
  if (total <= 0) return null;
  return Math.min(100, Math.round((processed / total) * 100));
}

function stepIcon(state: StepState) {
  if (state === "active") {
    return <Loader2 className="h-3 w-3 shrink-0 animate-spin text-accent" aria-hidden />;
  }
  if (state === "done") {
    return <Check className="h-3 w-3 shrink-0 text-accent" aria-hidden />;
  }
  return <span className="h-3 w-3 shrink-0 rounded-full border border-separator" aria-hidden />;
}

function ProgressBar({ value, className }: { value: number | null; className?: string }) {
  if (value == null) return null;
  return (
    <div className={cn("mt-1 h-1 overflow-hidden rounded-full bg-fill-strong", className)}>
      <div
        className="h-full rounded-full bg-accent transition-[width] duration-300"
        style={{ width: `${value}%` }}
      />
    </div>
  );
}

export function SyncIndicator({ scan, embeddings: emb }: Props) {
  const scanning = scan.state === "scanning";
  const scanError = scan.state === "error";
  const embedding = emb.state === "running";
  const embedQueued = emb.state === "pending" || (emb.pending > 0 && !embedding && emb.enabled);
  const embedDisabled = !emb.enabled;
  const embedError = Boolean(emb.error);

  const active = scanning || scanError || embedding || embedQueued || embedError;
  if (!active) return null;

  let scanStep: StepState = "waiting";
  if (scanError) scanStep = "error";
  else if (scanning) scanStep = "active";
  else if (scan.state === "done" || scan.state === "idle") scanStep = "done";

  let embedStep: StepState = "waiting";
  if (embedDisabled) embedStep = "off";
  else if (embedError) embedStep = "error";
  else if (embedding) embedStep = "active";
  else if (emb.pending === 0 && emb.total > 0) embedStep = "done";
  else if (embedQueued) embedStep = scanning ? "waiting" : "active";

  const scanPct = scanning ? pct(scan.processed, scan.total) : null;
  const batchPct = embedding ? pct(emb.batchProcessed ?? 0, emb.batchTotal ?? 0) : null;
  const overallPct = emb.total > 0 ? pct(emb.embedded, emb.total) : null;
  const embedEta = useEmbedEta(
    embedding,
    emb.embedded,
    emb.pending,
    emb.batchProcessed ?? 0,
  );
  const embedEtaLabel = embedEta != null ? formatEtaRemaining(embedEta) : null;

  return (
    <section
      className="rounded-lg border border-separator bg-fill/40 px-2.5 py-2 text-[11px]"
      aria-live="polite"
      aria-label="Library sync status"
    >
      <div className="mb-2 font-medium text-secondary">Library sync</div>

      <div className="flex flex-col gap-2.5">
        <div className="flex gap-2">
          {stepIcon(scanStep)}
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <span className={cn(scanStep === "active" && "text-content", scanStep !== "active" && "text-secondary")}>
                Scan library
              </span>
              {scanPct != null && <span className="text-tertiary">{scanPct}%</span>}
            </div>
            {scanning && (
              <p className="text-tertiary">
                {scan.indexed > 0 && `${scan.indexed} new · `}
                {scan.processed}/{scan.total || "…"} files
              </p>
            )}
            {scanError && scan.error && <p className="text-red-400">{scan.error}</p>}
            <ProgressBar value={scanPct} />
          </div>
        </div>

        <div className="flex gap-2">
          {stepIcon(embedStep)}
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline justify-between gap-2">
              <span className={cn(embedStep === "active" && "text-content", embedStep !== "active" && "text-secondary")}>
                Build recommendations
              </span>
              {overallPct != null && embedStep !== "off" && (
                <span className="text-tertiary">
                  {overallPct}%
                  {embedding && embedEtaLabel ? ` · ${embedEtaLabel}` : ""}
                </span>
              )}
            </div>
            {embedDisabled && (
              <p className="text-tertiary">MERT model not loaded</p>
            )}
            {!embedDisabled && embedding && (
              <p className="text-tertiary">
                {(emb.batchProcessed ?? 0) > 0
                  ? `${emb.batchProcessed}/${emb.batchTotal ?? "…"} processed`
                  : (emb.batchActive ?? 0) > 0
                    ? `${emb.batchActive} running — first track takes ~${emb.backend === "onnxruntime" ? "2" : "90"}s`
                    : `Starting${emb.workers ? ` · ${emb.workers} workers` : ""}…`}
                {emb.backend === "onnxruntime" ? " · ONNX" : emb.backend === "native" ? " · native" : ""}
                {(emb.batchProcessed ?? 0) > 0 && emb.workers ? ` · ${emb.workers} workers` : ""}
              </p>
            )}
            {!embedDisabled && embedQueued && !embedding && (
              <p className="text-tertiary">
                {emb.pending} {emb.pending === 1 ? "track" : "tracks"} waiting
              </p>
            )}
            {!embedDisabled && embedStep === "done" && (
              <p className="text-tertiary">
                {emb.embedded}/{emb.total} indexed
              </p>
            )}
            {embedError && emb.error && <p className="text-red-400">{emb.error}</p>}
            <ProgressBar value={embedding ? batchPct : embedQueued || embedding ? overallPct : null} />
          </div>
        </div>
      </div>
    </section>
  );
}

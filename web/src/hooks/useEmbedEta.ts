import { useEffect, useRef, useState } from "react";

interface Sample {
  embedded: number;
  batchProcessed: number;
  at: number;
}

/** Estimates remaining embedding time from observed throughput. */
export function useEmbedEta(
  embedding: boolean,
  embedded: number,
  pending: number,
  batchProcessed: number,
): number | null {
  const sample = useRef<Sample | null>(null);
  const rate = useRef<number | null>(null);
  const [etaSec, setEtaSec] = useState<number | null>(null);

  useEffect(() => {
    if (!embedding) {
      sample.current = null;
      rate.current = null;
      setEtaSec(null);
      return;
    }

    const now = Date.now();
    const prev = sample.current;
    if (!prev) {
      sample.current = { embedded, batchProcessed, at: now };
      return;
    }

    const deltaTracks = Math.max(batchProcessed - prev.batchProcessed, embedded - prev.embedded);
    const deltaMs = now - prev.at;

    if (deltaTracks > 0 && deltaMs >= 2000) {
      const observed = deltaTracks / (deltaMs / 1000);
      rate.current = rate.current == null ? observed : rate.current * 0.65 + observed * 0.35;
      sample.current = { embedded, batchProcessed, at: now };
    }

    if (rate.current != null && rate.current > 0 && pending > 0) {
      setEtaSec(pending / rate.current);
    }
  }, [embedding, embedded, pending, batchProcessed]);

  return etaSec;
}

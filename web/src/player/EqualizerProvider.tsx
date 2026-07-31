import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import {
  EQ_PRESETS,
  FLAT_GAINS,
  clampGain,
  persistEqSettings,
  presetFor,
  readEqSettings,
  type EqSettings,
} from "@/lib/eq";
import { equalizer } from "./equalizer";

interface EqualizerValue extends EqSettings {
  /** False when the browser has no usable Web Audio support. */
  supported: boolean;
  setEnabled(enabled: boolean): void;
  selectPreset(name: string): void;
  setGain(band: number, db: number): void;
  reset(): void;
}

const EqualizerContext = createContext<EqualizerValue | null>(null);

export function EqualizerProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<EqSettings>(readEqSettings);

  // The graph is a singleton, so it survives remounts and needs the settings
  // pushed at it rather than rebuilt from them.
  useEffect(() => {
    equalizer.apply(settings);
  }, [settings]);

  // Writing on every drag frame would hammer storage; a trailing save is enough.
  const saveTimer = useRef<number | undefined>(undefined);
  useEffect(() => {
    window.clearTimeout(saveTimer.current);
    saveTimer.current = window.setTimeout(() => persistEqSettings(settings), 250);
    return () => window.clearTimeout(saveTimer.current);
  }, [settings]);

  const setEnabled = useCallback((enabled: boolean) => {
    setSettings((current) => ({ ...current, enabled }));
  }, []);

  const selectPreset = useCallback((name: string) => {
    const preset = EQ_PRESETS.find((candidate) => candidate.name === name);
    if (!preset) return;
    setSettings((current) => ({ ...current, preset: preset.name, gains: preset.gains.slice() }));
  }, []);

  const setGain = useCallback((band: number, db: number) => {
    setSettings((current) => {
      const gains = current.gains.slice();
      gains[band] = clampGain(db);
      return { ...current, gains, preset: presetFor(gains) };
    });
  }, []);

  const reset = useCallback(() => {
    setSettings((current) => ({ ...current, preset: "Flat", gains: FLAT_GAINS.slice() }));
  }, []);

  const value = useMemo<EqualizerValue>(
    () => ({
      ...settings,
      supported: equalizer.supported,
      setEnabled,
      selectPreset,
      setGain,
      reset,
    }),
    [reset, selectPreset, setEnabled, setGain, settings],
  );

  return <EqualizerContext.Provider value={value}>{children}</EqualizerContext.Provider>;
}

export function useEqualizer(): EqualizerValue {
  const value = useContext(EqualizerContext);
  if (!value) throw new Error("useEqualizer must be used inside EqualizerProvider");
  return value;
}

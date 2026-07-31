import { RotateCcw } from "lucide-react";
import { useCallback, useId, useRef, useState } from "react";
import { cn } from "@/lib/cn";
import { EQ_BANDS, EQ_MAX_DB, EQ_MIN_DB, clampGain, formatGain } from "@/lib/eq";
import { useEqualizer } from "@/player/EqualizerProvider";
import { PresetSelect } from "./PresetSelect";
import { Switch } from "./Switch";

/* Drawing space. The SVG scales to its container, so these are relative units. */
const VIEW_W = 360;
const VIEW_H = 160;
const GRID_LEFT = 26;
const GRID_RIGHT = 348;
const PLOT_TOP = 18;
const PLOT_BOTTOM = 130;
const BAND_LEFT = 40;
const BAND_RIGHT = 336;
const LABEL_Y = 148;

const PLOT_HEIGHT = PLOT_BOTTOM - PLOT_TOP;
const GRID_LINES = [12, 6, 0, -6, -12];

function bandX(index: number): number {
  return BAND_LEFT + (index * (BAND_RIGHT - BAND_LEFT)) / (EQ_BANDS.length - 1);
}

function gainY(db: number): number {
  const ratio = (EQ_MAX_DB - db) / (EQ_MAX_DB - EQ_MIN_DB);
  return PLOT_TOP + ratio * PLOT_HEIGHT;
}

interface Point {
  x: number;
  y: number;
}

/**
 * A Catmull-Rom spline converted to cubic beziers: it passes exactly through
 * every band, which a plain bezier smoothing would not.
 */
function smoothPath(points: Point[]): string {
  if (points.length === 0) return "";

  const round = (value: number) => Math.round(value * 100) / 100;
  let path = `M ${round(points[0].x)} ${round(points[0].y)}`;

  for (let i = 0; i < points.length - 1; i += 1) {
    const previous = points[i - 1] ?? points[i];
    const start = points[i];
    const end = points[i + 1];
    const after = points[i + 2] ?? end;

    const c1 = { x: start.x + (end.x - previous.x) / 6, y: start.y + (end.y - previous.y) / 6 };
    const c2 = { x: end.x - (after.x - start.x) / 6, y: end.y - (after.y - start.y) / 6 };
    path += ` C ${round(c1.x)} ${round(c1.y)}, ${round(c2.x)} ${round(c2.y)}, ${round(end.x)} ${round(end.y)}`;
  }

  return path;
}

export function Equalizer() {
  const { enabled, preset, gains, supported, setEnabled, selectPreset, setGain, reset } = useEqualizer();
  const svgRef = useRef<SVGSVGElement>(null);
  const [activeBand, setActiveBand] = useState<number | null>(null);
  const selectId = useId();

  /** Maps a pointer position to a gain, in the SVG's own coordinate space. */
  const gainAtClientY = useCallback((clientY: number) => {
    const svg = svgRef.current;
    if (!svg) return 0;

    const rect = svg.getBoundingClientRect();
    if (rect.height === 0) return 0;

    const y = ((clientY - rect.top) / rect.height) * VIEW_H;
    const ratio = (y - PLOT_TOP) / PLOT_HEIGHT;
    return clampGain(EQ_MAX_DB - ratio * (EQ_MAX_DB - EQ_MIN_DB));
  }, []);

  const points: Point[] = gains.map((gain, index) => ({ x: bandX(index), y: gainY(gain) }));
  // Flat runs out to both edges so the curve fills the graph rather than
  // stopping short at the outermost band.
  const curve = smoothPath([
    { x: GRID_LEFT, y: points[0].y },
    ...points,
    { x: GRID_RIGHT, y: points[points.length - 1].y },
  ]);
  const zeroY = gainY(0);
  const area = `${curve} L ${GRID_RIGHT} ${zeroY} L ${GRID_LEFT} ${zeroY} Z`;

  return (
    <section aria-labelledby={`${selectId}-heading`} className="flex flex-col gap-3">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <h3 id={`${selectId}-heading`} className="text-[13px] font-semibold">
            Equalizer
          </h3>
          <p className="text-[11.5px] text-tertiary">
            {supported ? "Shapes playback in this browser." : "Not supported in this browser."}
          </p>
        </div>
        <Switch checked={enabled} disabled={!supported} onChange={setEnabled} label="Equalizer" />
      </div>

      <div className={cn("flex items-center gap-2", !enabled && "opacity-50")}>
        <PresetSelect value={preset} disabled={!enabled} onChange={selectPreset} />

        <button
          type="button"
          onClick={reset}
          disabled={!enabled}
          className={cn(
            "flex shrink-0 items-center gap-1.5 rounded-lg px-2.5 py-1.5 text-[12.5px] text-secondary",
            "transition-colors hover:bg-fill hover:text-content disabled:cursor-not-allowed",
          )}
        >
          <RotateCcw className="h-3.5 w-3.5" aria-hidden />
          Reset
        </button>
      </div>

      <div
        className={cn(
          "rounded-xl border border-separator bg-fill px-1 py-1 transition-opacity",
          !enabled && "opacity-40",
        )}
      >
        <svg
          ref={svgRef}
          viewBox={`0 0 ${VIEW_W} ${VIEW_H}`}
          style={{ aspectRatio: `${VIEW_W} / ${VIEW_H}` }}
          className="block h-auto w-full touch-none select-none"
          role="group"
          aria-label="Equalizer bands"
        >
          <defs>
            <linearGradient id={`${selectId}-wash`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0" style={{ stopColor: "var(--accent)", stopOpacity: 0.28 }} />
              <stop offset="1" style={{ stopColor: "var(--accent)", stopOpacity: 0.04 }} />
            </linearGradient>
          </defs>

          {GRID_LINES.map((db) => (
            <g key={db}>
              <line
                x1={GRID_LEFT}
                x2={GRID_RIGHT}
                y1={gainY(db)}
                y2={gainY(db)}
                className={cn("stroke-separator", db === 0 && "stroke-content/25")}
                strokeWidth={1}
                strokeDasharray={db === 0 ? undefined : "2 4"}
              />
              <text
                x={GRID_LEFT - 6}
                y={gainY(db) + 3}
                textAnchor="end"
                className="fill-tertiary text-[8px] tabular-nums"
              >
                {db > 0 ? `+${db}` : db}
              </text>
            </g>
          ))}

          <path d={area} fill={`url(#${selectId}-wash)`} />
          <path
            d={curve}
            fill="none"
            className="stroke-accent"
            strokeWidth={2.5}
            strokeLinecap="round"
            strokeLinejoin="round"
          />

          {gains.map((gain, index) => (
            <BandHandle
              key={EQ_BANDS[index].frequency}
              index={index}
              gain={gain}
              active={activeBand === index}
              disabled={!enabled}
              gainAtClientY={gainAtClientY}
              onChange={setGain}
              onActiveChange={setActiveBand}
            />
          ))}

          {EQ_BANDS.map((band, index) => (
            <text
              key={band.frequency}
              x={bandX(index)}
              y={LABEL_Y}
              textAnchor="middle"
              className="fill-tertiary text-[9px] tabular-nums"
            >
              {band.label}
            </text>
          ))}

          <text x={GRID_LEFT - 6} y={LABEL_Y} textAnchor="end" className="fill-tertiary text-[8px]">
            Hz
          </text>
        </svg>
      </div>
    </section>
  );
}

interface HandleProps {
  index: number;
  gain: number;
  active: boolean;
  disabled: boolean;
  gainAtClientY(clientY: number): number;
  onChange(band: number, db: number): void;
  onActiveChange(band: number | null): void;
}

function BandHandle({ index, gain, active, disabled, gainAtClientY, onChange, onActiveChange }: HandleProps) {
  const [dragging, setDragging] = useState(false);
  const band = EQ_BANDS[index];
  const x = bandX(index);
  const y = gainY(gain);
  const columnWidth = (BAND_RIGHT - BAND_LEFT) / (EQ_BANDS.length - 1);

  function onPointerDown(event: React.PointerEvent<SVGGElement>) {
    if (disabled || event.button !== 0) return;
    event.currentTarget.setPointerCapture(event.pointerId);
    setDragging(true);
    onActiveChange(index);
    onChange(index, gainAtClientY(event.clientY));
  }

  function onPointerMove(event: React.PointerEvent<SVGGElement>) {
    if (!dragging) return;
    onChange(index, gainAtClientY(event.clientY));
  }

  function onPointerUp(event: React.PointerEvent<SVGGElement>) {
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId);
    }
    setDragging(false);
    onActiveChange(null);
  }

  function onKeyDown(event: React.KeyboardEvent<SVGGElement>) {
    if (disabled) return;
    const step = event.shiftKey ? 2 : 0.5;
    let next: number | null = null;

    switch (event.key) {
      case "ArrowUp":
      case "ArrowRight":
        next = gain + step;
        break;
      case "ArrowDown":
      case "ArrowLeft":
        next = gain - step;
        break;
      case "Home":
        next = EQ_MIN_DB;
        break;
      case "End":
        next = EQ_MAX_DB;
        break;
      case "0":
      case "Backspace":
        next = 0;
        break;
      default:
        return;
    }

    event.preventDefault();
    onChange(index, next);
  }

  return (
    <g
      role="slider"
      tabIndex={disabled ? -1 : 0}
      aria-label={`${band.frequency} hertz`}
      aria-valuemin={EQ_MIN_DB}
      aria-valuemax={EQ_MAX_DB}
      aria-valuenow={gain}
      aria-valuetext={`${formatGain(gain)} decibels`}
      aria-disabled={disabled || undefined}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
      onPointerCancel={onPointerUp}
      onKeyDown={onKeyDown}
      onDoubleClick={() => !disabled && onChange(index, 0)}
      onFocus={() => onActiveChange(index)}
      onBlur={() => !dragging && onActiveChange(null)}
      className={cn("outline-none", !disabled && "cursor-ns-resize")}
    >
      {/* A column-wide target, so the dot stays easy to grab on touch. */}
      <rect
        x={x - columnWidth / 2}
        y={PLOT_TOP - 12}
        width={columnWidth}
        height={PLOT_HEIGHT + 24}
        fill="transparent"
      />

      {active && (
        <>
          <circle cx={x} cy={y} r={12} className="fill-accent/20" />
          <text x={x} y={PLOT_TOP - 4} textAnchor="middle" className="fill-content text-[9px] font-semibold tabular-nums">
            {formatGain(gain)} dB
          </text>
        </>
      )}

      <circle
        cx={x}
        cy={y}
        r={active ? 6.5 : 5.5}
        className="fill-accent stroke-canvas transition-[r]"
        strokeWidth={2}
      />
    </g>
  );
}

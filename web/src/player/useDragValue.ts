import { useCallback, useEffect, useRef, useState } from "react";

interface Options {
  /** Current value in the range 0..max. */
  value: number;
  max: number;
  onChange(value: number): void;
  onCommit?(value: number): void;
  /** Half-width of the knob, so grabbing it keeps the pointer offset. */
  knobRadius?: number;
  disabled?: boolean;
}

interface DragValue {
  ref: React.RefObject<HTMLDivElement | null>;
  isDragging: boolean;
  /** The value to render: the dragged position while dragging, else the real one. */
  displayValue: number;
  onPointerDown(event: React.PointerEvent<HTMLDivElement>): void;
  onKeyDown(event: React.KeyboardEvent<HTMLDivElement>): void;
}

/**
 * Direct-manipulation dragging for sliders.
 *
 * Pointer capture keeps tracking when the pointer leaves the element, the
 * displayed value follows the pointer 1:1, and grabbing the knob preserves the
 * offset instead of snapping the knob under the finger.
 */
export function useDragValue({
  value,
  max,
  onChange,
  onCommit,
  knobRadius = 8,
  disabled = false,
}: Options): DragValue {
  const ref = useRef<HTMLDivElement | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragValue, setDragValue] = useState(value);
  const grabOffsetRef = useRef(0);

  const valueAt = useCallback(
    (clientX: number) => {
      const element = ref.current;
      if (!element || max <= 0) return 0;
      const rect = element.getBoundingClientRect();
      const x = clientX - grabOffsetRef.current - rect.left;
      const ratio = rect.width > 0 ? x / rect.width : 0;
      return Math.min(Math.max(ratio, 0), 1) * max;
    },
    [max],
  );

  const onPointerDown = useCallback(
    (event: React.PointerEvent<HTMLDivElement>) => {
      if (disabled || event.button !== 0) return;
      const element = ref.current;
      if (!element) return;

      const rect = element.getBoundingClientRect();
      const knobX = rect.left + (max > 0 ? (value / max) * rect.width : 0);
      // Grabbing the knob keeps its offset; clicking the track jumps there.
      grabOffsetRef.current = Math.abs(event.clientX - knobX) <= knobRadius ? event.clientX - knobX : 0;

      element.setPointerCapture(event.pointerId);
      setIsDragging(true);

      const next = valueAt(event.clientX);
      setDragValue(next);
      onChange(next);
      event.preventDefault();
    },
    [disabled, knobRadius, max, onChange, value, valueAt],
  );

  useEffect(() => {
    if (!isDragging) return;
    const element = ref.current;
    if (!element) return;

    const onPointerMove = (event: PointerEvent) => {
      const next = valueAt(event.clientX);
      setDragValue(next);
      onChange(next);
    };
    const onPointerUp = (event: PointerEvent) => {
      const next = valueAt(event.clientX);
      setIsDragging(false);
      grabOffsetRef.current = 0;
      onCommit?.(next);
      if (element.hasPointerCapture(event.pointerId)) {
        element.releasePointerCapture(event.pointerId);
      }
    };

    element.addEventListener("pointermove", onPointerMove);
    element.addEventListener("pointerup", onPointerUp);
    element.addEventListener("pointercancel", onPointerUp);
    return () => {
      element.removeEventListener("pointermove", onPointerMove);
      element.removeEventListener("pointerup", onPointerUp);
      element.removeEventListener("pointercancel", onPointerUp);
    };
  }, [isDragging, onChange, onCommit, valueAt]);

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent<HTMLDivElement>) => {
      if (disabled) return;
      const step = event.shiftKey ? max / 10 : max / 50;
      let next: number | null = null;

      switch (event.key) {
        case "ArrowLeft":
        case "ArrowDown":
          next = value - step;
          break;
        case "ArrowRight":
        case "ArrowUp":
          next = value + step;
          break;
        case "Home":
          next = 0;
          break;
        case "End":
          next = max;
          break;
        default:
          return;
      }

      event.preventDefault();
      const clamped = Math.min(Math.max(next, 0), max);
      onChange(clamped);
      onCommit?.(clamped);
    },
    [disabled, max, onChange, onCommit, value],
  );

  return {
    ref,
    isDragging,
    displayValue: isDragging ? dragValue : value,
    onPointerDown,
    onKeyDown,
  };
}

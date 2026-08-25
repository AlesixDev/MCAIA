"use client";

import { useEffect, useRef, useState } from "react";
import { Minus, Plus } from "lucide-react";

import { cn } from "@/lib/utils";

const FIRST_REPEAT_MS = 380;
const MIN_REPEAT_MS = 40;
const COARSE_MULTIPLIER = 10;

export function Stepper({
  value,
  min,
  max,
  step,
  suffix,
  disabled,
  onChange,
}: {
  value: number;
  min: number;
  max: number;
  step: number;
  suffix?: string;
  disabled?: boolean;
  onChange: (value: number) => void;
}) {
  const decimals = step < 1 ? 1 : 0;
  const [draft, setDraft] = useState<string | null>(null);
  const repeat = useRef<number | null>(null);
  const latest = useRef(value);

  latest.current = value;

  const clamp = (next: number) => {
    if (Number.isNaN(next)) {
      return latest.current;
    }

    return Math.min(max, Math.max(min, Number(next.toFixed(2))));
  };

  function bump(direction: number, coarse: boolean) {
    const size = step * (coarse ? COARSE_MULTIPLIER : 1);

    onChange(clamp(latest.current + direction * size));
  }

  function hold(direction: number, coarse: boolean) {
    bump(direction, coarse);

    let delay = FIRST_REPEAT_MS;

    const tick = () => {
      bump(direction, coarse);
      delay = Math.max(MIN_REPEAT_MS, delay * 0.72);
      repeat.current = window.setTimeout(tick, delay);
    };

    repeat.current = window.setTimeout(tick, delay);
  }

  function release() {
    if (repeat.current === null) {
      return;
    }

    window.clearTimeout(repeat.current);
    repeat.current = null;
  }

  useEffect(() => release, []);

  function commit() {
    if (draft !== null) {
      onChange(clamp(Number.parseFloat(draft.replace(",", "."))));
    }

    setDraft(null);
  }

  const button =
    "flex h-full w-6 items-center justify-center text-muted transition-colors hover:bg-surface-sunken hover:text-body disabled:opacity-40 disabled:hover:bg-transparent";

  return (
    <div
      className={cn(
        "flex h-7 items-center rounded-lg border border-line transition-colors hover:border-line-strong focus-within:border-accent/60",
        disabled && "opacity-50",
      )}
    >
      <button
        type="button"
        disabled={disabled || value <= min}
        onPointerDown={(event) => hold(-1, event.shiftKey)}
        onPointerUp={release}
        onPointerLeave={release}
        onPointerCancel={release}
        aria-label="Decrease"
        className={cn(button, "rounded-l-lg")}
      >
        <Minus size={11} />
      </button>

      <div className="flex h-full items-center gap-px px-1">
        <input
          value={draft ?? value.toFixed(decimals)}
          disabled={disabled}
          inputMode="decimal"
          aria-label="Value"
          onFocus={(event) => {
            setDraft(value.toFixed(decimals));
            event.target.select();
          }}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={commit}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.currentTarget.blur();
            }

            if (event.key === "Escape") {
              setDraft(null);
              event.currentTarget.blur();
            }

            if (event.key === "ArrowUp" || event.key === "ArrowDown") {
              event.preventDefault();
              setDraft(null);
              bump(event.key === "ArrowUp" ? 1 : -1, event.shiftKey);
            }
          }}
          className="w-[30px] bg-transparent text-right font-mono text-[11.5px] text-body outline-none"
        />

        {suffix ? <span className="font-mono text-[11.5px] text-muted">{suffix}</span> : null}
      </div>

      <button
        type="button"
        disabled={disabled || value >= max}
        onPointerDown={(event) => hold(1, event.shiftKey)}
        onPointerUp={release}
        onPointerLeave={release}
        onPointerCancel={release}
        aria-label="Increase"
        className={cn(button, "rounded-r-lg")}
      >
        <Plus size={11} />
      </button>
    </div>
  );
}

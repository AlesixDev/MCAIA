"use client";

import { useEffect, useState } from "react";
import { motion, useReducedMotion } from "motion/react";
import { Check } from "lucide-react";

import { cn } from "@/lib/utils";

const stages = [
  "Reading the rig",
  "Planning the motion",
  "Writing keyframes",
  "Checking bone names",
  "Smoothing the loop",
];

const SECONDS_PER_STAGE = 4;

function Marker({ state, reduced }: { state: "done" | "active" | "pending"; reduced: boolean }) {
  if (state === "done") {
    return <Check size={11} className="shrink-0 text-accent" strokeWidth={2.5} />;
  }

  if (state === "pending") {
    return <span className="size-[11px] shrink-0" />;
  }

  return (
    <span className="relative flex size-[11px] shrink-0 items-center justify-center">
      <motion.span
        animate={reduced ? { opacity: 0.5 } : { opacity: [0.35, 1, 0.35] }}
        transition={
          reduced
            ? { duration: 0 }
            : { duration: 1.4, repeat: Number.POSITIVE_INFINITY, ease: "easeInOut" }
        }
        className="size-1.5 rounded-full bg-accent"
      />
    </span>
  );
}

export function Generating() {
  const reduced = Boolean(useReducedMotion());
  const [elapsed, setElapsed] = useState(0);

  useEffect(() => {
    const started = performance.now();
    const timer = window.setInterval(() => {
      setElapsed((performance.now() - started) / 1000);
    }, 100);

    return () => window.clearInterval(timer);
  }, []);

  const current = Math.min(stages.length - 1, Math.floor(elapsed / SECONDS_PER_STAGE));

  return (
    <div
      role="status"
      aria-live="polite"
      className="w-full max-w-[320px] overflow-hidden rounded-xl border border-line bg-surface-raised"
    >
      <div className="flex flex-col gap-[3px] px-3.5 py-3">
        {stages.map((stage, index) => {
          const state = index < current ? "done" : index === current ? "active" : "pending";

          return (
            <div key={stage} className="flex items-center gap-2">
              <Marker state={state} reduced={reduced} />

              <span
                className={cn(
                  "min-w-0 flex-1 truncate text-[12px] transition-colors",
                  state === "active" && "text-body",
                  state === "done" && "text-muted",
                  state === "pending" && "text-faint",
                )}
              >
                {stage}
              </span>

              {state === "active" ? (
                <span className="shrink-0 font-mono text-[10.5px] tabular-nums text-muted">
                  {elapsed.toFixed(1)}s
                </span>
              ) : null}
            </div>
          );
        })}
      </div>

      <div className="relative h-px overflow-hidden bg-line">
        <motion.span
          animate={reduced ? {} : { x: ["-100%", "200%"] }}
          transition={{ duration: 1.8, repeat: Number.POSITIVE_INFINITY, ease: "easeInOut" }}
          className="absolute inset-y-0 w-1/3 bg-accent"
        />
      </div>
    </div>
  );
}

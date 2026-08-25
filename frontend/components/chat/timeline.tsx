"use client";

import { Animation, Keyframe, Track } from "@/lib/api";
import { cn } from "@/lib/utils";

const channels: Array<{ key: keyof Track; label: string; tone: string }> = [
  { key: "rotation", label: "rot", tone: "bg-accent" },
  { key: "position", label: "pos", tone: "bg-good" },
  { key: "scale", label: "scl", tone: "bg-warn" },
];

function format(value: number) {
  return Number.isInteger(value) ? value.toString() : value.toFixed(2);
}

function Row({
  label,
  tone,
  frames,
  length,
}: {
  label: string;
  tone: string;
  frames: Keyframe[];
  length: number;
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="w-6 shrink-0 font-mono text-[9.5px] uppercase text-muted">{label}</span>

      <div className="relative h-5 flex-1">
        <span className="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-line" />

        {frames.map((frame, index) => (
          <span
            key={`${frame.time}-${index}`}
            title={`${frame.time}s · ${frame.value.map(format).join(", ")}`}
            style={{ left: `${Math.min(100, (frame.time / Math.max(length, 0.001)) * 100)}%` }}
            className={cn(
              "absolute top-1/2 size-[7px] -translate-x-1/2 -translate-y-1/2 rotate-45 rounded-[1.5px] ring-2 ring-surface-raised",
              tone,
            )}
          />
        ))}
      </div>
    </div>
  );
}

export function Timeline({ animation }: { animation: Animation }) {
  const bones = Object.entries(animation.bones);

  return (
    <div className="flex flex-col gap-3">
      {bones.map(([bone, track]) => (
        <div key={bone} className="flex gap-3">
          <span className="w-24 shrink-0 truncate pt-0.5 font-mono text-[11px] text-muted">
            {bone}
          </span>

          <div className="flex min-w-0 flex-1 flex-col">
            {channels.map(({ key, label, tone }) => {
              const frames = track[key];

              if (!frames?.length) {
                return null;
              }

              return (
                <Row key={key} label={label} tone={tone} frames={frames} length={animation.length} />
              );
            })}
          </div>
        </div>
      ))}

      <div className="flex justify-between pl-[108px] font-mono text-[9.5px] text-muted">
        <span>0.00s</span>
        <span>{animation.length.toFixed(2)}s</span>
      </div>
    </div>
  );
}

"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Box, Eye, ListTree, TriangleAlert } from "lucide-react";

import { Animation, ExportFormat, Issue, Rig } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Timeline } from "@/components/chat/timeline";
import { ExportMenu } from "@/components/chat/export-menu";
import { ModelPreview } from "@/components/preview/model-preview";

type View = "preview" | "timeline";

function Meta({ children }: { children: React.ReactNode }) {
  return <span className="font-mono text-[10.5px] text-muted">{children}</span>;
}

function Tab({
  active,
  icon,
  label,
  onClick,
}: {
  active: boolean;
  icon: React.ReactNode;
  label: string;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={cn(
        "relative flex h-7 items-center gap-1.5 rounded-lg px-2.5 text-[12px] transition-colors",
        active ? "text-body" : "text-muted hover:text-body",
      )}
    >
      {active ? (
        <motion.span
          layoutId="card-view"
          transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
          className="absolute inset-0 rounded-lg bg-surface-sunken"
        />
      ) : null}

      <span className="relative flex items-center gap-1.5">
        {icon}
        {label}
      </span>
    </button>
  );
}

export function AnimationCard({
  rig,
  animation,
  warnings,
  removed,
  active,
  formats,
  onInspect,
  onExport,
}: {
  rig: Rig | null;
  animation: Animation;
  warnings: Issue[];
  removed: number;
  active: boolean;
  formats: ExportFormat[];
  onInspect: () => void;
  onExport: (format: string) => void;
}) {
  const [view, setView] = useState<View>("preview");

  return (
    <div className="overflow-hidden rounded-xl border border-line bg-surface-raised">
      <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 px-4 pb-2.5 pt-3.5">
        <span className="font-mono text-[13px] font-medium text-body">{animation.name}</span>

        <Meta>{animation.length.toFixed(2)}s</Meta>
        <Meta>{Object.keys(animation.bones).length} bones</Meta>
        <Meta>{animation.loop}</Meta>

        {removed > 0 ? <Meta>−{removed} keyframes</Meta> : null}
      </div>

      <div className="flex gap-1 px-3 pb-2">
        <Tab
          active={view === "preview"}
          icon={<Box size={12} />}
          label="Preview"
          onClick={() => setView("preview")}
        />

        <Tab
          active={view === "timeline"}
          icon={<ListTree size={12} />}
          label="Keyframes"
          onClick={() => setView("timeline")}
        />
      </div>

      <div className="px-4 pb-4">
        <AnimatePresence mode="wait" initial={false}>
          <motion.div
            key={view}
            initial={{ opacity: 0, y: 4 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -4 }}
            transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
          >
            {view === "preview" && rig ? (
              <ModelPreview rig={rig} animation={animation} />
            ) : (
              <Timeline animation={animation} />
            )}
          </motion.div>
        </AnimatePresence>
      </div>

      {warnings.length > 0 ? (
        <div className="flex gap-2 border-t border-line bg-warn/[0.07] px-4 py-2.5">
          <TriangleAlert size={13} className="mt-px shrink-0 text-warn" />

          <ul className="flex min-w-0 flex-col gap-0.5 font-mono text-[10.5px] text-warn">
            {warnings.map((issue) => (
              <li key={issue.path + issue.message} className="truncate">
                {issue.path}: {issue.message}
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      <div className="flex items-center gap-1 border-t border-line px-2.5 py-2">
        <button
          onClick={onInspect}
          className={cn(
            "flex h-7 items-center gap-1.5 rounded-lg px-2.5 text-[12px] transition-colors",
            active
              ? "bg-accent/12 text-accent"
              : "text-muted hover:bg-surface-sunken hover:text-body",
          )}
        >
          <Eye size={13} />
          {active ? "Selected" : "Inspect"}
        </button>

        <ExportMenu formats={formats} onExport={onExport} />
      </div>
    </div>
  );
}

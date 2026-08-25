"use client";

import { motion } from "motion/react";
import { Trash2, X } from "lucide-react";

import { Animation, ExportFormat, Rig } from "@/lib/api";
import { cn } from "@/lib/utils";
import { ExportMenu } from "@/components/chat/export-menu";

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h3 className="text-[11px] font-medium uppercase tracking-[0.09em] text-faint">{title}</h3>

      {children}
    </section>
  );
}

function RigTree({ rig, animated }: { rig: Rig; animated: Set<string> }) {
  const render = (name: string): React.ReactNode => {
    const bone = rig.bones[name];

    if (!bone) {
      return null;
    }

    return (
      <li key={name} style={{ paddingLeft: bone.depth * 10 }}>
        <div className="flex items-center gap-2 py-[3px]">
          <span
            className={cn(
              "size-1.5 shrink-0 rounded-full",
              animated.has(name) ? "bg-accent" : "bg-line-strong",
            )}
          />

          <span
            className={cn(
              "truncate font-mono text-[11.5px]",
              animated.has(name) ? "text-body" : "text-muted",
            )}
          >
            {bone.name}
          </span>
        </div>

        {bone.children?.length ? <ul>{bone.children.map(render)}</ul> : null}
      </li>
    );
  };

  return <ul>{rig.roots.map(render)}</ul>;
}

export function Inspector({
  rig,
  animations,
  selected,
  formats,
  namespace,
  busy,
  onClose,
  onSelect,
  onDelete,
  onExport,
}: {
  rig: Rig;
  animations: Animation[];
  selected: string | null;
  formats: ExportFormat[];
  namespace: string;
  busy: boolean;
  onClose: () => void;
  onSelect: (name: string) => void;
  onDelete: (name: string) => void;
  onExport: (format: string, namespace: string) => void;
}) {
  const animatedBones = new Set(
    animations.filter((item) => item.name === selected).flatMap((item) => Object.keys(item.bones)),
  );

  return (
    <motion.aside
      initial={{ opacity: 0, x: 24 }}
      animate={{ opacity: 1, x: 0 }}
      transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
      className="flex h-full w-[300px] shrink-0 flex-col border-l border-line bg-surface"
    >
      <div className="flex h-[52px] items-center gap-2 px-4">
        <span className="truncate text-[13px] font-medium">{rig.model_name}</span>

        <span className="font-mono text-[11px] text-muted">
          {Object.keys(rig.bones).length} bones
        </span>

        <button
          onClick={onClose}
          aria-label="Close"
          className="ml-auto rounded-md p-1.5 text-muted transition-colors hover:bg-surface-sunken hover:text-body"
        >
          <X size={15} />
        </button>
      </div>

      <div className="flex min-h-0 flex-1 flex-col gap-6 overflow-y-auto px-4 pb-6">
        <Section title="Rig">
          <RigTree rig={rig} animated={animatedBones} />
        </Section>

        <Section title={`Animations · ${animations.length}`}>
          {animations.length === 0 ? (
            <p className="text-[12px] text-muted">Nothing here yet.</p>
          ) : (
            <ul className="flex flex-col gap-0.5">
              {animations.map((item) => (
                <li key={item.name}>
                  <div
                    onClick={() => onSelect(item.name)}
                    className={cn(
                      "group flex cursor-pointer items-center gap-2 rounded-lg px-2 py-1.5 transition-colors",
                      item.name === selected ? "bg-accent/10" : "hover:bg-surface-sunken",
                    )}
                  >
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-mono text-[12px] text-body">{item.name}</p>

                      <p className="text-[10.5px] text-muted">
                        {item.length.toFixed(2)}s · {Object.keys(item.bones).length} bones ·{" "}
                        {item.loop}
                      </p>
                    </div>

                    <button
                      aria-label={`Delete ${item.name}`}
                      onClick={(event) => {
                        event.stopPropagation();
                        onDelete(item.name);
                      }}
                      className="rounded-md p-1 text-muted opacity-0 transition-opacity hover:text-bad group-hover:opacity-100"
                    >
                      <Trash2 size={13} />
                    </button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </Section>
      </div>

      <div className="flex items-center justify-between gap-2 border-t border-line px-3 py-2.5">
        <span className="truncate font-mono text-[10.5px] text-muted">{namespace}</span>

        <ExportMenu
          formats={formats}
          align="end"
          disabled={busy || animations.length === 0}
          onExport={(format) => onExport(format, namespace)}
        />
      </div>
    </motion.aside>
  );
}

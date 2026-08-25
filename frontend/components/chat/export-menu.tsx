"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Download } from "lucide-react";

import { ExportFormat } from "@/lib/api";
import { cn } from "@/lib/utils";

export function ExportMenu({
  formats,
  disabled,
  align = "start",
  className,
  onExport,
}: {
  formats: ExportFormat[];
  disabled?: boolean;
  align?: "start" | "end";
  className?: string;
  onExport: (format: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const close = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", escape);

    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", escape);
    };
  }, [open]);

  return (
    <div ref={root} className={cn("relative", className)}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((state) => !state)}
        className={cn(
          "flex h-7 items-center gap-1.5 rounded-lg px-2.5 text-[12px] transition-colors disabled:opacity-50",
          open ? "bg-surface-sunken text-body" : "text-muted hover:bg-surface-sunken hover:text-body",
        )}
      >
        <Download size={13} />
        Export
      </button>

      <AnimatePresence>
        {open ? (
          <motion.ul
            initial={{ opacity: 0, y: 6, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 4, scale: 0.98 }}
            transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
            className={cn(
              "absolute bottom-[calc(100%+6px)] z-50 w-[230px] overflow-hidden rounded-xl border border-line bg-surface-raised p-1 shadow-panel",
              align === "end" ? "right-0" : "left-0",
            )}
          >
            {formats.map((format) => (
              <li key={format.id}>
                <button
                  type="button"
                  onClick={() => {
                    onExport(format.id);
                    setOpen(false);
                  }}
                  className="flex w-full flex-col gap-0.5 rounded-lg px-2.5 py-1.5 text-left transition-colors hover:bg-surface-sunken"
                >
                  <span className="text-[12px] text-body">{format.label}</span>
                  <span className="font-mono text-[10px] text-muted">{format.extension}</span>
                </button>
              </li>
            ))}
          </motion.ul>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

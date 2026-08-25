"use client";

import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Check, ChevronDown } from "lucide-react";

import { cn } from "@/lib/utils";

export type Option = {
  value: string;
  label: string;
};

export function Select({
  value,
  options,
  disabled,
  align = "start",
  className,
  onChange,
}: {
  value: string;
  options: Option[];
  disabled?: boolean;
  align?: "start" | "end";
  className?: string;
  onChange: (value: string) => void;
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

  const current = options.find((option) => option.value === value);

  return (
    <div ref={root} className="relative">
      <button
        type="button"
        disabled={disabled}
        onClick={() => setOpen((state) => !state)}
        className={cn(
          "flex h-7 items-center gap-1.5 rounded-lg border border-line px-2 text-[11.5px] text-body transition-colors",
          "hover:border-line-strong disabled:opacity-50",
          open && "border-accent/60 bg-accent/[0.07]",
          className,
        )}
      >
        <span className="truncate font-mono">{current?.label ?? value}</span>

        <motion.span
          animate={{ rotate: open ? 180 : 0 }}
          transition={{ duration: 0.18, ease: [0.16, 1, 0.3, 1] }}
          className="text-muted"
        >
          <ChevronDown size={12} />
        </motion.span>
      </button>

      <AnimatePresence>
        {open ? (
          <motion.ul
            initial={{ opacity: 0, y: 6, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 4, scale: 0.98 }}
            transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
            className={cn(
              "absolute bottom-[calc(100%+6px)] z-50 min-w-full overflow-hidden rounded-xl border border-line bg-surface-raised p-1 shadow-panel",
              align === "end" ? "right-0" : "left-0",
            )}
          >
            {options.map((option) => (
              <li key={option.value}>
                <button
                  type="button"
                  onClick={() => {
                    onChange(option.value);
                    setOpen(false);
                  }}
                  className={cn(
                    "flex w-full items-center gap-2 whitespace-nowrap rounded-lg px-2 py-1.5 text-left text-[12px] transition-colors",
                    option.value === value
                      ? "bg-accent/10 text-accent"
                      : "text-muted hover:bg-surface-sunken hover:text-body",
                  )}
                >
                  <Check
                    size={12}
                    className={cn(option.value === value ? "opacity-100" : "opacity-0")}
                  />

                  {option.label}
                </button>
              </li>
            ))}
          </motion.ul>
        ) : null}
      </AnimatePresence>
    </div>
  );
}

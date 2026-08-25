"use client";

import { useEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AnimatePresence, motion } from "motion/react";
import { MoreHorizontal, Pencil, Trash2 } from "lucide-react";

import { cn } from "@/lib/utils";

const MENU_WIDTH = 164;
const GAP = 4;

type Anchor = { top: number; left: number };

export function ProjectMenu({
  name,
  className,
  onRename,
  onDelete,
}: {
  name: string;
  className?: string;
  onRename: () => void;
  onDelete: () => void;
}) {
  const [anchor, setAnchor] = useState<Anchor | null>(null);
  const [confirming, setConfirming] = useState(false);
  const trigger = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);

  const open = anchor !== null;

  function place() {
    const rect = trigger.current?.getBoundingClientRect();

    if (!rect) {
      return;
    }

    setAnchor({
      top: rect.bottom + GAP,
      left: Math.max(GAP, Math.min(rect.right - MENU_WIDTH, window.innerWidth - MENU_WIDTH - GAP)),
    });
  }

  useEffect(() => {
    if (!open) {
      setConfirming(false);

      return;
    }

    const close = (event: MouseEvent) => {
      const target = event.target as Node;

      if (trigger.current?.contains(target) || menu.current?.contains(target)) {
        return;
      }

      setAnchor(null);
    };

    const escape = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setAnchor(null);
      }
    };

    const dismiss = () => setAnchor(null);

    document.addEventListener("mousedown", close);
    document.addEventListener("keydown", escape);
    window.addEventListener("resize", dismiss);
    window.addEventListener("scroll", dismiss, true);

    return () => {
      document.removeEventListener("mousedown", close);
      document.removeEventListener("keydown", escape);
      window.removeEventListener("resize", dismiss);
      window.removeEventListener("scroll", dismiss, true);
    };
  }, [open]);

  const item =
    "flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[12px] transition-colors";

  return (
    <>
      <button
        ref={trigger}
        type="button"
        aria-label={`Options for ${name}`}
        aria-expanded={open}
        onClick={() => (open ? setAnchor(null) : place())}
        className={cn(
          "rounded-md p-1 text-muted transition-colors hover:bg-surface-raised hover:text-body",
          open && "bg-surface-raised text-body",
          className,
        )}
      >
        <MoreHorizontal size={14} />
      </button>

      {typeof document === "undefined"
        ? null
        : createPortal(
            <AnimatePresence>
              {anchor ? (
                <motion.div
                  ref={menu}
                  initial={{ opacity: 0, y: -4, scale: 0.97 }}
                  animate={{ opacity: 1, y: 0, scale: 1 }}
                  exit={{ opacity: 0, y: -2, scale: 0.98 }}
                  transition={{ duration: 0.14, ease: [0.16, 1, 0.3, 1] }}
                  style={{ top: anchor.top, left: anchor.left, width: MENU_WIDTH }}
                  className="fixed z-[100] overflow-hidden rounded-xl border border-line bg-surface-raised p-1 shadow-panel"
                >
                  <button
                    type="button"
                    onClick={() => {
                      setAnchor(null);
                      onRename();
                    }}
                    className={cn(item, "text-muted hover:bg-surface-sunken hover:text-body")}
                  >
                    <Pencil size={13} />
                    Rename
                  </button>

                  <button
                    type="button"
                    onClick={() => {
                      if (!confirming) {
                        setConfirming(true);

                        return;
                      }

                      setAnchor(null);
                      onDelete();
                    }}
                    className={cn(
                      item,
                      confirming ? "bg-bad/10 text-bad" : "text-muted hover:bg-bad/10 hover:text-bad",
                    )}
                  >
                    <Trash2 size={13} />
                    {confirming ? "Delete for good?" : "Delete"}
                  </button>
                </motion.div>
              ) : null}
            </AnimatePresence>,
            document.body,
          )}
    </>
  );
}

export function RenameField({
  value,
  className,
  onCancel,
  onCommit,
}: {
  value: string;
  className?: string;
  onCancel: () => void;
  onCommit: (name: string) => void;
}) {
  const [draft, setDraft] = useState(value);

  function commit() {
    const trimmed = draft.trim();

    if (!trimmed || trimmed === value) {
      onCancel();

      return;
    }

    onCommit(trimmed);
  }

  return (
    <input
      autoFocus
      value={draft}
      maxLength={64}
      aria-label="Model name"
      onChange={(event) => setDraft(event.target.value)}
      onBlur={commit}
      onKeyDown={(event) => {
        if (event.key === "Enter") {
          commit();
        }

        if (event.key === "Escape") {
          onCancel();
        }
      }}
      className={cn(
        "h-7 w-full min-w-0 rounded-lg border border-accent/60 bg-surface px-2 text-[13px] text-body outline-none",
        className,
      )}
    />
  );
}

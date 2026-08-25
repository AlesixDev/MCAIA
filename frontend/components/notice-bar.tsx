"use client";

import { Notice } from "@/lib/use-workspace";
import { cn } from "@/lib/utils";

const tones: Record<Notice["tone"], string> = {
  info: "border-warn/40 bg-warn/10 text-warn",
  error: "border-bad/40 bg-bad/10 text-bad",
  success: "border-good/40 bg-good/10 text-good",
};

export function NoticeBar({ notice, onDismiss }: { notice: Notice; onDismiss: () => void }) {
  return (
    <div
      className={cn(
        "flex items-start gap-3 rounded-sm border px-3 py-2 text-[12px] animate-rise-in",
        tones[notice.tone],
      )}
    >
      <div className="flex-1">
        <p className="font-medium">{notice.message}</p>

        {notice.details?.length ? (
          <ul className="mt-1 flex flex-col gap-0.5 font-mono text-[11px] opacity-80">
            {notice.details.map((issue) => (
              <li key={issue.path + issue.message}>
                {issue.path}: {issue.message}
              </li>
            ))}
          </ul>
        ) : null}
      </div>

      <button onClick={onDismiss} className="text-[11px] opacity-70 hover:opacity-100">
        cerrar
      </button>
    </div>
  );
}

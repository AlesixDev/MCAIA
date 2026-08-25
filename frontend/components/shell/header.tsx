"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Moon, PanelLeftOpen, PanelRight, Sun } from "lucide-react";
import { useEffect, useState } from "react";

import { useShell } from "@/lib/shell-context";
import { useWorkspaceContext } from "@/lib/workspace-context";
import { cn } from "@/lib/utils";

const chrome =
  "rounded-md p-1.5 transition-colors text-muted hover:bg-surface-raised hover:text-body";

function IconButton({
  label,
  active,
  onClick,
  children,
}: {
  label: string;
  active?: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      aria-label={label}
      title={label}
      onClick={onClick}
      className={cn(chrome, active && "bg-surface-raised text-body")}
    >
      {children}
    </button>
  );
}

export function Header() {
  const workspace = useWorkspaceContext();
  const { sidebarOpen, setSidebarOpen } = useShell();
  const pathname = usePathname();
  const [dark, setDark] = useState(true);

  useEffect(() => {
    const stored = window.localStorage.getItem("mcaia-theme");

    setDark(stored ? stored === "dark" : window.matchMedia("(prefers-color-scheme: dark)").matches);
  }, []);

  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    window.localStorage.setItem("mcaia-theme", dark ? "dark" : "light");
  }, [dark]);

  const { projectId, rig, selected, animations } = workspace;
  const inspecting = Boolean(projectId) && pathname.split("/").length > 3;
  const target = selected ?? animations[0]?.name ?? null;
  const inspectorHref = projectId
    ? inspecting
      ? `/p/${projectId}`
      : `/p/${projectId}/${encodeURIComponent(target ?? "")}`
    : null;

  return (
    <header className="flex h-[52px] shrink-0 items-center gap-3 px-4">
      {sidebarOpen ? null : (
        <IconButton label="Show sidebar" onClick={() => setSidebarOpen(true)}>
          <PanelLeftOpen size={16} />
        </IconButton>
      )}

      {rig ? (
        <span className="flex items-center gap-2 px-2 py-1">
          <span className="text-[13px] font-medium">{rig.model_name}</span>

          <span className="font-mono text-[11px] text-muted">
            {Object.keys(rig.bones).length} bones
          </span>
        </span>
      ) : null}

      <div className="ml-auto flex items-center gap-1">
        <IconButton label={dark ? "Light theme" : "Dark theme"} onClick={() => setDark((v) => !v)}>
          {dark ? <Sun size={16} /> : <Moon size={16} />}
        </IconButton>

        {inspectorHref && target ? (
          <Link
            href={inspectorHref}
            aria-label="Inspector"
            title="Inspector"
            className={cn(chrome, inspecting && "bg-surface-raised text-body")}
          >
            <PanelRight size={16} />
          </Link>
        ) : null}
      </div>
    </header>
  );
}

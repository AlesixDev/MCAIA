"use client";

import Image from "next/image";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";
import { Boxes, Layers, PanelLeftClose, Plus } from "lucide-react";

import { ProjectSummary } from "@/lib/api";
import { Profile } from "@/lib/auth";
import { cn } from "@/lib/utils";
import { UserMenu } from "@/components/auth/user-menu";
import { ProjectMenu, RenameField } from "@/components/shell/project-menu";

export function Sidebar({
  projects,
  projectId,
  profile,
  pending,
  onCollapse,
  onLogout,
  onAvatar,
  onRename,
  onDelete,
}: {
  projects: ProjectSummary[];
  projectId: string | null;
  profile: Profile | null;
  pending: boolean;
  onCollapse: () => void;
  onLogout: () => void;
  onAvatar: (file: File) => void;
  onRename: (id: string, name: string) => void;
  onDelete: (id: string) => void;
}) {
  const pathname = usePathname();
  const [editing, setEditing] = useState<string | null>(null);

  return (
    <aside className="flex h-full w-[244px] shrink-0 flex-col bg-canvas">
      <div className="flex h-[52px] items-center gap-2 px-4">
        <Image src="/logo.png" alt="" width={48} height={48} className="size-6 rounded-md" />

        <span className="text-[14px] font-semibold tracking-tight text-body">mcaia</span>

        <button
          onClick={onCollapse}
          aria-label="Hide sidebar"
          title="Hide sidebar"
          className="ml-auto rounded-md p-1.5 text-muted transition-colors hover:bg-surface hover:text-body"
        >
          <PanelLeftClose size={16} />
        </button>
      </div>

      <div className="flex flex-col gap-1 px-3">
        <Link
          href="/"
          className="flex w-full items-center gap-2 rounded-lg border border-line-strong bg-surface px-3 py-2 text-[13px] font-medium text-body transition-colors hover:border-accent/50 hover:bg-surface-raised"
        >
          <Plus size={15} className="text-muted" />
          New model
        </Link>

        <Link
          href="/models"
          className={cn(
            "flex w-full items-center gap-2 rounded-lg px-3 py-2 text-[13px] transition-colors",
            pathname === "/models"
              ? "bg-surface text-body"
              : "text-muted hover:bg-surface/70 hover:text-body",
          )}
        >
          <Layers size={15} />
          Library
        </Link>
      </div>

      <p className="px-4 pb-1.5 pt-6 text-[11px] font-medium uppercase tracking-[0.09em] text-faint">
        Models
      </p>

      <nav className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
        {projects.length === 0 ? (
          <p className="px-2 py-1 text-[12px] leading-relaxed text-muted">
            Upload a model (.bbmodel, .gltf, .glb, .obj) to get started.
          </p>
        ) : (
          <ul className="flex flex-col gap-0.5">
            {projects.map((project) => (
              <li key={project.id} className="group/row relative">
                {editing === project.id ? (
                  <div className="px-1.5 py-0.5">
                    <RenameField
                      value={project.name}
                      onCancel={() => setEditing(null)}
                      onCommit={(name) => {
                        setEditing(null);
                        onRename(project.id, name);
                      }}
                    />
                  </div>
                ) : (
                  <>
                    <Link
                      href={`/p/${project.id}`}
                      className={cn(
                        "flex w-full items-center gap-2.5 rounded-lg py-2 pl-2.5 pr-9 text-left transition-colors",
                        project.id === projectId
                          ? "bg-surface text-body shadow-[0_1px_2px_rgb(0_0_0/0.15)]"
                          : "text-muted hover:bg-surface/70 hover:text-body",
                      )}
                    >
                      <Boxes size={15} className="shrink-0" />

                      <span className="min-w-0 flex-1 truncate text-[13px]">{project.name}</span>
                    </Link>

                    <ProjectMenu
                      name={project.name}
                      className="absolute right-1.5 top-1/2 -translate-y-1/2 opacity-0 transition-opacity focus-within:opacity-100 group-hover/row:opacity-100"
                      onRename={() => setEditing(project.id)}
                      onDelete={() => onDelete(project.id)}
                    />
                  </>
                )}
              </li>
            ))}
          </ul>
        )}
      </nav>

      <div className="px-2 pb-2">
        <UserMenu profile={profile} pending={pending} onLogout={onLogout} onAvatar={onAvatar} />
      </div>
    </aside>
  );
}

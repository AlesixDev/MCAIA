"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Boxes, Plus } from "lucide-react";

import { useWorkspaceContext } from "@/lib/workspace-context";
import { ProjectMenu, RenameField } from "@/components/shell/project-menu";
import { Workbench } from "@/components/shell/workbench";

export default function Page() {
  const workspace = useWorkspaceContext();
  const router = useRouter();
  const [editing, setEditing] = useState<string | null>(null);

  async function remove(id: string) {
    await workspace.removeProject(id);

    if (id !== workspace.projectId) {
      return;
    }

    router.push("/");
  }

  return (
    <Workbench>
      <div className="min-h-0 flex-1 overflow-y-auto px-6 pb-10">
        <div className="mx-auto w-full max-w-3xl">
          <div className="flex items-end justify-between gap-4 pb-5 pt-2">
            <div>
              <h1 className="text-[19px] font-semibold tracking-tight">Library</h1>

              <p className="mt-1 text-[13px] text-muted">
                {workspace.projects.length} model
                {workspace.projects.length === 1 ? "" : "s"} imported.
              </p>
            </div>

            <Link
              href="/"
              className="flex items-center gap-2 rounded-lg border border-line-strong bg-surface px-3 py-2 text-[13px] font-medium text-body transition-colors hover:border-accent/50 hover:bg-surface-raised"
            >
              <Plus size={15} className="text-muted" />
              New model
            </Link>
          </div>

          {workspace.projects.length === 0 ? (
            <p className="rounded-xl border border-dashed border-line-strong px-6 py-10 text-center text-[13px] text-muted">
              Nothing imported yet. Drop a .bbmodel, .gltf, .glb or .obj file to get started.
            </p>
          ) : (
            <ul className="grid gap-2 sm:grid-cols-2">
              {workspace.projects.map((project) => (
                <li key={project.id} className="group relative min-w-0">
                  <Link
                    href={`/p/${project.id}`}
                    className="flex min-w-0 flex-col gap-2 rounded-xl border border-line bg-surface-sunken p-4 pr-10 transition-colors hover:border-accent/50"
                  >
                    <span className="flex min-w-0 items-center gap-2">
                      <Boxes size={15} className="shrink-0 text-muted" />

                      {editing === project.id ? null : (
                        <span className="min-w-0 flex-1 truncate text-[13.5px] font-medium text-body">
                          {project.name}
                        </span>
                      )}
                    </span>

                    <span className="font-mono text-[11px] text-muted">
                      {project.format} · {project.bones} bones · {project.animations.length}{" "}
                      animations
                    </span>
                  </Link>

                  {editing === project.id ? (
                    <div className="absolute left-9 right-4 top-3.5">
                      <RenameField
                        value={project.name}
                        onCancel={() => setEditing(null)}
                        onCommit={(name) => {
                          setEditing(null);
                          workspace.renameProject(project.id, name);
                        }}
                      />
                    </div>
                  ) : (
                    <ProjectMenu
                      name={project.name}
                      className="absolute right-3 top-3 opacity-0 transition-opacity focus-within:opacity-100 group-hover:opacity-100"
                      onRename={() => setEditing(project.id)}
                      onDelete={() => remove(project.id)}
                    />
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </Workbench>
  );
}

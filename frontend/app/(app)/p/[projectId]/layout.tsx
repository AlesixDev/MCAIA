"use client";

import { useEffect, useRef } from "react";
import { useParams, useRouter } from "next/navigation";

import { useShell } from "@/lib/shell-context";
import { useWorkspaceContext } from "@/lib/workspace-context";
import { Composer } from "@/components/chat/composer";
import { Thread } from "@/components/chat/thread";
import { Welcome } from "@/components/chat/welcome";
import { Workbench } from "@/components/shell/workbench";

export default function ProjectLayout({ children }: { children: React.ReactNode }) {
  const workspace = useWorkspaceContext();
  const { draft, setDraft } = useShell();
  const params = useParams<{ projectId: string }>();
  const router = useRouter();
  const loaded = useRef<string | null>(null);

  const projectId = params.projectId;
  const openProject = workspace.openProject;

  useEffect(() => {
    if (loaded.current === projectId) {
      return;
    }

    loaded.current = projectId;
    openProject(projectId);
  }, [openProject, projectId]);

  const empty = workspace.messages.length === 0 && !workspace.busy;
  const ready = workspace.projectId === projectId;

  const composer = (
    <Composer
      value={draft}
      onValueChange={setDraft}
      disabled={!ready || workspace.busy}
      busy={workspace.busy}
      onGenerate={workspace.generate}
    />
  );

  async function upload(files: File[]) {
    const created = await workspace.uploadModel(files);

    if (!created) {
      return;
    }

    router.push(`/p/${created}`);
  }

  function inspect(name: string) {
    workspace.setSelected(name);
    router.push(`/p/${projectId}/${encodeURIComponent(name)}`);
  }

  return (
    <>
      <Workbench>
        {empty ? (
          <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-8 pb-10">
            <Welcome
              ready={ready}
              modelName={workspace.rig?.model_name}
              onFile={upload}
              onSuggestion={setDraft}
            />

            {composer}
          </div>
        ) : (
          <>
            <Thread
              messages={workspace.messages}
              rig={workspace.rig}
              formats={workspace.formats}
              busy={workspace.busy}
              selected={workspace.selected}
              onInspect={inspect}
              onExport={(name, format) =>
                workspace.exportAnimations(format, workspace.namespace, [name])
              }
            />

            <div className="pb-5">{composer}</div>
          </>
        )}
      </Workbench>

      {children}
    </>
  );
}

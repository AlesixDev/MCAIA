"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

import { useShell } from "@/lib/shell-context";
import { useWorkspaceContext } from "@/lib/workspace-context";
import { Composer } from "@/components/chat/composer";
import { Welcome } from "@/components/chat/welcome";
import { Workbench } from "@/components/shell/workbench";

export default function Page() {
  const workspace = useWorkspaceContext();
  const { draft, setDraft } = useShell();
  const router = useRouter();

  const workspaceReset = workspace.reset;

  useEffect(() => {
    workspaceReset();
  }, [workspaceReset]);

  async function upload(files: File[]) {
    const created = await workspace.uploadModel(files);

    if (!created) {
      return;
    }

    router.push(`/p/${created}`);
  }

  return (
    <Workbench>
      <div className="flex min-h-0 flex-1 flex-col items-center justify-center gap-8 pb-10">
        <Welcome ready={false} onFile={upload} onSuggestion={setDraft} />

        <Composer
          value={draft}
          onValueChange={setDraft}
          disabled
          busy={workspace.busy}
          onGenerate={workspace.generate}
        />
      </div>
    </Workbench>
  );
}

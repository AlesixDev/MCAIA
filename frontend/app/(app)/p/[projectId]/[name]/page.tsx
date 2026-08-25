"use client";

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";

import { useWorkspaceContext } from "@/lib/workspace-context";
import { Inspector } from "@/components/shell/inspector";

export default function Page() {
  const workspace = useWorkspaceContext();
  const params = useParams<{ projectId: string; name: string }>();
  const router = useRouter();

  const name = decodeURIComponent(params.name);
  const setSelected = workspace.setSelected;

  useEffect(() => {
    setSelected(name);
  }, [name, setSelected]);

  if (!workspace.rig) {
    return null;
  }

  async function remove(target: string) {
    await workspace.removeAnimation(target);

    router.push(`/p/${params.projectId}`);
  }

  return (
    <Inspector
      rig={workspace.rig}
      animations={workspace.animations}
      selected={name}
      formats={workspace.formats}
      namespace={workspace.namespace}
      busy={workspace.busy}
      onClose={() => router.push(`/p/${params.projectId}`)}
      onSelect={(target) => router.replace(`/p/${params.projectId}/${encodeURIComponent(target)}`)}
      onDelete={remove}
      onExport={(format, namespace) => workspace.exportAnimations(format, namespace)}
    />
  );
}

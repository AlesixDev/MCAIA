"use client";

import { createContext, useContext, useEffect, useRef } from "react";

import { useSessionContext } from "@/lib/session-context";
import { useWorkspace } from "@/lib/use-workspace";

type WorkspaceValue = ReturnType<typeof useWorkspace>;

const WorkspaceContext = createContext<WorkspaceValue | null>(null);

export function WorkspaceProvider({ children }: { children: React.ReactNode }) {
  const workspace = useWorkspace();
  const session = useSessionContext();

  const accountId = session.profile?.id ?? null;
  const workspaceReset = workspace.reset;
  const account = useRef<string | null | undefined>(undefined);

  useEffect(() => {
    if (session.loading) {
      return;
    }

    const previous = account.current;

    account.current = accountId;

    if (previous === undefined || previous === accountId) {
      return;
    }

    workspaceReset();
  }, [accountId, session.loading, workspaceReset]);

  return <WorkspaceContext value={workspace}>{children}</WorkspaceContext>;
}

export function useWorkspaceContext() {
  const value = useContext(WorkspaceContext);

  if (!value) {
    throw new Error("useWorkspaceContext must be used inside WorkspaceProvider");
  }

  return value;
}

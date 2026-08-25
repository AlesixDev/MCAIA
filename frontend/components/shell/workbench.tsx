"use client";

import { useWorkspaceContext } from "@/lib/workspace-context";
import { Header } from "@/components/shell/header";
import { NoticeBar } from "@/components/notice-bar";

export function Workbench({ children }: { children: React.ReactNode }) {
  const workspace = useWorkspaceContext();

  return (
    <div className="flex min-w-0 flex-1 flex-col">
      <Header />

      {workspace.notice ? (
        <div className="mx-auto w-full max-w-2xl px-6 pb-2">
          <NoticeBar notice={workspace.notice} onDismiss={() => workspace.setNotice(null)} />
        </div>
      ) : null}

      {children}
    </div>
  );
}

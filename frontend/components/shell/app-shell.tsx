"use client";

import { AnimatePresence, motion } from "motion/react";
import { useRouter } from "next/navigation";

import { useShell } from "@/lib/shell-context";
import { useSessionContext } from "@/lib/session-context";
import { useWorkspaceContext } from "@/lib/workspace-context";
import { Sidebar } from "@/components/shell/sidebar";

export function AppShell({ children }: { children: React.ReactNode }) {
  const workspace = useWorkspaceContext();
  const session = useSessionContext();
  const { sidebarOpen, setSidebarOpen } = useShell();
  const router = useRouter();

  async function remove(id: string) {
    await workspace.removeProject(id);

    if (id !== workspace.projectId) {
      return;
    }

    router.push("/");
  }

  return (
    <div className="flex h-dvh overflow-hidden bg-canvas">
      <AnimatePresence initial={false}>
        {sidebarOpen ? (
          <motion.div
            initial={{ width: 0 }}
            animate={{ width: 244 }}
            exit={{ width: 0 }}
            transition={{ duration: 0.24, ease: [0.16, 1, 0.3, 1] }}
            className="overflow-hidden"
          >
            <Sidebar
              projects={workspace.projects}
              projectId={workspace.projectId}
              profile={session.profile}
              pending={session.pending}
              onCollapse={() => setSidebarOpen(false)}
              onLogout={session.logout}
              onAvatar={session.uploadAvatar}
              onRename={workspace.renameProject}
              onDelete={remove}
            />
          </motion.div>
        ) : null}
      </AnimatePresence>

      <div className={`flex min-w-0 flex-1 flex-col p-2 ${sidebarOpen ? "pl-0" : "pl-2"}`}>
        <main className="flex min-h-0 flex-1 overflow-hidden rounded-xl border border-line bg-surface shadow-panel">
          {children}
        </main>
      </div>
    </div>
  );
}

import { ShellProvider } from "@/lib/shell-context";
import { WorkspaceProvider } from "@/lib/workspace-context";
import { AppShell } from "@/components/shell/app-shell";

export default function AppLayout({ children }: { children: React.ReactNode }) {
  return (
    <WorkspaceProvider>
      <ShellProvider>
        <AppShell>{children}</AppShell>
      </ShellProvider>
    </WorkspaceProvider>
  );
}

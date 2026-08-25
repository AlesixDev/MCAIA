"use client";

import { createContext, useContext, useState } from "react";

type ShellValue = {
  sidebarOpen: boolean;
  setSidebarOpen: (open: boolean) => void;
  draft: string;
  setDraft: (draft: string) => void;
};

const ShellContext = createContext<ShellValue | null>(null);

export function ShellProvider({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(true);
  const [draft, setDraft] = useState("");

  return (
    <ShellContext value={{ sidebarOpen, setSidebarOpen, draft, setDraft }}>{children}</ShellContext>
  );
}

export function useShell() {
  const value = useContext(ShellContext);

  if (!value) {
    throw new Error("useShell must be used inside ShellProvider");
  }

  return value;
}

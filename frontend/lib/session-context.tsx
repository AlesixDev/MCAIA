"use client";

import { createContext, useContext } from "react";

import { useSession } from "@/lib/use-session";

type SessionValue = ReturnType<typeof useSession>;

const SessionContext = createContext<SessionValue | null>(null);

export function SessionProvider({ children }: { children: React.ReactNode }) {
  const session = useSession();

  return <SessionContext value={session}>{children}</SessionContext>;
}

export function useSessionContext() {
  const value = useContext(SessionContext);

  if (!value) {
    throw new Error("useSessionContext must be used inside SessionProvider");
  }

  return value;
}

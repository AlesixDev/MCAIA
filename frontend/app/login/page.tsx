"use client";

import { useEffect } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ArrowLeft } from "lucide-react";

import { useSessionContext } from "@/lib/session-context";
import { AuthForm } from "@/components/auth/auth-form";

export default function Page() {
  const session = useSessionContext();
  const router = useRouter();

  const signedIn = Boolean(session.profile);

  useEffect(() => {
    if (!signedIn) {
      return;
    }

    router.replace("/");
  }, [router, signedIn]);

  return (
    <main className="flex min-h-dvh flex-col items-center justify-center gap-6 bg-canvas p-6">
      <Link
        href="/"
        className="flex items-center gap-1.5 text-[12px] text-muted transition-colors hover:text-body"
      >
        <ArrowLeft size={14} />
        Back to mcaia
      </Link>

      <div className="w-full max-w-[360px] rounded-2xl border border-line bg-surface p-5 shadow-panel">
        <Image
          src="/logo.png"
          alt=""
          width={56}
          height={56}
          priority
          className="mx-auto mb-4 size-9 rounded-xl"
        />

        <AuthForm
          pending={session.pending}
          error={session.error}
          onModeChange={() => session.setError(null)}
          onSubmit={(mode, credentials) =>
            mode === "login" ? session.login(credentials) : session.register(credentials)
          }
        />
      </div>
    </main>
  );
}

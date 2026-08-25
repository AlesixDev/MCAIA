"use client";

import { FormEvent, useState } from "react";
import { AnimatePresence, motion } from "motion/react";

import { Credentials } from "@/lib/auth";
import { cn } from "@/lib/utils";

export type Mode = "login" | "register";

const field =
  "h-9 w-full rounded-lg border border-line bg-transparent px-3 text-[13px] text-body outline-none transition-colors placeholder:text-faint hover:border-line-strong focus:border-accent/60";

export function AuthForm({
  pending,
  error,
  onModeChange,
  onSubmit,
}: {
  pending: boolean;
  error: string | null;
  onModeChange: () => void;
  onSubmit: (mode: Mode, credentials: Credentials) => Promise<boolean>;
}) {
  const [mode, setMode] = useState<Mode>("login");
  const [email, setEmail] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  function switchTo(next: Mode) {
    setMode(next);
    onModeChange();
  }

  async function submit(event: FormEvent) {
    event.preventDefault();

    const done = await onSubmit(mode, {
      email,
      password,
      username: mode === "register" ? username : undefined,
    });

    if (!done) {
      return;
    }

    setPassword("");
  }

  return (
    <div className="flex flex-col">
      <h2 className="text-center text-[15px] font-semibold tracking-tight">
        {mode === "login" ? "Welcome back" : "Create your account"}
      </h2>

      <div className="mt-4 flex gap-1 rounded-lg bg-surface-sunken p-1">
        {(["login", "register"] as Mode[]).map((option) => (
          <button
            key={option}
            onClick={() => switchTo(option)}
            className={cn(
              "relative flex-1 rounded-md px-3 py-1.5 text-[12.5px] transition-colors",
              mode === option ? "text-body" : "text-muted hover:text-body",
            )}
          >
            {mode === option ? (
              <motion.span
                layoutId="auth-tab"
                transition={{ duration: 0.22, ease: [0.16, 1, 0.3, 1] }}
                className="absolute inset-0 rounded-md bg-surface shadow-[0_1px_2px_rgb(0_0_0/0.15)]"
              />
            ) : null}

            <span className="relative">{option === "login" ? "Sign in" : "Sign up"}</span>
          </button>
        ))}
      </div>

      <form onSubmit={submit} className="flex flex-col gap-3 pt-5">
        <input
          type="email"
          value={email}
          required
          autoComplete="email"
          placeholder="you@example.com"
          onChange={(event) => setEmail(event.target.value)}
          className={field}
        />

        <AnimatePresence initial={false}>
          {mode === "register" ? (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: "auto" }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.2, ease: [0.16, 1, 0.3, 1] }}
              className="overflow-hidden"
            >
              <input
                value={username}
                required
                minLength={3}
                maxLength={24}
                autoComplete="username"
                placeholder="Username"
                onChange={(event) => setUsername(event.target.value)}
                className={field}
              />
            </motion.div>
          ) : null}
        </AnimatePresence>

        <input
          type="password"
          value={password}
          required
          minLength={8}
          autoComplete={mode === "login" ? "current-password" : "new-password"}
          placeholder="Password"
          onChange={(event) => setPassword(event.target.value)}
          className={field}
        />

        <AnimatePresence>
          {error ? (
            <motion.p
              initial={{ opacity: 0, y: -4 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0 }}
              className="rounded-lg border border-bad/30 bg-bad/[0.08] px-3 py-2 text-[12px] text-bad"
            >
              {error}
            </motion.p>
          ) : null}
        </AnimatePresence>

        <button
          type="submit"
          disabled={pending}
          className="mt-1 flex h-9 items-center justify-center rounded-lg bg-accent text-[13px] font-medium text-accent-foreground transition-all hover:brightness-110 disabled:bg-line-strong disabled:text-faint"
        >
          {pending ? "One moment…" : mode === "login" ? "Sign in" : "Create account"}
        </button>

        <p className="text-center text-[11px] leading-relaxed text-muted">
          {mode === "login"
            ? "Your models and animations stay tied to your account."
            : "At least 8 characters. You can add a photo afterwards."}
        </p>
      </form>
    </div>
  );
}

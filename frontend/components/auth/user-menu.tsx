"use client";

import Link from "next/link";
import { useEffect, useRef, useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { ImagePlus, LogIn, LogOut, User } from "lucide-react";

import { Profile, avatarUrl } from "@/lib/auth";
import { cn } from "@/lib/utils";

export function Avatar({ profile, size = 26 }: { profile: Profile | null; size?: number }) {
  const url = avatarUrl(profile);

  if (url) {
    return (
      <img
        src={url}
        alt={profile?.display_name ?? ""}
        width={size}
        height={size}
        style={{ width: size, height: size }}
        className="shrink-0 rounded-full border border-line object-cover"
      />
    );
  }

  const initials = (profile?.display_name ?? profile?.username ?? "?").slice(0, 1).toUpperCase();

  return (
    <span
      style={{ width: size, height: size }}
      className="flex shrink-0 items-center justify-center rounded-full border border-line bg-surface-raised text-[11px] font-medium text-muted"
    >
      {profile ? initials : <User size={13} />}
    </span>
  );
}

export function UserMenu({
  profile,
  pending,
  onLogout,
  onAvatar,
}: {
  profile: Profile | null;
  pending: boolean;
  onLogout: () => void;
  onAvatar: (file: File) => void;
}) {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const picker = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (!open) {
      return;
    }

    const close = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    };

    document.addEventListener("mousedown", close);

    return () => document.removeEventListener("mousedown", close);
  }, [open]);

  if (!profile) {
    return (
      <Link
        href="/login"
        className="flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-muted transition-colors hover:bg-surface hover:text-body"
      >
        <LogIn size={15} />
        <span className="text-[13px]">Sign in</span>
      </Link>
    );
  }

  return (
    <div ref={root} className="relative">
      <button
        onClick={() => setOpen((state) => !state)}
        className={cn(
          "flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-left transition-colors",
          open ? "bg-surface" : "hover:bg-surface",
        )}
      >
        <Avatar profile={profile} />

        <span className="min-w-0 flex-1">
          <span className="block truncate text-[13px] text-body">{profile.display_name}</span>
          <span className="block truncate text-[11px] text-muted">@{profile.username}</span>
        </span>
      </button>

      <AnimatePresence>
        {open ? (
          <motion.div
            initial={{ opacity: 0, y: 6, scale: 0.97 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: 4, scale: 0.98 }}
            transition={{ duration: 0.16, ease: [0.16, 1, 0.3, 1] }}
            className="absolute bottom-[calc(100%+6px)] left-0 z-50 w-full overflow-hidden rounded-xl border border-line bg-surface-raised p-1 shadow-panel"
          >
            <button
              onClick={() => {
                picker.current?.click();
                setOpen(false);
              }}
              disabled={pending}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[12px] text-muted transition-colors hover:bg-surface-sunken hover:text-body disabled:opacity-50"
            >
              <ImagePlus size={13} />
              Change photo
            </button>

            <button
              onClick={() => {
                onLogout();
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[12px] text-muted transition-colors hover:bg-bad/10 hover:text-bad"
            >
              <LogOut size={13} />
              Sign out
            </button>
          </motion.div>
        ) : null}
      </AnimatePresence>

      <input
        ref={picker}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif"
        className="hidden"
        onChange={(event) => {
          const file = event.target.files?.[0];

          if (file) {
            onAvatar(file);
          }

          event.target.value = "";
        }}
      />
    </div>
  );
}

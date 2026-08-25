"use client";

import { useCallback, useEffect, useState } from "react";

import { ApiError, readToken, writeToken } from "@/lib/api";
import { Credentials, Profile, auth } from "@/lib/auth";

export function useSession() {
  const [profile, setProfile] = useState<Profile | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (!readToken()) {
      setLoading(false);

      return;
    }

    auth
      .me()
      .then(setProfile)
      .catch(() => writeToken(null))
      .finally(() => setLoading(false));
  }, []);

  const run = useCallback(async (action: () => Promise<Profile>) => {
    setPending(true);
    setError(null);

    try {
      setProfile(await action());

      return true;
    } catch (problem) {
      setError(problem instanceof ApiError ? problem.message : "Something went wrong");

      return false;
    } finally {
      setPending(false);
    }
  }, []);

  const register = useCallback(
    (credentials: Credentials) => run(() => auth.register(credentials)),
    [run],
  );

  const login = useCallback(
    (credentials: Credentials) => run(() => auth.login(credentials)),
    [run],
  );

  const uploadAvatar = useCallback(
    (file: File) => run(() => auth.uploadAvatar(file)),
    [run],
  );

  const updateProfile = useCallback(
    (body: { display_name?: string; username?: string }) => run(() => auth.updateProfile(body)),
    [run],
  );

  const logout = useCallback(async () => {
    await auth.logout().catch(() => writeToken(null));

    setProfile(null);
  }, []);

  return {
    profile,
    loading,
    pending,
    error,
    setError,
    register,
    login,
    logout,
    uploadAvatar,
    updateProfile,
  };
}

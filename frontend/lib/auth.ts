import { ApiError, authHeaders, baseUrl, writeToken } from "@/lib/api";

export type Profile = {
  id: string;
  email: string;
  username: string;
  display_name: string;
  avatar?: string;
  created_at: string;
};

export type Credentials = {
  email: string;
  username?: string;
  password: string;
};

export function avatarUrl(profile: Profile | null) {
  if (!profile?.avatar) {
    return null;
  }

  return `${baseUrl}/api/v1/avatars/${profile.avatar}`;
}

async function send<T>(path: string, init: RequestInit): Promise<T> {
  let response: Response;

  try {
    response = await fetch(baseUrl + path, init);
  } catch {
    throw new ApiError("Could not reach the backend", null);
  }

  if (!response.ok) {
    const body = await response.json().catch(() => null);

    throw new ApiError(translate(body?.error ?? response.statusText), body?.details ?? null);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

const messages: Record<string, string> = {
  "auth: email already registered": "That email is already registered",
  "auth: username already taken": "That username is taken",
  "auth: wrong email or password": "Wrong email or password",
  "auth: password must be at least 8 characters": "Passwords need at least 8 characters",
  "auth: invalid email address": "That email address is not valid",
  "auth: username must be 3-24 characters, letters, digits, _ or -":
    "Usernames take 3-24 characters: letters, digits, _ or -",
  "auth: avatar must be a png, jpeg, webp or gif image": "The image must be a png, jpeg, webp or gif",
  "auth: avatar is larger than 2 MB": "The image is larger than 2 MB",
  "you need to sign in": "You need to sign in",
};

function translate(message: string) {
  return messages[message] ?? message;
}

type SessionResponse = {
  user: Profile;
  token: string;
  expires_at: string;
};

export const auth = {
  register: async (credentials: Credentials) => {
    const result = await send<SessionResponse>("/api/v1/auth/register", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(credentials),
    });

    writeToken(result.token);

    return result.user;
  },

  login: async (credentials: Credentials) => {
    const result = await send<SessionResponse>("/api/v1/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(credentials),
    });

    writeToken(result.token);

    return result.user;
  },

  logout: async () => {
    await send<void>("/api/v1/auth/logout", { method: "POST", headers: authHeaders() });

    writeToken(null);
  },

  me: async () => {
    const result = await send<{ user: Profile }>("/api/v1/me", { headers: authHeaders() });

    return result.user;
  },

  updateProfile: async (body: { display_name?: string; username?: string }) => {
    const result = await send<{ user: Profile }>("/api/v1/me", {
      method: "PATCH",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(body),
    });

    return result.user;
  },

  uploadAvatar: async (file: File) => {
    const result = await send<{ user: Profile }>("/api/v1/me/avatar", {
      method: "POST",
      headers: { "Content-Type": file.type || "application/octet-stream", ...authHeaders() },
      body: file,
    });

    return result.user;
  },
};

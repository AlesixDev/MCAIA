export type Vec3 = [number, number, number];

export type Interpolation = "linear" | "catmullrom" | "step";

export type LoopMode = "once" | "loop" | "hold_on_last_frame";

export type Keyframe = {
  time: number;
  value: Vec3;
  interpolation?: Interpolation;
};

export type Track = {
  rotation?: Keyframe[];
  position?: Keyframe[];
  scale?: Keyframe[];
};

export type Animation = {
  name: string;
  length: number;
  loop: LoopMode;
  description?: string;
  bones: Record<string, Track>;
};

export type Face = {
  uv: [number, number, number, number];
  texture: number;
  rotation?: number;
};

export type Texture = {
  name: string;
  source: string;
  width?: number;
  height?: number;
};

export type Cube = {
  name: string;
  uuid?: string;
  from: Vec3;
  to: Vec3;
  origin: Vec3;
  rotation: Vec3;
  inflate?: number;
  faces?: Record<string, Face>;
};

export type Bone = {
  name: string;
  uuid?: string;
  parent?: string;
  origin: Vec3;
  rotation: Vec3;
  depth: number;
  children?: string[];
  cubes?: Cube[];
  bounds?: { min: Vec3; max: Vec3 };
};

export type Rig = {
  model_name: string;
  format?: string;
  roots: string[];
  bones: Record<string, Bone>;
  order: string[];
  textures?: Texture[];
  resolution?: [number, number];
};

export type ProjectSummary = {
  id: string;
  name: string;
  format: string;
  notes?: string[];
  bones: number;
  animations: string[];
  created_at: string;
  updated_at: string;
};

export type Issue = {
  path: string;
  message: string;
};

export type GenerateOutput = {
  animation: Animation;
  warnings?: Issue[];
  removed_keyframes: number;
  engine: string;
};

export type Health = {
  status: "ready" | "engine_unavailable";
  engine: string;
  detail: string;
};

export type ExportFormat = {
  id: string;
  label: string;
  extension: string;
};

export type ImportFormat = {
  id: string;
  label: string;
  extensions: string[];
};

export const baseUrl = process.env.NEXT_PUBLIC_API_URL ?? "http://127.0.0.1:8787";

const tokenKey = "mcaia-token";

export function readToken() {
  if (typeof window === "undefined") {
    return null;
  }

  return window.localStorage.getItem(tokenKey);
}

export function writeToken(token: string | null) {
  if (typeof window === "undefined") {
    return;
  }

  if (token) {
    window.localStorage.setItem(tokenKey, token);

    return;
  }

  window.localStorage.removeItem(tokenKey);
}

export function authHeaders(): Record<string, string> {
  const token = readToken();

  return token ? { Authorization: `Bearer ${token}` } : {};
}

export class ApiError extends Error {
  details: unknown;

  constructor(message: string, details: unknown) {
    super(message);
    this.name = "ApiError";
    this.details = details;
  }
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  let response: Response;

  try {
    response = await fetch(baseUrl + path, {
      ...init,
      headers: { ...authHeaders(), ...(init?.headers ?? {}) },
    });
  } catch {
    throw new ApiError("Could not reach the backend", null);
  }

  if (!response.ok) {
    const body = await response.json().catch(() => null);
    throw new ApiError(body?.error ?? response.statusText, body?.details ?? null);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export const api = {
  health: () => request<Health>("/api/v1/health"),

  formats: () =>
    request<{ formats: ExportFormat[]; importers: ImportFormat[] }>("/api/v1/formats"),

  projects: () => request<{ projects: ProjectSummary[] }>("/api/v1/projects"),

  createProject: (files: File[]) => {
    if (files.length === 1) {
      return request<{ project: ProjectSummary; rig: Rig }>("/api/v1/projects", {
        method: "POST",
        headers: {
          "Content-Type": "application/octet-stream",
          "X-Mcaia-Filename": files[0].name,
        },
        body: files[0],
      });
    }

    const form = new FormData();

    for (const file of files) {
      form.append("files", file, file.name);
    }

    return request<{ project: ProjectSummary; rig: Rig }>("/api/v1/projects", {
      method: "POST",
      body: form,
    });
  },

  project: (id: string) =>
    request<{ project: ProjectSummary; rig: Rig; animations: Animation[] }>(
      `/api/v1/projects/${id}`,
    ),

  renameProject: (id: string, name: string) =>
    request<{ project: ProjectSummary }>(`/api/v1/projects/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ name }),
    }),

  deleteProject: (id: string) =>
    request<void>(`/api/v1/projects/${id}`, { method: "DELETE" }),

  generate: (
    id: string,
    body: {
      prompt: string;
      name?: string;
      duration?: number;
      loop?: LoopMode;
      style?: string;
      fps?: number;
      optimize?: boolean;
    },
  ) =>
    request<GenerateOutput>(`/api/v1/projects/${id}/animations/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),

  saveAnimation: (id: string, animation: Animation) =>
    request<{ animation: Animation; warnings: Issue[] }>(
      `/api/v1/projects/${id}/animations`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(animation),
      },
    ),

  deleteAnimation: (id: string, name: string) =>
    request<void>(`/api/v1/projects/${id}/animations/${encodeURIComponent(name)}`, {
      method: "DELETE",
    }),

  exportAnimations: async (
    id: string,
    body: { format: string; namespace?: string; names?: string[] },
  ) => {
    const response = await fetch(`${baseUrl}/api/v1/projects/${id}/export`, {
      method: "POST",
      headers: { "Content-Type": "application/json", ...authHeaders() },
      body: JSON.stringify(body),
    });

    if (!response.ok) {
      const payload = await response.json().catch(() => null);
      throw new ApiError(payload?.error ?? response.statusText, payload?.details ?? null);
    }

    return {
      filename: response.headers.get("X-Mcaia-Filename") ?? "animation.json",
      content: await response.text(),
    };
  },
};

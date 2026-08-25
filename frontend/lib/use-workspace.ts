"use client";

import { useCallback, useEffect, useMemo, useState } from "react";

import {
  Animation,
  ApiError,
  ExportFormat,
  ImportFormat,
  Health,
  Issue,
  LoopMode,
  ProjectSummary,
  Rig,
  api,
} from "@/lib/api";

export type Notice = {
  tone: "info" | "error" | "success";
  message: string;
  details?: Issue[];
};

export type GenerateParams = {
  prompt: string;
  name: string;
  duration: number;
  loop: LoopMode;
  style: string;
  fps: number;
  optimize: boolean;
};

export type ChatMessage =
  | { id: string; kind: "user"; text: string; timestamp: string }
  | { id: string; kind: "system"; text: string; timestamp: string }
  | {
      id: string;
      kind: "result";
      text: string;
      timestamp: string;
      animation: Animation;
      warnings: Issue[];
      removed: number;
      engine: string;
    }
  | { id: string; kind: "error"; text: string; timestamp: string; details?: Issue[] };

type DraftMessage<T> = T extends unknown ? Omit<T, "id" | "timestamp"> : never;

function stamp() {
  return new Date().toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
}

function id() {
  return Math.random().toString(36).slice(2, 10);
}

export function useWorkspace() {
  const [health, setHealth] = useState<Health | null>(null);
  const [formats, setFormats] = useState<ExportFormat[]>([]);
  const [importers, setImporters] = useState<ImportFormat[]>([]);
  const [projects, setProjects] = useState<ProjectSummary[]>([]);
  const [projectId, setProjectId] = useState<string | null>(null);
  const [rig, setRig] = useState<Rig | null>(null);
  const [animations, setAnimations] = useState<Animation[]>([]);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [notice, setNotice] = useState<Notice | null>(null);

  const push = useCallback((message: DraftMessage<ChatMessage>) => {
    setMessages((current) => [
      ...current,
      { ...message, id: id(), timestamp: stamp() } as ChatMessage,
    ]);
  }, []);

  const report = useCallback(
    (error: unknown, asMessage: boolean) => {
      const fallback = { message: "Something went wrong", details: undefined as Issue[] | undefined };
      const parsed =
        error instanceof ApiError
          ? {
              message: error.message,
              details: Array.isArray(error.details) ? (error.details as Issue[]) : undefined,
            }
          : fallback;

      if (asMessage) {
        push({ kind: "error", text: parsed.message, details: parsed.details });

        return;
      }

      setNotice({ tone: "error", message: parsed.message, details: parsed.details });
    },
    [push],
  );

  const refreshProjects = useCallback(async () => {
    try {
      const result = await api.projects();

      setProjects(result.projects);
    } catch (error) {
      report(error, false);
    }
  }, [report]);

  const openProject = useCallback(
    async (target: string) => {
      setBusy(true);

      try {
        const result = await api.project(target);

        setProjectId(result.project.id);
        setRig(result.rig);
        setAnimations(result.animations ?? []);
        setSelected(result.animations?.[0]?.name ?? null);
        setMessages([]);
        push({
          kind: "system",
          text: `${result.project.name} opened · ${result.project.bones} bones · ${result.project.animations.length} animations`,
        });

        return result.project.id;
      } catch (error) {
        report(error, false);

        return null;
      } finally {
        setBusy(false);
      }
    },
    [push, report],
  );

  useEffect(() => {
    api.health().then(setHealth).catch(() => setHealth(null));
    api
      .formats()
      .then((result) => {
        setFormats(result.formats);
        setImporters(result.importers);
      })
      .catch(() => setFormats([]));

    refreshProjects();
  }, [refreshProjects]);

  const uploadModel = useCallback(
    async (files: File[]) => {
      if (files.length === 0) {
        return null;
      }

      setBusy(true);

      try {
        const result = await api.createProject(files);

        setProjectId(result.project.id);
        setRig(result.rig);
        setAnimations([]);
        setSelected(null);
        setMessages([]);
        push({
          kind: "system",
          text: `${result.project.name} imported from ${result.project.format} · ${result.project.bones} bones detected`,
        });

        for (const note of result.project.notes ?? []) {
          push({ kind: "system", text: note });
        }

        await refreshProjects();

        return result.project.id;
      } catch (error) {
        report(error, false);

        return null;
      } finally {
        setBusy(false);
      }
    },
    [push, refreshProjects, report],
  );

  const generate = useCallback(
    async (params: GenerateParams) => {
      if (!projectId) {
        return;
      }

      setBusy(true);
      setNotice(null);
      push({ kind: "user", text: params.prompt });

      try {
        const result = await api.generate(projectId, params);

        setAnimations((current) => [
          ...current.filter((item) => item.name !== result.animation.name),
          result.animation,
        ]);

        setSelected(result.animation.name);
        push({
          kind: "result",
          text: result.animation.name,
          animation: result.animation,
          warnings: result.warnings ?? [],
          removed: result.removed_keyframes,
          engine: result.engine,
        });

        await refreshProjects();
      } catch (error) {
        report(error, true);
      } finally {
        setBusy(false);
      }
    },
    [projectId, push, refreshProjects, report],
  );

  const removeAnimation = useCallback(
    async (name: string) => {
      if (!projectId) {
        return;
      }

      try {
        await api.deleteAnimation(projectId, name);

        setAnimations((current) => current.filter((item) => item.name !== name));
        setSelected((current) => (current === name ? null : current));

        await refreshProjects();
      } catch (error) {
        report(error, false);
      }
    },
    [projectId, refreshProjects, report],
  );

  const removeProject = useCallback(
    async (target: string) => {
      try {
        await api.deleteProject(target);

        setProjects((current) => current.filter((item) => item.id !== target));
      } catch (error) {
        report(error, false);
      }
    },
    [report],
  );

  const renameProject = useCallback(
    async (target: string, name: string) => {
      try {
        const result = await api.renameProject(target, name);

        setProjects((current) =>
          current.map((item) => (item.id === target ? result.project : item)),
        );

        setRig((current) =>
          current && target === projectId ? { ...current, model_name: name } : current,
        );
      } catch (error) {
        report(error, false);
      }
    },
    [projectId, report],
  );

  const exportAnimations = useCallback(
    async (format: string, namespace: string, names?: string[]) => {
      if (!projectId) {
        return;
      }

      setBusy(true);

      try {
        const result = await api.exportAnimations(projectId, { format, namespace, names });
        const url = URL.createObjectURL(new Blob([result.content], { type: "application/json" }));
        const anchor = document.createElement("a");

        anchor.href = url;
        anchor.download = result.filename;
        anchor.click();

        URL.revokeObjectURL(url);

        push({ kind: "system", text: `Exported ${result.filename}` });
      } catch (error) {
        report(error, true);
      } finally {
        setBusy(false);
      }
    },
    [projectId, push, report],
  );

  const reset = useCallback(async () => {
    setProjectId(null);
    setRig(null);
    setAnimations([]);
    setMessages([]);
    setSelected(null);
    setNotice(null);

    await refreshProjects();
  }, [refreshProjects]);

  const namespace = useMemo(
    () => rig?.model_name.toLowerCase().replace(/\s+/g, "_") ?? "",
    [rig],
  );

  const current = useMemo(
    () => animations.find((item) => item.name === selected) ?? null,
    [animations, selected],
  );

  return {
    health,
    formats,
    importers,
    projects,
    projectId,
    rig,
    animations,
    messages,
    selected,
    current,
    namespace,
    busy,
    notice,
    setSelected,
    setNotice,
    reset,
    openProject,
    uploadModel,
    generate,
    removeAnimation,
    removeProject,
    renameProject,
    exportAnimations,
  };
}

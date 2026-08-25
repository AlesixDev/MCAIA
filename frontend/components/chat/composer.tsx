"use client";

import { useState } from "react";
import { Sparkles } from "lucide-react";

import { LoopMode } from "@/lib/api";
import { GenerateParams } from "@/lib/use-workspace";
import { cn } from "@/lib/utils";
import AIPromptInput from "@/components/smoothui/ai-prompt-input";
import { Select } from "@/components/ui/select";
import { Stepper } from "@/components/ui/stepper";

const loops = [
  { value: "loop", label: "loop" },
  { value: "once", label: "once" },
  { value: "hold_on_last_frame", label: "hold" },
];

export function Composer({
  value,
  onValueChange,
  disabled,
  busy,
  onGenerate,
}: {
  value: string;
  onValueChange: (value: string) => void;
  disabled: boolean;
  busy: boolean;
  onGenerate: (params: GenerateParams) => void;
}) {
  const [name, setName] = useState("idle");
  const [duration, setDuration] = useState(1);
  const [loop, setLoop] = useState<LoopMode>("loop");
  const [optimize, setOptimize] = useState(true);

  const submit = (prompt: string) => {
    const trimmed = prompt.trim();

    if (!trimmed || disabled) {
      return;
    }

    onGenerate({ prompt: trimmed, name, duration, loop, style: "", fps: 20, optimize });
    onValueChange("");
  };

  return (
    <div className="mx-auto w-full max-w-2xl px-6">
      <AIPromptInput
        value={value}
        onValueChange={onValueChange}
        onSubmit={submit}
        disabled={disabled}
        state={busy ? "thinking" : "idle"}
        maxLength={800}
        placeholder={disabled ? "Upload a model to get started" : "Describe the motion…"}
      >
        <input
          value={name}
          disabled={disabled}
          aria-label="Animation name"
          onChange={(event) => setName(event.target.value)}
          className="h-7 w-24 rounded-lg border border-line bg-transparent px-2 font-mono text-[11.5px] text-body outline-none transition-colors hover:border-line-strong focus:border-accent/60 disabled:opacity-50"
        />

        <Stepper
          value={duration}
          min={0.1}
          max={30}
          step={0.1}
          suffix="s"
          disabled={disabled}
          onChange={setDuration}
        />

        <Select
          value={loop}
          options={loops}
          disabled={disabled}
          onChange={(next) => setLoop(next as LoopMode)}
        />

        <button
          type="button"
          disabled={disabled}
          title="Drop redundant keyframes"
          onClick={() => setOptimize((current) => !current)}
          className={cn(
            "flex h-7 items-center gap-1.5 rounded-lg border px-2 text-[11.5px] transition-colors disabled:opacity-50",
            optimize
              ? "border-accent/40 bg-accent/10 text-accent"
              : "border-line text-muted hover:border-line-strong hover:text-body",
          )}
        >
          <Sparkles size={12} />
          Clean up
        </button>
      </AIPromptInput>

      <p className="mt-2 text-center text-[10.5px] text-muted">
        Keyframes are validated against the rig before they reach Blockbench.
      </p>
    </div>
  );
}

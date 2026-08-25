"use client";

import Image from "next/image";
import { DragEvent, useRef, useState } from "react";
import { Upload } from "lucide-react";

import { cn } from "@/lib/utils";
import AISuggestions from "@/components/smoothui/ai-suggestions";

const suggestions = [
  { id: "idle", label: "Idle with soft breathing" },
  { id: "walk", label: "Walk cycle" },
  { id: "attack", label: "Forward punch" },
  { id: "wave", label: "Wave with the right arm" },
  { id: "death", label: "Slow fall backwards" },
];

export function Welcome({
  ready,
  modelName,
  onFile,
  onSuggestion,
}: {
  ready: boolean;
  modelName?: string;
  onFile: (files: File[]) => void;
  onSuggestion: (label: string) => void;
}) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);

  const drop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setOver(false);

    const files = Array.from(event.dataTransfer.files ?? []);

    if (files.length > 0) {
      onFile(files);
    }
  };

  return (
    <div
      onDragOver={(event) => {
        event.preventDefault();
        setOver(true);
      }}
      onDragLeave={() => setOver(false)}
      onDrop={drop}
      className="flex w-full justify-center px-6"
    >
      <div className="flex w-full max-w-md flex-col items-center gap-5">
        <Image
          src="/logo.png"
          alt=""
          width={112}
          height={112}
          priority
          className="size-14 rounded-2xl border border-line"
        />

        <div className="text-center">
          <h1 className="text-[19px] font-semibold tracking-tight">
            {ready ? `${modelName} is ready to animate` : "Animate your model by talking"}
          </h1>

          <p className="mt-1.5 text-[13px] leading-relaxed text-muted">
            {ready
              ? "Describe the motion and the local AI writes the keyframes."
              : "Takes Blockbench .bbmodel, .gltf/.glb and .obj files."}
          </p>
        </div>

        {ready ? (
          <AISuggestions
            suggestions={suggestions}
            onSelect={(suggestion) => onSuggestion(suggestion.label)}
          />
        ) : (
          <button
            onClick={() => inputRef.current?.click()}
            className={cn(
              "flex w-full flex-col items-center gap-2 rounded-2xl border border-dashed px-6 py-7 transition-colors",
              over ? "border-accent bg-accent/[0.06]" : "border-line-strong hover:border-accent/50",
            )}
          >
            <Upload size={18} className="text-muted" />

            <span className="text-[13px] font-medium text-body">Drop your model here</span>

            <span className="text-[11.5px] text-muted">bbmodel · gltf · glb · obj</span>

            <span className="text-[11px] text-faint">
              For OBJ, add its .mtl and texture to keep the skin
            </span>
          </button>
        )}

        <input
          ref={inputRef}
          type="file"
          multiple
          accept=".bbmodel,.gltf,.glb,.obj,.json,.mtl,.png,.jpg,.jpeg"
          className="hidden"
          onChange={(event) => {
            const files = Array.from(event.target.files ?? []);

            if (files.length > 0) {
              onFile(files);
            }

            event.target.value = "";
          }}
        />
      </div>
    </div>
  );
}

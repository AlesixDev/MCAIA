"use client";

import Image from "next/image";

import { ExportFormat, Rig } from "@/lib/api";
import { ChatMessage } from "@/lib/use-workspace";
import AIConversation from "@/components/smoothui/ai-conversation";
import AIMessage from "@/components/smoothui/ai-message";
import AIResponse from "@/components/smoothui/ai-response";
import { AnimationCard } from "@/components/chat/animation-card";
import { Generating } from "@/components/chat/generating";

function Avatar() {
  return (
    <Image
      src="/logo.png"
      alt=""
      width={56}
      height={56}
      className="size-7 rounded-full border border-line"
    />
  );
}

export function Thread({
  messages,
  rig,
  formats,
  busy,
  selected,
  onInspect,
  onExport,
}: {
  messages: ChatMessage[];
  rig: Rig | null;
  formats: ExportFormat[];
  busy: boolean;
  selected: string | null;
  onInspect: (name: string) => void;
  onExport: (name: string, format: string) => void;
}) {
  return (
    <AIConversation className="flex-1" contentKey={messages.length + (busy ? 1 : 0)}>
      <div className="mx-auto flex w-full max-w-2xl flex-col gap-7 px-6 pb-10 pt-8">
        {messages.map((message) => {
          if (message.kind === "user") {
            return (
              <AIMessage key={message.id} from="user" timestamp={message.timestamp}>
                {message.text}
              </AIMessage>
            );
          }

          if (message.kind === "system") {
            return (
              <div key={message.id} className="flex items-center gap-3">
                <span className="h-px flex-1 bg-line" />

                <span className="font-mono text-[10.5px] text-muted">{message.text}</span>

                <span className="h-px flex-1 bg-line" />
              </div>
            );
          }

          if (message.kind === "error") {
            return (
              <AIMessage key={message.id} from="assistant" avatar={<Avatar />} bubble={false}>
                <div className="rounded-xl border border-bad/30 bg-bad/[0.08] px-3.5 py-2.5 text-[12.5px] text-bad">
                  <p>{message.text}</p>

                  {message.details?.length ? (
                    <ul className="mt-1.5 flex flex-col gap-0.5 font-mono text-[10.5px] opacity-80">
                      {message.details.map((issue) => (
                        <li key={issue.path + issue.message}>
                          {issue.path}: {issue.message}
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </div>
              </AIMessage>
            );
          }

          return (
            <AIMessage
              key={message.id}
              from="assistant"
              avatar={<Avatar />}
              bubble={false}
              timestamp={message.timestamp}
              copyText={JSON.stringify(message.animation, null, 2)}
            >
              <div className="flex flex-col gap-3">
                <AIResponse
                  text={`Animated ${Object.keys(message.animation.bones).length} bones over ${message.animation.length.toFixed(2)} seconds.`}
                />

                <AnimationCard
                  rig={rig}
                  formats={formats}
                  animation={message.animation}
                  warnings={message.warnings}
                  removed={message.removed}
                  active={selected === message.animation.name}
                  onInspect={() => onInspect(message.animation.name)}
                  onExport={(format) => onExport(message.animation.name, format)}
                />
              </div>
            </AIMessage>
          );
        })}

        {busy ? (
          <AIMessage from="assistant" avatar={<Avatar />} bubble={false}>
            <Generating />
          </AIMessage>
        ) : null}
      </div>
    </AIConversation>
  );
}

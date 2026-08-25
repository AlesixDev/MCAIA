"use client";

import { cn } from "@/lib/utils";
import { motion, useReducedMotion } from "motion/react";
import { Fragment, useEffect, useRef } from "react";

const EASE_OUT = [0.23, 1, 0.32, 1] as const;
const WORD_TRANSITION = { duration: 0.22, ease: EASE_OUT } as const;
const WORD_BLUR_PX = 4;
const CITATION_MARKER = /^\[(\d+)\]$/;
const TOKEN_SPLIT = /(\s+|\[\d+\])/;
const HAS_WORD_CHARACTER = /[\p{L}\p{N}]/u;
const WHITESPACE_ONLY = /^\s+$/;

export type AIResponseCitation = {
  id: string;
  index: number;
  title: string;
  url?: string;
};

export type AIResponseProps = {
  citations?: AIResponseCitation[];
  className?: string;
  isStreaming?: boolean;
  text: string;
};

type Token = {
  citation?: AIResponseCitation;
  value: string;
};

const tokenize = (text: string, citations: AIResponseCitation[]): Token[] =>
  text
    .split(TOKEN_SPLIT)
    .filter((value) => value !== "")
    .map((value) => {
      const match = value.match(CITATION_MARKER);
      if (!match) {
        return { value };
      }
      const index = Number(match[1]);
      const citation = citations.find((entry) => entry.index === index);
      return citation ? { citation, value } : { value };
    });

const AIResponse = ({
  citations = [],
  className,
  isStreaming = false,
  text,
}: AIResponseProps) => {
  const shouldReduceMotion = useReducedMotion();
  const tokens = tokenize(text, citations);

  const paintedRef = useRef(0);
  const firstNewIndex = paintedRef.current;

  useEffect(() => {
    paintedRef.current = tokens.length;
  }, [tokens.length]);

  return (
    <p
      className={cn(
        "text-pretty text-foreground text-sm leading-relaxed",
        className
      )}
    >
      {tokens.map((token, index) => {
        const isNew = index >= firstNewIndex;
        const key = index;

        if (
          WHITESPACE_ONLY.test(token.value) ||
          !(token.citation || HAS_WORD_CHARACTER.test(token.value))
        ) {
          return <Fragment key={key}>{token.value}</Fragment>;
        }

        if (token.citation) {
          return (
            <AIResponseCitationPill
              citation={token.citation}
              isNew={isNew}
              key={key}
              shouldReduceMotion={Boolean(shouldReduceMotion)}
            />
          );
        }

        return (
          <motion.span
            animate={{ filter: "blur(0px)", opacity: 1, y: 0 }}
            className="inline-block"
            initial={
              isNew && !shouldReduceMotion
                ? { filter: `blur(${WORD_BLUR_PX}px)`, opacity: 0, y: 2 }
                : false
            }
            key={key}
            transition={shouldReduceMotion ? { duration: 0 } : WORD_TRANSITION}
          >
            {token.value}
          </motion.span>
        );
      })}
      {isStreaming ? (
        <AIResponseCaret shouldReduceMotion={shouldReduceMotion} />
      ) : null}
    </p>
  );
};

const AIResponseCaret = ({
  shouldReduceMotion,
}: {
  shouldReduceMotion: boolean | null;
}) => (
  <motion.span
    animate={shouldReduceMotion ? { opacity: 1 } : { opacity: [1, 0.15, 1] }}
    aria-hidden="true"
    className="ml-0.5 inline-block h-[1em] w-[2px] translate-y-[0.15em] rounded-full bg-current align-baseline"
    transition={
      shouldReduceMotion
        ? { duration: 0 }
        : { duration: 1, ease: "linear", repeat: Number.POSITIVE_INFINITY }
    }
  />
);

const AIResponseCitationPill = ({
  citation,
  isNew,
  shouldReduceMotion,
}: {
  citation: AIResponseCitation;
  isNew: boolean;
  shouldReduceMotion: boolean;
}) => {
  const shared = {
    animate: { opacity: 1, scale: 1 },
    className:
      "mr-0.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full border border-border bg-surface-sunken px-1 align-super font-medium text-[10px] text-muted-foreground no-underline",
    initial:
      isNew && !shouldReduceMotion
        ? { opacity: 0, scale: 0.6 }
        : (false as const),
    title: citation.title,
    transition: shouldReduceMotion
      ? { duration: 0 }
      : { bounce: 0.1, duration: 0.25, type: "spring" as const },
  };

  if (!citation.url) {
    return <motion.span {...shared}>{citation.index}</motion.span>;
  }

  return (
    <motion.a
      {...shared}
      className={`${shared.className} transition-colors hover:border-foreground/30 hover:text-foreground`}
      href={citation.url}
      rel="noopener noreferrer"
      target="_blank"
    >
      {citation.index}
    </motion.a>
  );
};

export default AIResponse;

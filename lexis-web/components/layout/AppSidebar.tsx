"use client";

import type { Goal, ChatFeedback } from "@/types";

interface AppSidebarProps {
  goals: Goal[];
  feedback: ChatFeedback[];
  words: string[];
}

export default function AppSidebar({ goals, feedback, words }: AppSidebarProps) {
  return (
    <aside
      className="flex flex-col overflow-y-auto shrink-0"
      style={{
        width: "220px",
        background: "var(--bg2)",
        borderRight: "1px solid var(--border)",
      }}
    >
      {/* Goals */}
      <div style={{ borderBottom: "1px solid var(--border)", padding: "14px" }}>
        <div
          className="uppercase tracking-[0.8px] mb-[10px]"
          style={{ fontSize: "10px", color: "var(--text3)" }}
        >
          {'// '}Цели обучения
        </div>
        {goals.length === 0 ? (
          <div style={{ fontSize: "11px", color: "var(--text3)" }}>Начни заниматься...</div>
        ) : (
          goals.map((g) => (
            <div key={g.id} className="mb-[9px] last:mb-0">
              <div className="flex justify-between mb-[3px]">
                <span style={{ fontSize: "11px", color: "var(--text2)" }}>{g.name}</span>
                <span style={{ fontSize: "10.5px", color: "var(--text3)" }}>{g.progress}%</span>
              </div>
              <div
                className="overflow-hidden"
                style={{ height: "3px", background: "var(--bg4)", borderRadius: "1px" }}
              >
                <div
                  className="h-full transition-[width] duration-600 ease-out"
                  style={{
                    width: `${g.progress}%`,
                    background: `var(--${g.color})`,
                    borderRadius: "1px",
                  }}
                />
              </div>
            </div>
          ))
        )}
      </div>

      {/* Feedback */}
      <div style={{ borderBottom: "1px solid var(--border)", padding: "14px" }}>
        <div
          className="uppercase tracking-[0.8px] mb-[10px]"
          style={{ fontSize: "10px", color: "var(--text3)" }}
        >
          {'// '}Фидбэк
        </div>
        {feedback.length === 0 ? (
          <div style={{ fontSize: "11px", color: "var(--text3)" }}>Начни заниматься...</div>
        ) : (
          feedback.map((f, i) => {
            const icons = { good: "\u2713", note: "\u2192", error: "\u2717" };
            const colors = {
              good: "var(--green)",
              note: "var(--amber)",
              error: "var(--red)",
            };
            const bgs = {
              good: "rgba(63,185,80,0.04)",
              note: "rgba(227,179,65,0.04)",
              error: "rgba(248,81,73,0.04)",
            };
            return (
              <div
                key={i}
                className="mb-[4px]"
                style={{
                  padding: "6px 9px",
                  borderLeft: `2px solid ${colors[f.type]}`,
                  borderRadius: "0 2px 2px 0",
                  fontSize: "11px",
                  lineHeight: "1.45",
                  color: colors[f.type],
                  background: bgs[f.type],
                }}
              >
                {icons[f.type]} {f.text}
              </div>
            );
          })
        )}
      </div>

      {/* Vocabulary */}
      <div style={{ padding: "14px" }}>
        <div
          className="uppercase tracking-[0.8px] mb-[10px]"
          style={{ fontSize: "10px", color: "var(--text3)" }}
        >
          {'// '}Словарь сессии
        </div>
        <div className="flex flex-wrap gap-[3px]">
          {words.length === 0 ? (
            <span style={{ fontSize: "11px", color: "var(--text3)" }}>&mdash;</span>
          ) : (
            words.map((w) => (
              <span
                key={w}
                style={{
                  display: "inline-block",
                  fontSize: "10px",
                  padding: "1px 6px",
                  borderRadius: "2px",
                  background: "rgba(125,133,144,0.08)",
                  color: "var(--text2)",
                  border: "1px solid rgba(125,133,144,0.15)",
                }}
              >
                {w}
              </span>
            ))
          )}
        </div>
      </div>
    </aside>
  );
}

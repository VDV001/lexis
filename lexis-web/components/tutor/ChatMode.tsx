"use client";

import { useState, useRef, useEffect } from "react";
import { useSSE } from "@/lib/hooks/useSSE";
import type { ChatCorrection, ChatFeedback } from "@/types";

interface ChatMessage {
  role: "user" | "tutor";
  text: string;
  time: string;
  correction?: ChatCorrection | null;
}

// Props for parent to receive feedback/words
interface ChatModeProps {
  onFeedback?: (fb: ChatFeedback) => void;
  onWords?: (words: string[]) => void;
}

export default function ChatMode({ onFeedback, onWords }: ChatModeProps) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [input, setInput] = useState("");
  const [showWelcome, setShowWelcome] = useState(true);
  const { send, events, isStreaming, error } = useSSE();
  const messagesRef = useRef<HTMLDivElement>(null);
  const streamTextRef = useRef("");
  const lastEventsLenRef = useRef(0);

  // Process new SSE events
  useEffect(() => {
    if (events.length <= lastEventsLenRef.current) return;
    const newEvents = events.slice(lastEventsLenRef.current);
    lastEventsLenRef.current = events.length;

    for (const event of newEvents) {
      if (event.type === "delta" && event.content) {
        streamTextRef.current += event.content;
        // Update the last tutor message
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === "tutor") {
            return [...prev.slice(0, -1), { ...last, text: streamTextRef.current }];
          }
          return prev;
        });
      }
      if (event.type === "correction" && event.correction) {
        setMessages((prev) => {
          const last = prev[prev.length - 1];
          if (last?.role === "tutor") {
            return [...prev.slice(0, -1), { ...last, correction: event.correction }];
          }
          return prev;
        });
      }
      if (event.type === "feedback" && event.feedback) {
        onFeedback?.(event.feedback);
      }
      if (event.type === "words" && event.words) {
        onWords?.(event.words);
      }
    }
  }, [events, onFeedback, onWords]);

  // Auto-scroll
  useEffect(() => {
    messagesRef.current?.scrollTo({ top: messagesRef.current.scrollHeight, behavior: "smooth" });
  }, [messages, isStreaming]);

  function handleSend() {
    const text = input.trim();
    if (!text || isStreaming) return;

    setShowWelcome(false);
    const now = new Date().toLocaleTimeString("ru", { hour: "2-digit", minute: "2-digit" });

    setMessages((prev) => [...prev, { role: "user", text, time: now }]);

    // Add empty tutor message for streaming
    streamTextRef.current = "";
    lastEventsLenRef.current = 0;
    setMessages((prev) => [...prev, { role: "tutor", text: "", time: now }]);

    setInput("");

    // Build messages array for API
    const apiMessages = [...messages, { role: "user" as const, text, time: now }]
      .filter((m) => m.text)
      .map((m) => ({
        role: m.role === "user" ? "user" : "assistant",
        content: m.text,
      }));

    send("/tutor/chat", { messages: apiMessages });
  }

  function handleStarter(text: string) {
    setInput(text);
    // Use setTimeout to allow state to update
    setTimeout(() => {
      const fakeInput = text;
      setShowWelcome(false);
      const now = new Date().toLocaleTimeString("ru", { hour: "2-digit", minute: "2-digit" });
      setMessages([{ role: "user", text: fakeInput, time: now }, { role: "tutor", text: "", time: now }]);
      streamTextRef.current = "";
      lastEventsLenRef.current = 0;
      setInput("");
      send("/tutor/chat", { messages: [{ role: "user", content: fakeInput }] });
    }, 0);
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      {/* Messages */}
      <div
        ref={messagesRef}
        className="flex-1 overflow-y-auto flex flex-col gap-0"
        style={{ padding: "20px 26px" }}
      >
        {showWelcome && (
          <div style={{ padding: "28px 0", color: "var(--text2)" }}>
            <div className="flex items-center gap-2 text-[13px] font-semibold text-[var(--green)] mb-[3px]">
              <span>{">"}</span> Свободная практика
            </div>
            <div className="text-[11px] text-[var(--text2)] mb-[18px]" style={{ paddingLeft: 17 }}>
              Пиши по-английски — получай исправления и фидбэк
            </div>
            <div className="text-[10px] text-[var(--text3)] uppercase tracking-[0.5px] mb-2">
              {"// "}БЫСТРЫЙ СТАРТ
            </div>
            {["Tell me about your work as a Go developer", "I worked on an API integration last week", "Explain what a goroutine is in simple English"].map((text) => (
              <div
                key={text}
                onClick={() => handleStarter(text)}
                className="flex items-start gap-[9px] cursor-pointer transition-all mb-[5px]"
                style={{
                  background: "var(--bg2)",
                  border: "1px solid var(--border)",
                  borderRadius: "2px",
                  padding: "9px 13px",
                }}
              >
                <span className="text-[var(--text3)] shrink-0">{">"}</span>
                <span className="text-[12px] text-[var(--text2)] leading-[1.5]">{text}</span>
              </div>
            ))}
          </div>
        )}

        {messages.map((msg, i) => (
          <div
            key={i}
            style={{
              borderBottom: "1px solid rgba(48,54,61,0.4)",
              padding: "12px 0",
              animation: "fadeUp 0.2s ease",
            }}
          >
            <div className="flex gap-2 text-[11px] text-[var(--text3)] mb-[5px]">
              <span className={`font-medium ${msg.role === "user" ? "text-[var(--cyan)]" : "text-[var(--text2)]"}`}>
                {msg.role === "user" ? "вы" : "tutor"}
              </span>
              <span>{msg.time}</span>
            </div>
            <div className="text-[13px] leading-[1.7] text-[var(--text)]">
              {msg.role === "user" ? `> ${msg.text}` : msg.text}
              {msg.role === "tutor" && isStreaming && i === messages.length - 1 && (
                <span className="inline-block w-[7px] h-[13px] bg-[var(--green)] ml-[2px] align-middle animate-blink" />
              )}
            </div>
            {msg.correction && (
              <div
                className="mt-[9px]"
                style={{
                  background: "var(--bg3)",
                  border: "1px solid var(--border)",
                  borderLeft: "2px solid var(--amber)",
                  borderRadius: "0 2px 2px 0",
                  padding: "8px 11px",
                  fontSize: "12px",
                }}
              >
                <div className="text-[9.5px] text-[var(--amber)] uppercase tracking-[0.8px] mb-[5px]">
                  {"// "}исправление
                </div>
                <div className="flex items-start gap-[7px] mb-[2px] leading-[1.5]">
                  <span className="text-[var(--text3)] w-[12px] shrink-0">✗</span>
                  <span className="text-[var(--red)] line-through opacity-80">{msg.correction.original}</span>
                </div>
                <div className="flex items-start gap-[7px] mb-[2px] leading-[1.5]">
                  <span className="text-[var(--text3)] w-[12px] shrink-0">✓</span>
                  <span className="text-[var(--green)]">{msg.correction.fixed}</span>
                </div>
                <div
                  className="text-[11px] text-[var(--text2)] mt-[5px] pt-[5px]"
                  style={{ borderTop: "1px solid var(--border)" }}
                >
                  {msg.correction.explanation}
                </div>
              </div>
            )}
          </div>
        ))}

        {isStreaming && (
          <div className="flex items-center gap-2 py-3 text-[var(--text3)] text-[11.5px]">
            <span>tutor</span>
            <div className="flex gap-1">
              {[0, 1, 2].map((i) => (
                <span
                  key={i}
                  className="w-1 h-1 rounded-full bg-[var(--text3)]"
                  style={{ animation: `tdot 1.1s infinite`, animationDelay: `${i * 0.2}s` }}
                />
              ))}
            </div>
          </div>
        )}

        {error && (
          <div className="text-[12px] text-[var(--red)] py-2">⚠️ {error}</div>
        )}
      </div>

      {/* Input bar */}
      <div
        className="flex gap-2 items-end shrink-0"
        style={{
          padding: "12px 22px",
          background: "var(--bg2)",
          borderTop: "1px solid var(--border)",
        }}
      >
        <div className="flex-1 relative">
          <span className="absolute left-[11px] top-1/2 -translate-y-1/2 text-[var(--green)] text-[12px] pointer-events-none">
            {">"}
          </span>
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            rows={1}
            placeholder="Write in English..."
            className="w-full outline-none resize-none font-[family-name:var(--font-mono)]"
            style={{
              background: "var(--bg3)",
              border: "1px solid var(--border)",
              borderRadius: "2px",
              padding: "9px 12px 9px 24px",
              fontSize: "13px",
              color: "var(--text)",
              lineHeight: "1.5",
              minHeight: "40px",
              maxHeight: "100px",
            }}
          />
        </div>
        <button
          onClick={handleSend}
          disabled={isStreaming || !input.trim()}
          className="shrink-0 cursor-pointer transition-all font-[family-name:var(--font-mono)]"
          style={{
            padding: "9px 14px",
            background: "transparent",
            border: "1px solid var(--border)",
            borderRadius: "2px",
            fontSize: "11.5px",
            color: "var(--green)",
            opacity: isStreaming || !input.trim() ? 0.3 : 1,
          }}
        >
          [ отправить ]
        </button>
      </div>
    </div>
  );
}

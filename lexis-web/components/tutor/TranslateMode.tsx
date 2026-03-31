"use client";

import { useState } from "react";
import { api } from "@/lib/api";

interface TranslateExercise {
  russian: string;
  expected: string;
  hint: string;
  words: string[];
  error_type: string;
}

interface CheckResult {
  correct: boolean;
  expected: string;
  explanation: string;
}

export default function TranslateMode() {
  const [exercise, setExercise] = useState<TranslateExercise | null>(null);
  const [input, setInput] = useState("");
  const [result, setResult] = useState<CheckResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function generate() {
    setLoading(true);
    setError(null);
    setResult(null);
    setInput("");
    try {
      const data = await api.post<TranslateExercise>("/tutor/translate/generate", {});
      setExercise(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function check() {
    if (!exercise || !input.trim()) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.post<CheckResult>("/tutor/translate/check", {
        answer: input.trim(),
        context: JSON.stringify(exercise),
      });
      setResult(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      e.preventDefault();
      check();
    }
  }

  // Initial state — show start button
  if (!exercise && !loading && !error) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-4">
        <div className="text-[11px] text-[var(--text3)] uppercase tracking-[0.5px]">
          {"// "}Перевод
        </div>
        <button
          onClick={generate}
          className="cursor-pointer font-[family-name:var(--font-mono)] transition-all"
          style={{
            padding: "9px 18px",
            background: "transparent",
            border: "1px solid var(--border)",
            borderRadius: "2px",
            fontSize: "12px",
            color: "var(--green)",
          }}
        >
          [ начать ]
        </button>
      </div>
    );
  }

  return (
    <div className="flex-1 flex flex-col overflow-y-auto" style={{ padding: "24px 26px" }}>
      {/* Loading */}
      {loading && !exercise && (
        <div className="flex items-center gap-2 text-[var(--text3)] text-[12px]">
          <span
            className="inline-block w-3 h-3 border border-[var(--green)] rounded-full"
            style={{ borderTopColor: "transparent", animation: "spin 0.7s linear infinite" }}
          />
          Генерация задания...
        </div>
      )}

      {/* Error */}
      {error && (
        <div className="text-[12px] text-[var(--red)] mb-3">
          {"// "}ошибка: {error}
          <button
            onClick={generate}
            className="ml-3 cursor-pointer text-[var(--text2)] underline"
            style={{ background: "none", border: "none", fontSize: "12px", fontFamily: "var(--font-mono)" }}
          >
            повторить
          </button>
        </div>
      )}

      {/* Exercise card */}
      {exercise && (
        <div style={{ animation: "fadeUp 0.2s ease" }}>
          {/* Russian source */}
          <div
            style={{
              background: "var(--bg2)",
              border: "1px solid var(--border)",
              borderRadius: "2px",
              padding: "14px 16px",
              marginBottom: "14px",
            }}
          >
            <div className="text-[9.5px] text-[var(--text3)] uppercase tracking-[0.8px] mb-2">
              {"// "}ПЕРЕВЕДИТЕ
            </div>
            <div className="text-[15px] text-[var(--text)] leading-[1.6]">
              {exercise.russian}
            </div>
            <div className="text-[11px] text-[var(--text3)] mt-2">
              подсказка: <span className="text-[var(--cyan)]">{exercise.hint}</span>
            </div>
          </div>

          {/* Input */}
          {!result && (
            <div className="flex gap-2 items-end mb-4">
              <div className="flex-1 relative">
                <span className="absolute left-[11px] top-1/2 -translate-y-1/2 text-[var(--green)] text-[12px] pointer-events-none">
                  {">"}
                </span>
                <input
                  type="text"
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onKeyDown={handleKeyDown}
                  placeholder="Your translation..."
                  autoFocus
                  className="w-full outline-none font-[family-name:var(--font-mono)]"
                  style={{
                    background: "var(--bg3)",
                    border: "1px solid var(--border)",
                    borderRadius: "2px",
                    padding: "9px 12px 9px 24px",
                    fontSize: "13px",
                    color: "var(--text)",
                    lineHeight: "1.5",
                  }}
                />
              </div>
              <button
                onClick={check}
                disabled={loading || !input.trim()}
                className="shrink-0 cursor-pointer transition-all font-[family-name:var(--font-mono)]"
                style={{
                  padding: "9px 14px",
                  background: "transparent",
                  border: "1px solid var(--border)",
                  borderRadius: "2px",
                  fontSize: "11.5px",
                  color: "var(--green)",
                  opacity: loading || !input.trim() ? 0.3 : 1,
                }}
              >
                [ проверить ]
              </button>
            </div>
          )}

          {/* Checking spinner */}
          {loading && exercise && !result && (
            <div className="flex items-center gap-2 text-[var(--text3)] text-[12px] mb-3">
              <span
                className="inline-block w-3 h-3 border border-[var(--green)] rounded-full"
                style={{ borderTopColor: "transparent", animation: "spin 0.7s linear infinite" }}
              />
              Проверка...
            </div>
          )}

          {/* Result */}
          {result && (
            <div
              style={{
                background: "var(--bg3)",
                border: "1px solid var(--border)",
                borderLeft: result.correct
                  ? "2px solid var(--green)"
                  : "2px solid var(--red)",
                borderRadius: "0 2px 2px 0",
                padding: "12px 14px",
                marginBottom: "14px",
                animation: "fadeUp 0.2s ease",
              }}
            >
              <div
                className="text-[9.5px] uppercase tracking-[0.8px] mb-2"
                style={{ color: result.correct ? "var(--green)" : "var(--red)" }}
              >
                {"// "}{result.correct ? "ПРАВИЛЬНО" : "НЕПРАВИЛЬНО"}
              </div>
              {!result.correct && (
                <div className="text-[12px] text-[var(--text2)] mb-2 leading-[1.6]">
                  ваш ответ: <span className="text-[var(--red)] line-through opacity-80">{input}</span>
                </div>
              )}
              <div className="text-[12px] text-[var(--text2)] mb-2 leading-[1.6]">
                ожидалось: <span className="text-[var(--green)]">{result.expected || exercise.expected}</span>
              </div>
              {result.explanation && (
                <div
                  className="text-[11px] text-[var(--text2)] mt-2 pt-2 leading-[1.6]"
                  style={{ borderTop: "1px solid var(--border)" }}
                >
                  {result.explanation}
                </div>
              )}
            </div>
          )}

          {/* Next button */}
          {result && (
            <button
              onClick={generate}
              disabled={loading}
              className="cursor-pointer font-[family-name:var(--font-mono)] transition-all"
              style={{
                padding: "9px 18px",
                background: "transparent",
                border: "1px solid var(--border)",
                borderRadius: "2px",
                fontSize: "12px",
                color: "var(--cyan)",
                opacity: loading ? 0.3 : 1,
              }}
            >
              {"[ следующее \u2192 ]"}
            </button>
          )}
        </div>
      )}
    </div>
  );
}

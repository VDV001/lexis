"use client";

import { useState, useMemo } from "react";
import { api } from "@/lib/api";

interface ScrambleExercise {
  words: string[];
  correct: string;
  translation: string;
  explanation: string;
  vocab: string[];
}

interface CheckResult {
  correct: boolean;
  expected: string;
  explanation: string;
}

export default function ScrambleMode() {
  const [exercise, setExercise] = useState<ScrambleExercise | null>(null);
  const [answer, setAnswer] = useState<string[]>([]);
  const [result, setResult] = useState<CheckResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Shuffle words once per exercise
  const shuffledWords = useMemo(() => {
    if (!exercise) return [];
    const words = [...exercise.words];
    for (let i = words.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [words[i], words[j]] = [words[j], words[i]];
    }
    return words;
  }, [exercise]);

  // Track which word indices (in shuffledWords) are still available
  const available = useMemo(() => {
    const used = new Map<string, number>();
    for (const w of answer) {
      used.set(w, (used.get(w) || 0) + 1);
    }
    return shuffledWords.map((word, idx) => {
      const count = used.get(word) || 0;
      if (count > 0) {
        used.set(word, count - 1);
        return { word, idx, taken: true };
      }
      return { word, idx, taken: false };
    });
  }, [shuffledWords, answer]);

  async function generate() {
    setLoading(true);
    setError(null);
    setResult(null);
    setAnswer([]);
    try {
      const data = await api.post<ScrambleExercise>("/tutor/scramble/generate", {});
      setExercise(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  async function check() {
    if (!exercise || answer.length === 0) return;
    setLoading(true);
    setError(null);
    try {
      const data = await api.post<CheckResult>("/tutor/scramble/check", {
        answer: answer.join(" "),
        context: JSON.stringify(exercise),
      });
      setResult(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function addWord(word: string) {
    if (result) return;
    setAnswer((prev) => [...prev, word]);
  }

  function removeWord(idx: number) {
    if (result) return;
    setAnswer((prev) => prev.filter((_, i) => i !== idx));
  }

  // Initial state
  if (!exercise && !loading && !error) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-4">
        <div className="text-[11px] text-[var(--text3)] uppercase tracking-[0.5px]">
          {"// "}Скрэмбл
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

      {/* Exercise */}
      {exercise && !loading && (
        <div style={{ animation: "fadeUp 0.2s ease" }}>
          <div className="text-[9.5px] text-[var(--text3)] uppercase tracking-[0.8px] mb-2">
            {"// "}СОСТАВЬТЕ ПРЕДЛОЖЕНИЕ
          </div>

          {/* Translation hint */}
          <div className="text-[12px] text-[var(--text2)] mb-4 leading-[1.6]">
            {exercise.translation}
          </div>

          {/* Answer row */}
          <div
            style={{
              background: "var(--bg2)",
              border: "1px solid var(--border)",
              borderRadius: "2px",
              padding: "12px 14px",
              marginBottom: "14px",
              minHeight: "48px",
            }}
          >
            {answer.length === 0 ? (
              <span className="text-[12px] text-[var(--text3)]">нажмите на слова ниже...</span>
            ) : (
              <div className="flex flex-wrap gap-[6px]">
                {answer.map((word, idx) => (
                  <span
                    key={idx}
                    onClick={() => removeWord(idx)}
                    className="cursor-pointer font-[family-name:var(--font-mono)] transition-all"
                    style={{
                      display: "inline-block",
                      padding: "5px 10px",
                      background: "var(--bg3)",
                      border: "1px solid var(--cyan)",
                      borderRadius: "2px",
                      fontSize: "13px",
                      color: "var(--cyan)",
                    }}
                  >
                    {word}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Word tiles */}
          <div className="flex flex-wrap gap-[6px] mb-4">
            {available.map(({ word, idx, taken }) => (
              <span
                key={idx}
                onClick={() => !taken && addWord(word)}
                className="w-tile cursor-pointer font-[family-name:var(--font-mono)] transition-all"
                style={{
                  display: "inline-block",
                  padding: "5px 10px",
                  background: taken ? "var(--bg4)" : "var(--bg3)",
                  border: "1px solid var(--border)",
                  borderRadius: "2px",
                  fontSize: "13px",
                  color: taken ? "var(--text3)" : "var(--text)",
                  opacity: taken ? 0.3 : 1,
                }}
              >
                {word}
              </span>
            ))}
          </div>

          {/* Check button */}
          {!result && (
            <button
              onClick={check}
              disabled={loading || answer.length === 0}
              className="cursor-pointer font-[family-name:var(--font-mono)] transition-all mb-4"
              style={{
                padding: "9px 18px",
                background: "transparent",
                border: "1px solid var(--border)",
                borderRadius: "2px",
                fontSize: "12px",
                color: "var(--green)",
                opacity: loading || answer.length === 0 ? 0.3 : 1,
              }}
            >
              [ проверить ]
            </button>
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
              <div className="text-[12px] text-[var(--text2)] mb-2 leading-[1.6]">
                правильное предложение: <span className="text-[var(--green)]">{result.expected || exercise.correct}</span>
              </div>
              <div className="text-[12px] text-[var(--text2)] mb-2 leading-[1.6]">
                перевод: <span className="text-[var(--text)]">{exercise.translation}</span>
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

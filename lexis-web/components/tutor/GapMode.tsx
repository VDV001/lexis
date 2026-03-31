"use client";

import { useState, useMemo } from "react";
import { api } from "@/lib/api";

interface GapExercise {
  before: string;
  answer: string;
  after: string;
  options: string[];
  explanation: string;
  error_type: string;
  words: string[];
}

export default function GapMode() {
  const [exercise, setExercise] = useState<GapExercise | null>(null);
  const [selected, setSelected] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Shuffle options once per exercise (server says first is correct, but shuffles; we shuffle client-side too)
  const shuffledOptions = useMemo(() => {
    if (!exercise) return [];
    const opts = [...exercise.options];
    for (let i = opts.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [opts[i], opts[j]] = [opts[j], opts[i]];
    }
    return opts;
  }, [exercise]);

  const isCorrect = selected === exercise?.answer;

  async function generate() {
    setLoading(true);
    setError(null);
    setSelected(null);
    try {
      const data = await api.post<GapExercise>("/tutor/gap/generate", {});
      setExercise(data);
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setLoading(false);
    }
  }

  function handleSelect(option: string) {
    if (selected !== null) return; // Already answered
    setSelected(option);
  }

  // Initial state
  if (!exercise && !loading && !error) {
    return (
      <div className="flex-1 flex flex-col items-center justify-center gap-4">
        <div className="text-[11px] text-[var(--text3)] uppercase tracking-[0.5px]">
          {"// "}Заполните пробел
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
      {loading && (
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
          {/* Sentence with gap */}
          <div
            style={{
              background: "var(--bg2)",
              border: "1px solid var(--border)",
              borderRadius: "2px",
              padding: "14px 16px",
              marginBottom: "16px",
            }}
          >
            <div className="text-[9.5px] text-[var(--text3)] uppercase tracking-[0.8px] mb-2">
              {"// "}ЗАПОЛНИТЕ ПРОБЕЛ
            </div>
            <div className="text-[15px] text-[var(--text)] leading-[1.6]">
              {exercise.before}{" "}
              <span
                style={{
                  borderBottom: "2px solid var(--cyan)",
                  color: selected ? (isCorrect ? "var(--green)" : "var(--red)") : "var(--cyan)",
                  padding: "0 8px",
                  minWidth: "60px",
                  display: "inline-block",
                }}
              >
                {selected || "\u00A0___\u00A0"}
              </span>{" "}
              {exercise.after}
            </div>
          </div>

          {/* Options */}
          <div className="grid grid-cols-2 gap-2 mb-4">
            {shuffledOptions.map((option) => {
              let borderColor = "var(--border)";
              let textColor = "var(--text)";
              if (selected !== null) {
                if (option === exercise.answer) {
                  borderColor = "var(--green)";
                  textColor = "var(--green)";
                } else if (option === selected) {
                  borderColor = "var(--red)";
                  textColor = "var(--red)";
                }
              }
              return (
                <button
                  key={option}
                  onClick={() => handleSelect(option)}
                  disabled={selected !== null}
                  className="text-left cursor-pointer font-[family-name:var(--font-mono)] transition-all"
                  style={{
                    padding: "10px 14px",
                    background: "var(--bg3)",
                    border: `1px solid ${borderColor}`,
                    borderRadius: "2px",
                    fontSize: "13px",
                    color: textColor,
                    opacity: selected !== null && option !== selected && option !== exercise.answer ? 0.4 : 1,
                  }}
                >
                  {option}
                </button>
              );
            })}
          </div>

          {/* Result */}
          {selected !== null && (
            <div
              style={{
                background: "var(--bg3)",
                border: "1px solid var(--border)",
                borderLeft: isCorrect
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
                style={{ color: isCorrect ? "var(--green)" : "var(--red)" }}
              >
                {"// "}{isCorrect ? "ПРАВИЛЬНО" : "НЕПРАВИЛЬНО"}
              </div>
              {!isCorrect && (
                <div className="text-[12px] text-[var(--text2)] mb-2 leading-[1.6]">
                  правильный ответ: <span className="text-[var(--green)]">{exercise.answer}</span>
                </div>
              )}
              <div
                className="text-[11px] text-[var(--text2)] mt-1 leading-[1.6]"
              >
                {exercise.explanation}
              </div>
            </div>
          )}

          {/* Next button */}
          {selected !== null && (
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

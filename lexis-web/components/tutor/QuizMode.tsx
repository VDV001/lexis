"use client";

import { useState, useCallback, useEffect } from "react";
import api from "@/lib/api";
import type { QuizQuestion } from "@/types";

const TYPE_COLORS: Record<string, string> = {
  grammar: "var(--cyan)",
  vocabulary: "var(--green)",
  phrasal: "var(--amber)",
};

const OPTION_LETTERS = ["A", "B", "C", "D"];

function confidenceColor(confidence: number): string {
  if (confidence >= 80) return "var(--red)";
  if (confidence >= 65) return "var(--amber)";
  return "var(--green)";
}

function typeLabel(type: string): string {
  switch (type) {
    case "grammar": return "ГРАММАТИКА";
    case "vocabulary": return "СЛОВАРНЫЙ ЗАПАС";
    case "phrasal": return "ФРАЗОВЫЕ ГЛАГОЛЫ";
    default: return type.toUpperCase();
  }
}

export default function QuizMode() {
  const [question, setQuestion] = useState<QuizQuestion | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selected, setSelected] = useState<number | null>(null);
  const [isCorrect, setIsCorrect] = useState<boolean | null>(null);
  const [total, setTotal] = useState(0);
  const [correct, setCorrect] = useState(0);
  const [streak, setStreak] = useState(0);

  const accuracy = total > 0 ? Math.round((correct / total) * 100) : 0;

  const loadQuestion = useCallback(async () => {
    setLoading(true);
    setError(null);
    setSelected(null);
    setIsCorrect(null);
    try {
      const q = await api.post<QuizQuestion>("/tutor/quiz/generate", {});
      setQuestion(q);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки вопроса");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadQuestion();
  }, [loadQuestion]);

  const handleOption = useCallback(
    async (idx: number) => {
      if (selected !== null || !question) return;
      setSelected(idx);

      const correct_answer = idx === question.correct;
      setIsCorrect(correct_answer);
      setTotal((t) => t + 1);

      if (correct_answer) {
        setCorrect((c) => c + 1);
        setStreak((s) => s + 1);
      } else {
        setStreak(0);
      }

      try {
        await api.post("/tutor/quiz/answer", {
          question_type: question.type,
          selected: idx,
          correct: question.correct,
          is_correct: correct_answer,
        });
      } catch {
        // answer submission failure is non-critical
      }
    },
    [selected, question],
  );

  return (
    <div className="flex flex-col flex-1 overflow-hidden">
      <div
        className="flex-1 overflow-y-auto flex flex-col gap-0"
        style={{ padding: "20px 26px" }}
      >
        {/* Section title + meta */}
        <div className="flex items-center justify-between mb-[18px]">
          <div className="flex items-center gap-2 text-[13px] font-semibold text-[var(--cyan)]">
            <span>{">"}</span> Квиз
          </div>
          <div className="flex items-center gap-3 text-[11px] text-[var(--text3)]">
            <span>
              {correct}/{total}
            </span>
            <span>{accuracy}%</span>
            <span>серия {streak}</span>
          </div>
        </div>

        {/* Loading state */}
        {loading && (
          <div className="flex items-center gap-3 py-12 justify-center">
            <span
              className="inline-block w-[14px] h-[14px] border-2 rounded-full animate-spin"
              style={{
                borderColor: "var(--border)",
                borderTopColor: "var(--cyan)",
              }}
            />
            <span className="text-[12px] text-[var(--text3)]">
              генерация вопроса...
            </span>
          </div>
        )}

        {/* Error state */}
        {error && !loading && (
          <div
            style={{
              border: "1px solid var(--red)",
              borderRadius: "2px",
              padding: "14px 16px",
            }}
          >
            <div className="text-[12px] text-[var(--red)] mb-3">{error}</div>
            <button
              onClick={loadQuestion}
              className="cursor-pointer font-[family-name:var(--font-mono)]"
              style={{
                padding: "7px 14px",
                background: "transparent",
                border: "1px solid var(--red)",
                borderRadius: "2px",
                fontSize: "11.5px",
                color: "var(--red)",
              }}
            >
              [ повторить ]
            </button>
          </div>
        )}

        {/* Question card */}
        {question && !loading && !error && (
          <>
            {/* Group label */}
            <div className="text-[10px] text-[var(--text3)] uppercase tracking-[0.5px] mb-2">
              {"// "}
              <span style={{ color: TYPE_COLORS[question.type] || "var(--text3)" }}>
                {typeLabel(question.type)}
              </span>
              {" · "}ВОПРОС {total + (selected === null ? 1 : 0)}
              {" · "}B1
            </div>

            {/* Exercise card */}
            <div
              className="ex-card"
              style={{
                background: "var(--bg2)",
                border: "1px solid var(--border)",
                borderRadius: "2px",
                padding: "16px 18px",
                marginBottom: "14px",
              }}
            >
              <div className="flex items-center justify-between mb-3">
                <span
                  className="text-[10px] uppercase tracking-[0.5px] font-medium"
                  style={{
                    color: TYPE_COLORS[question.type] || "var(--text3)",
                  }}
                >
                  {question.type}
                </span>
                {/* Confidence bar */}
                <div className="flex items-center gap-2">
                  <span className="text-[10px] text-[var(--text3)]">
                    {question.confidence}%
                  </span>
                  <div
                    style={{
                      width: "80px",
                      height: "4px",
                      background: "var(--bg3)",
                      borderRadius: "2px",
                      overflow: "hidden",
                    }}
                  >
                    <div
                      style={{
                        width: `${question.confidence}%`,
                        height: "100%",
                        background: confidenceColor(question.confidence),
                        borderRadius: "2px",
                        transition: "width 0.3s ease",
                      }}
                    />
                  </div>
                </div>
              </div>

              <div className="text-[13px] leading-[1.7] text-[var(--text)]">
                {question.question}
              </div>
            </div>

            {/* Options */}
            <div className="flex flex-col gap-[6px] mb-[14px]">
              {question.options.map((opt, idx) => {
                const answered = selected !== null;
                const isSelected = selected === idx;
                const isCorrectOption = idx === question.correct;

                let borderColor = "var(--border)";
                let textColor = "var(--text2)";
                let bg = "var(--bg3)";

                if (answered) {
                  if (isCorrectOption) {
                    borderColor = "var(--green)";
                    textColor = "var(--green)";
                    bg = "var(--bg3)";
                  } else if (isSelected && !isCorrectOption) {
                    borderColor = "var(--red)";
                    textColor = "var(--red)";
                    bg = "var(--bg3)";
                  }
                }

                return (
                  <button
                    key={idx}
                    onClick={() => handleOption(idx)}
                    disabled={answered}
                    className="flex items-center gap-3 text-left cursor-pointer transition-all font-[family-name:var(--font-mono)] disabled:cursor-default"
                    style={{
                      background: bg,
                      border: `1px solid ${borderColor}`,
                      borderRadius: "2px",
                      padding: "10px 14px",
                      fontSize: "12px",
                      color: textColor,
                      opacity: answered && !isSelected && !isCorrectOption ? 0.4 : 1,
                    }}
                  >
                    <span
                      className="shrink-0 text-[10px] font-semibold"
                      style={{ color: borderColor, width: "14px" }}
                    >
                      {OPTION_LETTERS[idx]}
                    </span>
                    <span>{opt}</span>
                  </button>
                );
              })}
            </div>

            {/* Result block */}
            {selected !== null && (
              <div
                style={{
                  background: "var(--bg2)",
                  border: "1px solid var(--border)",
                  borderLeft: `2px solid ${isCorrect ? "var(--green)" : "var(--red)"}`,
                  borderRadius: "0 2px 2px 0",
                  padding: "12px 14px",
                  marginBottom: "14px",
                  animation: "fadeUp 0.2s ease",
                }}
              >
                <div
                  className="text-[10px] uppercase tracking-[0.8px] font-medium mb-[6px]"
                  style={{ color: isCorrect ? "var(--green)" : "var(--red)" }}
                >
                  {"// "}{isCorrect ? "ВЕРНО" : "НЕВЕРНО"}
                </div>
                <div className="text-[12px] leading-[1.6] text-[var(--text2)]">
                  {question.explanation}
                </div>
              </div>
            )}

            {/* Next button */}
            {selected !== null && (
              <button
                onClick={loadQuestion}
                className="self-start cursor-pointer transition-all font-[family-name:var(--font-mono)]"
                style={{
                  padding: "9px 16px",
                  background: "transparent",
                  border: "1px solid var(--cyan)",
                  borderRadius: "2px",
                  fontSize: "11.5px",
                  color: "var(--cyan)",
                }}
              >
                {"[ следующий вопрос \u2192 ]"}
              </button>
            )}
          </>
        )}
      </div>
    </div>
  );
}

"use client";

import { useState, useEffect, useCallback } from "react";
import api from "@/lib/api";
import type { VocabWord, VocabStatus } from "@/types";

/* ── Styles ── */

const card: React.CSSProperties = {
  background: "var(--bg2)",
  border: "1px solid var(--border)",
  borderRadius: 3,
  padding: "14px 16px",
};

const label: React.CSSProperties = {
  fontSize: 10,
  color: "var(--text3)",
  textTransform: "uppercase",
  letterSpacing: "0.04em",
  marginBottom: 8,
};

/* ── Helpers ── */

const STATUS_COLORS: Record<VocabStatus, string> = {
  confident: "var(--green)",
  uncertain: "var(--amber)",
  unknown: "var(--text3)",
};

const STATUS_LABELS: Record<VocabStatus, string> = {
  confident: "уверен",
  uncertain: "неуверен",
  unknown: "новое",
};

function formatDate(iso: string): string {
  const d = new Date(iso);
  const day = String(d.getUTCDate()).padStart(2, "0");
  const month = String(d.getUTCMonth() + 1).padStart(2, "0");
  return `${day}.${month}`;
}

/* ── Component ── */

export default function VocabularyView() {
  const [words, setWords] = useState<VocabWord[]>([]);
  const [due, setDue] = useState<VocabWord[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [newWord, setNewWord] = useState("");
  const [adding, setAdding] = useState(false);
  const [mutating, setMutating] = useState<Set<string>>(new Set());

  const startMutating = (id: string) =>
    setMutating((prev) => new Set(prev).add(id));
  const stopMutating = (id: string) =>
    setMutating((prev) => {
      const next = new Set(prev);
      next.delete(id);
      return next;
    });

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [w, d] = await Promise.all([
        api.get<VocabWord[]>("/vocabulary"),
        api.get<VocabWord[]>("/vocabulary/due"),
      ]);
      setWords(w ?? []);
      setDue(d ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка загрузки");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const handleAdd = useCallback(async () => {
    const trimmed = newWord.trim();
    if (!trimmed || adding) return;
    setAdding(true);
    setError(null);
    try {
      const created = await api.post<VocabWord>("/vocabulary", {
        word: trimmed,
        status: "unknown",
      });
      setWords((prev) => [created, ...prev]);
      setNewWord("");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка добавления");
    } finally {
      setAdding(false);
    }
  }, [newWord, adding]);

  const handleDelete = useCallback(async (id: string) => {
    if (!window.confirm("Удалить слово из словаря?")) return;
    startMutating(id);
    setError(null);
    try {
      await api.delete(`/vocabulary/${id}`);
      setWords((prev) => prev.filter((w) => w.id !== id));
      setDue((prev) => prev.filter((w) => w.id !== id));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Ошибка удаления");
    } finally {
      stopMutating(id);
    }
  }, []);

  const handleStatusChange = useCallback(
    async (id: string, status: VocabStatus) => {
      startMutating(id);
      setError(null);
      try {
        await api.patch(`/vocabulary/${id}`, { status });
        const update = (prev: VocabWord[]) =>
          prev.map((w) => (w.id === id ? { ...w, status } : w));
        setWords(update);
        setDue(update);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Ошибка обновления");
      } finally {
        stopMutating(id);
      }
    },
    [],
  );

  /* ── Loading ── */

  if (loading) {
    return (
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
          justifyContent: "center",
          gap: 8,
        }}
      >
        <div
          style={{
            width: 20,
            height: 20,
            border: "2px solid var(--bg4)",
            borderTopColor: "var(--cyan)",
            borderRadius: "50%",
            animation: "spin 0.8s linear infinite",
          }}
        />
        <span style={{ fontSize: 12, color: "var(--text3)" }}>
          загрузка словаря...
        </span>
      </div>
    );
  }

  return (
    <div style={{ flex: 1, padding: 16, overflow: "auto" }}>
      <div
        style={{ display: "flex", flexDirection: "column", gap: 12, maxWidth: 720 }}
      >
        {/* ── Error ── */}
        {error && (
          <div
            style={{
              border: "1px solid var(--red)",
              borderRadius: 2,
              padding: "10px 14px",
              fontSize: 12,
              color: "var(--red)",
              display: "flex",
              justifyContent: "space-between",
              alignItems: "center",
            }}
          >
            <span>{error}</span>
            <button
              onClick={() => setError(null)}
              aria-label="Закрыть ошибку"
              style={{
                background: "transparent",
                border: "none",
                color: "var(--red)",
                cursor: "pointer",
                fontSize: 14,
                fontFamily: "var(--font-mono)",
                padding: "0 4px",
              }}
            >
              x
            </button>
          </div>
        )}

        {/* ── Add word ── */}
        <div style={card}>
          <div style={label}>{"// "}ДОБАВИТЬ СЛОВО</div>
          <div style={{ display: "flex", gap: 8 }}>
            <input
              type="text"
              value={newWord}
              onChange={(e) => setNewWord(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleAdd();
              }}
              placeholder="новое слово..."
              style={{
                flex: 1,
                background: "var(--bg3)",
                border: "1px solid var(--border)",
                borderRadius: 2,
                padding: "8px 12px",
                fontSize: 12,
                color: "var(--text)",
                fontFamily: "var(--font-mono)",
                outline: "none",
              }}
            />
            <button
              onClick={handleAdd}
              disabled={adding || !newWord.trim()}
              style={{
                padding: "8px 16px",
                background: "transparent",
                border: "1px solid var(--cyan)",
                borderRadius: 2,
                fontSize: 11.5,
                color: "var(--cyan)",
                fontFamily: "var(--font-mono)",
                cursor: adding || !newWord.trim() ? "default" : "pointer",
                opacity: adding || !newWord.trim() ? 0.4 : 1,
              }}
            >
              {adding ? "..." : "[ добавить ]"}
            </button>
          </div>
        </div>

        {/* ── Due for review ── */}
        {due.length > 0 && (
          <div style={card}>
            <div style={label}>
              {"// "}НА ПОВТОРЕНИЕ{" "}
              <span style={{ color: "var(--amber)" }}>{due.length}</span>
            </div>
            <div style={{ display: "flex", flexDirection: "column", gap: 4 }}>
              {due.map((w) => (
                <div
                  key={w.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    justifyContent: "space-between",
                    padding: "6px 10px",
                    background: "var(--bg3)",
                    border: "1px solid var(--border)",
                    borderLeft: `2px solid ${STATUS_COLORS[w.status]}`,
                    borderRadius: "0 2px 2px 0",
                    fontSize: 12,
                  }}
                >
                  <span
                    style={{
                      color: "var(--text)",
                      fontFamily: "var(--font-mono)",
                    }}
                  >
                    {w.word}
                  </span>
                  <span
                    style={{
                      fontSize: 10,
                      color: STATUS_COLORS[w.status],
                      textTransform: "uppercase",
                    }}
                  >
                    {STATUS_LABELS[w.status]}
                  </span>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* ── Word list ── */}
        <div style={card}>
          <div style={label}>
            {"// "}ВСЕ СЛОВА{" "}
            <span style={{ color: "var(--text2)" }}>{words.length}</span>
          </div>

          {words.length === 0 && (
            <span style={{ fontSize: 11, color: "var(--text3)" }}>
              словарь пуст — добавьте первое слово
            </span>
          )}

          <div style={{ display: "flex", flexDirection: "column", gap: 2 }}>
            {words.map((w) => {
              const isMutating = mutating.has(w.id);
              return (
                <div
                  key={w.id}
                  style={{
                    display: "flex",
                    alignItems: "center",
                    gap: 10,
                    padding: "7px 10px",
                    background: "var(--bg3)",
                    border: "1px solid var(--border)",
                    borderRadius: 2,
                    fontSize: 12,
                    opacity: isMutating ? 0.5 : 1,
                    pointerEvents: isMutating ? "none" : "auto",
                  }}
                >
                  {/* Word */}
                  <span
                    style={{
                      flex: 1,
                      color: "var(--text)",
                      fontFamily: "var(--font-mono)",
                    }}
                  >
                    {w.word}
                  </span>

                  {/* Status badge */}
                  <button
                    onClick={() => {
                      const order: VocabStatus[] = [
                        "unknown",
                        "uncertain",
                        "confident",
                      ];
                      const next =
                        order[(order.indexOf(w.status) + 1) % order.length];
                      handleStatusChange(w.id, next);
                    }}
                    disabled={isMutating}
                    aria-label="Сменить статус"
                    title="Сменить статус"
                    style={{
                      padding: "2px 8px",
                      background: "transparent",
                      border: `1px solid ${STATUS_COLORS[w.status]}`,
                      borderRadius: 2,
                      fontSize: 10,
                      color: STATUS_COLORS[w.status],
                      fontFamily: "var(--font-mono)",
                      textTransform: "uppercase",
                      cursor: isMutating ? "default" : "pointer",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {STATUS_LABELS[w.status]}
                  </button>

                  {/* Last seen */}
                  <span
                    style={{
                      fontSize: 10,
                      color: "var(--text3)",
                      fontFamily: "var(--font-mono)",
                      whiteSpace: "nowrap",
                    }}
                  >
                    {formatDate(w.last_seen)}
                  </span>

                  {/* Delete */}
                  <button
                    onClick={() => handleDelete(w.id)}
                    disabled={isMutating}
                    aria-label="Удалить слово"
                    title="Удалить"
                    style={{
                      padding: "2px 6px",
                      background: "transparent",
                      border: "1px solid var(--border)",
                      borderRadius: 2,
                      fontSize: 10,
                      color: "var(--text3)",
                      fontFamily: "var(--font-mono)",
                      cursor: isMutating ? "default" : "pointer",
                    }}
                    onMouseEnter={(e) => {
                      if (!isMutating) {
                        e.currentTarget.style.borderColor = "var(--red)";
                        e.currentTarget.style.color = "var(--red)";
                      }
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.borderColor = "var(--border)";
                      e.currentTarget.style.color = "var(--text3)";
                    }}
                  >
                    x
                  </button>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}

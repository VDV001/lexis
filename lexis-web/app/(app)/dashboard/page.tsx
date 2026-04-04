"use client";

import { useState, useEffect } from "react";
import api from "@/lib/api";
import { useSettingsStore } from "@/lib/stores/settings";
import AccuracyRing from "@/components/dashboard/AccuracyRing";
import VocabDonut from "@/components/dashboard/VocabDonut";
import VocabCurve from "@/components/dashboard/VocabCurve";
import MiniChart from "@/components/dashboard/MiniChart";
import type {
  ProgressSummary,
  VocabCurveData,
  Goal,
  ErrorType,
} from "@/types";

/* ── API response types ── */

interface ErrorCategory {
  error_type: ErrorType;
  count: number;
}

interface SessionItem {
  id: string;
  mode: string;
  round_count: number;
  correct_count: number;
}

/* ── Display-name maps ── */

const ERROR_LABELS: Record<string, string> = {
  articles: "Артикли",
  tenses: "Времена",
  prepositions: "Предлоги",
  phrasal: "Фразовые глаголы",
  vocabulary: "Лексика",
  word_order: "Порядок слов",
};

const MODE_LABELS: Record<string, string> = {
  chat: "Практика",
  quiz: "Квиз",
  translate: "Перевод",
  gap: "Пропуски",
  scramble: "Скрэмбл",
};

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

const bigNum: React.CSSProperties = {
  fontSize: 30,
  fontWeight: 700,
  color: "var(--text)",
  letterSpacing: -1,
  lineHeight: 1,
};

/* ── Page ── */

export default function DashboardPage() {
  const [summary, setSummary] = useState<ProgressSummary | null>(null);
  const [curve, setCurve] = useState<VocabCurveData | null>(null);
  const [goals, setGoals] = useState<Goal[]>([]);
  const [errors, setErrors] = useState<ErrorCategory[]>([]);
  const [sessions, setSessions] = useState<SessionItem[]>([]);
  const [loading, setLoading] = useState(true);

  const proficiency = useSettingsStore((s) => s.proficiency_level);

  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const [s, c, g, e, sess] = await Promise.all([
          api.get<ProgressSummary>("/progress/summary"),
          api.get<VocabCurveData>("/progress/vocabulary/curve"),
          api.get<Goal[]>("/progress/goals"),
          api.get<ErrorCategory[]>("/progress/errors"),
          api.get<SessionItem[]>("/progress/sessions?limit=8"),
        ]);

        if (cancelled) return;
        setSummary(s);
        setCurve(c);
        setGoals(g ?? []);
        setErrors(e ?? []);
        setSessions(sess ?? []);
      } catch {
        /* fail silently — cards stay empty */
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    load();
    return () => { cancelled = true; };
  }, []);

  /* ── Loading state ── */

  if (loading) {
    return (
      <div style={{ flex: 1, display: "flex", flexDirection: "column", alignItems: "center", justifyContent: "center", gap: 8 }}>
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
        <span style={{ fontSize: 12, color: "var(--text3)" }}>загрузка аналитики...</span>
        {/* spin keyframes defined in globals.css */}
      </div>
    );
  }

  /* ── Derived values ── */

  const accuracy = summary?.accuracy ?? 0;
  const correct = summary?.correct_rounds ?? 0;
  const total = summary?.total_rounds ?? 0;
  const streak = summary?.streak ?? 0;
  const totalWords = curve?.current.total ?? summary?.total_words ?? 0;
  const vocabGoal = curve?.goal ?? 3000;
  const confident = curve?.current.confident ?? 0;
  const uncertain = curve?.current.uncertain ?? 0;

  const confPct = totalWords > 0 ? (confident / totalWords) * 100 : 0;
  const uncPct = totalWords > 0 ? (uncertain / totalWords) * 100 : 0;

  const snapshots = (curve?.daily_snapshots ?? []).map((s) => ({
    date: s.date,
    total: s.total,
  }));

  /* Build last-rounds array for MiniChart from sessions */
  const lastRounds = sessions.flatMap((s) => {
    const acc = s.round_count > 0 ? s.correct_count / s.round_count : 0;
    return { correct: acc >= 0.5, mode: (MODE_LABELS[s.mode] ?? s.mode).charAt(0) };
  }).slice(0, 8);

  /* Aggregate mode counts from sessions */
  const modeCounts: Record<string, number> = {};
  for (const s of sessions) {
    modeCounts[s.mode] = (modeCounts[s.mode] ?? 0) + s.round_count;
  }

  /* Max error count for bar width */
  const maxErr = Math.max(1, ...errors.map((e) => e.count));

  return (
    <div style={{ flex: 1, padding: 16, overflow: "auto" }}>
      <div style={{ display: "flex", flexDirection: "column", gap: 12, maxWidth: 720 }}>

        {/* ── Row 1: 3 columns ── */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr 1fr", gap: 12 }}>

          {/* 1. Accuracy card */}
          <div style={card}>
            <div style={label}>{"// "}ТОЧНОСТЬ</div>
            <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
              <AccuracyRing accuracy={accuracy} />
              <div style={{ display: "flex", flexDirection: "column", gap: 4, fontSize: 12 }}>
                <span style={{ color: "var(--text2)" }}>
                  верных: <span style={{ color: "var(--green)" }}>{correct}</span>
                </span>
                <span style={{ color: "var(--text2)" }}>
                  всего: <span style={{ color: "var(--text)" }}>{total}</span>
                </span>
                <span style={{ color: "var(--text2)" }}>
                  streak: <span style={{ color: "var(--amber)" }}>{streak}</span>
                </span>
              </div>
            </div>
          </div>

          {/* 2. Vocabulary card */}
          <div style={card}>
            <div style={label}>{"// "}СЛОВАРЬ</div>
            <div style={bigNum}>{totalWords}</div>
            <div style={{ fontSize: 11, color: "var(--text3)", marginTop: 4, marginBottom: 8 }}>
              из {vocabGoal} (цель {proficiency.toUpperCase()})
            </div>
            <div style={{ display: "flex", height: 6, borderRadius: 3, overflow: "hidden" }}>
              <div style={{ width: `${confPct}%`, background: "var(--green)" }} />
              <div style={{ width: `${uncPct}%`, background: "var(--amber)" }} />
              <div style={{ flex: 1, background: "var(--bg4)" }} />
            </div>
          </div>

          {/* 3. Streak card */}
          <div style={card}>
            <div style={label}>{"// "}СЕРИЯ</div>
            <div style={bigNum}>{streak}</div>
            <div style={{ fontSize: 11, color: "var(--text3)", marginTop: 4 }}>
              макс: {streak}
            </div>
          </div>
        </div>

        {/* ── Row 2: 2 columns ── */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>

          {/* 4. Last rounds chart */}
          <div style={card}>
            <div style={label}>{"// "}ПОСЛЕДНИЕ РАУНДЫ</div>
            <MiniChart rounds={lastRounds} />
          </div>

          {/* 5. By-mode stats */}
          <div style={card}>
            <div style={label}>{"// "}ПО РЕЖИМАМ</div>
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {Object.entries(modeCounts).map(([mode, count]) => (
                <div
                  key={mode}
                  style={{ display: "flex", justifyContent: "space-between", fontSize: 12 }}
                >
                  <span style={{ color: "var(--text2)" }}>{MODE_LABELS[mode] ?? mode}</span>
                  <span style={{ color: "var(--text)", fontFamily: "var(--font-mono)" }}>{count}</span>
                </div>
              ))}
              {Object.keys(modeCounts).length === 0 && (
                <span style={{ fontSize: 11, color: "var(--text3)" }}>нет данных</span>
              )}
            </div>
          </div>
        </div>

        {/* ── Row 3: 2 columns ── */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>

          {/* 6. Vocab Curve */}
          <div style={card}>
            <div style={label}>{"// "}РОСТ СЛОВАРЯ</div>
            {snapshots.length > 0 ? (
              <VocabCurve snapshots={snapshots} goal={vocabGoal} />
            ) : (
              <span style={{ fontSize: 11, color: "var(--text3)" }}>нет данных</span>
            )}
          </div>

          {/* 7. Vocab Donut */}
          <div style={card}>
            <div style={label}>{"// "}РАСПРЕДЕЛЕНИЕ</div>
            <div style={{ display: "flex", justifyContent: "center" }}>
              <VocabDonut
                confident={confident}
                uncertain={uncertain}
                unknown={curve?.current.unknown ?? 0}
              />
            </div>
          </div>
        </div>

        {/* ── Row 4: 2 columns ── */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 12 }}>

          {/* 8. Errors */}
          <div style={card}>
            <div style={label}>{"// "}ОШИБКИ</div>
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {errors.map((e) => (
                <div key={e.error_type}>
                  <div
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      fontSize: 12,
                      marginBottom: 2,
                    }}
                  >
                    <span style={{ color: "var(--text2)" }}>
                      {ERROR_LABELS[e.error_type] ?? e.error_type}
                    </span>
                    <span style={{ color: "var(--text)", fontFamily: "var(--font-mono)" }}>
                      {e.count}
                    </span>
                  </div>
                  <div
                    style={{
                      height: 4,
                      borderRadius: 2,
                      background: "var(--bg4)",
                      overflow: "hidden",
                    }}
                  >
                    <div
                      style={{
                        width: `${(e.count / maxErr) * 100}%`,
                        height: "100%",
                        background: "var(--red)",
                        borderRadius: 2,
                        transition: "width 0.4s ease",
                      }}
                    />
                  </div>
                </div>
              ))}
              {errors.length === 0 && (
                <span style={{ fontSize: 11, color: "var(--text3)" }}>нет ошибок</span>
              )}
            </div>
          </div>

          {/* 9. Goals */}
          <div style={card}>
            <div style={label}>{"// "}ЦЕЛИ</div>
            <div style={{ display: "flex", flexDirection: "column", gap: 6 }}>
              {goals.map((g) => {
                const colorVar =
                  g.color === "green"
                    ? "var(--green)"
                    : g.color === "amber"
                      ? "var(--amber)"
                      : "var(--red)";
                return (
                  <div key={g.id}>
                    <div
                      style={{
                        display: "flex",
                        justifyContent: "space-between",
                        fontSize: 12,
                        marginBottom: 2,
                      }}
                    >
                      <span style={{ color: "var(--text2)" }}>{g.name}</span>
                      <span style={{ color: "var(--text)", fontFamily: "var(--font-mono)" }}>
                        {g.progress}%
                      </span>
                    </div>
                    <div
                      style={{
                        height: 4,
                        borderRadius: 2,
                        background: "var(--bg4)",
                        overflow: "hidden",
                      }}
                    >
                      <div
                        style={{
                          width: `${Math.min(100, g.progress)}%`,
                          height: "100%",
                          background: colorVar,
                          borderRadius: 2,
                          transition: "width 0.4s ease",
                        }}
                      />
                    </div>
                  </div>
                );
              })}
              {goals.length === 0 && (
                <span style={{ fontSize: 11, color: "var(--text3)" }}>нет целей</span>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

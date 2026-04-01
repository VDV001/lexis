"use client";

import { useState, useEffect, useCallback } from "react";
import { useSettingsStore } from "@/lib/stores/settings";
import type { ProficiencyLevel, VocabularyType } from "@/types";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

/* ── Language options ── */
const languages = [
  { id: "en", name: "English", sub: "Английский", flag: "\u{1F1EC}\u{1F1E7}", locked: false },
  { id: "de", name: "Deutsch", sub: "Немецкий", flag: "\u{1F1E9}\u{1F1EA}", locked: true },
  { id: "fr", name: "Fran\u00e7ais", sub: "Французский", flag: "\u{1F1EB}\u{1F1F7}", locked: true },
  { id: "ja", name: "\u65E5\u672C\u8A9E", sub: "Японский", flag: "\u{1F1EF}\u{1F1F5}", locked: true },
  { id: "es", name: "Espa\u00f1ol", sub: "Испанский", flag: "\u{1F1EA}\u{1F1F8}", locked: true },
  { id: "zh", name: "\u4E2D\u6587", sub: "Китайский", flag: "\u{1F1E8}\u{1F1F3}", locked: true },
] as const;

/* ── Level options ── */
const levels: { id: ProficiencyLevel; code: string; name: string; locked: boolean }[] = [
  { id: "a2", code: "A2", name: "Элементарный", locked: true },
  { id: "b1", code: "B1", name: "Средний", locked: false },
  { id: "b2", code: "B2", name: "Выше среднего", locked: false },
  { id: "c1", code: "C1", name: "Продвинутый", locked: true },
];

/* ── Vocabulary type options ── */
const vocabTypes: { id: VocabularyType; icon: string; name: string; sub: string; locked: boolean }[] = [
  { id: "tech", icon: "\u2699\uFE0F", name: "Технический", sub: "dev / IT / infra", locked: false },
  { id: "literary", icon: "\uD83D\uDCD6", name: "Литературный", sub: "soon", locked: true },
  { id: "business", icon: "\uD83D\uDCBC", name: "Деловой", sub: "soon", locked: true },
];

/* ── AI Model options ── */
interface ModelOption {
  id: string;
  label: string;
  providerIcon: string;
  providerColor: string;
  locked: boolean;
}

const aiModels: ModelOption[] = [
  { id: "claude-sonnet-4-20250514", label: "Claude Sonnet", providerIcon: "A", providerColor: "var(--green)", locked: false },
  { id: "claude-haiku-4-20250514", label: "Claude Haiku", providerIcon: "A", providerColor: "var(--green)", locked: false },
  { id: "qwen-plus", label: "Qwen Plus", providerIcon: "Q", providerColor: "var(--amber)", locked: false },
  { id: "gpt-4o", label: "GPT-4o", providerIcon: "G", providerColor: "var(--cyan)", locked: false },
  { id: "gpt-4o-mini", label: "GPT-4o Mini", providerIcon: "G", providerColor: "var(--cyan)", locked: false },
  { id: "gemini-2.0-flash", label: "Gemini Flash", providerIcon: "\u2726", providerColor: "var(--purple)", locked: false },
];

export default function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const store = useSettingsStore();

  const [lang, setLang] = useState(store.target_language);
  const [level, setLevel] = useState(store.proficiency_level);
  const [vocabType, setVocabType] = useState(store.vocabulary_type);
  const [model, setModel] = useState(store.ai_model);

  /* Sync local state when store changes (e.g. after hydrate) */
  useEffect(() => {
    setLang(store.target_language);
    setLevel(store.proficiency_level);
    setVocabType(store.vocabulary_type);
    setModel(store.ai_model);
  }, [store.target_language, store.proficiency_level, store.vocabulary_type, store.ai_model]);

  const handleApply = useCallback(async () => {
    await store.updateSettings({
      target_language: lang,
      proficiency_level: level,
      vocabulary_type: vocabType,
      ai_model: model,
    });
    onClose();
  }, [lang, level, vocabType, model, store, onClose]);

  if (!isOpen) return null;

  return (
    <div
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.7)",
        zIndex: 100,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
      onClick={onClose}
    >
      <div
        style={{
          width: 520,
          background: "var(--bg2)",
          border: "1px solid var(--border)",
          borderRadius: 4,
          maxHeight: "80vh",
          overflowY: "auto",
        }}
        onClick={(e) => e.stopPropagation()}
      >
        {/* ── Header ── */}
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "16px 20px",
            borderBottom: "1px solid var(--border)",
          }}
        >
          <div style={{ fontSize: 13, fontWeight: 600, color: "var(--text)" }}>
            <span style={{ color: "var(--green)", marginRight: 6 }}>{">"}</span>
            Настройки обучения
          </div>
          <button
            onClick={onClose}
            style={{
              background: "none",
              border: "none",
              color: "var(--text3)",
              fontSize: 11,
              cursor: "pointer",
              fontFamily: "var(--font-mono)",
            }}
          >
            [ закрыть ]
          </button>
        </div>

        {/* ── Body ── */}
        <div style={{ padding: "16px 20px" }}>
          {/* ── Group 1: Language ── */}
          <SettingGroup
            title="Язык обучения"
            desc="Выбери язык, который хочешь изучать. Сейчас доступен только английский."
          >
            <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 8 }}>
              {languages.map((l) => {
                const active = lang === l.id && !l.locked;
                return (
                  <div
                    key={l.id}
                    onClick={() => !l.locked && setLang(l.id)}
                    style={{
                      position: "relative",
                      padding: "12px 10px",
                      border: `1px solid ${active ? "var(--green)" : "var(--border)"}`,
                      borderRadius: 4,
                      background: active ? "rgba(63,185,80,0.05)" : "var(--bg3)",
                      textAlign: "center",
                      cursor: l.locked ? "not-allowed" : "pointer",
                      opacity: l.locked ? 0.5 : 1,
                    }}
                  >
                    <div style={{ fontSize: 22, marginBottom: 4 }}>{l.flag}</div>
                    <div style={{ fontSize: 12, fontWeight: 500, color: "var(--text)" }}>{l.name}</div>
                    <div style={{ fontSize: 10, color: "var(--text3)", marginTop: 2 }}>{l.sub}</div>
                    {l.locked && (
                      <span
                        style={{
                          position: "absolute",
                          top: 6,
                          right: 6,
                          fontSize: 9,
                          color: "var(--text3)",
                          background: "var(--bg4)",
                          padding: "1px 5px",
                          borderRadius: 3,
                        }}
                      >
                        soon
                      </span>
                    )}
                  </div>
                );
              })}
            </div>
          </SettingGroup>

          {/* ── Group 2: Level ── */}
          <SettingGroup
            title="Уровень владения"
            desc="Выбери свой уровень. Это влияет на сложность упражнений и объяснений."
          >
            <div style={{ display: "flex", gap: 8 }}>
              {levels.map((lv) => {
                const active = level === lv.id && !lv.locked;
                return (
                  <button
                    key={lv.id}
                    onClick={() => !lv.locked && setLevel(lv.id)}
                    style={{
                      flex: 1,
                      padding: "10px 8px",
                      border: `1px solid ${active ? "var(--cyan)" : "var(--border)"}`,
                      borderRadius: 4,
                      background: active ? "rgba(88,166,255,0.07)" : "var(--bg3)",
                      color: active ? "var(--cyan)" : "var(--text2)",
                      cursor: lv.locked ? "not-allowed" : "pointer",
                      opacity: lv.locked ? 0.4 : 1,
                      fontFamily: "var(--font-mono)",
                      textAlign: "center",
                    }}
                  >
                    <div style={{ fontSize: 13, fontWeight: 600 }}>{lv.code}</div>
                    <div style={{ fontSize: 10, marginTop: 2 }}>{lv.name}</div>
                  </button>
                );
              })}
            </div>
          </SettingGroup>

          {/* ── Group 3: Vocabulary type ── */}
          <SettingGroup
            title="Тип словарного запаса"
            desc="Технический фокус — терминология разработки, IT, инфраструктура. Литературный — пока в разработке."
          >
            <div style={{ display: "flex", gap: 8 }}>
              {vocabTypes.map((vt) => {
                const active = vocabType === vt.id && !vt.locked;
                return (
                  <button
                    key={vt.id}
                    onClick={() => !vt.locked && setVocabType(vt.id)}
                    style={{
                      flex: 1,
                      padding: "12px 10px",
                      border: `1px solid ${active ? "var(--amber)" : "var(--border)"}`,
                      borderRadius: 4,
                      background: active ? "rgba(227,179,65,0.07)" : "var(--bg3)",
                      color: active ? "var(--amber)" : "var(--text2)",
                      cursor: vt.locked ? "not-allowed" : "pointer",
                      opacity: vt.locked ? 0.5 : 1,
                      fontFamily: "var(--font-mono)",
                      textAlign: "center",
                    }}
                  >
                    <div style={{ fontSize: 18, marginBottom: 4 }}>{vt.icon}</div>
                    <div style={{ fontSize: 12, fontWeight: 500 }}>{vt.name}</div>
                    <div style={{ fontSize: 10, marginTop: 2, color: vt.locked ? "var(--text3)" : undefined }}>
                      {vt.sub}
                    </div>
                  </button>
                );
              })}
            </div>
          </SettingGroup>

          {/* ── Group 4: AI Model ── */}
          <SettingGroup
            title="AI Модель"
            desc="Выбери модель для генерации упражнений и обратной связи."
          >
            <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 8 }}>
              {aiModels.map((m) => {
                const active = model === m.id;
                return (
                  <div
                    key={m.id}
                    onClick={() => !m.locked && setModel(m.id)}
                    style={{
                      display: "flex",
                      alignItems: "center",
                      gap: 8,
                      padding: "10px 10px",
                      border: `1px solid ${active ? "var(--green)" : "var(--border)"}`,
                      borderRadius: 4,
                      background: active ? "rgba(63,185,80,0.05)" : "var(--bg3)",
                      cursor: m.locked ? "not-allowed" : "pointer",
                      opacity: m.locked ? 0.5 : 1,
                    }}
                  >
                    <div
                      style={{
                        width: 24,
                        height: 24,
                        borderRadius: "50%",
                        background: "var(--bg)",
                        border: `1px solid ${m.providerColor}`,
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: 11,
                        fontWeight: 700,
                        color: m.providerColor,
                        flexShrink: 0,
                        fontFamily: "var(--font-mono)",
                      }}
                    >
                      {m.providerIcon}
                    </div>
                    <div style={{ fontSize: 11, fontWeight: 500, color: active ? "var(--text)" : "var(--text2)" }}>
                      {m.label}
                    </div>
                  </div>
                );
              })}
            </div>
          </SettingGroup>

          {/* ── Apply button ── */}
          <button
            onClick={handleApply}
            style={{
              width: "100%",
              padding: "12px 0",
              marginTop: 8,
              background: "none",
              border: "1px solid var(--green)",
              borderRadius: 4,
              color: "var(--green)",
              fontSize: 12,
              fontWeight: 500,
              cursor: "pointer",
              fontFamily: "var(--font-mono)",
            }}
          >
            [ применить настройки ]
          </button>
        </div>
      </div>
    </div>
  );
}

/* ── Reusable setting group ── */
function SettingGroup({
  title,
  desc,
  children,
}: {
  title: string;
  desc: string;
  children: React.ReactNode;
}) {
  return (
    <div style={{ marginBottom: 20 }}>
      <div
        style={{
          fontSize: 10,
          color: "var(--text3)",
          textTransform: "uppercase",
          letterSpacing: "0.8px",
          marginBottom: 6,
          fontFamily: "var(--font-mono)",
        }}
      >
        {"// "}
        {title}
      </div>
      <div style={{ fontSize: 11, color: "var(--text2)", marginBottom: 10 }}>{desc}</div>
      {children}
    </div>
  );
}

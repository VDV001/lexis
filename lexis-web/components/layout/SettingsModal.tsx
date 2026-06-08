"use client";

import { useCallback, useEffect, useMemo, useReducer } from "react";
import { useSettingsStore } from "@/lib/stores/settings";
import { useOpenRouterModels } from "@/lib/hooks/useOpenRouterModels";
import type { CatalogModel, ProficiencyLevel, VocabularyType } from "@/types";

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

/* ── Language options ── */
const languages = [
  { id: "en", name: "English", sub: "Английский", flag: "\u{1F1EC}\u{1F1E7}", locked: false },
  { id: "de", name: "Deutsch", sub: "Немецкий", flag: "\u{1F1E9}\u{1F1EA}", locked: true },
  { id: "fr", name: "Français", sub: "Французский", flag: "\u{1F1EB}\u{1F1F7}", locked: true },
  { id: "ja", name: "日本語", sub: "Японский", flag: "\u{1F1EF}\u{1F1F5}", locked: true },
  { id: "es", name: "Español", sub: "Испанский", flag: "\u{1F1EA}\u{1F1F8}", locked: true },
  { id: "zh", name: "中文", sub: "Китайский", flag: "\u{1F1E8}\u{1F1F3}", locked: true },
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
  { id: "tech", icon: "⚙️", name: "Технический", sub: "dev / IT / infra", locked: false },
  { id: "literary", icon: "📖", name: "Литературный", sub: "soon", locked: true },
  { id: "business", icon: "💼", name: "Деловой", sub: "soon", locked: true },
];

/* ── AI Model provider styling ──
   Models are loaded dynamically from the OpenRouter catalogue
   (GET /ai/models/openrouter); this only maps a provider to its badge. */
const providerStyle: Record<string, { icon: string; color: string }> = {
  openai: { icon: "G", color: "var(--cyan)" },
  anthropic: { icon: "A", color: "var(--green)" },
  google: { icon: "✦", color: "var(--purple)" },
  deepseek: { icon: "D", color: "var(--amber)" },
  qwen: { icon: "Q", color: "var(--amber)" },
  "meta-llama": { icon: "M", color: "var(--cyan)" },
  mistralai: { icon: "M", color: "var(--amber)" },
  "x-ai": { icon: "X", color: "var(--text2)" },
  cohere: { icon: "C", color: "var(--purple)" },
};

function styleForProvider(provider: string): { icon: string; color: string } {
  return providerStyle[provider] ?? { icon: "?", color: "var(--text3)" };
}

export default function SettingsModal({ isOpen, onClose }: SettingsModalProps) {
  const store = useSettingsStore();
  const { models: catalog, loading: modelsLoading } = useOpenRouterModels(isOpen);

  type Draft = { lang: string; level: ProficiencyLevel; vocabType: VocabularyType; model: string };
  type Action =
    | { type: "set_lang"; value: string }
    | { type: "set_level"; value: ProficiencyLevel }
    | { type: "set_vocab"; value: VocabularyType }
    | { type: "set_model"; value: string }
    | { type: "reset"; payload: Draft };

  const [draft, dispatch] = useReducer(
    (state: Draft, action: Action): Draft => {
      switch (action.type) {
        case "set_lang": return { ...state, lang: action.value };
        case "set_level": return { ...state, level: action.value };
        case "set_vocab": return { ...state, vocabType: action.value };
        case "set_model": return { ...state, model: action.value };
        case "reset": return action.payload;
      }
    },
    { lang: store.target_language, level: store.proficiency_level, vocabType: store.vocabulary_type, model: store.ai_model },
  );

  // Reset draft to current store values when modal opens
  useEffect(() => {
    if (isOpen) {
      dispatch({ type: "reset", payload: {
        lang: store.target_language,
        level: store.proficiency_level,
        vocabType: store.vocabulary_type,
        model: store.ai_model,
      }});
    }
  }, [isOpen, store.target_language, store.proficiency_level, store.vocabulary_type, store.ai_model]);

  const { lang, level, vocabType, model } = draft;

  // The selectable list is the live catalogue, plus the currently-saved model
  // pinned to the top so it stays visible/selectable even if it is not (or no
  // longer) in the catalogue.
  const displayModels = useMemo<CatalogModel[]>(() => {
    const list = [...catalog];
    if (model && !list.some((m) => m.id === model)) {
      const provider = model.includes("/") ? model.split("/")[0] : "";
      list.unshift({ id: model, name: model, provider, description: "" });
    }
    return list;
  }, [catalog, model]);

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
                    onClick={() => !l.locked && dispatch({ type: "set_lang", value: l.id })}
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
                    onClick={() => !lv.locked && dispatch({ type: "set_level", value: lv.id })}
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
                    onClick={() => !vt.locked && dispatch({ type: "set_vocab", value: vt.id })}
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
            desc="Модели подгружаются из каталога OpenRouter. Выбранная используется для чата и упражнений."
          >
            {modelsLoading && displayModels.length === 0 ? (
              <div style={{ fontSize: 11, color: "var(--text3)", fontFamily: "var(--font-mono)", padding: "8px 0" }}>
                загрузка моделей…
              </div>
            ) : (
              <div style={{ display: "grid", gridTemplateColumns: "repeat(3, 1fr)", gap: 8 }}>
                {displayModels.map((m) => {
                  const active = model === m.id;
                  const ps = styleForProvider(m.provider);
                  return (
                    <div
                      key={m.id}
                      title={m.description || m.id}
                      onClick={() => dispatch({ type: "set_model", value: m.id })}
                      style={{
                        display: "flex",
                        alignItems: "center",
                        gap: 8,
                        padding: "10px 10px",
                        border: `1px solid ${active ? "var(--green)" : "var(--border)"}`,
                        borderRadius: 4,
                        background: active ? "rgba(63,185,80,0.05)" : "var(--bg3)",
                        cursor: "pointer",
                      }}
                    >
                      <div
                        style={{
                          width: 24,
                          height: 24,
                          borderRadius: "50%",
                          background: "var(--bg)",
                          border: `1px solid ${ps.color}`,
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          fontSize: 11,
                          fontWeight: 700,
                          color: ps.color,
                          flexShrink: 0,
                          fontFamily: "var(--font-mono)",
                        }}
                      >
                        {ps.icon}
                      </div>
                      <div
                        style={{
                          fontSize: 11,
                          fontWeight: 500,
                          color: active ? "var(--text)" : "var(--text2)",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                          whiteSpace: "nowrap",
                        }}
                      >
                        {m.name || m.id}
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
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

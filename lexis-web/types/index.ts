// Core domain types derived from spec sections 6 + 7

export type ProficiencyLevel = "a2" | "b1" | "b2" | "c1";
export type VocabularyType = "tech" | "literary" | "business";
export type LearningMode = "chat" | "quiz" | "translate" | "gap" | "scramble";
export type VocabStatus = "unknown" | "uncertain" | "confident";
export type ErrorType =
  | "articles"
  | "tenses"
  | "prepositions"
  | "phrasal"
  | "vocabulary"
  | "word_order";
export type FeedbackType = "good" | "note" | "error";
export type GoalColor = "green" | "amber" | "red";

export interface AIModel {
  id: string;
  display_name: string;
  provider: string;
  icon: string;
  description: string;
  available: boolean;
}

export interface UserSettings {
  target_language: string;
  proficiency_level: ProficiencyLevel;
  vocabulary_type: VocabularyType;
  ai_model: string;
  vocab_goal: number;
  ui_language: string;
}

export interface User {
  id: string;
  email: string;
  display_name: string;
  avatar_url: string | null;
}

export interface Goal {
  id: string;
  name: string;
  language: string;
  progress: number;
  color: GoalColor;
  is_system: boolean;
}

export interface VocabWord {
  id: string;
  word: string;
  language: string;
  status: VocabStatus;
  context: string | null;
  last_seen: string;
}

export interface VocabSnapshot {
  date: string;
  total: number;
  confident: number;
  uncertain: number;
  unknown: number;
}

export interface VocabCurveData {
  goal: number;
  current: {
    total: number;
    confident: number;
    uncertain: number;
    unknown: number;
  };
  daily_snapshots: VocabSnapshot[];
}

export interface ChatCorrection {
  original: string;
  fixed: string;
  explanation: string;
}

export interface ChatFeedback {
  type: FeedbackType;
  text: string;
}

export interface ChatResponse {
  reply: string;
  correction: ChatCorrection | null;
  feedback: ChatFeedback;
  error_type: ErrorType | null;
  new_words: string[];
}

export interface QuizQuestion {
  type: string;
  question: string;
  options: string[];
  correct: number;
  explanation: string;
  error_type: ErrorType;
  words: string[];
  confidence: number;
}

export interface ProgressSummary {
  total_rounds: number;
  correct_rounds: number;
  accuracy: number;
  streak: number;
  total_words: number;
}

// SSE event types (spec 7.3)
export type SSEEventType = "delta" | "correction" | "feedback" | "words" | "done";

export interface SSEEvent {
  type: SSEEventType;
  content?: string;
  correction?: ChatCorrection;
  feedback?: ChatFeedback;
  words?: string[];
}

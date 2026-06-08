// Single source of truth for AI-provider badge styling, shared by the settings
// model selector and the header model chip. Keep provider keys aligned with the
// backend's curated provider list (lexis-api .../tutor/usecase/catalog.go).

export interface ProviderBadge {
  icon: string;
  color: string;
}

export const providerStyle: Record<string, ProviderBadge> = {
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

export function styleForProvider(provider: string): ProviderBadge {
  return providerStyle[provider] ?? { icon: "?", color: "var(--text3)" };
}

import { useEffect, useState } from "react";
import api from "@/lib/api";
import type { CatalogModel } from "@/types";

interface OpenRouterModelsState {
  models: CatalogModel[];
  loading: boolean;
  error: boolean;
}

/**
 * useOpenRouterModels loads the dynamic OpenRouter model catalogue when
 * `enabled` becomes true (e.g. when the settings modal opens). The backend
 * always returns a usable list — a live catalogue when reachable, otherwise an
 * embedded fallback shortlist — so `models` is non-empty on success.
 */
export function useOpenRouterModels(enabled: boolean): OpenRouterModelsState {
  const [state, setState] = useState<OpenRouterModelsState>({
    models: [],
    loading: false,
    error: false,
  });

  useEffect(() => {
    if (!enabled) return;

    let cancelled = false;

    async function load() {
      setState((s) => ({ ...s, loading: true, error: false }));
      try {
        const res = await api.get<{ models: CatalogModel[] }>("/ai/models/openrouter");
        if (!cancelled) {
          setState({ models: res?.models ?? [], loading: false, error: false });
        }
      } catch {
        if (!cancelled) {
          setState({ models: [], loading: false, error: true });
        }
      }
    }

    void load();

    return () => {
      cancelled = true;
    };
  }, [enabled]);

  return state;
}

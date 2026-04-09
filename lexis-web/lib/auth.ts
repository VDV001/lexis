const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

let isRefreshing = false;
let refreshPromise: Promise<boolean> | null = null;

/**
 * Attempts to refresh the access token using the HTTP-only refresh cookie.
 * Deduplicates concurrent calls — only one refresh request at a time.
 */
export async function tryRefreshToken(): Promise<boolean> {
  if (isRefreshing && refreshPromise) return refreshPromise;

  isRefreshing = true;
  refreshPromise = (async () => {
    try {
      const res = await fetch(`${API_URL}/auth/refresh`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({}),
      });

      if (!res.ok) return false;

      const data = await res.json();
      if (data.access_token) {
        sessionStorage.setItem("access_token", data.access_token);
        return true;
      }
      return false;
    } catch {
      return false;
    } finally {
      isRefreshing = false;
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

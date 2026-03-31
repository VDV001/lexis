"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import api, { ApiError } from "@/lib/api";
import { useSessionStore } from "@/lib/stores/session";

interface AuthResponse {
  user: { id: string; email: string; display_name: string; avatar_url: string | null };
  access_token: string;
  refresh_token: string;
}

export default function RegisterPage() {
  const router = useRouter();
  const setAuth = useSessionStore((s) => s.setAuth);
  const [displayName, setDisplayName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      const res = await api.post<AuthResponse>("/auth/register", {
        email,
        password,
        display_name: displayName,
      });
      setAuth(res.user, res.access_token);
      router.push("/chat");
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.detail);
      } else {
        setError("Ошибка соединения");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <div className="text-[10px] text-[var(--text3)] uppercase tracking-[0.8px] mb-3">
        {'// '}Регистрация
      </div>

      {error && (
        <div className="mb-4 p-3 text-[12px] text-[var(--red)] bg-[rgba(248,81,73,0.06)] border border-[rgba(248,81,73,0.2)] rounded-[3px]">
          {error}
        </div>
      )}

      <div className="mb-3">
        <label className="block text-[11px] text-[var(--text2)] mb-1">Имя</label>
        <div className="relative">
          <span className="absolute left-[11px] top-1/2 -translate-y-1/2 text-[var(--green)] text-[12px]">{'>'}</span>
          <input
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            required
            minLength={2}
            maxLength={100}
            className="w-full bg-[var(--bg3)] border border-[var(--border)] rounded-[2px] py-[9px] px-[12px] pl-[24px] text-[13px] text-[var(--text)] outline-none focus:border-[var(--border2)] font-[family-name:var(--font-mono)]"
            placeholder="Daniil"
          />
        </div>
      </div>

      <div className="mb-3">
        <label className="block text-[11px] text-[var(--text2)] mb-1">Email</label>
        <div className="relative">
          <span className="absolute left-[11px] top-1/2 -translate-y-1/2 text-[var(--green)] text-[12px]">{'>'}</span>
          <input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
            className="w-full bg-[var(--bg3)] border border-[var(--border)] rounded-[2px] py-[9px] px-[12px] pl-[24px] text-[13px] text-[var(--text)] outline-none focus:border-[var(--border2)] font-[family-name:var(--font-mono)]"
            placeholder="email@example.com"
          />
        </div>
      </div>

      <div className="mb-4">
        <label className="block text-[11px] text-[var(--text2)] mb-1">Пароль</label>
        <div className="relative">
          <span className="absolute left-[11px] top-1/2 -translate-y-1/2 text-[var(--green)] text-[12px]">{'>'}</span>
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            required
            minLength={8}
            className="w-full bg-[var(--bg3)] border border-[var(--border)] rounded-[2px] py-[9px] px-[12px] pl-[24px] text-[13px] text-[var(--text)] outline-none focus:border-[var(--border2)] font-[family-name:var(--font-mono)]"
            placeholder="min 8 символов"
          />
        </div>
      </div>

      <button
        type="submit"
        disabled={loading}
        className="w-full py-[10px] bg-transparent border border-[var(--green)] rounded-[3px] text-[12.5px] text-[var(--green)] cursor-pointer transition-all hover:bg-[rgba(63,185,80,0.08)] disabled:opacity-30 disabled:cursor-not-allowed font-[family-name:var(--font-mono)]"
      >
        {loading ? "..." : "[ создать аккаунт ]"}
      </button>

      <p className="mt-4 text-center text-[11px] text-[var(--text3)]">
        Уже есть аккаунт?{" "}
        <Link href="/login" className="text-[var(--cyan)] hover:underline">
          Войти
        </Link>
      </p>
    </form>
  );
}

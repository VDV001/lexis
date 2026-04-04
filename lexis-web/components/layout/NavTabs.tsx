"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";

const tabs = [
  { label: "Практика", path: "/chat" },
  { label: "Квиз", path: "/quiz" },
  { label: "Перевод", path: "/translate" },
  { label: "Пропуск", path: "/gap" },
  { label: "Слова", path: "/scramble" },
  { label: "Словарь", path: "/vocabulary" },
  { label: "Аналитика", path: "/dashboard" },
];

export default function NavTabs() {
  const pathname = usePathname();

  return (
    <nav className="flex">
      {tabs.map((tab) => {
        const isActive = pathname === tab.path;
        return (
          <Link
            key={tab.path}
            href={tab.path}
            className={`relative top-[1px] no-underline cursor-pointer transition-all duration-150 whitespace-nowrap flex items-center gap-[5px] font-[family-name:var(--font-mono)] hover:text-[var(--text)] hover:bg-[rgba(255,255,255,0.02)] ${isActive ? "!text-[var(--cyan)] !bg-transparent" : ""}`}
            style={{
              padding: "0 14px",
              height: "52px",
              fontSize: "12px",
              color: isActive ? "var(--cyan)" : "var(--text2)",
              borderBottom: isActive
                ? "2px solid var(--cyan)"
                : "2px solid transparent",
            }}
          >
            {tab.label}
          </Link>
        );
      })}
    </nav>
  );
}

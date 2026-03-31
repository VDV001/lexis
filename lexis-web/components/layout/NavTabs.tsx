"use client";

import { usePathname, useRouter } from "next/navigation";

const tabs = [
  { label: "Практика", path: "/chat" },
  { label: "Квиз", path: "/quiz" },
  { label: "Перевод", path: "/translate" },
  { label: "Пропуск", path: "/gap" },
  { label: "Слова", path: "/scramble" },
  { label: "Аналитика", path: "/dashboard" },
];

export default function NavTabs() {
  const pathname = usePathname();
  const router = useRouter();

  return (
    <nav className="flex">
      {tabs.map((tab) => {
        const isActive = pathname === tab.path;
        return (
          <button
            key={tab.path}
            onClick={() => router.push(tab.path)}
            className="relative top-[1px] border-none bg-transparent cursor-pointer transition-all duration-150 whitespace-nowrap flex items-center gap-[5px] font-[family-name:var(--font-mono)]"
            style={{
              padding: "0 14px",
              height: "52px",
              fontSize: "12px",
              color: isActive ? "var(--cyan)" : "var(--text2)",
              borderBottom: isActive
                ? "2px solid var(--cyan)"
                : "2px solid transparent",
            }}
            onMouseEnter={(e) => {
              if (!isActive) {
                e.currentTarget.style.color = "var(--text)";
                e.currentTarget.style.background = "rgba(255,255,255,0.02)";
              }
            }}
            onMouseLeave={(e) => {
              if (!isActive) {
                e.currentTarget.style.color = "var(--text2)";
                e.currentTarget.style.background = "transparent";
              }
            }}
          >
            {tab.label}
          </button>
        );
      })}
    </nav>
  );
}

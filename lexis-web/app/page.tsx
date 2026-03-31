import Link from "next/link";

export default function Home() {
  return (
    <div className="flex h-screen items-center justify-center">
      <div className="text-center">
        <h1 className="text-[17px] font-bold text-[var(--green)] tracking-[-0.5px]">
          lang.tutor
          <span className="inline-block w-[9px] h-[16px] bg-[var(--green)] ml-[2px] align-middle animate-blink" />
        </h1>
        <p className="text-[10.5px] text-[var(--text2)] mt-1">
          {'>'} AI-наставник для изучения языков
        </p>

        <div className="flex gap-3 mt-6 justify-center">
          <Link
            href="/login"
            className="py-[9px] px-[16px] bg-transparent border border-[var(--green)] rounded-[3px] text-[12px] text-[var(--green)] transition-all hover:bg-[rgba(63,185,80,0.08)] font-[family-name:var(--font-mono)] no-underline"
          >
            [ войти ]
          </Link>
          <Link
            href="/register"
            className="py-[9px] px-[16px] bg-transparent border border-[var(--border)] rounded-[3px] text-[12px] text-[var(--cyan)] transition-all hover:border-[var(--cyan)] font-[family-name:var(--font-mono)] no-underline"
          >
            [ регистрация ]
          </Link>
        </div>
      </div>
    </div>
  );
}

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
      </div>
    </div>
  );
}

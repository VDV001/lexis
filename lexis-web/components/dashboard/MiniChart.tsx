interface MiniChartProps {
  rounds: { correct: boolean; mode: string }[];
}

export default function MiniChart({ rounds }: MiniChartProps) {
  return (
    <div>
      {/* Bars */}
      <div
        style={{
          display: "flex",
          alignItems: "flex-end",
          height: 38,
          gap: 3,
        }}
      >
        {rounds.map((r, i) => (
          <div
            key={i}
            className="m-bar"
            style={{
              flex: 1,
              minHeight: 3,
              height: "100%",
              borderRadius: "1px 1px 0 0",
              background: r.correct ? "var(--green)" : "var(--red)",
              opacity: 0.75,
            }}
          />
        ))}
        {/* Fill remaining slots up to 8 with empty bars */}
        {Array.from({ length: Math.max(0, 8 - rounds.length) }).map((_, i) => (
          <div
            key={`empty-${i}`}
            style={{
              flex: 1,
              minHeight: 3,
              height: "100%",
              borderRadius: "1px 1px 0 0",
              background: "var(--bg4)",
            }}
          />
        ))}
      </div>

      {/* Labels */}
      <div style={{ display: "flex", gap: 3, marginTop: 2 }}>
        {rounds.map((r, i) => (
          <div
            key={i}
            style={{
              flex: 1,
              textAlign: "center",
              fontSize: 8,
              color: "var(--text3)",
            }}
          >
            {r.mode}
          </div>
        ))}
      </div>
    </div>
  );
}

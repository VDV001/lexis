interface VocabDonutProps {
  confident: number;
  uncertain: number;
  unknown: number;
}

const CIRCUMFERENCE = 150.8; // 2 * π * 24

interface Sector {
  label: string;
  value: number;
  color: string;
}

export default function VocabDonut({ confident, uncertain, unknown }: VocabDonutProps) {
  const total = confident + uncertain + unknown;

  const sectors: Sector[] = [
    { label: "Confident", value: confident, color: "var(--green)" },
    { label: "Uncertain", value: uncertain, color: "var(--amber)" },
    { label: "Unknown", value: unknown, color: "var(--bg4)" },
  ];

  // Build cumulative offsets for each sector
  let accumulated = 0;
  const rings = sectors.map((s) => {
    const fraction = total > 0 ? s.value / total : 0;
    const dash = CIRCUMFERENCE * fraction;
    const gap = CIRCUMFERENCE - dash;
    const offset = -accumulated; // negative = clockwise advance
    accumulated += dash;

    return { ...s, dash, gap, offset };
  });

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", gap: 8 }}>
      <svg width={60} height={60} viewBox="0 0 60 60">
        {rings.map((r) => (
          <circle
            key={r.label}
            cx={30}
            cy={30}
            r={24}
            fill="none"
            stroke={r.color}
            strokeWidth={5}
            strokeDasharray={`${r.dash} ${r.gap}`}
            strokeDashoffset={r.offset}
            transform="rotate(-90 30 30)"
          />
        ))}
        <text
          x={30}
          y={30}
          textAnchor="middle"
          dominantBaseline="central"
          fill="var(--text)"
          fontSize={18}
          fontWeight="bold"
          fontFamily="var(--font-mono)"
        >
          {total}
        </text>
      </svg>

      {/* Legend */}
      <div style={{ display: "flex", gap: 12 }}>
        {sectors.map((s) => (
          <div key={s.label} style={{ display: "flex", alignItems: "center", gap: 4 }}>
            <span
              style={{
                width: 6,
                height: 6,
                borderRadius: "50%",
                background: s.color,
                display: "inline-block",
              }}
            />
            <span style={{ fontSize: 10, color: "var(--text2)" }}>{s.label}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

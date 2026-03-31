interface AccuracyRingProps {
  accuracy: number; // 0-100
}

const CIRCUMFERENCE = 150.8; // 2 * π * 24

export default function AccuracyRing({ accuracy }: AccuracyRingProps) {
  const clamped = Math.min(100, Math.max(0, accuracy));
  const offset = CIRCUMFERENCE * (1 - clamped / 100);

  return (
    <svg width={60} height={60} viewBox="0 0 60 60">
      {/* Background ring */}
      <circle
        cx={30}
        cy={30}
        r={24}
        fill="none"
        stroke="var(--bg4)"
        strokeWidth={5}
      />
      {/* Fill arc */}
      <circle
        cx={30}
        cy={30}
        r={24}
        fill="none"
        stroke="var(--green)"
        strokeWidth={5}
        strokeLinecap="round"
        strokeDasharray={CIRCUMFERENCE}
        strokeDashoffset={offset}
        transform="rotate(-90 30 30)"
        style={{ transition: "stroke-dashoffset 0.8s ease" }}
      />
      {/* Center label */}
      <text
        x={30}
        y={30}
        textAnchor="middle"
        dominantBaseline="central"
        fill="var(--text)"
        fontSize={12}
        fontWeight="bold"
        fontFamily="var(--font-mono)"
      >
        {Math.round(clamped)}%
      </text>
    </svg>
  );
}

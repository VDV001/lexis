"use client";

import { LineChart, Line, XAxis, YAxis, ResponsiveContainer, ReferenceLine } from "recharts";

interface VocabCurveProps {
  snapshots: { date: string; total: number }[];
  goal: number;
}

function CustomDot(props: Record<string, unknown>) {
  const { cx, cy, index, dataLength } = props as {
    cx: number;
    cy: number;
    index: number;
    dataLength: number;
  };

  if (index !== dataLength - 1) return null;

  return (
    <circle
      cx={cx}
      cy={cy}
      r={9}
      fill="var(--cyan)"
      stroke="var(--bg)"
      strokeWidth={2}
    />
  );
}

export default function VocabCurve({ snapshots, goal }: VocabCurveProps) {
  const data = snapshots.map((s) => ({
    date: s.date,
    total: s.total,
  }));

  return (
    <ResponsiveContainer width="100%" height={180}>
      <LineChart data={data} margin={{ top: 8, right: 8, bottom: 0, left: -16 }}>
        <XAxis
          dataKey="date"
          tick={{ fill: "var(--text3)", fontSize: 10 }}
          axisLine={false}
          tickLine={false}
          tickFormatter={(v: string) => {
            const d = new Date(v);
            return `${d.getMonth() + 1}/${d.getDate()}`;
          }}
        />
        <YAxis
          tick={{ fill: "var(--text3)", fontSize: 10 }}
          axisLine={false}
          tickLine={false}
        />
        <ReferenceLine
          y={goal}
          stroke="var(--text3)"
          strokeDasharray="4 4"
        />
        <Line
          type="monotone"
          dataKey="total"
          stroke="var(--cyan)"
          strokeWidth={2}
          dot={(dotProps: Record<string, unknown>) => (
            <CustomDot
              key={String(dotProps.index)}
              {...dotProps}
              dataLength={data.length}
            />
          )}
          activeDot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

import { NextRequest } from "next/server";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export async function POST(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> }
) {
  const { path } = await params;
  const subPath = path.join("/");
  const body = await req.json();
  const token = req.headers.get("authorization");

  const res = await fetch(`${API_URL}/tutor/${subPath}`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: token } : {}),
    },
    body: JSON.stringify(body),
  });

  const contentType = res.headers.get("Content-Type") || "application/json";

  // Stream SSE through
  if (contentType.includes("text/event-stream")) {
    return new Response(res.body, {
      status: res.status,
      headers: {
        "Content-Type": "text/event-stream",
        "Cache-Control": "no-cache",
        Connection: "keep-alive",
      },
    });
  }

  // Pass through JSON responses
  const data = await res.text();
  return new Response(data, {
    status: res.status,
    headers: { "Content-Type": contentType },
  });
}

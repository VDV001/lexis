import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login", "/register"];

// TODO: In production (single domain), re-enable cookie-based server-side auth gating.
// Currently a pass-through — client-side auth in app/(app)/layout.tsx handles redirects.
// This is because in dev the API (:8080) and web (:3000) are on different ports,
// so cookies set by the API are not visible to this middleware.
export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (
    publicPaths.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname === "/favicon.ico" ||
    pathname === "/"
  ) {
    return NextResponse.next();
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};

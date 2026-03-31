import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const publicPaths = ["/login", "/register"];

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Allow public paths and static files
  if (
    publicPaths.some((p) => pathname.startsWith(p)) ||
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname === "/favicon.ico" ||
    pathname === "/"
  ) {
    return NextResponse.next();
  }

  // In development, check refresh_token cookie (set by Go API on same domain in prod)
  // Also check for access_token cookie as fallback
  const refreshToken = request.cookies.get("refresh_token");
  const accessToken = request.cookies.get("access_token");
  if (!refreshToken && !accessToken) {
    // Allow through — client-side auth check will redirect if needed
    // This avoids cookie domain mismatch in dev (API on :8080, web on :3000)
    return NextResponse.next();
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};

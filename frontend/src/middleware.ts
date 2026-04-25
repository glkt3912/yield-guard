import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const internalKey = process.env.APP_INTERNAL_API_KEY;
  if (!internalKey) {
    return NextResponse.next();
  }
  const requestHeaders = new Headers(request.headers);
  requestHeaders.set("X-Internal-Key", internalKey);
  return NextResponse.next({ request: { headers: requestHeaders } });
}

export const config = { matcher: "/api/:path*" };

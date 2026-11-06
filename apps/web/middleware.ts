import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  // Rotas públicas que não precisam de autenticação
  const publicRoutes = [
    "/",
    "/auth/login",
    "/auth/register",
    "/library",
    "/favorites",
    "/ranking",
    "/vip",
    "/series",
    "/manga",
  ];

  // Se é rota pública, permite acesso
  if (publicRoutes.some((route) => pathname === route || pathname.startsWith(route + "/"))) {
    return NextResponse.next();
  }

  // Rotas de admin exigem autenticação
  if (pathname.startsWith("/admin")) {
    const hasSession = request.cookies.has("ms_session");

    if (!hasSession) {
      // Redireciona para login com next para voltar depois
      const loginUrl = new URL("/auth/login", request.url);
      loginUrl.searchParams.set("next", pathname);
      return NextResponse.redirect(loginUrl);
    }

    return NextResponse.next();
  }

  // Outras rotas (perfil, etc.) também exigem autenticação
  if (pathname.startsWith("/profile")) {
    const hasSession = request.cookies.has("ms_session");

    if (!hasSession) {
      const loginUrl = new URL("/auth/login", request.url);
      loginUrl.searchParams.set("next", pathname);
      return NextResponse.redirect(loginUrl);
    }

    return NextResponse.next();
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/admin/:path*",
    "/profile/:path*",
    "/library/:path*",
    "/favorites/:path*",
  ],
};

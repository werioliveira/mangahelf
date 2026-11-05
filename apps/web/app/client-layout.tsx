"use client";

import { useEffect } from "react";
import { usePathname } from "next/navigation";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";
import { useThemeSettings } from "@/lib/api";

export default function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isAdmin = pathname.startsWith("/admin");
  const { data: theme } = useThemeSettings();

  // Injetar cores do CMS como CSS variables no <html>
  useEffect(() => {
    if (!theme) return;
    const root = document.documentElement;
    root.style.setProperty("--theme-primary", theme.primary);
    root.style.setProperty("--theme-secondary", theme.secondary);
    root.style.setProperty("--theme-accent", theme.accent);
  }, [theme]);

  return (
    <>
      {!isAdmin && <Header />}
      <main className={`min-h-screen ${isAdmin ? "bg-slate-50 text-slate-900" : "bg-background"}`}>
        {children}
      </main>
      {!isAdmin && <Footer />}
    </>
  );
}

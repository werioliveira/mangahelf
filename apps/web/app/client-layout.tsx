"use client";

import { usePathname } from "next/navigation";
import { Header } from "@/components/Header";
import { Footer } from "@/components/Footer";

export default function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const isAdmin = pathname.startsWith("/admin");

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

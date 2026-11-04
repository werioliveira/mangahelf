"use client";

import { useState } from "react";
import Link from "next/link";
import { Menu, X, Crown } from "lucide-react";

const links = [
  { href: "/", label: "Home" },
  { href: "/library", label: "Biblioteca" },
  { href: "/favorites", label: "Favoritos" },
  { href: "/ranking", label: "Ranking 🏆" },
  { href: "/vip", label: "VIP", vip: true },
  { href: "/profile", label: "Perfil" },
  { href: "/admin", label: "Admin" },
];

export function MobileMenu() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <button
        aria-label="Abrir menu"
        onClick={() => setOpen(!open)}
        className="md:hidden p-2 text-muted-foreground hover:text-foreground rounded-xl transition-all"
      >
        {open ? <X className="w-4 h-4" /> : <Menu className="w-4 h-4" />}
      </button>
      {open && (
        <nav className="absolute top-full left-0 right-0 md:hidden bg-background/98 backdrop-blur-xl border-b border-border">
          <div className="flex flex-col px-4 py-3">
            {links.map((l) => (
              <Link
                key={l.href}
                href={l.href}
                onClick={() => setOpen(false)}
                className={`flex items-center gap-2 px-3 py-2.5 text-sm font-medium rounded-lg ${
                  l.vip ? "text-accent" : "text-muted-foreground hover:text-foreground"
                }`}
              >
                {l.vip && <Crown className="w-3.5 h-3.5" />}
                {l.label}
              </Link>
            ))}
          </div>
        </nav>
      )}
    </>
  );
}

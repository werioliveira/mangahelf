import Link from "next/link";
import { SearchX } from "lucide-react";
import { btnOutline } from "@/lib/ui";

export default function NotFound() {
  return (
    <div className="max-w-md mx-auto px-6 py-24 text-center">
      <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-secondary mb-5">
        <SearchX className="w-6 h-6 text-muted-foreground" />
      </div>
      <h2 className="font-heading text-3xl font-black text-foreground mb-2">404</h2>
      <p className="text-sm text-muted-foreground mb-6">Este título saiu da estante ou nunca existiu.</p>
      <Link href="/library" className={btnOutline}>
        Ir para a Biblioteca
      </Link>
    </div>
  );
}

"use client";

import { useEffect, useRef, useState } from "react";
import { API_URL, Icon } from "./ui";

type Props = {
  values: string[];
  onChange: (tags: string[]) => void;
  suggestionsPath: string; // ex.: "/admin/tags?q="
  placeholder?: string;
};

export function TagsInput({
  values,
  onChange,
  suggestionsPath,
  placeholder = "Digite e pressione Enter",
}: Props) {
  const [input, setInput] = useState("");
  const [suggestions, setSuggestions] = useState<string[]>([]);
  const [showSug, setShowSug] = useState(false);
  const [activeSug, setActiveSug] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = { current: undefined as any };

  // Debounce para buscar sugestões
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    if (!input.trim()) {
      setSuggestions([]);
      return;
    }
    debounceRef.current = setTimeout(async () => {
      try {
        const res = await fetch(`${API_URL}${suggestionsPath}${encodeURIComponent(input.trim())}`, {
          credentials: "include",
        });
        if (res.ok) {
          const data = (await res.json()) as string[];
          // Filtra tags já adicionadas
          setSuggestions(data.filter((t) => !values.includes(t)).slice(0, 8));
          setShowSug(true);
          setActiveSug(-1);
        }
      } catch {
        // silencioso
      }
    }, 250);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [input, suggestionsPath, values]);

  // Fecha sugestões ao clicar fora
  useEffect(() => {
    const onClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setShowSug(false);
      }
    };
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

  const addTag = (raw: string) => {
    const t = raw.trim().toLowerCase();
    if (!t) return;
    if (values.includes(t)) return;
    onChange([...values, t]);
    setInput("");
    setActiveSug(-1);
    setShowSug(false);
  };

  const removeTag = (i: number) => {
    onChange(values.filter((_, idx) => idx !== i));
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter" || e.key === ",") {
      e.preventDefault();
      if (activeSug >= 0 && suggestions[activeSug]) {
        addTag(suggestions[activeSug]);
      } else if (input.trim()) {
        addTag(input);
      }
      return;
    }
    if (e.key === "Backspace" && !input && values.length > 0) {
      e.preventDefault();
      removeTag(values.length - 1);
      return;
    }
    if (showSug && suggestions.length > 0) {
      if (e.key === "ArrowDown") {
        e.preventDefault();
        setActiveSug((p) => (p + 1) % suggestions.length);
      }
      if (e.key === "ArrowUp") {
        e.preventDefault();
        setActiveSug((p) => (p <= 0 ? suggestions.length - 1 : p - 1));
      }
      if (e.key === "Escape") {
        setShowSug(false);
      }
    }
  };

  return (
    <div ref={containerRef} className="relative">
      <div
        onClick={() => inputRef.current?.focus()}
        className="flex min-h-[42px] flex-wrap items-center gap-1.5 rounded-lg border border-white/10 bg-white/5 px-2 py-1.5 focus-within:border-indigo-400/50"
      >
        {values.map((t, i) => (
          <span
            key={t}
            className="inline-flex items-center gap-1 rounded-md bg-indigo-500/20 px-2 py-1 text-xs font-medium text-indigo-200"
          >
            {t}
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                removeTag(i);
              }}
              aria-label={`Remover tag ${t}`}
              className="rounded hover:bg-indigo-500/30"
            >
              <Icon name="x" className="h-3 w-3" />
            </button>
          </span>
        ))}
        <input
          ref={inputRef}
          value={input}
          placeholder={values.length === 0 ? placeholder : ""}
          className="min-w-[120px] flex-1 bg-transparent px-1 py-0.5 text-sm text-slate-200 outline-none placeholder:text-slate-500"
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={onKeyDown}
          aria-label="Adicionar tag"
        />
      </div>

      {showSug && suggestions.length > 0 && (
        <ul
          role="listbox"
          className="absolute z-10 mt-1 max-h-48 w-full overflow-y-auto rounded-lg border border-white/10 bg-[#0e1426] py-1 shadow-xl"
        >
          {suggestions.map((s, i) => (
            <li
              key={s}
              role="option"
              aria-selected={i === activeSug}
              onMouseEnter={() => setActiveSug(i)}
              onMouseDown={(e) => {
                e.preventDefault();
                addTag(s);
              }}
              className={`cursor-pointer px-3 py-2 text-sm ${
                i === activeSug ? "bg-indigo-500/15 text-indigo-300" : "text-slate-300 hover:bg-white/5"
              }`}
            >
              {s}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

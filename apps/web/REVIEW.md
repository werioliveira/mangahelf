# Revisão de código — apps/web (2026-11-04)

Revisão `/review uncommitted` do frontend MangaShelf (Next.js 15 App Router).
Sem falhas de segurança; build e typecheck verdes; First Load JS 116 kB (< 250 kB do budget).

## Recommendation: NEEDS CHANGES (4 warnings + 9 sugestões)

### WARNINGs (corrigir antes do próximo milestone)

1. **`.gitignore` sem `.env*`** — `apps/web/.gitignore:1` (+ repo sem `.gitignore` raiz).
   A primeira env file com URL/credenciais da API Go seria commitável.
   *Fix:* `.env*` / `!.env.example` no apps/web e um `.gitignore` raiz com `/node_modules`, `/.next`, `.env*`.

2. **Fontes Google no build** — `app/layout.tsx:2` usa `next/font/google`; `next build`
   baixa os woff2 e falha sem rede. O alvo de deploy é VPS self-hosted (roadmap).
   *Fix:* vendorizar e usar `next/font/local`, ou documentar egress no Docker/CI.

3. **Imagem de fundo `priority` + `sizes="100vw"`** — `components/HeroCarousel.tsx:49` e
   `app/manga/[slug]/page.tsx:35`: o fundo que recebe `blur-3xl` baixa a variante em
   resolução de viewport do PNG de ~2 MB e compete com o LCP real (capa de 288 px).
   *Fix:* remover `priority` e usar `sizes="320px"` só no fundo.

4. **Datas de capítulo contraditórias** — `app/manga/[slug]/page.tsx:117` fabrica
   `i === 0 ? "1 hora atrás" : i+1 + " dias atrás"` em vez de usar `updates[].ago` de
   `lib/data.ts` (cap 178: "3 horas atrás" na home vs "2 dias atrás" no detalhe).
   *Fix:* buscar `updates.find(u => u.slug === s.slug)` e renderizar `c.ago`.

### SUGGESTIONs

5. `app/manga/[slug]/page.tsx:56` — chips de gênero linkam `/library` sem o filtro
   (nada lê `searchParams`); clique é no-op. *Fix:* `?genero=` + seed do `LibraryBrowser`.
6. `app/layout.tsx:20` — sem `metadataBase`, `og:image` sai relativo; share previews
   quebram em Facebook/WhatsApp/X. *Fix:* `metadataBase: new URL(NEXT_PUBLIC_SITE_URL)`.
7. `components/LibraryBrowser.tsx:12` — `sort()` code-unit misplaced "Ação" em pt-BR.
   *Fix:* `localeCompare(b, "pt")`.
8. `components/HeroCarousel.tsx:16` — timer roda com aba oculta. *Fix:* pausar em
   `visibilitychange` (uso de `useEffect` com sistema externo, permitido pela skill).
9. `lib/data.ts:206` — `export const ranking` nunca importado; `/ranking` reordena
   `series` (fonte de verdade dupla morta). *Fix:* apagar ou consumir em um lugar só.
10. `components/Header.tsx:5` + `components/MobileMenu.tsx:7` — links de nav hardcodeados
    duas vezes; drift desktop↔mobile ao renomear rota. *Fix:* `NAV_LINKS` compartilhado.
11. `app/manga/[slug]/capitulos/[n]/page.tsx:43` — classes `disabled:` em `<span>`/`<Link>`
    e `aria-disabled` sem role são dead code; direção indisponível deveria ser
    `<button disabled>`.
12. `app/ranking/page.tsx:9` — description "mais lidos da semana" vs ordenação all-time
    da página. *Fix:* alinhar o texto.
13. `public/covers/*.png` — 14,4 MB de PNGs demo no git servidos pelo otimizador do
    Next, contra a linha 45 do roadmap ("esse papel é do imgproxy ou do modo static").
    Aceitável como demo com o `shortcut:` declarado; ao ligar a API, usar URLs imgproxy
    via `remotePatterns` e variantes WebP pequenas.

### Verificado e deliberadamente NÃO reportado
- Hrefs relativos do leitor (`${num-1}`/`${num+1}`) resolvem corretamente em Next 15.
- Bounds de capítulo (1..chapters), cores de ranking >3º, todos os slugs existentes.
- Sem `<img>`, sem `useEffect` para fetch, listas com key, `sizes` em todos os `Image`.

# Patch notes — MangaShelf

## 0.1.0 — 2026-11-04 · Frontend web (`apps/web/`)

**Adicionado**
- Frontend Next.js 15 (App Router) em `apps/web/`, réplica fiel do design base
  (`base44html/`): tokens (índigo `239 84% 67%` · ciano `187 92% 42%` · dark `240 10% 6%`),
  Space Grotesk + Inter, radius 0.75rem — marca **MangaShelf**.
- Home completa: hero de destaque com carrossel e capas empilhadas, carrossel TOP 10 com
  número em contorno, grid "Recentemente Atualizadas" com badges NOVO, sidebar POPULARES
  com abas Diário/Semanal/Mensal.
- Rotas: `/library` (busca + filtro por gênero), `/favorites`, `/ranking`, `/vip`,
  `/profile`, `/admin` (stub), `/manga/[slug]` (detalhe + capítulos) e
  `/manga/[slug]/capitulos/[n]` (leitor com prev/next) — com `loading`/`error`/`not-found`
  nas rotas dinâmicas e metadata/OG em todas as páginas.
- Camada de dados demo `lib/data.ts` no shape da futura API (`api/openapi.yaml`), marcada
  com `shortcut:` para troca por `fetch` + cache tags.
- `REVIEW.md` com as recomendações da revisão de código (4 warnings, 9 sugestões).

**Validado**
- `next build` verde; First Load JS 116 kB (budget ≤ 250 kB); zero violações da skill
  `mangashelf-frontend-optimized` (sem `useEffect` para fetch, sem `<img>`, server
  components por padrão).

**Conhecido / próximo**
- Warnings da revisão pendentes: `.env*` no gitignore (coberto nesta versão), fontes
  Google no build (vendorizar para deploy self-hosted), imagens de fundo `priority`,
  datas de capítulo na página de detalhe.
- Sem backend: dados 100% demo; leitor usa páginas placeholder até o pipeline imgproxy.

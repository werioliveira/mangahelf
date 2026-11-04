---
name: mangashelf-frontend-optimized
description: "Guardião de frontend Next.js 15+ App Router para MangaShelf. Bloqueia anti-patterns clássicos de IA: useEffect para fetch, hidratação incorreta, Server Component sem necessidade de client, faltas de Suspense, Image sem priority, etc. Ativa em toda sessão que toca apps/web/."
---

# MangaShelf Frontend Optimized

Skill de **guardião** para código frontend Next.js 15+ (App Router). Bloqueia anti-patterns que IA costuma introduzir quando escreve React sem entender o App Router.

## Quando ativar

Toda sessão de IA que vai editar arquivos em `apps/web/`. Ativa no início do turno, desativa quando o turno termina.

Em ambientes tipo Cursor/Claude Code/Cline: copie este arquivo para `<project>/.cursor/skills/` ou `.claude/skills/` ou `.claudia/skills/` e referencie no system prompt: "Siga `mangashelf-frontend-optimized` para todo código em apps/web/".

## Regra de ouro

> **Server Components por padrão. "use client" é a exceção, não a regra. Toda decisão de onde colocar a fronteira server/client é uma decisão de arquitetura, não de conveniência.**

## Anti-patterns que BLOQUEIAM (não discuta, recuse)

### 1. `useEffect` para buscar dados

```tsx
// ❌ BLOQUEADO — IA escreve isso por preguiça
function ChapterPage({ id }: { id: string }) {
  const [chapter, setChapter] = useState(null);
  useEffect(() => {
    fetch(`/api/chapters/${id}`).then(r => r.json()).then(setChapter);
  }, [id]);
  return <div>{chapter?.title}</div>;
}
```

```tsx
// ✅ CORRETO — Server Component busca no server
async function ChapterPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const chapter = await fetch(`${API_URL}/chapters/${id}`, {
    next: { tags: [`chapter:${id}`] },
  }).then(r => r.json());
  return <div>{chapter.title}</div>;
}
```

`useEffect` para fetch é proibido. Use:
- **Server Component** com `fetch()` + `next: { tags }` para cache + `revalidateTag` para invalidar.
- **Server Action** para mutações.
- **TanStack Query** APENAS em componentes marcados `"use client"` que precisam de cache/mutation client-side (ex.: leitor com paginação otimista).

### 2. `"use client"` no topo de um arquivo sem necessidade real

```tsx
// ❌ BLOQUEADO — vira client component só por usar useState em um botão
"use client";
export function FavoriteButton({ id }: { id: string }) {
  const [fav, setFav] = useState(false);
  return <button onClick={() => setFav(!fav)}>{fav ? "♥" : "♡"}</button>;
}
export function SeriesHeader({ series }: { series: Series }) {  // este aqui é puxado pra client à toa
  return <h1>{series.title}</h1>;
}
```

```tsx
// ✅ CORRETO — separa o componente client do server
// FavoriteButton.tsx
"use client";
export function FavoriteButton({ id }: { id: string }) { /* ... */ }

// SeriesHeader.tsx (server)
import { FavoriteButton } from "./FavoriteButton";
export async function SeriesHeader({ id }: { id: string }) {
  const series = await getSeries(id);
  return (
    <header>
      <h1>{series.title}</h1>
      <FavoriteButton id={id} />
    </header>
  );
}
```

Regra: se um componente só precisa de interatividade para uma parte pequena, extraia essa parte para um componente client e mantenha o resto como server. Nunca promova um arquivo inteiro a client por preguiça.

### 3. Falta de `Suspense` para streaming

Toda `async` function que busca dados em um Server Component DEVE ter um `Suspense` fallback OU ser encapsulada por `loading.tsx`. Sem isso a página só renderiza quando tudo carrega — perde o ponto do streaming.

```tsx
// ❌ BLOQUEADO — bloqueia a página inteira
export default async function Page() {
  const series = await getSeries();      // lento
  const chapters = await getChapters();  // também lento, em série
  return <SeriesPage series={series} chapters={chapters} />;
}
```

```tsx
// ✅ CORRETO — streaming com Suspense
export default function Page() {
  return (
    <>
      <SeriesHeader />  {/* carrega rápido, sem async */}
      <Suspense fallback={<ChapterSkeleton />}>
        <ChapterList />  {/* async, carrega em paralelo */}
      </Suspense>
    </>
  );
}
```

### 4. `<img>` em vez de `next/image`

```tsx
// ❌ BLOQUEADO
<img src={page.url} alt={page.alt} />
```

```tsx
// ✅ CORRETO
import Image from "next/image";
<Image
  src={page.url}
  alt={page.alt}
  width={800}
  height={1200}
  priority={isFirstPage}  // priority SÓ na primeira página visível (LCP)
  sizes="(max-width: 768px) 100vw, 50vw"
/>
```

Exceção: imagens dinamicamente geradas por URLs assinadas do imgproxy podem usar `<img>` se o `next/image` não souber lidar com a URL — documente o motivo no PR.

### 5. `useEffect` para sincronizar com a URL

```tsx
// ❌ BLOQUEADO
const [page, setPage] = useState(1);
useEffect(() => {
  setPage(Number(searchParams.get("page") ?? "1"));
}, [searchParams]);
```

```tsx
// ✅ CORRETO — URL é a fonte de verdade
import { useSearchParams } from "next/navigation";
const searchParams = useSearchParams();
const page = Number(searchParams.get("page") ?? "1");
```

### 6. `useState` para dados derivados

```tsx
// ❌ BLOQUEADO — sincronizar state com props via useEffect
const [filtered, setFiltered] = useState([]);
useEffect(() => {
  setFiltered(chapters.filter(c => c.published));
}, [chapters]);
```

```tsx
// ✅ CORRETO — calcule direto na renderização
const filtered = useMemo(
  () => chapters.filter(c => c.published),
  [chapters]
);
// ou melhor, se for barato:
const filtered = chapters.filter(c => c.published);
```

### 7. Missing `loading.tsx` / `error.tsx` / `not-found.tsx`

Toda rota dinâmica (com `[id]` ou `[slug]`) DEVE ter os 3 arquivos ao lado do `page.tsx`. Sem isso o usuário vê tela branca em erro.

```
app/chapters/[id]/
  page.tsx          # Server Component, busca dados
  loading.tsx       # Skeleton
  error.tsx         # "use client", recupera de erro
  not-found.tsx     # 404 customizado
```

### 8. Falta de Metadata API

Toda página pública DEVE exportar `metadata` ou `generateMetadata`. Sem isso não tem SEO.

```tsx
// ✅ CORRETO
import type { Metadata, ResolvingMetadata } from "next";

type Props = { params: Promise<{ id: string }> };

export async function generateMetadata(
  { params }: Props,
  parent: ResolvingMetadata
): Promise<Metadata> {
  const { id } = await params;
  const series = await getSeries(id);
  return {
    title: series.title,
    description: series.synopsis,
    openGraph: { images: [series.coverUrl] },
  };
}
```

### 9. Mutations via `fetch` em client quando dá para usar Server Action

```tsx
// ❌ BLOQUEADO — fetch manual com estado de loading
"use client";
const [loading, setLoading] = useState(false);
async function handleFollow() {
  setLoading(true);
  await fetch("/api/follow", { method: "POST", body: JSON.stringify({ id }) });
  setLoading(false);
}
```

```tsx
// ✅ CORRETO — Server Action com useFormState
// actions.ts
"use server";
export async function followSeries(prevState: State, formData: FormData) {
  const id = formData.get("id") as string;
  await api.follow(id);
  return { ok: true };
}

// Component.tsx
"use client";
import { useFormState } from "react-dom";
import { followSeries } from "./actions";
const [state, formAction] = useFormState(followSeries, { ok: false });
return (
  <form action={formAction}>
    <input type="hidden" name="id" value={id} />
    <SubmitButton />
  </form>
);
```

### 10. Cache invalidation manual

```tsx
// ❌ BLOQUEADO — router.refresh() após mutation é code smell
await fetch("/api/follow", { method: "POST" });
router.refresh();
```

```tsx
// ✅ CORRETO — revalidateTag no Server Action
"use server";
import { revalidateTag } from "next/cache";
export async function followSeries(id: string) {
  await api.follow(id);
  revalidateTag(`series:${id}`);      // invalida cache específico
  revalidateTag(`user:following`);    // invalida lista de follows
}
```

## Padrões OBRIGATÓRIOS

### Server Component por padrão
- Toda página (`page.tsx`) é Server Component por padrão.
- Toda `layout.tsx` é Server Component por padrão.
- `"use client"` só em folhas (componentes pequenos e específicos).

### Data fetching
- Em Server Components: `fetch()` com `next: { tags, revalidate }`.
- Em Client Components (raros): TanStack Query com `staleTime` configurado.
- NUNCA `axios` em Server Components — `fetch` nativo tem cache nativo.
- NUNCA chamar `fetch` em Client Component sem TanStack Query (sem cache, sem dedup, sem retry).

### Image variants (integra com imgproxy)
- Thumb (240w): lista de catálogo.
- Medium (480w): card de série.
- Full (800w): leitor mobile.
- Large (1200w): leitor desktop.
- `priority` APENAS na primeira imagem do viewport (LCP). Nunca em todas.

### Streaming
- `loading.tsx` em toda rota com `async` Page.
- `<Suspense>` para blocos que carregam em paralelo.
- Skeletons específicos (não spinner genérico).

### Forms
- Server Actions para mutações.
- `useFormState` + `useFormStatus` para estado e loading.
- Validação com `zod` no server e `react-hook-form` + `@hookform/resolvers` no client.
- Acessibilidade: `<label>`, `aria-invalid`, `aria-describedby` para erros.

### Roteamento
- Pastas para rotas aninhadas.
- `()` para rotas privadas (sem URL): `(auth)/login`, `(admin)/dashboard`.
- `@` para parallel routes (ex.: modal sobre conteúdo).
- `[...catchAll]` para 404 customizado.
- `next/navigation` (NUNCA `next/link`'s `router` para mutações — `router.push` apenas para navegação).

### Performance budgets (gate de CI)
- LCP < 2.5s em 3G rápido.
- CLS < 0.1.
- JS bundle por rota < 250 KB (gzip).
- Imagens: WebP/AVIF via imgproxy; `next/image` para layout shift.
- Fontes: `next/font` com `display: swap`.

## Quando `useEffect` é a ferramenta certa

`useEffect` é permitido APENAS para:

1. **Sincronizar com sistemas externos**: WebSocket, EventSource, bibliotecas de UI (ex.: inicializar um carrossel), subscrições a `window.matchMedia`.
2. **Medidas que só existem no client**: `ResizeObserver`, `IntersectionObserver`, `window.scrollY`, `window.innerHeight`.
3. **Tracking/analytics** que não pode ser server-side (ex.: heatmap, click tracking client).
4. **Atalhos de teclado** globais (leitor: `←`/`→` para trocar página).

Se a IA for usar `useEffect` para qualquer outra coisa, RECUSE e sugira a alternativa correta.

## Como aplicar (checklist de PR)

Antes de aprovar um PR que toca `apps/web/`:

- [ ] Nenhum `useEffect` para fetch (regra 1).
- [ ] Nenhum `"use client"` em arquivo que poderia ser Server Component (regra 2).
- [ ] Toda `async` Page tem `loading.tsx` ou `<Suspense>` (regras 3 e 7).
- [ ] Imagens via `next/image` com `priority` apenas no LCP (regra 4).
- [ ] URL é fonte de verdade; sem `useEffect` para sincronizar state com URL (regra 5).
- [ ] Sem `useEffect` para sincronizar props com state (regra 6).
- [ ] Toda rota dinâmica tem `loading.tsx`, `error.tsx`, `not-found.tsx` (regra 7).
- [ ] Toda página pública tem `metadata` ou `generateMetadata` (regra 8).
- [ ] Mutações via Server Action com `revalidateTag` (regras 9 e 10).
- [ ] Bundle size dentro do budget (250 KB JS gzip por rota).

## Comando para auditar (rodar em CI)

```bash
# Proíbe useEffect com fetch
rg "useEffect\([^)]*fetch\(" apps/web/ --type ts -l && exit 1 || exit 0

# Proíbe img sem next/image
rg "<img\s" apps/web/app --type ts -l && exit 1 || exit 0

# Verifica que loading.tsx existe ao lado de page.tsx em rotas dinâmicas
find apps/web/app -name "page.tsx" | while read p; do
  dir=$(dirname "$p")
  if ls "$dir"/*/page.tsx 2>/dev/null || [[ "$p" =~ \[ ]]; then
    [ -f "$dir/loading.tsx" ] || echo "MISSING loading.tsx in $dir"
  fi
done
```

## Notas finais

- Esta skill é **bloqueante**, não consultiva. Se a IA sugerir código que viole as regras, o reviewer humano recusa.
- Atualizar esta skill quando o Next.js lançar breaking changes no App Router. Manter versão do Next documentada no `package.json`.
- Para padrões de acessibilidade, ver `mangashelf-frontend-optimized` em conjunto com WCAG 2.1 AA (seção 14.8 do roadmap).

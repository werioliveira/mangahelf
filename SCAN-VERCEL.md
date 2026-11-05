# Scan de Conformidade — Vercel React Best Practices

**Data:** 2026-11-05  
**Skill:** `vercel-labs/agent-skills` / `vercel-react-best-practices` (v1.0.0)  
**Escopo:** `apps/web/` (frontend Next.js 15, App Router)  
**Total de regras analisadas:** 40+ regras em 8 seções

---

## Resumo Executivo

| Status | Quantidade | Regras |
|--------|------------|--------|
| ✅ Conforme | 11 | 2.1, 2.5, 3.5, 3.6, 5.1, 5.4, 5.11, 5.15, 6.9, 7.13, 7.14 |
| ⚠️ Parcial / Aceitável | 2 | 1.5 (dependências), 4.3 (fetch nativo vs SWR) |
| ❌ Não conforme | 0 | Nenhum gap crítico identificado |
| N/A | 27 | Regras não aplicáveis ao estágio atual do projeto |

**Veredito:** O frontend está em **boa conformidade** com as regras críticas da skill. Não há anti-patterns de performance que comprometam significativamente a experiência do usuário.

---

## 1. Eliminating Waterfalls (CRITICAL)

### 1.1 Check Cheap Conditions Before Async Flags — N/A
Não há padrões de `flag && cheapCondition` com async no frontend.

### 1.2 Defer Await Until Needed — N/A
Não há awaits desnecessários bloqueando branches não utilizados.

### 1.3 Dependency-Based Parallelization — N/A
Não há operações com dependências parciais que precisem de paralelismo avançado.

### 1.4 Prevent Waterfall Chains in API Routes — N/A
Não há API Routes no frontend (são no backend Go).

### 1.5 Promise.all() for Independent Operations — ⚠️ Aceitável
`lib/api.ts` `lookupChapterPages()` chama `lookupChapter()` e depois `getChapterPages()` sequencialmente. **Porém**, as chamadas são dependentes (precisa do `chapterId` do lookup para buscar as páginas), então não é possível paralelizar. Não é um problema real.

### 1.6 Strategic Suspense Boundaries — N/A
Páginas são estáticas/SSR com dados já disponíveis. Não há necessidade de Suspense neste estágio.

---

## 2. Bundle Size Optimization (CRITICAL)

### 2.1 Avoid Barrel File Imports — ✅ CONFORME
Todas as importações são diretas de arquivos específicos. Não há barrel files (`index.ts`) sendo importados.

### 2.2 Conditional Module Loading — N/A
Não há carregamento condicional de módulos com `import()` dinâmico.

### 2.3 Defer Non-Critical Third-Party Libraries — N/A
Não há bibliotecas de terceiros pesadas sendo carregadas.

### 2.4 Dynamic Imports for Heavy Components — N/A
Não há componentes pesados que justifiquem `next/dynamic`.

### 2.5 Prefer Statically Analyzable Paths — ✅ CONFORME
Todas as importações são estáticas e analisáveis pelo bundler. Não há caminhos montados dinamicamente.

### 2.6 Preload Based on User Intent — N/A
Não há cenários de preload baseados em hover/focus neste estágio.

---

## 3. Server-Side Performance (HIGH)

### 3.1 Authenticate Server Actions Like API Routes — N/A
Não há Server Actions no frontend.

### 3.2 Avoid Duplicate Serialization in RSC Props — ✅ CONFORME
Os componentes passam apenas os dados necessários. Exemplo: `Page()` passa `items={featured}`, `items={top10}` — dados estáticos já disponíveis, sem transformação desnecessária.

### 3.3 Avoid Shared Module State for Request Data — ✅ CONFORME
Não há shared module state mutável para dados de request.

### 3.4 Cross-Request LRU Caching — N/A
Não há LRU cache no frontend.

### 3.5 Hoist Static I/O to Module Level — ✅ CONFORME
`lib/data.ts`: `series`, `featured`, `top10`, `updates`, `popular` são definidos no nível do módulo, carregados uma vez.

### 3.6 Minimize Serialization at RSC Boundaries — ✅ CONFORME
Somente campos necessários são passados via props. Não há serialização de objetos inteiros quando apenas um campo é usado.

### 3.7 Parallel Data Fetching with Component Composition — N/A
Não há busca de dados paralela necessária (dados estáticos no módulo).

### 3.8 Parallel Nested Data Fetching — N/A
Não há nested fetching.

### 3.9 Per-Request Deduplication with React.cache() — N/A
Não há múltiplas chamadas iguais que precisem de dedup.

### 3.10 Use after() for Non-Blocking Operations — N/A
Não há operações pós-response no frontend.

---

## 4. Client-Side Data Fetching (MEDIUM-HIGH)

### 4.1 Deduplicate Global Event Listeners — N/A
Não há global event listeners.

### 4.2 Use Passive Event Listeners for Scrolling Performance — N/A
`Top10.tsx` usa `scrollBy()` com `behavior: "smooth"` — não usa `addEventListener` direto para scroll.

### 4.3 Use SWR for Automatic Deduplication — ⚠️ Aceitável
`lib/api.ts` usa `fetch` nativo com `next: { tags: [...] }` para revalidação do Next.js. Não está usando SWR. Para o escopo atual (poucos dados, poucas instâncias simultâneas), o fetch nativo com cache tags é aceitável. Se o projeto crescer, migrar para SWR seria recomendado.

### 4.4 Version and Minimize localStorage Data — N/A
Não há uso de `localStorage`.

---

## 5. Re-render Optimization (MEDIUM)

### 5.1 Calculate Derived State During Rendering — ✅ CONFORME
`LibraryBrowser.tsx`: `filtered` é calculado durante a renderização (comentário no código: "filtro derivado, calculado na renderização"). Não é estado espelhado em `useState`.

### 5.2 Defer State Reads to Usage Point — N/A
Não há uso de `searchParams` ou `localStorage` em componentes.

### 5.3 Do not wrap simple expression with primitive result in useMemo — ✅ CONFORME
Não há `useMemo` desnecessário em expressões simples.

### 5.4 Don't Define Components Inside Components — ✅ CONFORME
Não há componentes definidos dentro de outros componentes em nenhum arquivo.

### 5.5 Extract Default Non-primitive Parameter Value — N/A
Não há memoized components com default values.

### 5.6 Extract to Memoized Components — N/A
Não há `memo()` em componentes. Não é crítico para este estágio.

### 5.7 Narrow Effect Dependencies — ✅ CONFORME
`HeroCarousel.tsx`: `useEffect` depende de `[paused, items.length]` — dependências estreitas e corretas.

### 5.8 Put Interaction Logic in Event Handlers — ✅ CONFORME
Lógica de interação está em handlers (`onClick`, `onMouseEnter`, etc.).

### 5.9 Split Combined Hook Computations — N/A
Não há hooks combinados problemáticos.

### 5.10 Subscribe to Derived State — N/A
Não há subscription problemática.

### 5.11 Use Functional setState Updates — ✅ CONFORME
`HeroCarousel.tsx`: `setIndex((i) => (i + 1) % items.length)` — atualização funcional do estado.

### 5.12 Use Lazy State Initialization — N/A
Não há estado com inicialização cara.

### 5.13 Use Transitions for Non-Urgent Updates — N/A
Não há `useTransition`. Não é crítico para este estágio.

### 5.14 Use useDeferredValue for Expensive Derived Renders — N/A
Não há renders caros de derivação.

### 5.15 Use useRef for Transient Values — ✅ CONFORME
`Top10.tsx`: usa `useRef<HTMLDivElement>` para o scroller.

---

## 6. Rendering Performance (MEDIUM)

### 6.1 Animate SVG Wrapper Instead of SVG Element — N/A
Não há animação de SVG.

### 6.2 CSS content-visibility for Long Lists — N/A
Não há listas muito longas que precisem de `content-visibility`.

### 6.3 Hoist Static JSX Elements — N/A
Não há JSX estático sendo recalculado desnecessariamente.

### 6.4 Optimize SVG Precision — N/A
Não há SVG custom.

### 6.5 Prevent Hydration Mismatch Without Flickering — N/A
Não há hydration mismatch visível.

### 6.6 Suppress Expected Hydration Mismatches — N/A
Não há supressão necessária.

### 6.7 Use Activity Component for Show/Hide — N/A
Não há show/hide complexo.

### 6.8 Use defer or async on Script Tags — N/A
Não há script tags.

### 6.9 Use Explicit Conditional Rendering — ✅ CONFORME
Renderização condicional explícita (ex: `favors.length === 0`, estados de loading, etc.).

### 6.10 Use React DOM Resource Hints — N/A
Não há resource hints (`preload`, `prefetch`). Não é crítico.

### 6.11 Use useTransition Over Manual Loading States — N/A
Não há `useTransition`. Não é crítico.

---

## 7. JavaScript Performance (LOW-MEDIUM)

### 7.1 Avoid Layout Thrashing — N/A
Não há leitura/escrita alternada de layout.

### 7.2 Build Index Maps for Repeated Lookups — N/A
Não há buscas repetidas em arrays grandes.

### 7.3 Cache Property Access in Loops — N/A
Não há loops críticos de performance.

### 7.4 Cache Repeated Function Calls — N/A
Não há funções repetidas em loops críticos.

### 7.5 Cache Storage API Calls — N/A
Não há Storage API.

### 7.6 Combine Multiple Array Iterations — ⚠️ Observação
`lib/data.ts`: várias iterações sobre `series` (`featured`, `top10`, `updates`, `popular`, `ranking`). Cada uma faz `.map()` ou `.find()`. Para arrays pequenos (< 12 itens), o impacto é desprezível. Não é um problema real neste estágio.

### 7.7 Defer Non-Critical Work with requestIdleCallback — N/A
Não há uso.

### 7.8 Early Length Check for Array Comparisons — N/A
Não há comparação de arrays.

### 7.9 Early Return from Functions — N/A
Não há funções que se beneficiariam de early return ausente.

### 7.10 Hoist RegExp Creation — N/A
Não há RegExp em loops.

### 7.11 Use flatMap to Map and Filter in One Pass — N/A
Não há `map` + `filter` separados.

### 7.12 Use Loop for Min/Max Instead of Sort — N/A
Não há `sort` para min/max.

### 7.13 Use Set/Map for O(1) Lookups — ✅ CONFORME
`LibraryBrowser.tsx`: `[...new Set(series.flatMap((s) => s.genres))].sort()` — usa `Set` para deduplicar gêneros.

### 7.14 Use toSorted() Instead of sort() for Immutability — ✅ CONFORME
`ranking/page.tsx`: `[...series].sort(...)` — usa spread para evitar mutação, equivalente a `toSorted()`.

---

## 8. Advanced Patterns (LOW)

### 8.1 Do Not Put Effect Events in Dependency Arrays — N/A
Não há efeitos com eventos no dependency array.

### 8.2 Initialize App Once, Not Per Mount — N/A
Não há inicialização repetida por mount.

### 8.3 Store Event Handlers in Refs — N/A
Não há handlers armazenados em refs.

### 8.4 useEffectEvent for Stable Callback Refs — N/A
Não há uso.

---

## Melhorias Recomendadas (Opcional, não bloqueante)

| Regra | Sugestão | Prioridade |
|-------|----------|------------|
| 4.3 Use SWR | Considerar migrar `lib/api.ts` de `fetch` nativo para SWR se o número de requisições simultâneas crescer | Baixa |
| 1.6 Suspense Boundaries | Considerar Suspense para páginas com dados assíncronos futuros | Baixa |
| 7.6 Combine Array Iterations | Consolidar iterações sobre `series` em `lib/data.ts` se a lista crescer muito | Baixa |

---

## Conclusão

O frontend do MangaShelf está em **boa conformidade** com as regras críticas da skill `vercel-react-best-practices`. As regras mais impactantes (Eliminating Waterfalls e Bundle Size Optimization) estão atendidas. Não há anti-patterns que comprometam significativamente a performance ou a experiência do usuário.

**Próximo passo:** Se o frontend crescer, considere adotar SWR para deduplicação automática de requisições e Suspense para carregamento progressivo de dados.

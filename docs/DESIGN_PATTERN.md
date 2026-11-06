# MangaShelf Admin — Design Pattern

Guia para qualquer IA (ou dev) que for criar ou alterar páginas do painel admin do MangaShelf. Siga este documento para que todas as telas pareçam do mesmo produto.

## 1. Contexto do produto

- **O que é:** um CMS para scans de mangá. Administradores cadastram séries, enviam e publicam capítulos, gerenciam usuários e planos VIP.
- **Quem usa:** equipe interna de scan. Usam o painel todo dia, então prioridade é **rapidez, clareza e baixo atrito**, não decoração.
- **Idioma da interface:** português do Brasil, sempre. Datas em `pt-BR`.
- **Estilo:** dashboard escuro, moderno, limpo. Referência visual: templates tipo NextAdmin (sidebar lateral, cards de estatística, tabelas em painéis arredondados).

## 2. Stack e convenções de código

- Next.js (App Router) + TypeScript estrito + Tailwind CSS. Sem biblioteca de UI externa.
- Ícones: SVG inline (24x24, `stroke="currentColor"`, `strokeWidth 1.8`, linhas arredondadas). Não usar emojis nem libs de ícone.
- Estrutura:

```
app/admin/
├── layout.tsx             # server, só metadata, renderiza <ClientLayout>
├── client-layout.tsx      # sidebar + header (client)
├── page.tsx               # dashboard (server component, fetch no servidor)
├── _components/ui.tsx     # componentes compartilhados (fonte única de UI)
├── series/page.tsx
└── chapters/page.tsx
```

- **Páginas de dados interativos** (listas com busca, edição, modais) são `"use client"`. **Dashboard de leitura** é server component com `fetch(..., { cache: "no-store" })`.
- API base: `process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"`. Requisições do navegador usam `credentials: "include"`.
- **Sempre reutilize** `_components/ui.tsx`. Se precisar de um componente novo e ele puder servir a mais de uma página, adicione lá em vez de duplicar.
- Erros: nunca use `err.message` direto em `catch`. Use o helper `errMsg(e, "mensagem padrão")`.
- Nunca use `alert()`. Feedback é por toast (`useToast`).

## 3. Design tokens

Use exatamente estes valores (Tailwind arbitrary values). Não invente tons novos.

### Cores

| Papel | Valor | Uso |
|---|---|---|
| Fundo da página | `#0b1020` | `bg-[#0b1020]` no layout e em inputs |
| Fundo da sidebar | `#0e1426` | `bg-[#0e1426]` |
| Superfície (cards, painéis, modais) | `#111830` | `bg-[#111830]` |
| Borda de card | `white/[0.06]` | `border border-white/[0.06]` |
| Borda de divisão interna | `white/5` | `divide-white/5`, `border-white/5` |
| Borda de input | `white/10` | `border-white/10` |
| Hover de linha/botão neutro | `white/[0.03]` a `white/5` | |
| Texto principal | `text-white` / `text-slate-100` | títulos, valores |
| Texto secundário | `text-slate-300` / `text-slate-400` | corpo, labels |
| Texto terciário | `text-slate-500` | legendas, datas, ajudas |
| **Destaque (accent)** | indigo: `indigo-500` (botão), `indigo-300` (texto/link/ativo), `indigo-500/15` (fundo ativo) | única cor de ação |
| Destaque secundário | `sky-400` | só em gradientes com o indigo e gráficos |

Regra: **uma cor de ação (indigo)**. As outras cores só aparecem como **semântica de status** (ver badges).

### Cores semânticas (badges e status)

Padrão de badge: `bg-{cor}-400/10 text-{cor}-300 ring-1 ring-inset ring-{cor}-400/20`, em forma de pílula (`rounded-full px-2.5 py-0.5 text-xs font-medium`).

| Tom | Significado |
|---|---|
| `green` | sucesso, ativo, publicado, público |
| `amber` | em andamento, atenção (processando, early) |
| `violet` | VIP / premium |
| `sky` | informativo (acesso público em capítulos, early em séries) |
| `red` | erro, inativo, removido, ação destrutiva |
| `gray` | neutro, rascunho, sem valor |

### Tipografia

- Fonte do sistema do projeto (sem fonte customizada).
- Título de página: `text-2xl font-bold tracking-tight text-white`.
- Título de card/painel: `text-base font-semibold text-white`.
- Valor grande (stat): `text-[28px] font-bold leading-none tracking-tight`.
- Corpo: `text-sm`. Legenda: `text-xs text-slate-500`.
- **Cabeçalho de tabela:** `text-xs font-medium text-slate-500`, em caixa normal. **Não usar CAIXA ALTA com tracking**.
- Slug/IDs/números de capítulo: `font-mono text-xs`.

### Forma e espaçamento

- Cards e painéis: `rounded-2xl`. Modais: `rounded-2xl`. Inputs e botões: `rounded-lg`. Campos de busca e filtros de topo: `rounded-xl`. Badges: `rounded-full`.
- **Sem sombras decorativas** nos cards. Profundidade vem de borda + diferença de fundo. Só o modal tem `shadow-2xl`.
- Padding de card: `p-5` (stat) ou `p-6` (gráfico/conteúdo). Células de tabela: `px-5 py-3.5` (ou `py-4` quando há 2 linhas de texto).
- Espaço vertical entre blocos da página: `space-y-6`. Container: `mx-auto max-w-7xl`.
- Sem gradientes decorativos em fundos. Gradiente só em: logo, barras de progresso (`from-indigo-500 to-sky-400`) e miniaturas de capa.

## 4. Shell (layout)

### Sidebar (`client-layout.tsx`)
- Largura `w-64`, recolhida `w-[72px]` (só ícones, com `title` em cada item). Transição de largura de 200ms.
- Topo: logo (quadrado `rounded-lg` com gradiente indigo→sky e letra "M") + "MangaShelf" + legenda "Painel admin".
- Grupos de navegação: **Geral** (Dashboard), **Catálogo** (Séries, Capítulos), **Usuários**, **VIP** (Planos), **Configurações** (Geral, Busca, Interface, Interno).
- Grupo com **1 item** vira link direto (sem acordeão). Grupo com vários itens é acordeão com chevron que gira; subitens ficam sob uma linha vertical `border-l border-white/10`.
- Item ativo: `bg-indigo-500/15 text-indigo-300`. Inativo: `text-slate-400 hover:bg-white/5 hover:text-white`.
- Ao clicar num grupo com a sidebar recolhida, ela expande.
- Rodapé: botão "Sair" (hover vermelho sutil).

### Header
- Altura `h-16`, fundo `bg-[#0b1020]/80 backdrop-blur`, borda inferior `border-white/5`.
- Esquerda: botão de menu (recolhe sidebar) + **título dinâmico da página** (derivado da rota) + subtítulo.
- Direita: busca em pílula (escondida em mobile), indicador "Sistema online" (pílula verde com ponto), avatar redondo.

### Conteúdo
- `main` com `p-6 lg:p-8` e rolagem própria (`overflow-y-auto`). O layout ocupa a tela toda (`h-screen`), só o `main` rola.

## 5. Anatomia de uma página admin

Toda página segue esta ordem:

1. **PageHeader**: título + subtítulo com contexto útil (ex.: "12 no catálogo. Cadastre, edite e controle o acesso…") + ação primária à direita (ex.: "Nova série").
2. **Barra de ferramentas**: `SearchBox` + filtros. Filtros de poucas opções usam `Segmented` **com contadores**. Filtros de muitas opções usam `<select>` estilizado.
3. **Conteúdo** dentro de `Panel` (tabela) ou grid de cards.
4. **Modais** (criar/editar/confirmar exclusão) e **toast** no fim do JSX.

### Estados obrigatórios (nunca omitir)
- **Carregando:** `<Spinner label="Carregando …" />`.
- **Erro de carga:** faixa vermelha `border-red-400/20 bg-red-400/10 text-red-300` acima do conteúdo.
- **Vazio de verdade** (sem registros): `<Empty>` com ícone, título, explicação e **botão da ação principal** quando existir.
- **Vazio por filtro:** `<Empty>` com "Nada encontrado" e sugestão de ajustar busca/filtros, sem botão de criar.
- **Salvando:** botão desabilitado com texto "Salvando…". Nunca permitir clique duplo.

## 6. Biblioteca de componentes (`_components/ui.tsx`)

| Componente | Função |
|---|---|
| `Icon` | SVG por nome (`plus, search, edit, trash, x, layers, book, check, alert`). Adicione novos paths ali. |
| `Badge` | Pílula de status com `tone`. |
| `Btn` | Botão. `variant`: `primary` (indigo), `ghost` (borda), `danger` (vermelho). |
| `inputCls` | String de classes para `input`, `select` e `textarea`. |
| `Field` | Label acima do input + texto de ajuda opcional abaixo. |
| `Modal` | Overlay escuro com blur, fecha com **Esc** e clique fora, `role="dialog"`. Header, corpo e rodapé de ações. |
| `Segmented` | Filtro em grupo de botões com contador opcional. |
| `SearchBox` | Busca com ícone. |
| `PageHeader` | Título, subtítulo, ação. |
| `Panel` | Container de tabela (`rounded-2xl`, borda, `overflow-hidden`). |
| `Th` | Célula de cabeçalho de tabela. |
| `Spinner`, `Empty` | Estados de carga e vazio. |
| `useToast` | Retorna `{ node, show(msg, ok?) }`. Renderizar `{toast.node}` no fim da página. Some em 3,5s. |
| `fmtDate`, `errMsg`, `API_URL` | Helpers. |

## 7. Padrões de UX

### Tabelas
- Cabeçalho com fundo `bg-white/[0.02]`, linhas separadas por `divide-white/5`, hover `bg-white/[0.03]`.
- `overflow-x-auto` no wrapper para não quebrar em telas pequenas.
- Coluna principal mostra identidade em 2–3 linhas: nome (`font-medium text-slate-100`), slug em mono, resumo em `line-clamp-1`.
- **Ações por linha são botões de ícone** (editar, excluir) alinhados à direita, com `aria-label` descritivo. Excluir tem hover vermelho.
- Números que levam a outra lista são **links** (ex.: contagem de capítulos de uma série leva a `/admin/chapters?series=<id>`).

### Formulários (em modal)
- Um modal por tarefa. Campo principal com `autoFocus`.
- Slug gerado automaticamente a partir do título (sem acento, minúsculo, hífens) até o usuário editá-lo manualmente.
- Texto de ajuda (`hint`) explica o campo quando não é óbvio.
- Rodapé: `Cancelar` (ghost) à esquerda do botão primário. O botão primário diz o que faz: "Criar série", "Salvar alterações". Nunca "Enviar" ou "OK".
- Validação mínima antes de enviar, com toast de erro.

### Ações destrutivas
- Sempre com **modal de confirmação** citando o nome do item em destaque.
- Se houver efeito colateral (ex.: série com capítulos), avisar em âmbar dentro do modal.
- Botão de confirmação é `danger` e diz "Excluir".

### Feedback
- Sucesso e erro de mutações: toast (`Série criada.`, `Capítulo atualizado.`, `Erro ao salvar série`).
- Atualização **otimista no estado local** após PATCH/DELETE bem-sucedido. Após POST, recarregar a lista (o servidor gera id e campos).

### Filtros e busca
- Filtragem no cliente com `useMemo`. Busca por título, slug e número.
- Filtros combinam entre si (busca + série + estado).
- Contadores nos filtros refletem o total, não o resultado filtrado.

### Acessibilidade mínima
- `aria-label` em botões só com ícone. `role="dialog" aria-modal` nos modais. Esc fecha modal.
- Contraste: nunca usar texto abaixo de `slate-500` sobre `#111830`.
- Foco visível nos inputs (`focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-400/20`).
- `[color-scheme:dark]` em inputs/selects para o menu nativo ficar escuro.

## 8. Redação (copy)

- Português do Brasil, frases curtas, verbos diretos, **caixa de frase** (só a primeira letra maiúscula).
- Nomes de ação consistentes: "Nova série", "Editar", "Excluir", "Salvar alterações", "Cancelar". O mesmo nome no botão e no toast.
- Estados em português: Publicado, Processando, Rascunho, Removido. Acesso: Público, Acesso antecipado (ou "Early" em filtros curtos), VIP.
- Erros dizem o que houve; vazios dizem o que fazer. Sem pedir desculpas, sem tom de marketing.

## 9. Domínio e dados

### Valores de enum
- **Acesso** (série e capítulo): `public`, `early`, `vip_only`. A API às vezes devolve `vip` em capítulos: normalizar para `vip_only`.
- **Estado do capítulo:** `draft`, `processing`, `published`, `removed`.

### Tipos
```ts
type Series = { id: string; title: string; slug: string; synopsis: string; access: string; chapter_count: number; created_at: string };
type Chapter = { id: string; series_id: string; series_title?: string; number: number; title: string; state: string; access: string; created_at: string };
type User = { id: string; email: string; is_active: boolean; roles: string; has_vip: boolean; vip_expires: string | null; created_at: string };
```

### Endpoints
- Existentes: `GET /admin/series`, `GET /admin/users`, `GET /admin/chapters`, `PATCH /admin/chapters/:id` (`{state, access}`).
- Assumidos para séries (confirmar com o backend): `POST /admin/series`, `PATCH /admin/series/:id`, `DELETE /admin/series/:id`, corpo `{title, slug, synopsis, access}`.

## 10. Dashboard (`/admin`)

- Linha 1: 4 `StatCard` (Séries, Capítulos, Usuários, Assinantes VIP). Cada um tem ícone em círculo colorido, valor grande, rótulo e uma dica em `text-xs`. **Só dados reais**; não inventar variação percentual se a API não fornece histórico.
- Linha 2: gráfico de área em SVG ("Novos usuários", 12 meses, calculado de `created_at`) ocupando 3/5, e ranking de séries com mais capítulos (barras horizontais com gradiente) mais barra empilhada de distribuição de acesso, ocupando 2/5.
- Linha 3: tabelas "Séries recentes" (3/5) e "Usuários recentes" (2/5), cada uma com link "Ver todas/todos".
- Gráficos são SVG/divs feitos à mão (sem lib), cor da linha `#818cf8`, grade `white` a 5% de opacidade.

## 11. O que NÃO fazer

- Não usar tema claro, fundos brancos ou `slate-900` como texto.
- Não usar `alert()`, `confirm()` ou `prompt()` nativos.
- Não adicionar sombras coloridas, glows, gradientes de fundo ou animações de entrada em cards.
- Não usar CAIXA ALTA com espaçamento largo em rótulos e cabeçalhos.
- Não criar uma cor de ação nova; indigo é a única.
- Não duplicar componentes que já existem em `_components/ui.tsx`.
- Não usar dados fictícios em métricas. Se não houver dado, mostrar estado vazio.
- Não usar emojis como ícones.
- Não esquecer os estados de carga, erro e vazio.

## 12. Checklist para uma página nova

- [ ] Usa `PageHeader` com subtítulo informativo e ação primária.
- [ ] Busca e filtros relevantes com contadores.
- [ ] Dados dentro de `Panel` com tabela padrão, ou grid de cards no mesmo estilo.
- [ ] Estados de carregando, erro, vazio e vazio-por-filtro.
- [ ] Criar/editar em `Modal`, exclusão com confirmação.
- [ ] Feedback via `useToast`, sem `alert`.
- [ ] Botões de ícone com `aria-label`.
- [ ] Textos em português, caixa de frase, nomes de ação consistentes.
- [ ] Item correspondente existe no `GROUPS` da sidebar.
- [ ] Só tokens e componentes deste documento.

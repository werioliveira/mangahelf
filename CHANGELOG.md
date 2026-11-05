# Patch notes — MangaShelf

## 0.2.0 — 2026-11-05 · Auth & AuthZ Backend (`internal/auth/`, `internal/authz/`, `internal/audit/`)

**Adicionado**
- `internal/auth`: sistema completo de autenticação conforme AUTH.md
  - Password hashing: Argon2id com semáforo de concorrência, política 12+ chars, bloqueio senhas comuns, NFKC normalization
  - Sessions: SCS com store hashed (SHA256 token), `__Host-` cookie, SameSite=Lax, idle/absolute TTL, revogação imediata
  - CSRF: Fetch Metadata + Origin allowlist + token por sessão (header `X-CSRF-Token`)
  - Rate limiting: in-memory com backoff progressivo, HMAC email hash, fail-closed em auth
  - One-time tokens: email verify (24h), password reset (30min), email change (1h), consumo atômico
  - MFA: TOTP (AES-256-GCM encrypted secrets, key rotation, replay protection), recovery codes (HMAC-SHA256)
  - WebAuthn/Passkeys: discoverable credentials, sign count verification, clone detection
  - Flows: register, login (com MFA challenge), logout, forgot/reset/change password, change email, verify email, reauth (step-up)
  - Auditoria: append-only `audit_log` com trigger imutável
- `internal/authz`: autorização deny-by-default
  - Permission catalog tipado (32 permissions, scopes global/group, MFA flags)
  - Resolver: avalia deny overrides → allow overrides → group role → global role (deny vence allow)
  - Anti-escalação: position-based, ator deve possuir todas permissões do role, proteção último owner
  - Policy router: Public / Authenticated / Require(perm, scopeFn) com teste CI que falha se rota sem policy
- `internal/audit`: logging estruturado com actor, action, target, before/after, IP, UA
- Database schema (migration 0009): users, sessions, user_sessions, auth_tokens, TOTP, recovery codes, WebAuthn, permissions, roles, role_permissions, user_roles, group_members, user_overrides, bans, audit_log
- Dependencies: scs, pgxstore, pgx/v5, webauthn, otp, x/time, x/text, pgconn

**Removido**
- `internal/services/auth.go`, `internal/services/auth/service.go`, `internal/services/auth_test.go` (old gorilla/sessions auth)
- `internal/services/oidc/auth.go` (old OIDC stub)
- `internal/adapters/httpapi/middleware/auth.go` (old gorilla/sessions middleware)

**Atualizado**
- `go.mod`/`go.sum`: novas dependências
- `internal/adapters/httpapi/auth.go`: novo AuthHandler com endpoints completos
- `internal/adapters/httpapi/oidc.go`: integração OIDC stub
- `internal/ports/ports.go`: interfaces Authenticator/Authorizer atualizadas
- `internal/services/oidc/oidc.go`: serviço OIDC
- `internal/services/auth_service.go`: novo entry point do serviço de auth
- `deploy/.env.example`: variáveis de configuração auth (SESSION_SECRET, RPID, RP_ORIGINS, ENCRYPTION_KEYS, etc.)

**Validado**
- `go build ./...` verde
- `go test ./...` verde

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
# Patchnote — MangaShelf

## v0.1.0 — Fundação e autenticação (2026-11-04)

### Estrutura do repositório (monorepo)
- `/cmd/api` — entrypoint da API Go (chi)
- `/cmd/worker` — entrypoint do worker de mídia
- `/internal` — pacotes Go (config, ports, adapters, services)
- `/migrations` — migrations SQL (up.sql + down.sql)
- `/api` — OpenAPI (`openapi.yaml`)
- `/deploy` — docker-compose, Caddyfile, .env.example
- `/docs` — ADRs, self-host, contribuição

### Fase 0 — Fundação
- `go.mod` — módulo único `github.com/mangashelf/mangashelf` (Go 1.22)
- `internal/config/config.go` — loader de config com validação de boot (drivers válidos, combinações inválidas falham com mensagem clara)
- `internal/ports/ports.go` — interfaces: `Storage`, `SearchEngine`, `ImageURLBuilder`, `Cache`, `Authenticator`, `Authorizer`
- Adaptadores: `storage/local` (funcional), `storage/s3` (stub), `search/postgres` (stub), `cache/memory` (funcional), `auth/builtin` (stub)
- `cmd/api/main.go` — servidor chi com `/healthz`, middleware (request ID, real IP, recovery, logger, timeout), desligamento graceful
- `cmd/worker/main.go` — entrypoint do worker
- `migrations/0001_initial/{up,down}.sql` — schema inicial: users, groups, permissions/roles/role_permissions/user_roles/group_members/user_overrides (sistema de permissão atômica §4.2), series, chapters (access/public_at §5.4), vip_plans/memberships/vouchers, audit_log (+ índices). Reversível.
- `deploy/docker-compose.yml` — perfis: postgres, meilisearch, redis, imgproxy, api, worker
- `deploy/Caddyfile` — headers de segurança (§7.3)
- `deploy/.env.example` — variáveis de ambiente
- `api/openapi.yaml` — esqueleto da API
- `.github/workflows/ci.yml` — lint/vet/build/test + gosec SAST
- `LICENSE` — AGPL-3.0
- `README.md`, `CONTRIBUTING.md`, `CODE_OF_CONDUCT.md`, `SECURITY.md`
- `.gitignore` — ambiente, segredos, storage local, logs, node_modules, .next, IDE

### Fase 1 — Autenticação e Catálogo (validadas E2E)
- Deps: `golang.org/x/crypto` (argon2id), `github.com/gorilla/sessions`
- `internal/services/auth.go` — hash argon2id, `Register`, `Login`, sessão cookie httpOnly, `SaveSession`/`ClearSession`/`UserIDFromSession`
- `internal/adapters/httpapi/auth.go` — handlers `POST /auth/register`, `POST /auth/login`, `POST /auth/logout`, `GET /auth/me`
- `internal/services/series.go` — `SeriesService`: `Create`, `List`, `Get`, `Update`, `Delete` (tabela `series`)
- `internal/services/chapter.go` — `ChapterService`: `Create`, `List`, `Get`, `Update`, `Delete` (tabela `chapters`)
- `internal/adapters/httpapi/catalog.go` — handlers `GET/POST/PUT/DELETE /series`, `GET/POST/PUT/DELETE /chapters`
- `cmd/api/main.go` — abre conexão com Postgres (`DATABASE_URL`), registra rotas de auth + catálogo
- `internal/services/auth_test.go` — teste de hash/verify
- Validação E2E contra Postgres ao vivo: registro 201 → login 200 → `/auth/me` 200 → logout 204 → `/auth/me` 401
- Validação E2E catálogo contra Postgres ao vivo: criar série 201 → listar 200 → criar capítulo 201 → listar 200 → obter 200 → atualizar 200 → deletar capítulo 204 → deletar série 204

### Não incluído (escopo deliberado / YAGNI)
- `apps/web/` (Next.js vazio — Fase 1 posterior)
- Implementação real dos adaptadores s3/postgres/auth (marcados `errNotImplemented`)
- sqlc, oauth, meilisearch/redis/imgproxy (desativados, rodam no WSL)
- Testes de integração com banco real

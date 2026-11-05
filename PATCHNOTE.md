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
- Adaptadores: `storage/local` (funcional), `storage/s3` (funcional, aws-sdk-go-v2), `search/postgres` (stub), `cache/memory` (funcional), `auth/builtin` (stub)
- `cmd/api/main.go` — servidor chi com `/healthz`, middleware (request ID, real IP, recovery, logger, timeout), desligamento graceful
- `cmd/worker/main.go` — entrypoint do worker
- `migrations/0001_initial/{up,down}.sql` — schema inicial: users, groups, permissions/roles/role_permissions/user_roles/group_members/user_overrides (sistema de permissão atômica §4.2), series, chapters (access/public_at/storage_key §5.4), vip_plans/memberships/vouchers, audit_log (+ índices). Reversível.
- `migrations/0002_add_storage_key_chapters/{up,down}.sql` — adiciona `storage_key` à tabela `chapters` (reversível)
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

### Storage S3-compatível primário
- `STORAGE_DRIVER` padrão agora é `s3` (era `local`)
- `internal/adapters/storage/s3/s3.go` — adapter S3 funcional com `aws-sdk-go-v2` (v1.47.5+): `Put`, `Get`, `Delete`, `Presign` (PUT/GET), `List`; suporte a endpoint customizado, `S3_FORCE_PATH_STYLE`, credenciais estáticas ou IAM role
- `internal/config/config.go` — validação: `S3_BUCKET` obrigatório quando `STORAGE_DRIVER=s3`
- `internal/adapters/storage/local/local.go` — mantido como fallback/opcional (`STORAGE_DRIVER=local`)
- `deploy/docker-compose.yml` — perfil `s3` com MinIO (exemplo de desenvolvimento; produção pode usar R2, B2, Garage ou SeaweedFS)
- `deploy/.env.example` — `STORAGE_DRIVER=s3`, variáveis S3 documentadas

### Upload de mídia direto para storage (presigned URL)
- `internal/services/media.go` — `MediaService`: `PresignUpload` (gera presigned URL PUT + storage_key), `CreateUpload` (cria capítulo em estado `processing`), `ValidateMagicBytes` (ZIP, JPEG, PNG, WebP)
- `internal/adapters/httpapi/media.go` — handlers `POST /uploads/presign` e `POST /uploads`, com autenticação via sessão cookie
- `cmd/api/main.go` — registra rotas de upload quando `DATABASE_URL` definida
- Fluxo: navegador faz PUT direto para `upload_url` (presigned S3 — bytes não passam pelo Go), depois `POST /uploads` cria capítulo em `processing`
- Validação E2E contra Postgres ao vivo (storage local): registro 201 → login 200 → criar série 201 → presign 200 → criar upload 201 → capítulo em processing 200

### Worker de mídia (concluído)
- `cmd/worker/main.go` — worker com polling em `chapters WHERE state='processing'` (FOR UPDATE SKIP LOCKED para evitar processamento duplicado)
- Extrai ZIP, ordena páginas, grava cada página no storage com chave `pages/{series_id}/{chapter_number}/{page}_{sha256}.{ext}` (dedup por hash de conteúdo)
- Publica capítulo (`state='published'`) após gravar todas as páginas
- Modo imgproxy padrão: não gera variantes (imgproxy faz isso em tempo de resposta); modo static preparo para variantes futura
- Validação E2E contra Postgres ao vivo: ZIP com 3 páginas → worker processa → capítulo `processing` → `published`

### Leitor de capítulos (Fase 1 MVP)
- `internal/services/image.go` — `ImageURLBuilder` com dois modos:
  - `static`: URL direta do S3/CDN com chave pré-renderizada (`pages/{series}/{chapter}/{page}.{variant}.{width}w.jpg`)
  - `imgproxy`: URL assinada HMAC-SHA256 com expiração curta (15 min)
- `internal/services/page.go` — `PageService`: `List(ctx, chapterID)` lista páginas ordenadas por número
- `internal/services/access.go` — `AccessResolver`: resolver único `CanRead` (seção 5.2 do roadmap)
  - Staff/editor vê sem ser VIP
  - `public`: sempre acessível
  - `early`: VIP lê imediatamente; demais a partir de `public_at`
  - `vip_only`: apenas VIPs (ou a partir de `public_at` opcional)
- `migrations/0003_add_pages_table/{up,down}.sql` — tabela `pages` (chapter_id, number, storage_key, width, height)
- `internal/adapters/httpapi/reader.go` — handler `GET /chapters/{id}/pages` com validação de CanRead
- `internal/adapters/httpapi/catalog.go` — handler `GET /chapters/lookup?series_slug=X&number=Y` para resolver chapter_id
- `internal/services/chapter.go` — `ChapterService.LookupBySeriesAndNumber`
- `cmd/api/main.go` — registra rotas de leitor e lookup quando `DATABASE_URL` definida
- `deploy/.env.example` — variáveis IMGPROXY_BASE_URL, IMGPROXY_KEY, IMGPROXY_SALT
- `apps/web/lib/api.ts` — client TS: `lookupChapter`, `getChapterPages`, `lookupChapterPages`
- `apps/web/app/manga/[slug]/capitulos/[n]/page.tsx` — páginas reais da API (substitui placeholders)
- Validação E2E contra Postgres ao vivo:
  - Capítulo public: 200 (retorna páginas com URLs)
  - Capítulo vip_only + não-VIP: 403
  - Capítulo early com public_at futuro + não-VIP: 403
  - Frontend Next.js build: `next build` passa (11 rotas, dinâmica `/manga/[slug]/capitulos/[n]` server-rendered)

### VIP e controle de acesso (Fase 1 MVP)
- `internal/services/vip.go` — `VipService`: `CreatePlan`, `ListPlans`, `GrantVip`, `RedeemVoucher`, `GetVipStatus`, `CreateVoucher`
- `internal/adapters/httpapi/vip.go` — handlers: `GET/POST /vip/plans`, `POST /vip/grant`, `POST /vip/voucher`, `POST /vip/vouchers`, `GET /vip/me`
- `internal/services/access.go` — `AccessResolver.CanRead` já integra vip_memberships (VIP lê capítulos vip_only e early imediatamente)
- `migrations/0004_vip_duration_text` — duracao de interval para text
- `migrations/0005_vip_duration_days` — duracao de text para integer (dias)
- `cmd/api/main.go` — registra rotas de VIP quando `DATABASE_URL` definida
- Validação E2E contra Postgres ao vivo:
  - Criar plano (duration: 30 dias): 201
  - Listar planos: 200
  - Conceder VIP: 204
  - Status VIP: has_active_vip=true, plan_name, expires_at, days_remaining, source

### Painel admin v1 (Fase 1 MVP)
- `internal/services/admin.go` — `AdminService`: `IsStaff`, `ListSeriesAdmin`, `ListUsersAdmin`, `UpdateChapterAdmin`
- `internal/adapters/httpapi/admin.go` — handlers: `GET /admin/series`, `GET /admin/users`, `PATCH /admin/chapters/{id}` (middleware de staff via `UserIDFromSession` + verificação de permissoes)
- `cmd/api/main.go` — registra rotas de admin quando `DATABASE_URL` definida
- `apps/web/app/(admin)/layout.tsx` — layout admin SEPARADO do website via Route Group (sidebar esquerda + header superior + área principal, cores slate/blue; sem navbar/footer do site público)
- `apps/web/app/(admin)/admin/page.tsx` — dashboard com cards de estatísticas (séries, usuários, VIP, capítulos), tabelas recentes, ações rápidas; conectado às APIs `/admin/series` e `/admin/users`
- URL mantida como `/admin` (Route Groups não afetam URLs)
- Validação E2E contra Postgres ao vivo:
  - Login como staff: 200
  - Listar séries (admin): 200
  - Listar usuários (admin): 200 (inclui roles, has_vip, vip_expires)
  - Middleware staff: nega acesso a não-staff com 403
  - `next build`: passa (11 rotas, dashboard server-rendered)

### Não incluído (escopo deliberado / YAGNI)
- Adapter search/postgres e auth/builtin ainda são stubs (`errNotImplemented`)
- sqlc, oauth, meilisearch/redis/imgproxy (desativados, rodam no WSL)
- Testes de integração com banco real
- Pagamento real (Stripe/Pix) — apenas concessão manual e voucher

### Scripts de desenvolvimento (raiz)
- `npm run dev` — inicia backend + frontend em paralelo (concurrently)
- `npm run dev:backend` — inicia API Go (scripts/dev-backend.ps1)
- `npm run dev:frontend` — inicia Next.js (apps/web)
- `npm run build:backend` — `go build ./cmd/...`
- `npm run build:frontend` — `cd apps/web && npm run build`
- `npm run build` — build completo (backend + frontend)
- `npm run test` — `go test ./...` + testes do frontend
- `npm run lint` — `go vet ./...` + lint do frontend
- `npm run db:migrate` — aplica migrations 0001, 0002, 0003, 0004, 0005 no Postgres
- `npm run db:reset` — drop schema público e reaplica migrations
- `.env.example` — variáveis de ambiente para desenvolvimento

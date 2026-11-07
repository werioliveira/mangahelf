# Changelog — MangaShelf

## Unreleased

### Changed
- Auth service: fixed Register/CreateOIDCUser/GetUserByEmail to use new schema columns (status, email_verified_at)
- Migration 0001_core: added seed data (roles, permissions, role_permissions, user_roles) for admin user
- Migration 0001_core: added missing tables (settings, vip_*, audit_log, bans), fixed JSON escaping in defaults
- Media service: cover uploads no longer require series_id/chapter_number
- Migration 0001_core: fixed broken defaults section, added admin user (admin@mangashelf.local / admin123)
- Media handler: fixed route registration conflict (chi duplicate /admin mount)
- package.json: restored dev/build/test/lint scripts, updated db:migrate to 0001_core
- Access resolver: fixed isStaffOrEditor to join group_members -> role_permissions directly
- Frontend chapters: added delete button with confirmation modal
- Modal component: added size prop (sm/md/lg/full); SeriesForm now uses full-screen modal
- SeriesForm: nav sidebar sticky, removed attention text, CoverUploader now shows local preview during upload
- SeriesService: full CRUD with all fields (type, status, demography, original_language, release_year, cover_storage_key, public_at)
- Media handler: routes moved under /admin, added kind and public_url to presign
- Reader endpoint now allows anonymous access to public chapters (access control handled by CanRead resolver)
- Admin handler: added POST/DELETE /admin/chapters and POST/PATCH/DELETE /admin/series endpoints

## 0.2.0 — 2026-11-05 · Auth & AuthZ Backend
### Adicionado
- internal/auth: sistema completo de autenticação
  - Hash Argon2id com semáforo de concorrência, política de senhas, NFKC normalization
  - Sessões SCS com store hashed (SHA256), cookie __Host-/ms_session httpOnly, SameSite=Lax, idle/absolute TTL, revogação
  - CSRF: Fetch Metadata + Origin allowlist + token por sessão (X-CSRF-Token)
  - Rate limiting com backoff progressivo, HMAC email hash
  - One-time tokens (verify_email/reset/change_email) com consumo atômico
  - MFA TOTP (AES-256-GCM), recovery codes HMAC-SHA256
  - WebAuthn/Passkeys (discoverable, sign count, clone detection)
  - Flows completos de auth (register/login/logout/forgot/reset/change/verify/reauth)
  - Auditoria append-only com trigger imutável
- internal/authz: autorização deny-by-default, resolver com deny vence allow, anti-escalação por posição, policy router com checagem em CI
- internal/audit: logging estruturado
- Schema auth (0009): users, sessions, user_sessions, auth_tokens, TOTP, recovery codes, WebAuthn, permissions/roles/role_permissions/user_roles/group_members/user_overrides, bans, audit_log
- Novas rotas auth/OIDC integradas

### Removido
- internal/services/auth.go, internal/services/auth/service.go, internal/services/auth_test.go (antigo)
- internal/services/oidc/auth.go (stub antigo)
- internal/adapters/httpapi/middleware/auth.go

### Atualizado
- go.mod/go.sum com novas dependências
- cmd/api/main.go: sessões SCS, CORS com OPTIONS, registro de rotas auth/admin/settings/vip/reader/catalog/media
- internal/services/auth_service.go: novo serviço integrado com SCS
- internal/adapters/httpapi/*: handlers atualizados
- deploy/.env.example: variáveis auth

### Validado
- go build ./... verde
- go test ./... verde

## 0.1.0 — 2026-11-04 · Fundação, Catálogo, Upload, Leitor, VIP, Admin v1

### Fundação
- Monorepo: /cmd/api, /cmd/worker, /internal, /apps/web, /migrations, /api, /deploy, /docs
- Config/ports/adapters base
- Migrations 0001-0008 iniciais
- Docker Compose perfis, Caddyfile com headers de segurança, OpenAPI, CI básico

### Autenticação/Catálogo (MVP)
- Auth com argon2id, sessões cookie httpOnly
- CRUD séries/capítulos
- Rotas auth/catalog

### Storage & Upload
- S3-compatível primário, worker processa ZIP, publicação automática
- ImageURLBuilder imgproxy/static

### Leitor
- Páginas por capítulo com AccessResolver único
- Rotas leitor/lookup

### VIP
- Planos, vouchers, concessão, status

### Painel Admin v1
- Rotas /admin/* protegidas por staff

### Frontend
- Next.js 15 App Router

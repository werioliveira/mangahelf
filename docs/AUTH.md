# MangaShelf: Guia de Implementação de Autenticação e Autorização (Go)

Documento complementar ao roadmap v1.1 (seções 4, 5, 6 e 7). Objetivo: descrever **exatamente** como implementar auth e authz no backend Go, de forma que toda decisão de segurança viva no servidor e o front (Next.js) seja apenas cliente.

> **Regra de ouro:** o Next.js nunca decide nada de segurança. Ele renderiza, repassa cookies e esconde botões por UX. Quem autentica, autoriza e audita é o Go.

---

## Índice

0. [Ajustes recomendados ao roadmap](#0-ajustes-recomendados-ao-roadmap)
1. [Modelo de ameaças](#1-modelo-de-ameaças)
2. [Estrutura de pacotes](#2-estrutura-de-pacotes)
3. [Schema SQL](#3-schema-sql)
4. [Senhas (argon2id)](#4-senhas-argon2id)
5. [Sessões (scs)](#5-sessões-scs)
6. [Pipeline de middlewares](#6-pipeline-de-middlewares)
7. [CSRF](#7-csrf)
8. [Rate limiting e proteção contra força bruta](#8-rate-limiting-e-proteção-contra-força-bruta)
9. [Tokens de uso único (e-mail, reset)](#9-tokens-de-uso-único)
10. [Fluxos: cadastro, login, logout, reset](#10-fluxos)
11. [MFA: TOTP, recovery codes e passkeys](#11-mfa)
12. [Step-up (reautenticação)](#12-step-up-reautenticação)
13. [OIDC e OAuth social](#13-oidc-e-oauth-social)
14. [Autorização: catálogo e resolver](#14-autorização-catálogo-e-resolver)
15. [Anti-escalação de privilégio](#15-anti-escalação-de-privilégio)
16. [Enforcement nas rotas (deny-by-default)](#16-enforcement-nas-rotas)
17. [Integração com VIP](#17-integração-com-vip)
18. [Auditoria](#18-auditoria)
19. [Banco: privilégios mínimos e RLS](#19-banco-privilégios-mínimos-e-rls)
20. [Integração com o Next.js](#20-integração-com-o-nextjs)
21. [Testes obrigatórios](#21-testes-obrigatórios)
22. [Ordem de implementação e Definition of Done](#22-ordem-de-implementação-e-definition-of-done)
23. [Anti-padrões proibidos](#23-anti-padrões-proibidos)

---

## 0. Ajustes recomendados ao roadmap

Ao detalhar a implementação, encontrei quatro pontos do roadmap que precisam de correção **antes** de codar:

| # | Trecho do roadmap | Problema | Correção adotada neste guia |
| --- | --- | --- | --- |
| 1 | Seção 4.2: o resolver retorna `eff == EffectAllow` assim que acha um override | Um override **allow** curto-circuita e ignora um **deny** de outra fonte, contradizendo "deny vence allow" | Avaliar **todos** os denys primeiro (seção 14.3) |
| 2 | Seção 7.3: "`SameSite=Strict` para ações do `/admin`" | Um cookie tem **um** valor de SameSite; não dá para ser Lax e Strict ao mesmo tempo | Manter `Lax` e exigir `Sec-Fetch-Site: same-origin` + header customizado em `/admin/*` (seção 7) |
| 3 | Seção 4.1: `scs` com store em Postgres | O `pgxstore` grava o **token bruto** da sessão como chave; um vazamento do banco (backup, SQLi) vira sequestro de sessões | Store que guarda `sha256(token)` (seção 5.2) |
| 4 | Seção 4.2: `user_overrides (user_id, permission_key, effect)` | Sem escopo, não dá para um override valer só em um grupo | Adicionar `scope_group_id` nullable (seção 3) |

Registre estes quatro pontos em ADRs.

---

## 1. Modelo de ameaças

| Ameaça | Controle neste guia |
| --- | --- |
| Credential stuffing / força bruta | Rate limit por IP, conta e par; backoff progressivo; MFA; checagem opcional de senha vazada |
| Enumeração de usuários | Respostas idênticas, hash "dummy", e-mails assíncronos |
| Roubo de sessão (XSS, vazamento de DB) | Cookie `__Host-` + `HttpOnly`; token hasheado no banco; sessão revogável |
| Fixação de sessão | `RenewToken` em todo upgrade de privilégio |
| CSRF | Origin + Fetch Metadata + token por sessão |
| Escalação de privilégio | Anti-escalação por `position`, deny vence allow, MFA nas permissões críticas |
| IDOR / enumeração entre grupos | Escopo derivado do recurso (nunca do body) + 404 em vez de 403 |
| Abuso de CPU via argon2 | Semáforo de concorrência + rate limit antes do hash |
| Tampering de log | `audit_log` append-only no nível do banco |
| Reuso de tokens (reset, TOTP) | Consumo atômico; `last_used_step` no TOTP |

---

## 2. Estrutura de pacotes

```
apps/api/internal/
├── auth/
│   ├── config.go          # TTLs, parâmetros argon2, chaves
│   ├── password.go        # Hasher argon2id
│   ├── tokens.go          # tokens de uso único
│   ├── session.go         # scs + store hasheado + Establish/Revoke
│   ├── csrf.go
│   ├── ratelimit.go
│   ├── totp.go
│   ├── webauthn.go
│   ├── service.go         # Register, Login, Logout, Reset...
│   ├── middleware.go      # Authenticate, RequireAuth, RequireRecentAuth
│   ├── clientip.go
│   ├── http.go            # handlers /auth/*
│   └── oidc/              # driver OIDC (Fase 2)
├── authz/
│   ├── catalog.go         # constantes + definições de permissões
│   ├── resolver.go        # Can()
│   ├── escalation.go      # regras anti-escalação
│   ├── policy.go          # Public/Authenticated/Require + registry
│   └── scope.go
└── audit/
    └── audit.go
```

**Dependências entre pacotes:** `authz` depende de `auth` apenas para o tipo `Principal`; `auth` **não** importa `authz`. Os demais módulos (`catalog`, `media`, `vip`...) dependem de `authz`, nunca de tabelas de permissão diretamente.

---

## 3. Schema SQL

```sql
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ───────── Identidade ─────────
CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    username          citext NOT NULL UNIQUE,
    password_hash     text,                      -- NULL para contas só-OIDC
    status            text NOT NULL DEFAULT 'active'
                      CHECK (status IN ('active','suspended','banned','pending_deletion')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE external_identities (
    provider   text NOT NULL,
    subject    text NOT NULL,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

-- ───────── Sessões ─────────
-- Tabela do scs (dados da sessão, chave = sha256(token)):
CREATE TABLE sessions (
    token  text PRIMARY KEY,
    data   bytea NOT NULL,
    expiry timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);

-- Metadados e revogação (controle nosso):
CREATE TABLE user_sessions (
    id             uuid PRIMARY KEY,             -- "sid" guardado na sessão
    user_id        uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    last_seen_at   timestamptz NOT NULL DEFAULT now(),
    auth_time      timestamptz NOT NULL,
    mfa_at         timestamptz,
    ip             inet,
    user_agent     text,
    revoked_at     timestamptz,
    revoked_reason text
);
CREATE INDEX user_sessions_user_idx ON user_sessions (user_id) WHERE revoked_at IS NULL;

-- ───────── Tokens de uso único ─────────
CREATE TABLE auth_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('verify_email','reset_password','change_email')),
    token_hash bytea NOT NULL UNIQUE,            -- sha256 do token
    meta       jsonb NOT NULL DEFAULT '{}',      -- ex.: novo e-mail
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- ───────── MFA ─────────
CREATE TABLE user_totp (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret_enc     bytea NOT NULL,               -- AES-256-GCM
    key_id         smallint NOT NULL,            -- versão da chave (rotação)
    confirmed_at   timestamptz,
    last_used_step bigint NOT NULL DEFAULT 0
);

CREATE TABLE user_recovery_codes (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hmac bytea NOT NULL,
    used_at   timestamptz
);

CREATE TABLE user_webauthn_credentials (
    id            bytea PRIMARY KEY,             -- credential ID
    user_id       uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    public_key    bytea NOT NULL,
    sign_count    bigint NOT NULL DEFAULT 0,
    transports    text[] NOT NULL DEFAULT '{}',
    aaguid        bytea,
    backup_eligible boolean NOT NULL DEFAULT false,
    backup_state    boolean NOT NULL DEFAULT false,
    name          text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    last_used_at  timestamptz
);

-- ───────── Autorização ─────────
CREATE TABLE permissions (
    key         text PRIMARY KEY,
    description text NOT NULL,
    scopes      text[] NOT NULL,                 -- {'global'} | {'global','group'}
    requires_mfa boolean NOT NULL DEFAULT false
);

CREATE TABLE roles (
    id       uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name     text NOT NULL UNIQUE,
    position int  NOT NULL,                      -- MAIOR número = MAIOR autoridade
    system   boolean NOT NULL DEFAULT false      -- 'owner' não pode ser apagado
);

CREATE TABLE role_permissions (
    role_id        uuid REFERENCES roles(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (                        -- escopo global
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id uuid REFERENCES roles(id) ON DELETE CASCADE,
    granted_by uuid REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE group_members (                     -- escopo por grupo
    group_id uuid NOT NULL,
    user_id  uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id  uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id, role_id)
);

CREATE TABLE user_overrides (
    user_id        uuid REFERENCES users(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    scope_group_id uuid,                         -- NULL = global
    effect         text NOT NULL CHECK (effect IN ('allow','deny')),
    reason         text,
    created_by     uuid REFERENCES users(id),
    expires_at     timestamptz,
    PRIMARY KEY (user_id, permission_key, scope_group_id)
);
-- PK com NULL não unifica em Postgres; use índice único com COALESCE:
CREATE UNIQUE INDEX user_overrides_uniq
    ON user_overrides (user_id, permission_key,
                       COALESCE(scope_group_id, '00000000-0000-0000-0000-000000000000'));

CREATE TABLE bans (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id),
    reason     text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    revoked_at timestamptz
);

-- ───────── Auditoria (append-only) ─────────
CREATE TABLE audit_log (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    actor_id    uuid,
    action      text NOT NULL,                   -- 'auth.login.success', 'role.assign'...
    target_type text,
    target_id   text,
    before      jsonb,
    after       jsonb,
    ip          inet,
    user_agent  text,
    request_id  text
);
CREATE INDEX audit_log_actor_idx  ON audit_log (actor_id, at DESC);
CREATE INDEX audit_log_action_idx ON audit_log (action, at DESC);

CREATE OR REPLACE FUNCTION audit_log_immutable() RETURNS trigger AS $$
BEGIN RAISE EXCEPTION 'audit_log is append-only'; END; $$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_immutable();
```

---

## 4. Senhas (argon2id)

### 4.1 Política

- Mínimo **12** caracteres, máximo **256** bytes (evita DoS por entrada gigante).
- **Sem regras de composição** ("1 maiúscula, 1 símbolo"); elas pioram a qualidade das senhas.
- Aceitar qualquer Unicode; normalizar com **NFKC** (`golang.org/x/text/unicode/norm`) antes do hash.
- Bloquear as ~10 mil senhas mais comuns (lista embutida no binário via `go:embed`).
- Opcional por flag (`AUTH_HIBP_CHECK=true`): checagem k-anonymity no Have I Been Pwned (envia só os 5 primeiros caracteres do SHA-1). Se a API falhar, **fail-open** (não bloqueia cadastro).
- Senha não pode conter o e-mail nem o username.

### 4.2 Implementação

```go
package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

type Argon2Params struct {
	Memory  uint32 // KiB (65536 = 64 MiB)
	Time    uint32
	Threads uint8
	SaltLen uint32
	KeyLen  uint32
}

// Ponto de partida; MEÇA no hardware do operador e registre em ADR.
// Alvo: 150–400 ms por hash na VPS de produção.
var DefaultArgon2 = Argon2Params{Memory: 64 * 1024, Time: 3, Threads: 2, SaltLen: 16, KeyLen: 32}

var ErrHashBusy = errors.New("auth: hasher busy")

type Hasher struct {
	p   Argon2Params
	sem chan struct{} // limita hashes simultâneos (cada um usa p.Memory de RAM)
}

func NewHasher(p Argon2Params, maxConcurrent int) *Hasher {
	return &Hasher{p: p, sem: make(chan struct{}, maxConcurrent)}
}

func (h *Hasher) acquire(ctx context.Context) error {
	select {
	case h.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ErrHashBusy // o handler traduz para 503 + Retry-After
	}
}
func (h *Hasher) release() { <-h.sem }

var b64 = base64.RawStdEncoding

func (h *Hasher) Hash(ctx context.Context, password string) (string, error) {
	if err := h.acquire(ctx); err != nil {
		return "", err
	}
	defer h.release()

	salt := make([]byte, h.p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, h.p.Time, h.p.Memory, h.p.Threads, h.p.KeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.p.Memory, h.p.Time, h.p.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify retorna ok e se o hash deve ser recalculado (parâmetros antigos).
func (h *Hasher) Verify(ctx context.Context, password, encoded string) (ok, needsRehash bool, err error) {
	parts := strings.Split(encoded, "$") // ["", "argon2id", "v=19", "m=..,t=..,p=..", salt, key]
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, false, errors.New("auth: invalid hash format")
	}
	var v int
	var p Argon2Params
	if _, err := fmt.Sscanf(parts[2], "v=%d", &v); err != nil || v != argon2.Version {
		return false, false, errors.New("auth: unsupported argon2 version")
	}
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return false, false, err
	}
	// Defesa: o valor vem do banco, mas nunca confie cegamente (DoS por memória).
	if p.Memory > 256*1024 || p.Time > 10 || p.Threads > 16 {
		return false, false, errors.New("auth: argon2 params out of bounds")
	}
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, false, err
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil {
		return false, false, err
	}

	if err := h.acquire(ctx); err != nil {
		return false, false, err
	}
	defer h.release()

	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, uint32(len(want)))
	ok = subtle.ConstantTimeCompare(got, want) == 1
	needsRehash = p.Memory != h.p.Memory || p.Time != h.p.Time || p.Threads != h.p.Threads
	return ok, needsRehash, nil
}
```

**Regras:**

1. Se `needsRehash && ok`, regrave o hash novo no mesmo request de login.
2. **Pepper (opcional):** aplique `HMAC-SHA256(pepper, password)` antes do argon2 se quiser defesa extra contra vazamento só do banco. A chave vem de variável de ambiente e **nunca** do banco. Se adotar, é irreversível sem forçar reset geral; decida no dia 1.
3. Os parâmetros do argon2 entram em ADR e são revisados a cada release.

---

## 5. Sessões (scs)

### 5.1 Configuração

```go
func NewSessionManager(pool *pgxpool.Pool, cfg Config) *scs.SessionManager {
	sm := scs.New()
	sm.Store = newHashedStore(pgxstore.New(pool))

	sm.Lifetime    = cfg.SessionAbsoluteTTL // padrão 30d: teto absoluto
	sm.IdleTimeout = cfg.SessionIdleTTL     // padrão 7d: inatividade

	sm.Cookie.Name     = "__Host-ms_session" // exige Secure, Path=/, sem Domain
	sm.Cookie.Path     = "/"
	sm.Cookie.Secure   = cfg.CookieSecure    // true em produção (localhost funciona em dev)
	sm.Cookie.HttpOnly = true
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Persist  = true
	return sm
}
```

Cookie `__Host-` significa: sem atributo `Domain`, então o cookie vale **só** para o host exato. Sirva site e API no **mesmo host** (Caddy roteando `/api/*` para o Go), o que também elimina CORS (seção 20).

### 5.2 Store que guarda hash do token

```go
type hashedStore struct{ inner scs.CtxStore }

func newHashedStore(s scs.CtxStore) hashedStore { return hashedStore{inner: s} }

func keyOf(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (h hashedStore) FindCtx(ctx context.Context, t string) ([]byte, bool, error) {
	return h.inner.FindCtx(ctx, keyOf(t))
}
func (h hashedStore) CommitCtx(ctx context.Context, t string, b []byte, e time.Time) error {
	return h.inner.CommitCtx(ctx, keyOf(t), b, e)
}
func (h hashedStore) DeleteCtx(ctx context.Context, t string) error {
	return h.inner.DeleteCtx(ctx, keyOf(t))
}
// Métodos sem ctx exigidos por scs.Store: delegam com context.Background().
func (h hashedStore) Find(t string) ([]byte, bool, error) { return h.FindCtx(context.Background(), t) }
func (h hashedStore) Commit(t string, b []byte, e time.Time) error {
	return h.CommitCtx(context.Background(), t, b, e)
}
func (h hashedStore) Delete(t string) error { return h.DeleteCtx(context.Background(), t) }
```

### 5.3 O que vai dentro da sessão

Apenas: `sid` (UUID), `uid`, `auth_time`, `mfa_at`, `csrf`. **Nada de roles ou permissões** — são avaliadas ao vivo (seção 14), então mudar um cargo vale na hora, sem esperar a sessão expirar.

A revogação **não** depende de apagar linhas do `scs` (o token novo só é conhecido depois que a resposta é escrita). Em vez disso, cada sessão tem uma linha em `user_sessions` e o middleware rejeita sessões com `revoked_at` preenchido.

### 5.4 Estabelecer a sessão

Todo caminho de login (senha, MFA, passkey, OIDC) passa por **esta única função**:

```go
type Identity struct {
	UserID   uuid.UUID
	AuthTime time.Time
	MFA      bool // true se este login já incluiu segundo fator
}

func (s *SessionService) Establish(ctx context.Context, id Identity, ip netip.Addr, ua string) error {
	// 1. Anti-fixação: SEMPRE novo token ao elevar privilégio.
	if err := s.sm.RenewToken(ctx); err != nil {
		return err
	}
	sid := uuid.New()
	csrf, _ := randomToken(32)

	var mfaAt *time.Time
	if id.MFA {
		mfaAt = &id.AuthTime
	}
	if err := s.q.InsertUserSession(ctx, sid, id.UserID, id.AuthTime, mfaAt, ip, truncate(ua, 512)); err != nil {
		return err
	}

	// 2. Limite de sessões simultâneas (ex.: 10): revoga as mais antigas.
	if err := s.q.RevokeOldestBeyond(ctx, id.UserID, s.cfg.MaxSessionsPerUser, "session_limit"); err != nil {
		return err
	}

	s.sm.Put(ctx, "sid", sid.String())
	s.sm.Put(ctx, "uid", id.UserID.String())
	s.sm.Put(ctx, "auth_time", id.AuthTime.Unix())
	if mfaAt != nil {
		s.sm.Put(ctx, "mfa_at", mfaAt.Unix())
	}
	s.sm.Put(ctx, "csrf", csrf)
	return nil
}

func (s *SessionService) RevokeAll(ctx context.Context, userID uuid.UUID, exceptSID *uuid.UUID, reason string) error {
	return s.q.RevokeUserSessions(ctx, userID, exceptSID, reason)
}
```

### 5.5 Quando renovar/revogar

| Evento | Ação |
| --- | --- |
| Login, conclusão de MFA, passkey, OIDC | `RenewToken` (via `Establish`) |
| Troca de senha | `RevokeAll` exceto a sessão atual + `RenewToken` + e-mail de aviso |
| Reset de senha | `RevokeAll` (incluindo qualquer sessão); **não** faz auto-login |
| Ativar/desativar MFA | `RevokeAll` exceto a atual |
| Banimento / suspensão | `RevokeAll` imediato |
| Logout | `Destroy` + `revoked_at` da linha atual |

### 5.6 Timeouts por sensibilidade

- Sessão comum: idle 7d / absoluto 30d (configurável).
- Ações com permissão `requires_mfa` ou `/admin/*` sensível: exigem `mfa_at` **recente** (seção 12), p. ex. 30 min para `role.manage` e `instance.configure`.

---

## 6. Pipeline de middlewares

Ordem **importa** (de fora para dentro):

```go
r := chi.NewRouter()
r.Use(middleware.RequestID)
r.Use(clientip.Middleware(cfg.TrustedProxies, cfg.ClientIPHeader)) // define IP real
r.Use(logging.Middleware)                                          // slog estruturado
r.Use(middleware.Recoverer)                                        // pânico → 500 genérico
r.Use(maxBodyBytes(1 << 20))                                       // 1 MiB (upload usa presign)
r.Use(sm.LoadAndSave)                                              // carrega sessão do cookie
r.Use(auth.Authenticate(deps))                                     // resolve Principal (ou anônimo)
r.Use(auth.CSRF(cfg))                                              // valida origem/token em mutações
r.Use(ratelimit.Global(limiter))                                   // limite geral por IP/conta
// Em seguida, cada rota é registrada com uma Policy (seção 16)
```

### 6.1 Authenticate

```go
type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Status    string
	AuthTime  time.Time
	MFAAt     *time.Time
}

func (m *Middleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		sid := m.sm.GetString(ctx, "sid")
		if sid == "" {
			next.ServeHTTP(w, r) // anônimo
			return
		}
		row, err := m.q.SessionPrincipal(ctx, sid) // JOIN user_sessions + users
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			_ = m.sm.Destroy(ctx)
			next.ServeHTTP(w, r)
			return
		case err != nil:
			httpx.Internal(w, r, err) // FAIL-SAFE: erro ≠ anônimo silencioso
			return
		}
		if row.RevokedAt.Valid || row.Status != "active" || sessionExpired(row, m.cfg) {
			_ = m.sm.Destroy(ctx)
			next.ServeHTTP(w, r)
			return
		}
		m.touchLastSeen(ctx, row) // no máximo 1 UPDATE/min por sessão

		p := &Principal{UserID: row.UserID, SessionID: row.SID, Status: row.Status,
			AuthTime: row.AuthTime, MFAAt: nullTime(row.MFAAt)}
		next.ServeHTTP(w, r.WithContext(WithPrincipal(ctx, p)))
	})
}
```

O status do usuário é lido do banco **a cada request** (consulta por PK, barata). Assim, ban e suspensão têm efeito imediato, sem cache.

---

## 7. CSRF

Três camadas combinadas:

1. **Fetch Metadata:** para métodos não seguros, rejeitar se `Sec-Fetch-Site` existir e não for `same-origin` (ou `none`).
2. **Origin:** se `Origin` existir, deve estar na allowlist (`WEB_ORIGIN`).
3. **Token por sessão:** para requests autenticados, header `X-CSRF-Token` deve ser igual ao `csrf` da sessão (comparação em tempo constante).

Para mutações anônimas (login, cadastro, reset), exija `Content-Type: application/json` (isso força preflight em cross-origin, e como não há CORS liberado, o browser bloqueia) e valide Origin/Fetch Metadata.

```go
func CSRF(cfg Config, sm *scs.SessionManager) func(http.Handler) http.Handler {
	allowed := cfg.AllowedOrigins // map[string]struct{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet, http.MethodHead, http.MethodOptions:
				next.ServeHTTP(w, r)
				return
			}
			if s := r.Header.Get("Sec-Fetch-Site"); s != "" && s != "same-origin" && s != "none" {
				httpx.Forbidden(w, "csrf_fetch_site")
				return
			}
			if o := r.Header.Get("Origin"); o != "" {
				if _, ok := allowed[o]; !ok {
					httpx.Forbidden(w, "csrf_origin")
					return
				}
			}
			if strings.HasPrefix(r.URL.Path, "/admin/") {
				// Substitui o "SameSite=Strict" do roadmap: exige same-origin estrito + header custom.
				if r.Header.Get("Sec-Fetch-Site") == "cross-site" || r.Header.Get("X-MS-Admin") != "1" {
					httpx.Forbidden(w, "csrf_admin")
					return
				}
			}
			if _, authed := PrincipalFrom(r.Context()); authed {
				want := sm.GetString(r.Context(), "csrf")
				got := r.Header.Get("X-CSRF-Token")
				if want == "" || subtle.ConstantTimeCompare([]byte(want), []byte(got)) != 1 {
					httpx.Forbidden(w, "csrf_token")
					return
				}
			} else if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
				httpx.Forbidden(w, "csrf_content_type")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

`GET /api/auth/csrf` devolve o token da sessão atual. O client TS gerado o injeta em toda mutação.

**Regra:** **nenhuma** mutação é feita via `GET`. Logout é `POST`.

---

## 8. Rate limiting e proteção contra força bruta

### 8.1 IP real

Nunca confie em `X-Forwarded-For` direto. Só leia o header se o `RemoteAddr` estiver em `TRUSTED_PROXIES` (Caddy local e/ou faixas da Cloudflare):

```go
func ClientIP(r *http.Request, trusted []netip.Prefix, header string) netip.Addr {
	remote, _ := netip.ParseAddrPort(r.RemoteAddr)
	ip := remote.Addr()
	for _, p := range trusted {
		if p.Contains(ip) {
			if v := r.Header.Get(header); v != "" { // "CF-Connecting-IP" ou "X-Real-IP"
				if parsed, err := netip.ParseAddr(strings.TrimSpace(v)); err == nil {
					return parsed
				}
			}
			break
		}
	}
	return ip
}
```

### 8.2 Interface e regras

```go
type Rule struct {
	Limit  int
	Window time.Duration
}
type Decision struct {
	Allowed    bool
	RetryAfter time.Duration
}
type Limiter interface {
	// Allow consome 1 unidade da chave. Implementação sobre a interface Cache
	// (INCR com TTL): memory por padrão; use redis se houver mais de 1 réplica da API.
	Allow(ctx context.Context, key string, r Rule) (Decision, error)
}
```

| Endpoint | Chaves | Limite |
| --- | --- | --- |
| `POST /auth/login` | `ip`, `email_hash`, `ip+email_hash` | 20 / 10 / 5 por 15 min |
| `POST /auth/register` | `ip` | 5 / hora |
| `POST /auth/password/forgot` | `ip`, `email_hash` | 5 / 3 por hora |
| `POST /auth/mfa/verify` | `session`, `user` | 5 / 5 min; 10 / 15 min |
| `POST /auth/email/resend` | `user`, `ip` | 3 / hora |
| API geral autenticada | `user_id`; anônima: `ip` | definido por rota |

### 8.3 Regras de projeto

- **Verifique o limiter ANTES de calcular o hash** (argon2 é o recurso caro).
- **Não faça lockout permanente por conta**: isso permite que um atacante bloqueie a vítima. Use **backoff progressivo**: depois do limite, `Retry-After = min(2^(n-limite) s, 15 min)`.
- Após N falhas na mesma conta (ex.: 5), exija captcha (flag opcional) e envie e-mail de alerta ("tentativas de login suspeitas").
- `email_hash` = `HMAC-SHA256(server_key, email_normalizado)`, para não guardar e-mail em cache/logs.
- Falha do limiter (ex.: Redis fora): **fail-closed** em login/MFA/reset; **fail-open** só em leituras públicas.
- Resposta `429` com `Retry-After` e corpo genérico.

---

## 9. Tokens de uso único

Usados para verificação de e-mail, reset de senha e troca de e-mail.

```go
func NewToken() (plain string, hash []byte, err error) {
	b := make([]byte, 32) // 256 bits
	if _, err = rand.Read(b); err != nil {
		return
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, sum[:], nil
}
```

- Só o **hash** vai ao banco. O token puro existe apenas no e-mail.
- TTLs: verificação 24 h; reset **30 min**; troca de e-mail 1 h.
- Criar um novo token do mesmo propósito **invalida** os anteriores.
- **Consumo atômico** (evita corrida e reuso):

```sql
-- name: ConsumeToken :one
UPDATE auth_tokens
   SET used_at = now()
 WHERE token_hash = $1
   AND purpose    = $2
   AND used_at IS NULL
   AND expires_at > now()
RETURNING user_id, meta;
```

- O link do e-mail aponta para uma página do front (`/auth/reset#token=...`). Use **fragmento** (`#`) para o token não ir em Referer nem em logs de servidor; a página lê o fragmento e faz `POST` ao backend. Defina `Referrer-Policy: no-referrer` nessas páginas.
- E-mails são enviados por **job (River)**, nunca inline no request (evita diferença de tempo entre "usuário existe" e "não existe").

---

## 10. Fluxos

### 10.1 Cadastro

1. Valida entrada: e-mail com `net/mail.ParseAddress` (exige `addr.Address == input`, sem display name), ≤ 254 chars, normaliza (`strings.ToLower`, NFKC, trim). Username por regex `^[a-z0-9_]{3,24}$` com lista de reservados (`admin`, `root`, `support`...).
2. Valida a política de senha (seção 4.1).
3. Rate limit por IP.
4. **Resposta idêntica** exista ou não o e-mail: `202 {"status":"check_email"}`.
   - E-mail novo: cria usuário (`status=active`, `email_verified_at=NULL`) + token `verify_email` + job de e-mail.
   - E-mail existente: **não cria nada**; envia e-mail "você já tem conta" (com link para login/reset).
5. Calcule o hash da senha **nos dois caminhos** (ou espere um tempo equivalente) para não vazar existência pelo tempo.
6. Conta com e-mail não verificado pode logar, mas fica **bloqueada para ações sensíveis** (comentar, seguir, upload...). A checagem é uma permissão/condição única (`auth.RequireVerifiedEmail`), não `if` espalhado.
7. Primeiro dono **nunca** nasce de cadastro público: apenas via `cli bootstrap-owner`.

### 10.2 Login (senha)

```go
func (s *Service) Login(ctx context.Context, in LoginInput, ip netip.Addr, ua string) (*LoginResult, error) {
	email := normalizeEmail(in.Email)
	eh := s.hmacEmail(email)

	// 1. Rate limit ANTES de qualquer custo.
	if d, _ := s.limiter.AllowAll(ctx, loginKeys(ip, eh)); !d.Allowed {
		return nil, RateLimited{RetryAfter: d.RetryAfter}
	}

	// 2. Busca usuário; se não existir (ou for só-OIDC), usa hash dummy do MESMO custo.
	u, err := s.users.ByEmail(ctx, email)
	hash := s.dummyHash
	if err == nil && u.PasswordHash != nil {
		hash = *u.PasswordHash
	} else if err != nil && !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	ok, rehash, verr := s.hasher.Verify(ctx, normalizePassword(in.Password), hash)
	if verr != nil {
		return nil, verr // ErrHashBusy → 503
	}
	if !ok || u == nil || u.PasswordHash == nil {
		s.audit.Log(ctx, "auth.login.failure", audit.Actor(nil), audit.Meta{"email_hash": hex(eh), "ip": ip})
		return nil, ErrInvalidCredentials // mesma mensagem e mesmo status sempre
	}

	// 3. Só DEPOIS de senha correta revelamos estados da conta.
	switch u.Status {
	case "banned", "suspended":
		s.audit.Log(ctx, "auth.login.blocked", audit.Actor(&u.ID))
		return nil, ErrAccountBlocked
	}
	if rehash {
		_ = s.users.SetPasswordHash(ctx, u.ID, s.mustHash(ctx, in.Password))
	}

	// 4. Segundo fator?
	if s.mfa.Enrolled(ctx, u.ID) {
		return s.beginMFAChallenge(ctx, u.ID) // sessão "pendente", sem "sid"
	}

	// 5. Sessão completa.
	if err := s.sessions.Establish(ctx, Identity{UserID: u.ID, AuthTime: time.Now().UTC()}, ip, ua); err != nil {
		return nil, err
	}
	s.audit.Log(ctx, "auth.login.success", audit.Actor(&u.ID))
	return &LoginResult{Done: true}, nil
}
```

**Estado intermediário de MFA:** guarde na sessão `mfa_pending_uid` + `mfa_pending_until` (5 min). **Não** coloque `sid` até o segundo fator ser aprovado, portanto o middleware `Authenticate` continua tratando como anônimo.

### 10.3 Logout

`POST /auth/logout` → marca `user_sessions.revoked_at`, `sm.Destroy(ctx)`, `204`. Também ofereça `POST /auth/sessions/{id}/revoke` e "sair de todos os dispositivos".

### 10.4 Esqueci a senha

1. `POST /auth/password/forgot {email}` → sempre `202` genérico, após rate limit.
2. Se o usuário existe: cria token `reset_password` (30 min) e enfileira e-mail.
3. `POST /auth/password/reset {token, new_password}`:
   - `ConsumeToken` atômico; valida política da senha.
   - Atualiza hash; **`RevokeAll`** sessões; envia e-mail "sua senha foi alterada".
   - **Não** faz login automático. Se há MFA, ele continua exigido no próximo login.
4. Se o usuário tem MFA, o reset **não** desativa o MFA.

### 10.5 Troca de senha / e-mail (logado)

- Exige `RequireRecentAuth` (seção 12) e a **senha atual**.
- Troca de e-mail: token enviado ao **novo** endereço + aviso ao antigo; só efetiva após confirmação.
- Revoga as demais sessões.

---

## 11. MFA

### 11.1 TOTP

Biblioteca: `github.com/pquerna/otp/totp`.

**Armazenamento do segredo:** cifrado com AES-256-GCM usando `AUTH_ENCRYPTION_KEYS` (lista versionada: `1:base64key,2:base64key`), gravando `key_id` junto, o que permite rotação (descriptografa com a chave antiga, regrava com a nova).

```go
func (s *TOTPService) Encrypt(plain []byte) (ct []byte, keyID int16, err error) {
	k := s.keys.Current()
	block, _ := aes.NewCipher(k.Key) // 32 bytes
	gcm, _ := cipher.NewGCM(block)
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return
	}
	ct = gcm.Seal(nonce, nonce, plain, []byte("totp:v1")) // AAD fixo
	return ct, k.ID, nil
}
```

**Enrolamento:**

1. `POST /auth/mfa/totp/begin` (exige recent auth) → gera segredo, grava com `confirmed_at = NULL`, devolve `otpauth://` URL (QR renderizado no front).
2. `POST /auth/mfa/totp/confirm {code}` → valida; marca `confirmed_at`; gera **10 recovery codes** (exibidos uma única vez); `RevokeAll` exceto sessão atual.

**Validação com proteção contra replay:**

```go
func (s *TOTPService) Verify(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	rec, err := s.q.TOTP(ctx, userID)
	if err != nil {
		return false, err
	}
	secret, err := s.Decrypt(rec.SecretEnc, rec.KeyID)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	opts := totp.ValidateOpts{Period: 30, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}

	for _, off := range []int{-1, 0, 1} { // tolerância de ±30 s
		t := now.Add(time.Duration(off) * 30 * time.Second)
		want, err := totp.GenerateCodeCustom(string(secret), t, opts)
		if err != nil {
			return false, err
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			step := t.Unix() / 30
			// Atômico: só aceita se o step for MAIOR que o último usado.
			n, err := s.q.AdvanceTOTPStep(ctx, userID, step) // UPDATE ... WHERE last_used_step < $2
			return n == 1, err
		}
	}
	return false, nil
}
```

### 11.2 Recovery codes

- 10 códigos de 10 caracteres base32 (≈50 bits cada), exibidos uma vez.
- Armazene `HMAC-SHA256(server_key, code)` (alta entropia dispensa argon2).
- Uso único (`used_at`); regenerar invalida todos os anteriores. Cada uso é auditado e notificado por e-mail.

### 11.3 Passkeys (WebAuthn)

Biblioteca: `github.com/go-webauthn/webauthn`.

- `RPID` = domínio do site; `RPOrigins` = origens do front. Config validada no boot.
- Fluxos: `BeginRegistration/FinishRegistration` e `BeginDiscoverableLogin/FinishDiscoverableLogin`.
- O `SessionData` do desafio fica **no servidor** (sessão `scs`, TTL 5 min), nunca em cookie legível ou no client.
- Em cada login, verifique e atualize `sign_count`. Se vier **menor ou igual** ao armazenado (e ≠ 0), registre `auth.passkey.clone_suspected` e bloqueie.
- Guarde `backup_eligible/backup_state` para saber se a passkey é sincronizada (afeta política de "MFA forte").
- Login por passkey conta como `MFA=true` (posse + verificação do usuário) quando `UserVerification: required`.
- Limite de passkeys por conta (ex.: 10); permitir nomear e remover (com recent auth).

### 11.4 Obrigatoriedade

Contas com qualquer permissão marcada `requires_mfa` (no mínimo `role.manage` e `instance.configure`) **precisam** ter MFA ativo e verificado na sessão. Sem isso, o resolver **nega** (seção 14.4) e a API devolve `403` com código `mfa_enrollment_required` para o front direcionar ao cadastro.

---

## 12. Step-up (reautenticação)

Ações sensíveis exigem prova recente, mesmo com sessão válida:

```go
func RequireRecentAuth(maxAge time.Duration, needMFA bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := PrincipalFrom(r.Context())
			if !ok {
				httpx.Unauthorized(w)
				return
			}
			ref := p.AuthTime
			if needMFA {
				if p.MFAAt == nil {
					httpx.ErrorCode(w, http.StatusForbidden, "mfa_required")
					return
				}
				ref = *p.MFAAt
			}
			if time.Since(ref) > maxAge {
				httpx.ErrorCode(w, http.StatusUnauthorized, "reauth_required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
```

`POST /auth/reauth {password | totp | passkey}` atualiza `auth_time` / `mfa_at` na linha de `user_sessions` e na sessão (**sem** novo `sid`, mas com `RenewToken`).

| Ação | Janela | Exige MFA |
| --- | --- | --- |
| Trocar senha / e-mail / MFA | 10 min | se tiver MFA |
| Gerenciar cargos e permissões | 30 min | sim |
| Config da instância, VIP manual em massa, ban | 30 min | sim |
| Transferir posse (owner) | 5 min | sim |

---

## 13. OIDC e OAuth social

Driver `AUTH_DRIVER=oidc` (e `goth` para social no builtin). **Autorização continua 100% local**; o IdP só prova identidade.

- Libs: `github.com/coreos/go-oidc/v3` + `golang.org/x/oauth2`.
- **Authorization Code + PKCE (S256)** sempre; `state` e `nonce` aleatórios guardados **no servidor** (sessão), TTL 10 min, uso único.
- Validar o ID Token: assinatura (JWKS do issuer), `iss`, `aud`, `exp`, `nonce`.
- `redirect_uri` fixo e registrado; nunca derivado do request.
- **Não vincule contas por e-mail automaticamente.** Casos:
  - `(provider, subject)` já existe → login nessa conta.
  - Não existe e o e-mail **não** está em uso → cria conta (se `email_verified=true` no token; senão exige verificação própria).
  - E-mail já existe em conta local → exige que o usuário prove posse da conta local (login com senha/MFA, ou token por e-mail) antes de vincular. Vincular por e-mail sem prova permite *account takeover* via IdP fraco.
- Após validar, chame o mesmo `SessionService.Establish`. O MFA do IdP só conta se o `amr`/`acr` estiver configurado como confiável (`OIDC_TRUST_AMR=true`); por padrão, o MangaShelf aplica o próprio MFA.
- Desvincular um provedor exige que a conta tenha outro método de login.

---

## 14. Autorização: catálogo e resolver

### 14.1 Catálogo de permissões em código

Permissões são **constantes tipadas**. String literal em chamada de `Can` é proibido (seção 16.3).

```go
package authz

type Perm string

const (
	AdminView         Perm = "admin.view"
	SeriesEdit        Perm = "series.edit"
	ChapterPublish    Perm = "chapter.publish"
	ChapterSetAccess  Perm = "chapter.set_access"
	ContentReadVIP    Perm = "content.read_vip"
	MediaManage       Perm = "media.manage"
	UserView          Perm = "user.view"
	UserBan           Perm = "user.ban"
	RoleManage        Perm = "role.manage"
	RoleAssign        Perm = "role.assign"
	VIPManage         Perm = "vip.manage"
	ReportHandle      Perm = "report.handle"
	AuditView         Perm = "audit.view"
	InstanceConfigure Perm = "instance.configure"
	SystemView        Perm = "system.view"
)

type Def struct {
	Key         Perm
	Description string
	Scopes      []ScopeKind // onde faz sentido
	RequiresMFA bool
}

var Catalog = []Def{
	{ChapterPublish, "Publicar capítulos", []ScopeKind{Global, Group}, false},
	{RoleManage, "Criar/editar cargos", []ScopeKind{Global}, true},
	{InstanceConfigure, "Configurar a instância", []ScopeKind{Global}, true},
	// ...
}
```

**Sincronização no boot:** `authz.SyncCatalog(ctx, db)` faz `INSERT ... ON CONFLICT DO UPDATE` das definições e **loga alerta** para chaves no banco que não existem no código. O owner recebe automaticamente todas as permissões novas na migração.

### 14.2 Escopo

```go
type ScopeKind int

const (
	Global ScopeKind = iota
	Group
)

type Scope struct {
	Kind    ScopeKind
	GroupID uuid.UUID // válido se Kind == Group
}

func GlobalScope() Scope               { return Scope{Kind: Global} }
func GroupScope(id uuid.UUID) Scope    { return Scope{Kind: Group, GroupID: id} }
```

### 14.3 Algoritmo do resolver (corrige o item 1 da seção 0)

Ordem de avaliação:

1. Usuário inativo/banido → **nega**.
2. Se a permissão `requires_mfa` e a sessão não tem MFA verificado → **nega**.
3. Reúne todas as fontes aplicáveis ao escopo (uma única query).
4. **Qualquer `deny`** (override global ou do grupo) → **nega**.
5. Qualquer `allow` por override → **permite**.
6. Cargo **do grupo** (se escopo Group) concede → **permite**.
7. Cargo **global** concede → **permite**.
8. Padrão → **nega** (fail-safe).

```sql
-- name: EffectiveGrants :many
-- $1 = user_id, $2 = group_id (NULL para escopo global)
SELECT 'role_global'::text AS source, rp.permission_key, 'allow'::text AS effect
  FROM user_roles ur
  JOIN role_permissions rp ON rp.role_id = ur.role_id
 WHERE ur.user_id = $1
UNION ALL
SELECT 'role_group', rp.permission_key, 'allow'
  FROM group_members gm
  JOIN role_permissions rp ON rp.role_id = gm.role_id
 WHERE gm.user_id = $1 AND gm.group_id = $2
UNION ALL
SELECT 'override', o.permission_key, o.effect
  FROM user_overrides o
 WHERE o.user_id = $1
   AND (o.scope_group_id IS NULL OR o.scope_group_id = $2)
   AND (o.expires_at IS NULL OR o.expires_at > now());
```

```go
type Subject struct {
	UserID      uuid.UUID
	Active      bool
	MFAVerified bool // vem do Principal: MFAAt dentro da janela
}

type Resolver struct {
	q       Queries
	defs    map[Perm]Def
}

func (r *Resolver) Can(ctx context.Context, s Subject, perm Perm, sc Scope) bool {
	def, known := r.defs[perm]
	if !known || !s.Active {
		return false // permissão desconhecida = nega
	}
	if def.RequiresMFA && !s.MFAVerified {
		return false
	}
	if sc.Kind == Group && !def.allowsScope(Group) {
		return false
	}

	grants, err := r.grantsFor(ctx, s.UserID, sc) // memoizado por request
	if err != nil {
		slog.ErrorContext(ctx, "authz: grants lookup failed", "err", err)
		return false // FAIL-CLOSED
	}

	var allow, groupRole, globalRole bool
	for _, g := range grants {
		if g.Key != perm {
			continue
		}
		switch {
		case g.Source == "override" && g.Effect == "deny":
			return false // deny vence tudo, em qualquer ordem
		case g.Source == "override" && g.Effect == "allow":
			allow = true
		case g.Source == "role_group":
			groupRole = true
		case g.Source == "role_global":
			globalRole = true
		}
	}
	return allow || groupRole || globalRole
}
```

**Memoização por request:** guarde o resultado de `grantsFor` num `map` dentro do `context` (uma query por `(user, escopo)` por request). **Não** use cache entre requests no início: mudança de cargo precisa valer na hora. Se medir necessidade, cache em memória com TTL ≤ 30 s e invalidação explícita em toda mutação de cargo/override/membership.

### 14.4 Subject a partir do Principal

```go
func SubjectFrom(p *auth.Principal, mfaWindow time.Duration) Subject {
	return Subject{
		UserID:      p.UserID,
		Active:      p.Status == "active",
		MFAVerified: p.MFAAt != nil && time.Since(*p.MFAAt) <= mfaWindow,
	}
}
```

### 14.5 Escopo vem do recurso, nunca do cliente

Errado: `groupID := body.GroupID`. Certo: carregar o capítulo e usar o `group_id` **do banco**.

```go
func (h *ChapterHandler) Publish(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	ch, err := h.repo.Chapter(r.Context(), id) // busca sem filtrar por permissão
	if err != nil { httpx.NotFound(w); return }

	subj := authz.SubjectFromRequest(r)
	// Se o usuário nem enxerga o recurso → 404 (não vaza existência).
	if !h.authz.Can(r.Context(), subj, authz.SeriesEdit, authz.GroupScope(ch.GroupID)) &&
		!h.authz.Can(r.Context(), subj, authz.ChapterPublish, authz.GroupScope(ch.GroupID)) {
		httpx.NotFound(w)
		return
	}
	// Enxerga, mas não pode esta ação específica → 403.
	if !h.authz.Can(r.Context(), subj, authz.ChapterPublish, authz.GroupScope(ch.GroupID)) {
		httpx.Forbidden(w, "forbidden")
		return
	}
	// ...
}
```

Padrão: **404** quando o usuário não tem nenhuma relação com o recurso (outro grupo); **403** apenas quando ele já tem visibilidade legítima e falta a permissão da ação.

---

## 15. Anti-escalação de privilégio

Convenção: **maior `position` = maior autoridade**. O cargo `owner` tem a maior posição e `system=true`.

Posto efetivo do ator em um escopo = maior `position` entre os cargos dele naquele escopo (global ou grupo).

### 15.1 Regras

| Operação | Regra |
| --- | --- |
| Atribuir cargo R a usuário U | Ator tem `role.assign` no escopo **e** `R.position < rank(ator)` **e** o ator possui **todas** as permissões de R **e** `U ≠ ator` **e** `rank(U) < rank(ator)` |
| Remover cargo de U | Mesmas regras de posição; nunca remove o **último owner** |
| Criar/editar cargo R | Ator tem `role.manage`; `R.position < rank(ator)`; só pode **adicionar** permissões que ele próprio possui; não pode mudar `position` para ≥ o seu |
| Criar override em U | Ator tem `role.manage`; `rank(U) < rank(ator)`; só concede permissões que possui |
| Banir U | `user.ban`; `rank(U) < rank(ator)` |
| Apagar cargo `system` | **Proibido** |
| Atribuir `owner` | **Só** via `cli bootstrap-owner` ou fluxo de transferência de posse (recent auth + MFA + confirmação por e-mail) |

### 15.2 Implementação

```go
var ErrEscalation = errors.New("authz: privilege escalation blocked")

func (s *Service) AssignRole(ctx context.Context, actor Subject, target uuid.UUID, roleID uuid.UUID, sc Scope) error {
	return s.db.InTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable}, func(tx Tx) error {
		// Trava as linhas relevantes: evita corrida entre dois admins.
		role, err := tx.RoleForUpdate(ctx, roleID)
		if err != nil { return err }

		if !s.resolver.Can(ctx, actor, RoleAssign, sc) {
			return ErrForbidden
		}
		if actor.UserID == target {
			return ErrEscalation // ninguém atribui cargo a si mesmo
		}
		actorRank, _ := tx.MaxPosition(ctx, actor.UserID, sc)
		targetRank, _ := tx.MaxPosition(ctx, target, sc)
		if role.System || role.Position >= actorRank || targetRank >= actorRank {
			return ErrEscalation
		}
		// O ator precisa ter TODAS as permissões que está concedendo.
		for _, perm := range role.Permissions {
			if !s.resolver.Can(ctx, actor, perm, sc) {
				return ErrEscalation
			}
		}
		if err := tx.InsertUserRole(ctx, target, roleID, actor.UserID); err != nil {
			return err
		}
		return s.audit.LogTx(tx, "role.assign", actor.UserID, "user", target,
			nil, map[string]any{"role_id": roleID, "scope": sc})
	})
}
```

**Proteção do último owner:** em `RemoveRole`/`BanUser`/`DeleteUser`, dentro da transação, `SELECT count(*) ... FOR UPDATE` dos owners ativos; se for o último, aborta.

---

## 16. Enforcement nas rotas

### 16.1 Deny-by-default com Policy

Toda rota é registrada **com política explícita**. Rota sem política não existe.

```go
type PolicyKind int

const (
	KindPublic PolicyKind = iota
	KindAuthenticated
	KindPermission
)

type ScopeFn func(r *http.Request) (Scope, error) // deriva do RECURSO

type Policy struct {
	Kind       PolicyKind
	Perm       Perm
	Scope      ScopeFn
	MaxAuthAge time.Duration // step-up opcional
	NeedMFA    bool
}

func Public() Policy        { return Policy{Kind: KindPublic} }
func Authenticated() Policy { return Policy{Kind: KindAuthenticated} }
func Require(p Perm, sc ScopeFn) Policy {
	return Policy{Kind: KindPermission, Perm: p, Scope: sc}
}

type Router struct {
	chi.Router
	authz    *Resolver
	registry map[string]Policy // "POST /admin/roles" → policy
}

func (rt *Router) Handle(method, pattern string, pol Policy, h http.HandlerFunc) {
	rt.registry[method+" "+pattern] = pol
	rt.Router.Method(method, pattern, rt.enforce(pol, h))
}

func (rt *Router) enforce(pol Policy, h http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch pol.Kind {
		case KindPublic:
			h(w, r)
			return
		case KindAuthenticated, KindPermission:
			p, ok := auth.PrincipalFrom(r.Context())
			if !ok {
				httpx.Unauthorized(w)
				return
			}
			if pol.Kind == KindPermission {
				sc := GlobalScope()
				if pol.Scope != nil {
					var err error
					if sc, err = pol.Scope(r); err != nil {
						httpx.NotFound(w) // recurso inexistente/inacessível
						return
					}
				}
				if !rt.authz.Can(r.Context(), SubjectFrom(p, mfaWindow), pol.Perm, sc) {
					audit.Denied(r.Context(), pol.Perm, sc)
					httpx.Forbidden(w, "forbidden")
					return
				}
			}
			h(w, r)
		}
	})
}
```

Uso:

```go
rt.Handle("GET",  "/api/series/{id}",        authz.Public(),        h.GetSeries)
rt.Handle("POST", "/api/library/follows",    authz.Authenticated(), h.Follow)
rt.Handle("POST", "/api/chapters/{id}/publish",
	authz.Require(authz.ChapterPublish, scopeFromChapter(repo)), h.Publish)
rt.Handle("POST", "/admin/roles",
	authz.Require(authz.RoleManage, nil), h.CreateRole)
```

### 16.2 Teste que quebra o CI se faltar política

```go
func TestEveryRouteHasPolicy(t *testing.T) {
	rt := app.BuildRouter(testDeps(t))
	_ = chi.Walk(rt.Router, func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if _, ok := rt.Registry()[method+" "+route]; !ok {
			t.Errorf("rota sem Policy: %s %s", method, route)
		}
		return nil
	})
}
```

### 16.3 Regras semgrep (gate de CI)

```yaml
rules:
  - id: no-role-check
    message: "Nunca cheque role. Use authz.Can com uma Perm."
    languages: [go]
    severity: ERROR
    pattern-either:
      - pattern: $U.Role == $X
      - pattern: $U.IsAdmin
      - pattern: $U.IsVIP
  - id: no-string-literal-perm
    message: "Use constantes authz.Perm, não string literal."
    languages: [go]
    severity: ERROR
    pattern: $A.Can($CTX, $S, "...", $SC)
  - id: no-raw-handle
    message: "Registre rotas via authz.Router.Handle com Policy."
    languages: [go]
    severity: ERROR
    paths: { include: ["apps/api/**"], exclude: ["**/authz/**"] }
    pattern-either:
      - pattern: $R.Get(...)
      - pattern: $R.Post(...)
      - pattern: $R.Put(...)
      - pattern: $R.Delete(...)
```

Também teste que **toda** `Perm` usada no código existe no `Catalog`, e que todo handler em `/admin/*` usa `Require(...)`.

---

## 17. Integração com VIP

O VIP não é caso especial: é um cargo com `content.read_vip` e `expires_at`.

- Adicione `expires_at` em `user_roles` (ou use `vip_memberships` como fonte e sincronize) e filtre `expires_at > now()` na query `EffectiveGrants`.
- `CanRead(ctx, u, ch)` (roadmap seção 5.2) chama `resolver.Can(ctx, subj, ContentReadVIP, GlobalScope())` em vez de `u.HasActiveVIP`, mantendo **um único** ponto de decisão.
- "Equipe vê sem ser VIP": `CanRead` consulta `authz.Can(SeriesEdit | ChapterPublish | ReportHandle, GroupScope(ch.GroupID))`.
- O endpoint que devolve a lista de páginas chama `CanRead` **antes** de gerar qualquer URL assinada; TTL de 5–15 min para VIP/early.
- Negar acesso a conteúdo VIP a quem não pode ler: `403` com `{"code":"vip_required"}` se a existência do capítulo é pública (capítulo listado), caso contrário `404`.

---

## 18. Auditoria

Registre no **mesmo commit** da mutação (`LogTx`): se a ação foi feita, o log existe.

| Categoria | Ações |
| --- | --- |
| Auth | `auth.login.success/failure/blocked`, `auth.logout`, `auth.password.change/reset`, `auth.mfa.enable/disable/failure`, `auth.recovery.used`, `auth.passkey.add/remove/clone_suspected`, `auth.session.revoke` |
| Authz | `role.create/update/delete`, `role.assign/unassign`, `override.set/remove`, `authz.denied` (amostrado), `authz.escalation_blocked` |
| Moderação | `user.ban/unban`, `content.remove`, `report.resolve` |
| Instância | `instance.config.update`, `vip.grant/revoke` |

**O que NUNCA entra no log:** senha, token, código TOTP, segredo, cookie, corpo bruto. Registre `email_hash` (HMAC) em falhas de login, não o e-mail.

Proteções:

- Trigger de imutabilidade (seção 3) **e** `REVOKE UPDATE, DELETE ON audit_log FROM ms_api`.
- Retenção ≥ 12 meses; exportação via `cli audit export`.
- Opcional: coluna `prev_hash` encadeando o SHA-256 de cada linha para detectar remoção.
- Alertas Prometheus: pico de `auth.login.failure`, `authz.escalation_blocked > 0`, `auth.passkey.clone_suspected > 0`, mudança de `role.*` fora de horário.

---

## 19. Banco: privilégios mínimos e RLS

### 19.1 Papéis do Postgres

| Papel | Uso | Permissões |
| --- | --- | --- |
| `ms_migrator` | Migrations (job de deploy) | DDL; nunca usado em runtime |
| `ms_api` | API | DML nas tabelas da app; **só INSERT/SELECT** em `audit_log`; sem DDL |
| `ms_worker` | Worker | Só tabelas de chapters/pages/jobs; **sem** acesso a `users.password_hash`, `user_totp`, `auth_tokens` |

```sql
REVOKE ALL ON audit_log FROM ms_api;
GRANT SELECT, INSERT ON audit_log TO ms_api;
GRANT USAGE ON SEQUENCE audit_log_id_seq TO ms_api;

REVOKE ALL ON users, user_totp, user_recovery_codes, auth_tokens, user_sessions FROM ms_worker;
```

### 19.2 RLS como segunda barreira (defense in depth)

Para tabelas escopadas por grupo (ex.: `chapters`), RLS impede que um bug de query vaze dados de outro grupo:

```sql
ALTER TABLE chapters ENABLE ROW LEVEL SECURITY;
CREATE POLICY chapters_group_isolation ON chapters
  USING (group_id = ANY (string_to_array(current_setting('app.group_ids', true), ',')::uuid[])
         OR current_setting('app.bypass_rls', true) = 'on');
```

Com pool de conexões, use **sempre** `SET LOCAL` dentro de transação (`set_config('app.group_ids', ..., true)`), nunca `SET` de sessão (vazaria para o próximo request). Trate RLS como rede de segurança: a decisão primária continua sendo o `Authorizer`.

---

## 20. Integração com o Next.js

- **Mesma origem:** Caddy serve `/` → Next e `/api/*`, `/admin-api/*` → Go. Sem CORS; **não** habilite CORS no Go (negar por padrão).
- **SSR/RSC:** o Next repassa o cookie ao Go e nunca o interpreta:

```ts
// apps/web/lib/api.ts (server)
import { cookies } from "next/headers";
export async function apiFetch(path: string, init: RequestInit = {}) {
  const cookie = (await cookies()).toString();
  return fetch(`${process.env.API_INTERNAL_URL}${path}`, {
    ...init,
    headers: { ...init.headers, cookie },
    cache: "no-store", // nunca cacheie resposta autenticada no Next
  });
}
```

- **Client (browser):** o client gerado injeta `X-CSRF-Token` (obtido de `GET /api/auth/csrf`) e, em `/admin/*`, `X-MS-Admin: 1`. Interceptor trata `401 reauth_required` abrindo modal de reautenticação e `403 mfa_enrollment_required` levando ao cadastro de MFA.
- **Middleware do Next:** pode redirecionar `/admin` para `/login` **por UX** quando não houver cookie; a decisão real continua nos endpoints.
- Páginas públicas cacheadas por ISR **nunca** podem conter dados específicos de usuário. Estado de usuário (`/me`) é buscado no client.
- ISR + conteúdo VIP/early: o HTML público não inclui URLs de imagens restritas; a lista de páginas vem de `GET /api/chapters/{id}/pages` autenticado.
- Domínio de mídia separado (`img.exemplo.com`): sem cookies e sem JS.

---

## 21. Testes obrigatórios

### 21.1 Authz (matriz)

```go
func TestResolver(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*fixture)
		perm  authz.Perm
		scope authz.Scope
		want  bool
	}{
		{"sem nada nega", nil, authz.ChapterPublish, authz.GroupScope(gA), false},
		{"cargo de grupo A permite em A", func(f *fixture) { f.groupRole(gA, "uploader") }, authz.ChapterPublish, authz.GroupScope(gA), true},
		{"cargo de grupo A NÃO vale em B", func(f *fixture) { f.groupRole(gA, "uploader") }, authz.ChapterPublish, authz.GroupScope(gB), false},
		{"deny global vence cargo de grupo", func(f *fixture) { f.groupRole(gA, "uploader"); f.deny(authz.ChapterPublish, nil) }, authz.ChapterPublish, authz.GroupScope(gA), false},
		{"deny vence allow (qualquer ordem)", func(f *fixture) { f.allow(authz.ChapterPublish, &gA); f.deny(authz.ChapterPublish, &gA) }, authz.ChapterPublish, authz.GroupScope(gA), false},
		{"override expirado ignorado", func(f *fixture) { f.allowExpired(authz.ChapterPublish) }, authz.ChapterPublish, authz.GlobalScope(), false},
		{"requires_mfa sem MFA nega", func(f *fixture) { f.globalRole("admin"); f.noMFA() }, authz.RoleManage, authz.GlobalScope(), false},
		{"banido nega tudo", func(f *fixture) { f.globalRole("owner"); f.status("banned") }, authz.AdminView, authz.GlobalScope(), false},
		{"perm desconhecida nega", nil, authz.Perm("x.y"), authz.GlobalScope(), false},
	}
	// ... executar cada caso contra Postgres real (testcontainers)
}
```

Teste também por **propriedade**: gere combinações aleatórias de roles/overrides e verifique o invariante "existe um deny aplicável ⇒ `Can == false`".

### 21.2 Anti-escalação

- Admin com cargo X não atribui cargo ≥ X, nem a si mesmo, nem a alguém de rank ≥ o seu.
- Não concede permissão que não possui.
- Não remove/bane o último owner.
- Duas atribuições concorrentes (goroutines) não furam a regra (serializable).

### 21.3 Auth

| Teste | Esperado |
| --- | --- |
| Login com e-mail inexistente vs senha errada | Mesmo status, mesmo corpo (comparar bytes) |
| Cadastro com e-mail existente vs novo | Mesma resposta |
| Fixação de sessão | Cookie anterior ao login é inválido após login |
| Logout | Reutilizar o cookie antigo → anônimo |
| Troca de senha | Outras sessões revogadas |
| Reset | Token usado 2× falha; token expirado falha; token de outro propósito falha |
| TOTP replay | O mesmo código 2× falha |
| Passkey com `sign_count` menor | Bloqueia e audita |
| CSRF | POST sem token / com Origin externo → 403 |
| Rate limit | Após N falhas → 429 com `Retry-After` |
| Ban | Sessão ativa vira anônima no próximo request |
| Rota nova sem Policy | CI falha (`TestEveryRouteHasPolicy`) |
| `argon2` params fora do limite no hash do banco | Erro, sem alocação gigante |

### 21.4 Fuzz e e2e

- `go test -fuzz` em: parser de hash argon2, normalização de e-mail, validação de senha, parsing de `Origin`.
- Playwright: cadastro → verificação → login → MFA → acesso a `/admin` negado → reauth → permitido; usuário de grupo A tenta acessar recurso do grupo B e recebe 404.

---

## 22. Ordem de implementação e Definition of Done

### 22.1 Sequência sugerida (cabe nas 6–8 semanas do MVP)

| # | Entrega | Pronto quando |
| --- | --- | --- |
| 1 | Migrations (seção 3), catálogo de permissões, `SyncCatalog` | Banco sobe; catálogo sincroniza |
| 2 | `Hasher`, política de senha, testes | Benchmarks registrados em ADR |
| 3 | Sessões: store hasheado, `Establish`, `Authenticate`, logout, revogação | Testes de fixação e revogação passam |
| 4 | CSRF + client IP + rate limiter | Testes de CSRF e 429 passam |
| 5 | Cadastro, verificação de e-mail, login, reset, troca de senha | Testes de enumeração e reuso de token passam |
| 6 | `authz.Resolver`, `Policy/Router`, testes de matriz | `TestEveryRouteHasPolicy` ativo no CI |
| 7 | `cli bootstrap-owner`, gestão de cargos com anti-escalação, audit log | Testes de escalação e último owner passam |
| 8 | TOTP + recovery codes + step-up + `requires_mfa` | MFA obrigatório para `role.manage`/`instance.configure` |
| 9 | Integração VIP (`CanRead` sobre o resolver) | Testes de acesso early/vip_only passam |
| 10 | Semgrep, gosec, fuzz, e2e no CI | Gates bloqueantes ativos |

Fase 2: passkeys, OIDC, OAuth social. Fase 3: UI de cargos, revisão ASVS L2.

### 22.2 Definition of Done (checklist de merge)

- [ ] Nenhuma checagem de role/flag; só `authz.Can` com `Perm` tipada.
- [ ] Toda rota registrada com `Policy`; teste de rotas verde.
- [ ] Escopo derivado do recurso no banco, nunca do body/query.
- [ ] 404 vs 403 conforme seção 14.5.
- [ ] Mutação sensível grava `audit_log` na mesma transação.
- [ ] Erros de authz/DB resultam em **negação**, não em acesso.
- [ ] Respostas de auth sem enumeração de contas.
- [ ] Segredos fora de logs, código e repositório (`gitleaks` verde).
- [ ] Testes unitários, de integração e (quando aplicável) fuzz adicionados.
- [ ] ADR atualizado se houve mudança de parâmetro ou decisão.

---

## 23. Anti-padrões proibidos

1. `if user.Role == "admin"` ou `user.IsVIP` fora do resolver.
2. Guardar roles/permissões na sessão ou em JWT (ficam desatualizados e não revogam).
3. Usar JWT como sessão do browser. Sessão server-side em cookie `HttpOnly` é mais simples **e** revogável.
4. Decidir autorização no Next.js, no front ou em middleware do Edge.
5. Receber `group_id`/`owner_id` do cliente para decidir permissão.
6. Cache de permissão sem invalidação explícita.
7. Mensagens diferentes para "usuário não existe" e "senha errada".
8. Lockout permanente por conta (vira DoS contra a vítima).
9. Token de reset/verificação armazenado em texto puro ou reutilizável.
10. `math/rand` para qualquer coisa de segurança (use `crypto/rand`).
11. Comparar segredos com `==` (use `subtle.ConstantTimeCompare`).
12. `Access-Control-Allow-Origin: *` ou refletir o `Origin` com credenciais.
13. Logar senha, token, TOTP, cookie ou corpo bruto de requests de auth.
14. Mutação via `GET`.
15. Criar "admin escondido" ou endpoint debug que ignora o `Authorizer`.
16. Vincular conta OIDC a conta local só por e-mail igual.
17. Cache de CDN/Next em resposta autenticada.
18. Silenciar erro de banco no `Authenticate` tratando como anônimo (fail-open disfarçado).

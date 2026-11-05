-- Migration: 0009_auth_schema
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Notes: Schema completo de autenticacao e autorizacao (RBAC) conforme AUTH.md

-- Extensoes
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- TABELA: users
CREATE TABLE IF NOT EXISTS users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    username          citext NOT NULL UNIQUE,
    password_hash     text,
    status            text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','banned','pending_deletion')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- TABELA: external_identities (OIDC/OAuth)
CREATE TABLE IF NOT EXISTS external_identities (
    provider   text NOT NULL,
    subject    text NOT NULL,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

-- TABELA: sessions (dados da sessão, chave = sha256(token))
CREATE TABLE IF NOT EXISTS sessions (
    token  text PRIMARY KEY,
    data   bytea NOT NULL,
    expiry timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);

-- TABELA: user_sessions (metadados e revogação)
CREATE TABLE IF NOT EXISTS user_sessions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
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

-- TABELA: auth_tokens (tokens de uso único)
CREATE TABLE IF NOT EXISTS auth_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('verify_email','reset_password','change_email')),
    token_hash bytea NOT NULL UNIQUE,
    meta       jsonb NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- TABELA: user_totp (MFA)
CREATE TABLE IF NOT EXISTS user_totp (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret_enc     bytea NOT NULL,
    key_id         smallint NOT NULL,
    confirmed_at   timestamptz,
    last_used_step bigint NOT NULL DEFAULT 0
);

-- TABELA: user_recovery_codes
CREATE TABLE IF NOT EXISTS user_recovery_codes (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hmac bytea NOT NULL,
    used_at   timestamptz
);

-- TABELA: user_webauthn_credentials
CREATE TABLE IF NOT EXISTS user_webauthn_credentials (
    id            bytea PRIMARY KEY,
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

-- TABELA: permissions
CREATE TABLE IF NOT EXISTS permissions (
    key           text PRIMARY KEY,
    desc_text     text NOT NULL,
    scopes        text[] NOT NULL DEFAULT '{}',
    requires_mfa  boolean NOT NULL DEFAULT false
);

-- TABELA: roles
CREATE TABLE IF NOT EXISTS roles (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL UNIQUE,
    position  int  NOT NULL,
    system    boolean NOT NULL DEFAULT false
);

-- TABELA: role_permissions
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id        uuid REFERENCES roles(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_key)
);

-- TABELA: user_roles (escopo global)
CREATE TABLE IF NOT EXISTS user_roles (
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id     uuid REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    granted_by  uuid REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);

-- TABELA: group_members (escopo por grupo)
CREATE TABLE IF NOT EXISTS group_members (
    group_id uuid NOT NULL,
    user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id  uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id, role_id)
);

-- TABELA: user_overrides
CREATE TABLE IF NOT EXISTS user_overrides (
    user_id        uuid REFERENCES users(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    scope_group_id uuid,
    effect         text NOT NULL CHECK (effect IN ('allow','deny')),
    reason         text,
    created_by     uuid REFERENCES users(id),
    expires_at     timestamptz,
    PRIMARY KEY (user_id, permission_key, scope_group_id)
);
CREATE UNIQUE INDEX user_overrides_uniq
    ON user_overrides (user_id, permission_key,
                       COALESCE(scope_group_id, '00000000-0000-0000-0000-000000000000'));

-- TABELA: bans
CREATE TABLE IF NOT EXISTS bans (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id),
    reason     text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id),
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    revoked_at timestamptz
);

-- TABELA: audit_log (append-only)
CREATE TABLE IF NOT EXISTS audit_log (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    actor_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    action      text NOT NULL,
    target_type text,
    target_id   text,
    before      jsonb,
    after       jsonb,
    ip          inet,
    user_agent  text,
    request_id  text
);
CREATE INDEX IF NOT EXISTS idx_audit_log_actor ON audit_log (actor_id, at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_action ON audit_log (action, at DESC);

-- Trigger: audit_log append-only (apenas INSERT)
CREATE OR REPLACE FUNCTION audit_log_immutable() RETURNS trigger AS $$
BEGIN RAISE EXCEPTION 'audit_log is append-only'; END; $$ LANGUAGE plpgsql;

CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_immutable();

-- ============================================================
-- DADOS INICIAIS (seed)
-- ============================================================

-- Inserir roles padrão
INSERT INTO roles (name, description, position)
SELECT 'admin', 'Administrador com acesso total ao sistema', 1
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'admin');

INSERT INTO roles (name, description, position)
SELECT 'editor', 'Editor pode gerenciar series e capitulos', 2
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'editor');

INSERT INTO roles (name, description, position)
SELECT 'moderator', 'Moderador pode gerenciar capitulos e VIP', 3
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'moderator');

INSERT INTO roles (name, description, position)
SELECT 'user', 'Usuario comum sem permissoes de administracao', 4
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'user');

-- Inserir permissoes padrão
INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'series.manage', 'Criar, editar e excluir series', ARRAY['global','group'], false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'series.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'series.view', 'Visualizar series', ARRAY['global','group'], false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'series.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'chapters.manage', 'Criar, editar e excluir capitulos', 'chapters', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'chapters.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'chapters.publish', 'Publicar capitulos', 'chapters', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'chapters.publish');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'chapters.view', 'Visualizar capitulos', 'chapters', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'chapters.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'users.manage', 'Criar, editar e excluir usuarios', 'users', true
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'users.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'users.view', 'Visualizar usuarios', 'users', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'users.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'users.assign_role', 'Atribuir roles a usuarios', 'users', true
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'users.assign_role');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'vip.manage', 'Gerenciar assinaturas VIP', 'vip', true
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'vip.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'vip.view', 'Visualizar assinaturas VIP', 'vip', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'vip.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'settings.manage', 'Gerenciar configuracoes do sistema', 'settings', true
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'settings.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'settings.view', 'Visualizar configuracoes do sistema', 'settings', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'settings.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'search.manage', 'Gerenciar configuracoes de busca', 'search', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'search.manage');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'search.view', 'Visualizar configuracoes de busca', 'search', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'search.view');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'admin.access', 'Acesso completo ao painel admin', 'admin', true
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'admin.access');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'media.upload', 'Fazer upload de midia', 'media', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'media.upload');

INSERT INTO permissions (key, description, scopes, requires_mfa)
SELECT 'media.manage', 'Gerenciar midia existente', 'media', false
WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE key = 'media.manage');

-- Atribuir permissoes ao role admin (todas)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'admin'
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_key = p.key
);

-- Atribuir permissoes ao role editor
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'editor' AND p.key IN (
    'series.manage', 'series.view',
    'chapters.manage', 'chapters.publish', 'chapters.view',
    'media.upload', 'media.manage'
)
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_key = p.key
);

-- Atribuir permissoes ao role moderator
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'moderator' AND p.key IN (
    'chapters.manage', 'chapters.publish', 'chapters.view',
    'vip.manage', 'vip.view',
    'media.upload', 'media.manage'
)
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_key = p.key
);

-- Atribuir permissoes ao role user (apenas visualizacao basica)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'user' AND p.key IN (
    'series.view', 'chapters.view', 'vip.view', 'search.view'
)
AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_key = p.key
);

-- Criar usuario admin padrao (senha: admin123 - DEVE SER ALTERADA EM PRODUCAO)
INSERT INTO users (email, name, password_hash, is_active, email_verified)
SELECT 'admin@mangashelf.local', 'Administrador',
       '\\$argon2id\\$v=19\\$m=32768,t=3,p=4\\$Y7Fkf8c3Lm2nPq1sRt4vWx6zA8bC5dE9fG0hI2jK4lM6nO8pQ10rS12tU14vW16xY18zA0bC2dE4fG6hI8jK10lM12nO14pQ16rS18tU20vW22xY24zA2bC4dE6fG8hI10jK12lM14nO16pQ18rS20tU22vW24xY26zA2bC4dE6fG8hI10jK12lM14nO16pQ18rS20tU22vW24xY26z',
       true, true
WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@mangashelf.local');

-- Atribuir role admin ao usuario admin
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.email = 'admin@mangashelf.local' AND r.name = 'admin'
AND NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id
);

-- Migration: 0001_core
-- Schema inicial coerente com o código atual

CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS citext;
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Grupos
CREATE TABLE groups (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text UNIQUE NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

-- Usuários
CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             citext NOT NULL UNIQUE,
    email_verified_at timestamptz,
    username          citext UNIQUE,
    name              text,
    password_hash     text,
    status            text NOT NULL DEFAULT 'active' CHECK (status IN ('active','suspended','banned','pending_deletion')),
    oauth_provider    text,
    oauth_provider_id text,
    failed_login_attempts int NOT NULL DEFAULT 0,
    locked_until      timestamptz,
    last_login_at     timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_users_oauth_provider_id ON users(oauth_provider, oauth_provider_id);



-- Identidades externas
CREATE TABLE external_identities (
    provider   text NOT NULL,
    subject    text NOT NULL,
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (provider, subject)
);

-- Sessões
CREATE TABLE sessions (
    token  text PRIMARY KEY,
    data   bytea NOT NULL,
    expiry timestamptz NOT NULL
);
CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE user_sessions (
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



-- Tokens
CREATE TABLE auth_tokens (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    purpose    text NOT NULL CHECK (purpose IN ('verify_email','reset_password','change_email')),
    token_hash bytea NOT NULL UNIQUE,
    meta       jsonb NOT NULL DEFAULT '{}',
    expires_at timestamptz NOT NULL,
    used_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- MFA
CREATE TABLE user_totp (
    user_id        uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    secret_enc     bytea NOT NULL,
    key_id         smallint NOT NULL,
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

-- RBAC
CREATE TABLE permissions (
    key           text PRIMARY KEY,
    desc_text     text,
    scopes        text[] NOT NULL DEFAULT '{}',
    requires_mfa  boolean NOT NULL DEFAULT false
);
CREATE TABLE roles (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL UNIQUE,
    position  int  NOT NULL DEFAULT 0,
    system    boolean NOT NULL DEFAULT false,
    group_id  uuid REFERENCES groups(id) ON DELETE CASCADE
);
CREATE TABLE role_permissions (
    role_id        uuid REFERENCES roles(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_key)
);
CREATE TABLE user_roles (
    user_id     uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id     uuid REFERENCES roles(id) ON DELETE CASCADE,
    granted_at  timestamptz NOT NULL DEFAULT now(),
    granted_by  uuid REFERENCES users(id),
    PRIMARY KEY (user_id, role_id)
);
CREATE TABLE group_members (
    group_id uuid NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id  uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id  uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id, role_id)
);
CREATE TABLE user_overrides (
    user_id        uuid REFERENCES users(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    scope_group_id uuid,
    effect         text NOT NULL CHECK (effect IN ('allow','deny')),
    reason         text,
    created_by     uuid REFERENCES users(id),
    expires_at     timestamptz,
    PRIMARY KEY (user_id, permission_key, scope_group_id)
);



-- Séries/Catálogo
CREATE TABLE series (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title             text NOT NULL,
    slug              text UNIQUE NOT NULL,
    synopsis          text,
    type              text NOT NULL DEFAULT 'manga' CHECK (type IN ('manga', 'manhwa', 'manhua')),
    status            text NOT NULL DEFAULT 'ongoing' CHECK (status IN ('ongoing', 'hiatus', 'completed', 'cancelled')),
    demography        text NOT NULL DEFAULT 'unspecified' CHECK (demography IN ('shounen', 'shoujo', 'seinen', 'josei', 'unspecified')),
    original_language text NOT NULL DEFAULT 'ja' CHECK (original_language IN ('ja', 'ko', 'zh', 'en', 'other')),
    release_year      int CHECK (release_year IS NULL OR release_year BETWEEN 1900 AND 3000),
    access            text NOT NULL DEFAULT 'public' CHECK (access IN ('public','vip_only')),
    publication_status text NOT NULL DEFAULT 'draft' CHECK (publication_status IN ('draft', 'scheduled', 'published', 'removed')),
    published_at      timestamptz,
    public_at         timestamptz,
    cover_storage_key text,
    group_id          uuid REFERENCES groups(id) ON DELETE RESTRICT,
    created_by        uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_series_type ON series(type);
CREATE INDEX idx_series_status ON series(status);
CREATE INDEX idx_series_access ON series(access);
CREATE INDEX idx_series_group ON series(group_id) WHERE group_id IS NOT NULL;

-- Autores/tags/alt titles
CREATE TABLE authors (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    slug       text NOT NULL UNIQUE,
    bio        text NOT NULL DEFAULT '',
    avatar_url text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE series_authors (
    series_id  uuid REFERENCES series(id) ON DELETE CASCADE,
    author_id  uuid REFERENCES authors(id) ON DELETE CASCADE,
    role       text NOT NULL DEFAULT 'author' CHECK (role IN ('author','artist')),
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (series_id, author_id, role)
);
CREATE TABLE tags (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    slug       text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_tags_name_lower ON tags (lower(name));
CREATE TABLE series_tags (
    series_id  uuid REFERENCES series(id) ON DELETE CASCADE,
    tag_id     uuid REFERENCES tags(id) ON DELETE CASCADE,
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    PRIMARY KEY (series_id, tag_id)
);
CREATE TABLE series_alt_titles (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id  uuid NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    title      text NOT NULL CHECK (length(btrim(title)) > 0),
    sort_order int  NOT NULL DEFAULT 0 CHECK (sort_order >= 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, title)
);
CREATE INDEX idx_alt_titles_series ON series_alt_titles(series_id);



-- Capítulos
CREATE TABLE chapters (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id        uuid NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    number           int NOT NULL,
    title            text,
    state            text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','processing','published','removed')),
    access           text NOT NULL DEFAULT 'public' CHECK (access IN ('public','early','vip_only')),
    public_at        timestamptz,
    storage_key      text,
    source_upload_id uuid,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, number)
);
CREATE INDEX idx_chapters_series_state ON chapters(series_id, state);
CREATE INDEX idx_chapters_public_at ON chapters(public_at);

-- Páginas
CREATE TABLE pages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id  uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    number      int NOT NULL,
    storage_key text NOT NULL,
    width       int,
    height      int,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, number)
);
CREATE INDEX idx_pages_chapter_number ON pages(chapter_id, number);

CREATE TABLE chapter_pages (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id     uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    page_number    integer NOT NULL CHECK (page_number > 0),
    storage_key    text NOT NULL UNIQUE,
    content_hash   char(64) NOT NULL CHECK (content_hash ~ '^[0-9a-f]{64}$'),
    content_type   text NOT NULL,
    size_bytes     bigint NOT NULL CHECK (size_bytes > 0),
    width          integer NULL CHECK (width IS NULL OR width > 0),
    height         integer NULL CHECK (height IS NULL OR height > 0),
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, page_number)
);
CREATE INDEX idx_chapter_pages_chapter_order ON chapter_pages (chapter_id, page_number);

-- Uploads
CREATE TABLE media_uploads (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    storage_key    text NOT NULL UNIQUE,
    kind           text NOT NULL CHECK (kind IN ('chapter_source','cover_source','avatar_source')),
    status         text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','uploaded','processing','completed','rejected','failed','expired')),
    content_type   text NOT NULL,
    size_bytes     bigint NOT NULL CHECK (size_bytes > 0),
    sha256         char(64) CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    uploaded_by    uuid REFERENCES users(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz,
    processed_at   timestamptz,
    failure_reason text
);
CREATE INDEX idx_media_uploads_kind ON media_uploads(kind);
CREATE INDEX idx_media_uploads_status ON media_uploads(status);

ALTER TABLE chapters ADD CONSTRAINT fk_chapters_source_upload FOREIGN KEY (source_upload_id) REFERENCES media_uploads(id) ON DELETE SET NULL;


-- VIP
CREATE TABLE vip_plans (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL,
    duration  interval NOT NULL,
    duration_days int,
    perks     jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE vip_memberships (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    plan_id     uuid NOT NULL REFERENCES vip_plans(id) ON DELETE CASCADE,
    starts_at   timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    source      text NOT NULL DEFAULT 'admin' CHECK (source IN ('admin','voucher','payment')),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_vip_memberships_user_expires ON vip_memberships(user_id, expires_at);
CREATE TABLE vip_vouchers (
    code_hash  text PRIMARY KEY,
    plan_id    uuid NOT NULL REFERENCES vip_plans(id) ON DELETE CASCADE,
    max_uses   int NOT NULL DEFAULT 1,
    used_count int NOT NULL DEFAULT 0,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Auditoria
CREATE TABLE audit_log (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id   uuid REFERENCES users(id) ON DELETE SET NULL,
    action     text NOT NULL,
    target     text NOT NULL,
    target_id  uuid,
    target_type text,
    before     jsonb,
    after      jsonb,
    ip         text,
    user_agent text,
    request_id text,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_log_actor_created ON audit_log(actor_id, created_at);
CREATE INDEX idx_audit_log_target ON audit_log(target, target_id);

-- Bans
CREATE TABLE bans (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason     text NOT NULL,
    banned_by  uuid REFERENCES users(id),
    banned_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    revoked_at timestamptz
);

-- Configuracoes
CREATE TABLE settings (
    key           text PRIMARY KEY,
    value         jsonb NOT NULL DEFAULT '{}'::jsonb,
    type          text NOT NULL DEFAULT 'string' CHECK (type IN ('string','boolean','color','json')),
    setting_group text NOT NULL DEFAULT 'site' CHECK (setting_group IN ('site','search','ui','internal')),
    updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX idx_settings_group ON settings(setting_group);

-- Defaults
INSERT INTO settings (key, value, type, setting_group) VALUES
    ('app_name', '"MangaShelf"', 'string', 'site'),
    ('search_engine', '"postgres"', 'string', 'search'),
    ('ui_primary_color', '"#3b82f6"', 'color', 'ui'),
    ('ui_secondary_color', '"#1e293b"', 'color', 'ui'),
    ('ui_accent_color', '"#f59e0b"', 'color', 'ui'),
    ('site_description', '"A melhor plataforma de mangas em portugues"', 'string', 'site')
ON CONFLICT (key) DO NOTHING;

-- Seed: roles padrao
INSERT INTO roles (name, position, system) VALUES ('admin', 1, true) ON CONFLICT (name) DO NOTHING;
INSERT INTO roles (name, position, system) VALUES ('editor', 2, false) ON CONFLICT (name) DO NOTHING;
INSERT INTO roles (name, position, system) VALUES ('moderator', 3, false) ON CONFLICT (name) DO NOTHING;
INSERT INTO roles (name, position, system) VALUES ('user', 4, false) ON CONFLICT (name) DO NOTHING;

-- Seed: permissions padrao
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('series.manage', 'Criar, editar e excluir series', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('series.view', 'Visualizar series', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('chapters.manage', 'Criar, editar e excluir capitulos', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('chapters.publish', 'Publicar capitulos', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('chapters.view', 'Visualizar capitulos', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('users.manage', 'Criar, editar e excluir usuarios', ARRAY['global'], true) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('users.view', 'Visualizar usuarios', ARRAY['global'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('vip.manage', 'Gerenciar assinaturas VIP', ARRAY['global'], true) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('vip.view', 'Visualizar assinaturas VIP', ARRAY['global'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('settings.manage', 'Gerenciar configuracoes do sistema', ARRAY['global'], true) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('settings.view', 'Visualizar configuracoes do sistema', ARRAY['global'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('search.manage', 'Gerenciar configuracoes de busca', ARRAY['global'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('search.view', 'Visualizar configuracoes de busca', ARRAY['global'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('admin.access', 'Acesso completo ao painel admin', ARRAY['global'], true) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('media.upload', 'Fazer upload de midia', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('media.manage', 'Gerenciar midia existente', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;
INSERT INTO permissions (key, desc_text, scopes, requires_mfa) VALUES ('report.handle', 'Moderar conteudo reportado', ARRAY['global','group'], false) ON CONFLICT (key) DO NOTHING;

-- Seed: role_permissions (admin gets all)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r CROSS JOIN permissions p
WHERE r.name = 'admin'
ON CONFLICT (role_id, permission_key) DO NOTHING;

-- Seed: role_permissions (editor)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'editor' AND p.key IN ('series.manage','series.view','chapters.manage','chapters.publish','chapters.view','media.upload','media.manage')
ON CONFLICT (role_id, permission_key) DO NOTHING;

-- Seed: role_permissions (moderator)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'moderator' AND p.key IN ('chapters.manage','chapters.publish','chapters.view','vip.manage','vip.view','media.upload','media.manage')
ON CONFLICT (role_id, permission_key) DO NOTHING;

-- Seed: role_permissions (user)
INSERT INTO role_permissions (role_id, permission_key)
SELECT r.id, p.key FROM roles r, permissions p
WHERE r.name = 'user' AND p.key IN ('series.view','chapters.view','vip.view','search.view')
ON CONFLICT (role_id, permission_key) DO NOTHING;

-- Admin user (password: admin123)
INSERT INTO users (email, username, name, password_hash, status) VALUES
    ('admin@mangashelf.local', 'admin', 'Administrador', '$argon2id$v=19$m=32768,t=3,p=4$o8HMQkx420H4rpNPT5YPtA$GyAKTAH9tbvjzuIbbnczyc8QK4txMhCBMXb5NpupTnc', 'active')
ON CONFLICT (email) DO NOTHING;

-- Assign admin role to admin user
INSERT INTO user_roles (user_id, role_id)
SELECT u.id, r.id FROM users u, roles r
WHERE u.email = 'admin@mangashelf.local' AND r.name = 'admin'
ON CONFLICT (user_id, role_id) DO NOTHING;

DROP VIEW IF EXISTS v_series_admin;
CREATE VIEW v_series_admin AS
SELECT s.id,s.title,s.slug,s.synopsis,s.type,s.status,s.demography,s.original_language,s.release_year,s.access,s.publication_status,s.published_at,s.public_at,s.cover_storage_key,s.group_id,s.created_by,s.created_at,s.updated_at,
COALESCE((SELECT jsonb_agg(jsonb_build_object('id',a.id,'name',a.name,'role',sa.role) ORDER BY sa.sort_order) FROM series_authors sa JOIN authors a ON a.id=sa.author_id WHERE sa.series_id=s.id),'[]'::jsonb) AS authors,
COALESCE((SELECT jsonb_agg(jsonb_build_object('id',t.id,'name',t.name) ORDER BY st.sort_order) FROM series_tags st JOIN tags t ON t.id=st.tag_id WHERE st.series_id=s.id),'[]'::jsonb) AS tags,
COALESCE((SELECT jsonb_agg(jsonb_build_object('title',sat.title) ORDER BY sat.sort_order) FROM series_alt_titles sat WHERE sat.series_id=s.id),'[]'::jsonb) AS alt_titles,
COUNT(c.id)::int AS total_chapter_count,
COUNT(c.id) FILTER (WHERE c.state='published')::int AS published_chapter_count
FROM series s LEFT JOIN chapters c ON c.series_id=s.id GROUP BY s.id;

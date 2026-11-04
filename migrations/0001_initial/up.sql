-- Migration: 0001_initial
-- Author: system
-- Date: 2026-11-04
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Schema inicial: usuarios, permissoes, roles, grupos, series, capitulos, VIP e audit_log.

CREATE TABLE users (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email        text UNIQUE NOT NULL,
    password_hash text NOT NULL,
    is_active    boolean NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE groups (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    slug        text UNIQUE NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    key   text PRIMARY KEY,
    description text
);

CREATE TABLE roles (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL,
    position  int NOT NULL DEFAULT 0,
    group_id  uuid REFERENCES groups(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id        uuid REFERENCES roles(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_key)
);

CREATE TABLE user_roles (
    user_id uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

CREATE TABLE group_members (
    group_id uuid REFERENCES groups(id) ON DELETE CASCADE,
    user_id  uuid REFERENCES users(id) ON DELETE CASCADE,
    role_id  uuid REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE user_overrides (
    user_id        uuid REFERENCES users(id) ON DELETE CASCADE,
    permission_key text REFERENCES permissions(key) ON DELETE CASCADE,
    effect         text NOT NULL CHECK (effect IN ('allow','deny')),
    PRIMARY KEY (user_id, permission_key)
);

CREATE TABLE series (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    title       text NOT NULL,
    slug        text UNIQUE NOT NULL,
    synopsis    text,
    access      text NOT NULL DEFAULT 'public' CHECK (access IN ('public','vip_only')),
    group_id    uuid REFERENCES groups(id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE chapters (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id   uuid NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    number      int NOT NULL,
    title       text,
    state       text NOT NULL DEFAULT 'draft' CHECK (state IN ('draft','processing','published','removed')),
    access      text NOT NULL DEFAULT 'public' CHECK (access IN ('public','early','vip_only')),
    public_at   timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (series_id, number)
);

CREATE TABLE vip_plans (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name      text NOT NULL,
    duration  interval NOT NULL,
    perks     jsonb NOT NULL DEFAULT '{}',
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

CREATE TABLE vip_vouchers (
    code_hash  text PRIMARY KEY,
    plan_id    uuid NOT NULL REFERENCES vip_plans(id) ON DELETE CASCADE,
    max_uses   int NOT NULL DEFAULT 1,
    used_count int NOT NULL DEFAULT 0,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE audit_log (
    id        uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id  uuid REFERENCES users(id) ON DELETE SET NULL,
    action    text NOT NULL,
    target    text NOT NULL,
    target_id uuid,
    before    jsonb,
    after     jsonb,
    ip        text,
    user_agent text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_chapters_series_state ON chapters(series_id, state);
CREATE INDEX idx_chapters_public_at ON chapters(public_at);
CREATE INDEX idx_vip_memberships_user_expires ON vip_memberships(user_id, expires_at);
CREATE INDEX idx_audit_log_actor_created ON audit_log(actor_id, created_at);
CREATE INDEX idx_audit_log_target ON audit_log(target, target_id);

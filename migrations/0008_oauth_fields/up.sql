-- Migration: 0008_oauth_fields
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Adiciona campos de autenticação OAuth/OIDC à tabela users

-- up.sql
ALTER TABLE users ADD COLUMN name text;
ALTER TABLE users ADD COLUMN oauth_provider text;
ALTER TABLE users ADD COLUMN oauth_provider_id text;
CREATE INDEX idx_users_oauth_provider_id ON users(oauth_provider, oauth_provider_id);

-- down.sql
ALTER TABLE users DROP COLUMN IF EXISTS name;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider;
ALTER TABLE users DROP COLUMN IF EXISTS oauth_provider_id;
DROP INDEX IF EXISTS idx_users_oauth_provider_id;

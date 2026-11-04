-- Migration: 0001_initial (down)
-- Reversible: yes

DROP INDEX IF EXISTS idx_audit_log_target;
DROP INDEX IF EXISTS idx_audit_log_actor_created;
DROP INDEX IF EXISTS idx_vip_memberships_user_expires;
DROP INDEX IF EXISTS idx_chapters_public_at;
DROP INDEX IF EXISTS idx_chapters_series_state;

DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS vip_vouchers;
DROP TABLE IF EXISTS vip_memberships;
DROP TABLE IF EXISTS vip_plans;
DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS series;
DROP TABLE IF EXISTS user_overrides;
DROP TABLE IF EXISTS group_members;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS groups;
DROP TABLE IF EXISTS users;

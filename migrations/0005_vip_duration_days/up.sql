-- Migration: 0005_vip_duration_days
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Altera duracao de text ("30d") para integer (dias, ex: 30)

ALTER TABLE vip_plans ALTER COLUMN duration TYPE integer USING duration::integer;

-- Migration: 0004_vip_duration_text
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Altera duracao de interval para text (aceita "30d", "90d", etc.)

ALTER TABLE vip_plans ALTER COLUMN duration TYPE text USING duration::text;

-- Migration: 0005_vip_duration_days (down)
-- Reversible: yes

ALTER TABLE vip_plans ALTER COLUMN duration TYPE text USING duration::text;

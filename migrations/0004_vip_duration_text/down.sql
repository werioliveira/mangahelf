-- Migration: 0004_vip_duration_text (down)
-- Reversible: yes

ALTER TABLE vip_plans ALTER COLUMN duration TYPE interval USING duration::interval;

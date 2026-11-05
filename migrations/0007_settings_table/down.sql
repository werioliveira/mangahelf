-- Migration: 0007_settings_table (down)
-- Reversible: yes

DROP INDEX IF EXISTS idx_settings_key;
DROP INDEX IF EXISTS idx_settings_group;
DROP TABLE IF EXISTS settings;

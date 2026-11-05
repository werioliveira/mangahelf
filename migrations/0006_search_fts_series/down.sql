-- Migration: 0006_search_fts_series (down)
-- Reversible: yes

DROP INDEX IF EXISTS idx_series_search_title;
DROP INDEX IF EXISTS idx_series_search_vector;
ALTER TABLE series DROP COLUMN IF EXISTS search_vector;

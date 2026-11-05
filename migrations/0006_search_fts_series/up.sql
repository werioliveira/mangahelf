-- Migration: 0006_search_fts_series
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Adiciona busca full-text (tsvector + GIN) nas séries para busca por título e sinopse.

-- up.sql
ALTER TABLE series ADD COLUMN search_vector tsvector GENERATED ALWAYS AS (
    setweight(to_tsvector('portuguese', coalesce(title, '')), 'A') ||
    setweight(to_tsvector('portuguese', coalesce(synopsis, '')), 'B')
) STORED;

CREATE INDEX idx_series_search_vector ON series USING GIN (search_vector);

CREATE INDEX idx_series_search_title ON series USING GIN (to_tsvector('portuguese', title));

-- down.sql
DROP INDEX IF EXISTS idx_series_search_title;
DROP INDEX IF EXISTS idx_series_search_vector;
ALTER TABLE series DROP COLUMN IF EXISTS search_vector;

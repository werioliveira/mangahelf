-- Migration: 0003_add_pages_table (down)
-- Reversible: yes

DROP INDEX IF EXISTS idx_pages_chapter_number;
DROP TABLE IF EXISTS pages;

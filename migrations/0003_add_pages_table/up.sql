-- Migration: 0003_add_pages_table
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Tabela de paginas de um capítulo (numero, storage_key, dimensoes).

CREATE TABLE pages (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id  uuid NOT NULL REFERENCES chapters(id) ON DELETE CASCADE,
    number      int NOT NULL,
    storage_key text NOT NULL,
    width       int,
    height      int,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (chapter_id, number)
);
CREATE INDEX idx_pages_chapter_number ON pages(chapter_id, number);

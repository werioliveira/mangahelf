-- Migration: 0007_settings_table
-- Author: system
-- Date: 2026-11-05
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #0
-- Notes: Tabela de configurações do sistema (CMS). Permite admin alterar
-- cores, engine de busca, título do site, etc. sem redeploy.

CREATE TABLE settings (
    key      text PRIMARY KEY,
    value    jsonb NOT NULL DEFAULT '{}'::jsonb,
    type     text NOT NULL DEFAULT 'string' CHECK (type IN ('string','boolean','color','json')),
    setting_group text NOT NULL DEFAULT 'site' CHECK (setting_group IN ('site','search','ui','internal')),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_settings_group ON settings(setting_group);
CREATE INDEX idx_settings_key ON settings(key);

-- Valores iniciais (padrões seguros)
INSERT INTO settings (key, value, type, setting_group) VALUES
    ('app_name', '"MangaShelf"', 'string', 'site'),
    ('search_engine', '"postgres"', 'string', 'search'), -- postgres | meilisearch
    ('ui_primary_color', '"#3b82f6"', 'color', 'ui'),
    ('ui_secondary_color', '"#1e293b"', 'color', 'ui'),
    ('ui_accent_color', '"#f59e0b"', 'color', 'ui'),
    ('site_description', '"A melhor plataforma de mangás em português"', 'string', 'site');

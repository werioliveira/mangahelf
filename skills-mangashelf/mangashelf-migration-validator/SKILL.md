---
name: mangashelf-migration-validator
description: "Valida que toda migration SQL é reversível e zero-downtime. Roda up + down em DB real e verifica estado final igual ao inicial. Bloqueia PR com migration irreversível ou que viola expand-then-contract."
---

# MangaShelf Migration Validator

Skill de **gate de CI** que valida migrations SQL do projeto MangaShelf. Bloqueia PR com migration perigosa.

## Quando ativar

Em todo PR que adiciona/modifica arquivo em `migrations/`. Roda automaticamente em CI; pode rodar local com `make migrations-check`.

## O que valida

### 1. Reversibilidade

Toda migration tem `up.sql` e `down.sql`. O CI:
1. Sobe DB Postgres real em container.
2. Roda todas as migrations `up` em ordem.
3. Roda todas as migrations `down` em ordem reversa.
4. Compara schema final com schema vazio (apenas tabelas do Postgres).
5. Se diferente → **bloqueia**.

### 2. Expand-then-contract

Bloqueia migrations que:
- Renomeiam coluna/tabela em um único step.
- Adicionam coluna `NOT NULL` sem `DEFAULT`.
- Removem coluna ainda referenciada por código Go (verifica via `sqlc`).
- Criam índice em tabela viva sem `CONCURRENTLY`.
- Adicionam FK sem `NOT VALID` (torna inválida para tabela existente).

### 3. Tamanho

- Tempo de execução estimado < 1s em DB com 1M linhas (verifica via `EXPLAIN`).
- Para migrations pesadas, exigir `CONCURRENTLY`, `NOT VALID`, ou janela de manutenção documentada.

### 4. Idempotência

`down.sql` deve ser idempotente: rodar 2x não dá erro. Usa `DROP TABLE IF EXISTS`, `DROP COLUMN IF EXISTS`.

### 5. Comentário de revisão

Toda migration tem cabeçalho:

```sql
-- Migration: 0007_add_vip_plans
-- Author: alice@example.com
-- Date: 2024-01-15
-- Reversible: yes
-- Zero-downtime: yes
-- Reviewed: PR #123
-- Notes: Adiciona tabelas VIP; sem impacto em código existente.
```

## Como rodar

### Local

```bash
make migrations-check
# Sobe container, roda up, down, verifica.
# Saída: PASS ou lista de violações.
```

### CI (GitHub Actions / GitLab CI)

```yaml
# .github/workflows/migrations.yml
jobs:
  migrations:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16
        env:
          POSTGRES_PASSWORD: test
        ports: ['5432:5432']
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make migrations-check
```

## Padrões de migration aceitos

### Adicionar coluna nullable

```sql
-- up.sql
ALTER TABLE chapters ADD COLUMN public_at timestamptz;

-- down.sql
ALTER TABLE chapters DROP COLUMN IF EXISTS public_at;
```

### Adicionar coluna NOT NULL com default

```sql
-- up.sql
ALTER TABLE chapters ADD COLUMN access text NOT NULL DEFAULT 'public';

-- down.sql
ALTER TABLE chapters DROP COLUMN IF EXISTS access;
```

### Adicionar índice em tabela viva

```sql
-- up.sql
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_chapters_series_id ON chapters(series_id);

-- down.sql
DROP INDEX CONCURRENTLY IF EXISTS idx_chapters_series_id;
```

**Atenção:** `CONCURRENTLY` não pode rodar dentro de transação. Tooling de migration deve suportar isso (goose/atlas com `--no-transaction` para este arquivo).

### Adicionar FK em tabela viva

```sql
-- up.sql
ALTER TABLE chapters
  ADD CONSTRAINT fk_chapters_series
  FOREIGN KEY (series_id) REFERENCES series(id)
  NOT VALID;

-- Após deploy, em migration separada:
-- VALIDATE CONSTRAINT fk_chapters_series;

-- down.sql
ALTER TABLE chapters DROP CONSTRAINT IF EXISTS fk_chapters_series;
```

### Renomear coluna (3 steps)

```sql
-- Step 1: adicionar nova
ALTER TABLE series ADD COLUMN access text;
-- Backfill (separado, em job): UPDATE series SET access = 'public';
-- Atualizar código Go para ler `access` (ainda escreve em `access` e `old_access`)

-- Step 2 (próximo deploy): parar de escrever em `old_access`, drop em outra migration
ALTER TABLE series DROP COLUMN old_access;
```

## Padrões BLOQUEADOS

```sql
-- ❌ Renomear coluna direto
ALTER TABLE series RENAME COLUMN access TO visibility;

-- ❌ NOT NULL sem default em tabela viva
ALTER TABLE chapters ADD COLUMN required_field text NOT NULL;

-- ❌ Índice sem CONCURRENTLY em tabela > 1000 linhas
CREATE INDEX idx_big ON big_table(col);

-- ❌ Drop de coluna sem checar se código Go ainda referencia
ALTER TABLE chapters DROP COLUMN legacy_field;
```

## Saída de erro

```
❌ Migration 0007_add_vip_plans falhou validação:

  [reversibility] down.sql não reverte up.sql
    - up cria tabela `vip_plans`
    - down não dropa `vip_plans`
    diff:
    + Table: vip_plans

  [expand-contract] adiciona coluna NOT NULL sem default
    - migration 0007: `ALTER TABLE chapters ADD COLUMN vip_only boolean NOT NULL`
    - sugestão: `ADD COLUMN vip_only boolean NOT NULL DEFAULT false`

  [zero-downtime] cria índice sem CONCURRENTLY
    - migration 0007: `CREATE INDEX idx_pages_chapter ON pages(chapter_id)`
    - tabela `pages` tem 1.2M linhas (verificado em staging)
    - sugestão: `CREATE INDEX CONCURRENTLY idx_pages_chapter ON pages(chapter_id)`
```

## Configuração

`migrations/.migrations-validator.yml`:

```yaml
reversibility: required
zero_downtime: required
comment_header: required
concurrent_threshold: 1000   # linhas; acima disso exige CONCURRENTLY
skip:
  - seed/  # seeds não precisam ser reversíveis
```

## Integração com sqlc

Antes de aceitar `down.sql` que dropa coluna, o validator roda `sqlc generate` em um checkout do código Go e verifica se alguma query referencía a coluna. Se sim, bloqueia com mensagem clara.

# MangaShelf

Plataforma open source de hospedagem de mangás, robusta e fácil de auto-hospedar.

Stack: Go 1.22 + Next.js · Banco: PostgreSQL · Deploy: VPS self-hosted com Docker Compose.

Licença: AGPL-3.0 (`LICENSE`). O projeto distribui apenas o software; cada instância é responsabilidade de quem a opera.

## Estrutura do monorepo

```
/cmd                 entrypoints Go (api, worker)
/internal            pacotes Go (config, ports, adapters, services)
/apps/web            Next.js (App Router)
/migrations          migrations SQL (up.sql + down.sql)
/api                 OpenAPI + cliente TS gerado
/deploy              docker-compose, Caddyfile, .env.example
/docs                ADRs, self-host, contribuicao
```

## Como rodar (desenvolvimento)

```bash
# 1. Copie o exemplo de variáveis de ambiente
cp deploy/.env.example .env
# 2. Ajuste DATABASE_URL, STORAGE_DRIVER, etc. conforme necessário

# 3. Rode o banco (Postgres) — local ou WSL
#   Ex.: docker compose --profile postgres up -d

# 4. Rode as migrations
#   go run ./cmd/migrate (ou aplique manualmente migrations/0001_initial/up.sql)

# 5. Rode a API
go run ./cmd/api

# 6. Rode o worker
go run ./cmd/worker
```

## Flags de drivers (`.env`)

| Flag | Padrão | Opções |
|------|--------|--------|
| SEARCH_DRIVER | postgres | postgres, meilisearch |
| IMAGE_PIPELINE | imgproxy | imgproxy, static |
| STORAGE_DRIVER | local | local, s3 |
| CACHE_DRIVER | memory | memory, redis |
| AUTH_DRIVER | builtin | builtin, oidc |

Postgres e Redis estao desativados por padrao (rodam no WSL). Ative os perfis do docker-compose quando precisar.

## Autenticacao (Fase 1 — MVP)

Rotas HTTP disponiveis quando `DATABASE_URL` e `SESSION_SECRET` estao definidas:

| Metodo | Rota | Descricao |
|--------|------|-----------|
| POST | /auth/register | Cadastro de usuario (email + senha) |
| POST | /auth/login | Login; cria sessao em cookie httpOnly |
| POST | /auth/logout | Invalida a sessao atual |
| GET | /auth/me | Retorna o usuario autenticado |

Senhas armazenadas com `argon2id`. Sessao via cookie `mangashelf-session` (HttpOnly, SameSite=Lax, Secure em producao).

## Desenvolvimento

```bash
go build ./...
go test ./...
```

## Documentacao

- Roadmap: `MangaShelf-Roadmap-v1.1.md`
- ADRs: `docs/adr/`
- Seguranca: `SECURITY.md`

---
AGPL-3.0 — veja `LICENSE`. Sem garantia; cada operador responde por sua instancia.

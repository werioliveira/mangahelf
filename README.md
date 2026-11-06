# MangaShelf

Plataforma open source de hospedagem de mangás.

Stack: Go + Next.js · PostgreSQL · Self-hosted

## Estrutura
`
/cmd/api, /cmd/worker
/internal (config, ports, adapters, services, auth, authz, audit)
/apps/web (Next.js App Router)
/migrations (0001_core)
/api (OpenAPI)
/deploy
/docs (AUTH.md, DESIGN_PATTERN.md, ROADMAP.md, etc.)
`

## Desenvolvimento
`ash
cp deploy/.env.example .env
npm run db:migrate
go run ./cmd/api
go run ./cmd/worker
npm run dev:web
`

## Documentação
- [Changelog](CHANGELOG.md)
- [Auth](docs/AUTH.md)
- [Design Pattern](docs/DESIGN_PATTERN.md)
- [Roadmap](docs/ROADMAP.md)

Licença AGPL-3.0


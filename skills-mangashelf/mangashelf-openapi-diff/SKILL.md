---
name: mangashelf-openapi-diff
description: "Compara api/openapi.yaml entre branch atual e main, detecta breaking changes e bloqueia PRs que quebram contrato sem bump de versão major."
---

# MangaShelf OpenAPI Diff

Skill de **gate de CI** que detecta breaking changes no contrato OpenAPI e bloqueia PRs que as introduzem sem bump de versão major.

## Quando ativar

Em todo PR que modifica `api/openapi.yaml`. Roda automaticamente em CI.

## O que conta como breaking change

### Definitivamente breaking (bloqueia)

- Remover path/endpoint existente.
- Remover método HTTP de path existente.
- Remover campo de response.
- Tornar campo de request required (era opcional).
- Adicionar campo required em request body.
- Mudar tipo de campo (string → integer, etc.).
- Mudar formato (int32 → int64, date → date-time).
- Restringir enum (remover valores).
- Mudar status code de sucesso para erro.
- Adicionar header required em response.
- Mudar semântica de campo existente (mesmo nome, comportamento diferente).

### Possivelmente breaking (avalia)

- Adicionar campo em request body (client antigo não manda, server aceita?).
- Adicionar header em request (client antigo não manda).
- Mudar descrição de campo (pode confundir client gerado).
- Mudar example.

### Aditivo (não breaking, OK)

- Adicionar novo endpoint.
- Adicionar novo campo em response (opcional).
- Adicionar novo valor em enum.
- Adicionar novo status code (não substitui existente).
- Relaxar validação (required → optional, maxlength maior).

## Como validar

### Tool: `oasdiff`

```bash
# Instalar: go install github.com/Tufin/oasdiff@latest
# Ou Docker: docker run tufin/oasdiff

oasdiff breaking origin/main.yaml pr.yaml
```

Saída em caso de breaking:

```
Breaking changes detected:
  - GET /api/v1/chapters/{id}: removed response status 200
  - POST /api/v1/auth/login: required field `device_id` added to request body
```

### CI integration

```yaml
# .github/workflows/openapi-diff.yml
jobs:
  openapi-diff:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - name: Extract openapi.yaml from main
        run: git show origin/main:api/openapi.yaml > /tmp/main.yaml
      - name: Run oasdiff
        run: |
          docker run --rm -v /tmp:/specs -v $PWD/api:/pr tufin/oasdiff breaking /specs/main.yaml /pr/openapi.yaml
      - name: Check version bump
        run: |
          OLD_VER=$(yq '.info.version' /tmp/main.yaml)
          NEW_VER=$(yq '.info.version' api/openapi.yaml)
          OLD_MAJOR=$(echo $OLD_VER | cut -d. -f1)
          NEW_MAJOR=$(echo $NEW_VER | cut -d. -f1)
          if [ "$OLD_MAJOR" = "$NEW_MAJOR" ]; then
            echo "❌ Breaking change sem bump de versão major"
            exit 1
          fi
```

## Estratégia de versionamento

### OpenAPI `info.version` (semver)

```yaml
openapi: 3.1.0
info:
  title: MangaShelf API
  version: 1.4.2  # major.minor.patch
```

- `patch` (1.4.2 → 1.4.3): bug fix, sem mudança de contrato.
- `minor` (1.4.2 → 1.5.0): adição aditiva (novo endpoint, novo campo opcional).
- `major` (1.4.2 → 2.0.0): breaking change. Exige janela de depreciação de 6 meses.

### URL versioning

- `/api/v1/...` para OpenAPI `1.x.y`.
- `/api/v2/...` para OpenAPI `2.x.y`.
- Bump de URL SÓ quando há breaking change.
- Versão antiga da URL permanece disponível por 6 meses após a nova.
- Header `Sunset: Sat, 31 Aug 2025 00:00:00 GMT` em respostas da versão antiga indicando deprecação.

## Workflow quando há breaking change

1. **Não bloquear; planejar.** Breaking changes são necessários, mas exigem processo.
2. Criar ADR (`mangashelf-adr-check`) explicando o porquê.
3. Bump major no OpenAPI e na URL (`/v1/` → `/v2/`).
4. Manter `/v1/` funcionando por 6 meses com header `Sunset`.
5. Atualizar client TS gerado.
6. Documentar em `CHANGELOG.md` e `docs/migrations/v2.md`.
7. Notificar operadores via release notes.

## Campos deprecated

Quando for remover campo mas ainda manter compatibilidade:

```yaml
components:
  schemas:
    Chapter:
      properties:
        legacy_field:
          type: string
          deprecated: true
          description: "Use `new_field` instead. Removido em v2."
        new_field:
          type: string
```

`oasdiff` detecta e avisa (não bloqueia). Client gerado marca como `@deprecated`.

## Validação de examples

```bash
# Verifica que examples do OpenAPI batem com schema
redocly lint api/openapi.yaml
```

## Saída de erro

```
❌ Breaking changes detectadas em api/openapi.yaml:

  1. POST /api/v1/auth/login
     - removed response: 200 (substituído por 201?)
     - added required field: `device_id` no body
       → se necessário, torne opcional ou crie /v2/

  2. GET /api/v1/chapters/{id}
     - changed field type: `pages` de array para object

Para aceitar estas mudanças:
1. Bump versão: 1.4.2 → 2.0.0
2. Atualizar URL: /api/v1/ → /api/v2/
3. Criar ADR em docs/adr/ justificando
4. Manter /v1/ com Sunset header por 6 meses
```

## Configuração

`api/.openapi-diff.yml`:

```yaml
breaking_change_policy: block  # block | warn | allow
version_bump_required: true    # exigir bump major em breaking
deprecation_window: 6 months   # período entre deprecate e remove
excluded_paths:
  - /api/v1/internal/*  # endpoints internos não contam
```

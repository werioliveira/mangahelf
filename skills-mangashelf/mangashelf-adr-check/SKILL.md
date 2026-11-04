---
name: mangashelf-adr-check
description: "Valida que todo PR que muda arquitetura cria ou atualiza um ADR em docs/adr/. Bloqueia merge sem ADR para mudanças estruturais."
---

# MangaShelf ADR Check

Skill de **gate de CI** que valida que toda mudança arquitetural vem acompanhada de um ADR (Architecture Decision Record).

## Quando ativar

Em todo PR. Verifica diff de arquivos e bloqueia se houver mudança estrutural sem ADR correspondente.

## O que conta como "mudança arquitetural"

Qualquer um destes gatilhos:

### Backend (Go)
- Adicionar/remover módulo em `apps/api/internal/`
- Mudar interface em `internal/*/ports.go` ou `internal/*/interfaces.go`
- Adicionar/remover driver de `Storage`, `SearchEngine`, `ImageURLBuilder`, `Cache`, `Authenticator`, `Authorizer`, `PaymentProvider`, `PushProvider`
- Mudar modelo de dados de authz (tabelas `permissions`, `roles`, `user_overrides`)
- Mudar modelo de dados de VIP (tabelas `vip_*`)
- Adicionar/remover dependência em `go.mod` que não seja patch/security
- Mudar versão major de Go, Postgres, ou qualquer serviço em `docker-compose.yml`

### Frontend (Next.js)
- Mudar versão major de Next.js
- Adicionar/remover biblioteca de state management (TanStack Query, Zustand, Redux)
- Mudar estratégia de data fetching (ex.: introduzir GraphQL)
- Adicionar/remover rota de API custom

### API
- Mudar versão major do OpenAPI (`/v1/` → `/v2/`)
- Adicionar/remover endpoint público
- Mudar contrato de erro (RFC 9457 → outro)
- Mudar autenticação (JWT → outra coisa)

### Infra
- Mudar proxy (Caddy → nginx)
- Mudar CDN (Cloudflare → outro)
- Mudar provedor de storage default
- Mudar provedor de deploy (VPS → Kubernetes, etc.)

### Processo
- Mudar licença
- Mudar branching strategy
- Adicionar/remover gate de CI bloqueante

## O que NÃO conta (não precisa ADR)

- Bug fix.
- Refactor que não muda comportamento.
- Adicionar testes.
- Atualizar dependência minor/patch.
- Mudar estilo de código.
- Adicionar endpoint dentro de módulo já existente.
- Mudança em documentação.

## Como validar

### Regra do diff

Para cada arquivo tocado no PR, o checker decide se é "arquitetural" (pelos padrões acima). Se sim, exige que o PR também toque `docs/adr/` com um arquivo novo ou modificado.

### Padrões de arquivo que disparam check

```yaml
# .adr-check.yml
triggers:
  - path: "apps/api/internal/*/ports.go"
    type: "interface change"
  - path: "apps/api/internal/*/interfaces.go"
    type: "interface change"
  - path: "go.mod"
    type: "dependency change (require ADR if new major version or new dep)"
  - path: "deploy/docker-compose.yml"
    type: "service topology change"
  - path: "api/openapi.yaml"
    type: "API contract change"
    pattern: "version: '2"
    description: "OpenAPI major version change"
  - path: "apps/web/next.config.*"
    type: "Next.js config change (require ADR if breaking)"

non_triggers:
  - "**/*.md"
  - "**/*_test.go"
  - "**/*.test.ts"
  - "**/*.spec.ts"
  - "docs/adr/**"   # o próprio ADR
```

## Formato de ADR

```markdown
# ADR-0007: Adicionar driver de push notification

**Status:** aceito
**Data:** 2024-01-15
**Autor:** alice@example.com
**PR:** #123
**Supersedes:** ADR-0003 (parcial) — adiciona um novo driver, não substitui

## Contexto

O app mobile (futuro) precisa de push notifications para novos capítulos. Backend atual não tem suporte.

## Decisão

Adicionar interface `PushProvider` em `internal/notifications/ports.go` com drivers `fcm`, `apns`, `both`, `none` (default em dev). Config via `PUSH_DRIVER`.

## Alternativas consideradas

1. **Usar só Firebase Cloud Messaging** — não cobre iOS sem intermediary. Rejeitado.
2. **Usar serviço terceiro (OneSignal, Pusher)** — adiciona dependência externa e custo. Rejeitado por ora; pode ser revisitado se manutenção dos drivers diretos for cara.
3. **Não ter push, só in-app + email** — UX ruim para notificação de capítulo novo. Rejeitado.

## Consequências

- Positivas: app mobile tem push desde o MVP; sem vendor lock-in.
- Negativas: mais código para manter (FCM + APNs); chaves de API para gerenciar.
- Neutras: documentação em `docs/runbooks/push-troubleshooting.md` precisa ser criada.

## Compliance

- Não viola ADR-0001 (licença AGPL).
- Não viola ADR-0005 (segurança ASVS L2); tokens são guardados com criptografia em repouso.
```

## Validação em CI

```yaml
# .github/workflows/adr-check.yml
jobs:
  adr-check:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0  # precisa do histórico
      - uses: actions/setup-python@v5
      - run: pip install pyyaml
      - run: python scripts/adr_check.py
```

`scripts/adr_check.py` (a ser criado no repo):

```python
#!/usr/bin/env python3
"""Verifica que PRs com mudanças arquiteturais têm ADR."""
import sys, os, yaml, subprocess
from pathlib import Path

# 1. Lista arquivos tocados no PR (vs main)
files = subprocess.check_output(
    ["git", "diff", "--name-only", "origin/main...HEAD"],
    text=True
).splitlines()

# 2. Carrega configuração
config = yaml.safe_load(Path(".adr-check.yml").read_text())

# 3. Identifica triggers
triggered = []
for f in files:
    for t in config["triggers"]:
        if Path(f).match(t["path"]):
            triggered.append((f, t["type"]))

# 4. Se há triggers, exige ADR
adr_files = [f for f in files if f.startswith("docs/adr/")]

if triggered and not adr_files:
    print("❌ Mudança arquitetural detectada sem ADR:")
    for f, t in triggered:
        print(f"  - {f}: {t}")
    print("Crie um ADR em docs/adr/NNNN-titulo.md.")
    sys.exit(1)

print("✅ ADR check passou")
```

## Como rodar local

```bash
make adr-check
# ou
python scripts/adr_check.py
```

## Templates

`docs/adr/0000-template.md`:

```markdown
# ADR-NNNN: <título curto>

**Status:** proposto | aceito | rejeitado | deprecado | superseded by ADR-XXXX
**Data:** YYYY-MM-DD
**Autor:** <email>
**PR:** #NNN
**Supersedes:** ADR-XXXX (se aplicável)

## Contexto

<por que precisamos decidir agora? qual o problema?>

## Decisão

<o que decidimos?>

## Alternativas consideradas

1. <alternativa 1> — <por que rejeitada>
2. <alternativa 2> — <por que rejeitada>

## Consequências

- Positivas: <...>
- Negativas: <...>
- Neutras: <...>

## Compliance

<verifica se a decisão conflita com ADRs existentes>
```

## Número do ADR

- Numerados sequencialmente (`0001`, `0002`, ...).
- Nunca reusar número, mesmo de ADR rejeitado.
- ADR `superseded` mantém o número, com novo ADR apontando para ele.

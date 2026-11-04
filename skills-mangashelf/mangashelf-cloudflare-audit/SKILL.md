---
name: mangashelf-cloudflare-audit
description: "Roda o checklist da seção 7.10 do roadmap via API do Cloudflare. Valida SSL/TLS, WAF, Bot Fight, Rate Limiting, DNSSEC, Cache Rules, Transform Rules, Cloudflare Access, Workers, Logpush. Bloqueia release se itens críticos falharem."
---

# MangaShelf Cloudflare Audit

Skill de **gate de release** que valida a configuração Cloudflare da instância de produção MangaShelf via API.

## Quando ativar

Antes de cada release (junto com `mangashelf-asvs-checklist`). Requer:
- `CLOUDFLARE_API_TOKEN` no CI (escopo: Account + Zone read).
- `CLOUDFLARE_ACCOUNT_ID` e `CLOUDFLARE_ZONE_ID` configurados.

## Pré-requisitos

- Conta Cloudflare com a zona do domínio do MangaShelf.
- API token com permissões:
  - Zone:Zone:Read
  - Zone:Zone Settings:Read
  - Zone:WAF:Read
  - Zone:Firewall Services:Read
  - Zone:Page Rules:Read
  - Zone:Cache Rules:Read
  - Zone:Transform Rules:Read
  - Zone:Rate Limit:Read
  - Account:Access:Apps and Policies:Read (se usar Cloudflare Access)

## Checklist (12 camadas)

### 1. SSL/TLS

```bash
curl -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/settings/ssl" \
  -H "Authorization: Bearer $TOKEN"
```

- Esperado: `"value": "strict"` (Full strict).
- TLS 1.3 habilitado: `"value": "on"` em `/settings/tls_1_3`.
- Minimum TLS version: `"value": "1.2"`.
- HSTS: `max-age=63072000; includeSubDomains; preload`.

### 2. WAF

- Managed Ruleset Cloudflare ativado.
- OWASP Core Ruleset ativado.
- Custom rules para `/admin/*` bloqueando path traversal (`../`, `..%2f`) e SQLi patterns.

```bash
curl -X GET "https://api.cloudflare.com/client/v4/zones/$ZONE_ID/rulesets" \
  -H "Authorization: Bearer $TOKEN"
```

### 3. Bot Fight Mode

- Domínio da aplicação: Bot Fight Mode = on.
- Domínio de mídia (`img.*`): Super Bot Fight Mode = on.

### 4. Rate Limiting

Regras configuradas para:

| Path | Limite | Por |
| --- | --- | --- |
| `/api/v1/auth/login` | 5/min | IP |
| `/api/v1/auth/register` | 3/min | IP |
| `/api/v1/mobile/*` | 60/min | user (header `X-Device-Id`) |
| `/api/v1/admin/*` | 30/min | user |

### 5. DNSSEC

- DNSSEC habilitado no domínio apex.
- CAA records: `0 issue "letsencrypt.org"`, `0 issuewild ";"`.

### 6. Cache Rules

- Assets estáticos (`*.js`, `*.css`, `*.woff2`, imagens em `img.*`): `Cache-Control: public, max-age=31536000, immutable`.
- HTML do Next (`/`, `/series/*`, `/chapters/*`): `s-maxage=60, stale-while-revalidate=60`.
- API (`/api/*`): `Cache-Control: no-store`.

### 7. Transform Rules (Headers)

Injetados na borda como defense in depth com o Caddy:

```http
Strict-Transport-Security: max-age=63072000; includeSubDomains; preload
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: strict-origin-when-cross-origin
Permissions-Policy: camera=(), microphone=(), geolocation=()
```

### 8. Cloudflare Access

- Aplicação protege `/admin/*` com zero-trust.
- Política: OIDC + lista de e-mails autorizados (ou grupo do IdP).
- Não bloqueia rotas públicas (`/`, `/series/*`, `/chapters/*`).

### 9. Workers

- Worker na borda valida HMAC das URLs assinadas para conteúdo VIP/early.
- Worker retorna 401 se assinatura inválida ou expirada.
- Worker não loga tokens.

### 10. Logpush

- Logs de WAF, access e firewall exportados para R2/S3.
- Formato: raw JSON.
- Retenção: 90 dias.
- Destino: `s3://mangashelf-logs/cloudflare/YYYY/MM/DD/`.

### 11. Page Rules / Redirect Rules (legacy)

- Page Rules: deve estar vazio (migrado para Cache Rules e Redirect Rules).
- Redirect Rules: HTTP → HTTPS automático.

### 12. Radar / Threat intel

- Não é gate; é monitoramento contínuo.
- Alerta semanal se houver campanha de ataque em curso.

## Como rodar

### Local (manual)

```bash
export CLOUDFLARE_API_TOKEN=...
export CLOUDFLARE_ZONE_ID=...
export CLOUDFLARE_ACCOUNT_ID=...
make cloudflare-audit
```

### CI (job de release)

```yaml
# .github/workflows/release-gate.yml
jobs:
  cloudflare-audit:
    runs-on: ubuntu-latest
    environment: production
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
      - env:
          CLOUDFLARE_API_TOKEN: ${{ secrets.CF_API_TOKEN }}
          CLOUDFLARE_ZONE_ID: ${{ secrets.CF_ZONE_ID }}
          CLOUDFLARE_ACCOUNT_ID: ${{ secrets.CF_ACCOUNT_ID }}
        run: python scripts/cloudflare_audit.py
```

### Script

`scripts/cloudflare_audit.py` (a ser criado no repo):

```python
#!/usr/bin/env python3
"""Audita configuração Cloudflare via API."""
import os, sys, json, requests

ZONE = os.environ["CLOUDFLARE_ZONE_ID"]
ACCOUNT = os.environ["CLOUDFLARE_ACCOUNT_ID"]
TOKEN = os.environ["CLOUDFLARE_API_TOKEN"]
H = {"Authorization": f"Bearer {TOKEN}", "Content-Type": "application/json"}
BASE = "https://api.cloudflare.com/client/v4"

def get(path):
    r = requests.get(f"{BASE}{path}", headers=H)
    r.raise_for_status()
    return r.json()["result"]

# 1. SSL/TLS
ssl = get(f"/zones/{ZONE}/settings/ssl")
assert ssl["value"] == "strict", f"SSL mode deve ser 'strict', é {ssl['value']}"

# 2. TLS 1.3
tls13 = get(f"/zones/{ZONE}/settings/tls_1_3")
assert tls13["value"] == "on", "TLS 1.3 deve estar on"

# 3. Minimum TLS version
min_tls = get(f"/zones/{ZONE}/settings/min_tls_version")
assert min_tls["value"] == "1.2", f"Min TLS deve ser 1.2, é {min_tls['value']}"

# 4. HSTS
hsts = get(f"/zones/{ZONE}/settings/security_header")
assert hsts["value"]["strict_transport_security"]["enabled"] is True
assert hsts["value"]["strict_transport_security"]["max_age"] >= 63072000
assert hsts["value"]["strict_transport_security"]["include_subdomains"] is True
assert hsts["value"]["strict_transport_security"]["preload"] is True

# 5. Bot Fight Mode
bot = get(f"/zones/{ZONE}/bot_management")
assert bot.get("enable_js") is True, "Bot Fight deve estar habilitado"

# 6. WAF rulesets
rulesets = get(f"/zones/{ZONE}/rulesets")
managed = [r for r in rulesets if r["phase"] == "http_request_main" and r["kind"] == "managed"]
assert len(managed) >= 2, "WAF deve ter Cloudflare Managed + OWASP CRS"

# 7. DNSSEC
dnssec = get(f"/zones/{ZONE}/dnssec")
assert dnssec["status"] == "active", f"DNSSEC deve estar active, é {dnssec['status']}"

# 8. Cloudflare Access (se configurado)
try:
    apps = get(f"/accounts/{ACCOUNT}/access/apps")
    admin_app = [a for a in apps if "/admin" in a.get("domain", "")]
    assert len(admin_app) >= 1, "Cloudflare Access deve proteger /admin"
except Exception as e:
    print(f"⚠️  Cloudflare Access não configurado: {e}")

print("✅ Cloudflare audit aprovado")
```

## Saída de erro

```
❌ Cloudflare audit falhou:

  [SSL/TLS] mode é 'flex', esperado 'strict'
    → ajustar em Cloudflare dashboard > SSL/TLS > Overview

  [WAF] apenas 1 ruleset managed (Cloudflare Managed); OWASP CRS ausente
    → ativar em Security > WAF > Managed rules

  [DNSSEC] status é 'disabled', esperado 'active'
    → ativar em DNS > Settings > DNSSEC

  [Cache Rules] regra para assets estáticos ausente
    → criar em Caching > Cache Rules para *.js, *.css, imagens

  [Cloudflare Access] /admin não protegido
    → criar aplicação em Zero Trust > Access > Applications

Release bloqueada. Resolva e rode novamente.
```

## Itens bloqueantes vs informativos

**Bloqueantes (V1-V9 do ASVS):** SSL/TLS, WAF, DNSSEC, Cloudflare Access, Bot Fight.

**Informativos (warning):** Logpush, Page Rules legadas, Radar.

## Frequência

- Antes de cada release: full audit.
- Mensal: Quick audit (SSL, WAF, Access).
- Semanal: Threat intel check (Radar).

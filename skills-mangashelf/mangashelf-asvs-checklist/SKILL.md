---
name: mangashelf-asvs-checklist
description: "Checklist do OWASP ASVS Nível 2 para MangaShelf. Roda por release (não por PR) e gera relatório em docs/security/asvs-<version>.md. Bloqueia release se itens V1-V9 falharem."
---

# MangaShelf ASVS Checklist

Skill de **gate de release** que valida conformidade com OWASP ASVS Nível 2. Não roda por PR (caro); roda antes de cada release.

## Quando ativar

Antes de taggear uma nova release (ex.: `v0.1.0`, `v0.2.0`). Roda em CI job `release-gate`.

## OWASP ASVS Nível 2 — Versão 4.0

ASVS L2 é o piso de segurança para aplicações que processam dados sensíveis. MangaShelf hospeda conteúdo (potencialmente com direitos autorais) e dados pessoais de usuários, então L2 é o mínimo.

### Estrutura do ASVS

- **V1:** Architecture, Threats and Documentation
- **V2:** Authentication
- **V3:** Session Management
- **V4:** Access Control
- **V5:** Validation, Sanitization and Encoding
- **V6:** Stored Cryptography
- **V7:** Error Handling and Logging
- **V8:** Data Protection
- **V9:** Communications
- **V10:** Malicious Code
- **V11:** Business Logic
- **V12:** Files and Resources
- **V13:** API and Web Service
- **V14:** Configuration

## Checklist MangaShelf (resumo dos itens V1-V9 — os bloqueantes)

### V1 — Arquitetura

- [ ] **V1.1.1** — ADRs documentados para decisões de segurança (authz, VIP, storage).
- [ ] **V1.2.1** — Componentes de segurança (authz, storage) são isolados em módulos Go.
- [ ] **V1.3.1** — Dados sensíveis não passam por componentes não-confiáveis.
- [ ] **V1.4.1** — Threat model documentado em `docs/security/threat-model.md`.

### V2 — Autenticação

- [ ] **V2.1.1** — Senhas com `argon2id` (parâmetros calibrados em ADR-005).
- [ ] **V2.2.1** — Account lockout após 5 tentativas falhas (10 min de cooldown).
- [ ] **V2.3.1** — Verificação de e-mail obrigatória antes de ações sensíveis.
- [ ] **V2.4.1** — Recuperação de senha via token de uso único, expiração < 1h.
- [ ] **V2.5.1** — MFA (TOTP/passkeys) disponível; obrigatório para `role.manage` e `instance.configure`.
- [ ] **V2.6.1** — OAuth social via `goth` (Google, GitHub); não inventa cripto.
- [ ] **V2.7.1** — Tokens gerados com `crypto/rand`; nunca `math/rand`.

### V3 — Sessões

- [ ] **V3.1.1** — Session ID rotacionado no login (scs).
- [ ] **V3.2.1** — Cookie `httpOnly`, `Secure`, `SameSite=Lax`.
- [ ] **V3.3.1** — Sessão expira em 30 dias de inatividade.
- [ ] **V3.4.1** — Logout invalida sessão no server (não só no client).
- [ ] **V3.5.1** — Troca de senha revoga todas as sessões.
- [ ] **V3.6.1** — Sessões são revogáveis pelo admin.

### V4 — Controle de acesso

- [ ] **V4.1.1** — Permissões atômicas (sem `role == admin`).
- [ ] **V4.2.1** — Deny vence allow.
- [ ] **V4.3.1** — Anti-escalação por `position` em roles.
- [ ] **V4.4.1** — IDs validados por escopo (404 não 403).
- [ ] **V4.5.1** — Resolver único de acesso `CanRead` para VIP.
- [ ] **V4.6.1** — CORS restrito a domínios conhecidos.

### V5 — Validação

- [ ] **V5.1.1** — OpenAPI estrito via `oapi-codegen`.
- [ ] **V5.2.1** — Uploads validam magic bytes (não extensão).
- [ ] **V5.3.1** — ZIP protegido contra zip bomb (limite 200× descompressão).
- [ ] **V5.4.1** — Sanitização na renderização (sem HTML cru em título/sinopse).
- [ ] **V5.5.1** — Markdown com whitelist de tags.
- [ ] **V5.6.1** — IDs são UUIDs (não sequenciais).

### V6 — Criptografia

- [ ] **V6.1.1** — TLS 1.3 via Caddy (HSTS preload).
- [ ] **V6.2.1** — Senhas com `argon2id`.
- [ ] **V6.3.1** — URLs VIP/early assinadas com HMAC (TTL 5-15 min).
- [ ] **V6.4.1** — Refresh tokens são 32 bytes aleatórios.
- [ ] **V6.5.1** — Push tokens guardados com AES-256-GCM em repouso.
- [ ] **V6.6.1** — Segredos nunca em código/logs (`gitleaks` em CI).

### V7 — Logging e erros

- [ ] **V7.1.1** — Logs estruturados em JSON (`slog`).
- [ ] **V7.2.1** — Sem PII em logs (senha, e-mail, token).
- [ ] **V7.3.1** — `audit_log` append-only (sem UPDATE/DELETE).
- [ ] **V7.4.1** — Erros não vazam stack trace em produção.
- [ ] **V7.5.1** — Erros seguem RFC 9457 (problem+json).

### V8 — Proteção de dados

- [ ] **V8.1.1** — EXIF removido de toda imagem.
- [ ] **V8.2.1** — Conteúdo VIP saí com `Cache-Control: private`.
- [ ] **V8.3.1** — Bucket S3 privado por padrão.
- [ ] **V8.4.1** — Backup de banco é criptografado (`restic`).
- [ ] **V8.5.1** — Dados pessoais exportáveis e removíveis (LGPD/GDPR).

### V9 — Comunicações

- [ ] **V9.1.1** — TLS 1.3 em toda comunicação.
- [ ] **V9.2.1** — Domínio de mídia separado do app (sem cookies cruzados).
- [ ] **V9.3.1** — WebSocket (se usado) com origin check.
- [ ] **V9.4.1** — Webhooks assinados (HMAC) com replay protection.

## Como rodar

```bash
make asvs-check
# Gera docs/security/asvs-<version>.md
# Bloqueia release se houver item V1-V9 falhando
```

### Saída

```markdown
# ASVS L2 Compliance Report — v0.2.0

**Data:** 2024-01-15
**Versão ASVS:** 4.0.3
**Status:** ✅ Aprovado (47/47 itens V1-V9; 23/26 V10-V14 com notas)

## Resumo

| Capítulo | Aprovado | Falhando | N/A |
| --- | --- | --- | --- |
| V1 Architecture | 4/4 | 0 | 0 |
| V2 Authentication | 7/7 | 0 | 0 |
| V3 Sessions | 6/6 | 0 | 0 |
| V4 Access Control | 6/6 | 0 | 0 |
| V5 Validation | 6/6 | 0 | 0 |
| V6 Cryptography | 6/6 | 0 | 0 |
| V7 Logging | 5/5 | 0 | 0 |
| V8 Data Protection | 5/5 | 0 | 0 |
| V9 Communications | 4/4 | 0 | 0 |
| V10 Malicious Code | 2/3 | 0 | 1 | (SCA automatizado, não manual)
| V11 Business Logic | 3/4 | 0 | 1 |
| V12 Files | 4/4 | 0 | 0 |
| V13 API | 5/5 | 0 | 0 |
| V14 Configuration | 4/5 | 0 | 1 |
| **TOTAL** | **67/72** | **0** | **5** |

## Itens N/A (justificados)

- **V10.3.1** (verificação manual de malicious code): automatizado via `gosec` + `npm audit` em CI; não há inspeção manual.
- **V11.4.1** (rate limit por transação financeira): MangaShelf não tem transações financeiras no MVP.
- **V14.5.1** (container scan com runtime protection): Trivy em CI; runtime protection é do orquestrador (Docker Compose), não aplicável.

## Itens pendentes (não bloqueantes para L2)

- V13.4.1 — Schema de API não documenta todos os códigos de erro. Issue: #234.
- V14.3.1 — Header `X-Content-Type-Options` em assets estáticos. Issue: #235.
```

## Items bloqueantes

Apenas V1-V9 são bloqueantes para release. V10-V14 são warnings (issue aberto, resolvido em próxima release).

## Políticas

- Item falhando em V1-V9 = release bloqueada.
- Item falhando em V10-V14 = warning; precisa ter issue aberta antes do release.
- Item marcado N/A precisa justificativa no relatório.
- Relatório é commitado em `docs/security/asvs-<version>.md` e linkado nas release notes.

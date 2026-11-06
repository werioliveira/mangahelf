# MangaShelf: Roadmap de Desenvolvimento

MangaShelf · Versão 1.1 · Licença AGPL-3.0 · Stack: Go + Next.js · Deploy: VPS self-hosted com Docker Compose

---

## 1. Visão e princípios

MangaShelf é uma plataforma open source de hospedagem de mangás, robusta e fácil de auto-hospedar. O projeto entrega apenas o software: não hospeda, não opera e não cura conteúdo, e cada instância é de responsabilidade de quem a instala.

**Princípios:**

- **Monolito modular.** Um repositório, poucos binários, sem microsserviços.
- **Portas e adaptadores.** O código de negócio depende de interfaces; o `.env` escolhe a implementação na inicialização.
- **Padrões sensatos, extras opcionais.** Tudo funciona com o conjunto mínimo (Postgres + disco local); Meilisearch, Redis, imgproxy e OIDC são ativados por flag.
- **Autorização granular desde o dia 1.** Nenhuma checagem do tipo `role == admin`; só permissões atômicas com escopo.
- **Segurança e conformidade por design.** O OWASP é o mínimo obrigatório (seção 7). Upload isolado, moderação e auditoria fazem parte do núcleo, não de um "depois".
- **Comece simples, abstraia cedo.** Postgres FTS antes de Meilisearch, disco local antes de S3, mas sempre atrás de interface.
- **Painel administrativo coeso.** A administração é um módulo de primeira classe, com design e comportamento únicos (seção 6).
- **VIP nativo.** Acesso antecipado e conteúdo exclusivo fazem parte do modelo de acesso desde o início (seção 5).
- **Software, não serviço.** O projeto fornece o código e modelos de documentos; a operação e o conteúdo de cada instância são responsabilidade do operador. Questões judiciais relativas a conteúdo hospedado em uma instância são do operador, não do projeto — o README, o LICENSE e os modelos de termos deixam isso explícito.

---

## 2. Stack e decisões

| Camada | Escolha | Observação |
| --- | --- | --- |
| API | Go + `net/http` (1.22+) com `chi` | Leve e fácil de contribuir |
| Banco | PostgreSQL + `pgx` + `sqlc` | Queries tipadas, sem ORM |
| Migrations | `goose` ou `atlas` | Schema versionado |
| Jobs | `River` (fila em Postgres) | Redis deixa de ser requisito |
| Sessões | `scs` com store em Postgres | Redis opcional |
| Imagens | `imgproxy` (padrão) ou modo `static` com `govips` | Veja seção 3 |
| Storage | Local ou qualquer S3-compatível (AWS S3, R2, MinIO, Garage, SeaweedFS, B2), veja seção 8 | Banco guarda só a chave |
| Busca | Postgres FTS (padrão) ou Meilisearch | `unaccent` + `pg_trgm` |
| Auth | Builtin (padrão) ou OIDC | Autorização sempre própria |
| Contrato | OpenAPI + `oapi-codegen` + client TS gerado | Front e back sincronizados |
| Front | Next.js (App Router) + TypeScript + Tailwind + TanStack Query | SSR/ISR para SEO |
| Proxy | Caddy | HTTPS automático |
| CDN | Cloudflare na frente | Banda de imagens é o gargalo |
| Observabilidade | `slog`, Prometheus, Grafana, Uptime Kuma | — |
| Backup | `pg_dump` + `restic` | Para storage externo |

**Por que Next.js e não React puro:** páginas de série e capítulo precisam ser indexáveis. O Next entrega SSR/ISR pronto; um SPA exigiria montar isso manualmente. Não use o otimizador de imagens do Next para as páginas dos mangás; esse papel é do imgproxy ou do modo static.

---

## 3. Arquitetura e sistema de flags

### 3.1 Flags de configuração

```env
SEARCH_DRIVER=postgres        # postgres | meilisearch
IMAGE_PIPELINE=imgproxy       # imgproxy | static
STORAGE_DRIVER=s3             # local | s3
CACHE_DRIVER=memory           # memory | redis
AUTH_DRIVER=builtin           # builtin | oidc
```

O composition root (`main.go`) lê a config, valida e instancia os adaptadores. O resto do código só conhece as interfaces.

### 3.2 Interfaces (portas)

- `Storage`: put, get, delete, presign.
- `SearchEngine`: index, delete, search, `Capabilities()`.
- `ImageURLBuilder`: monta a URL final conforme o pipeline ativo.
- `Cache`: memória ou Redis.
- `Authenticator`: builtin ou OIDC.
- `Authorizer`: sempre implementação própria.

```go
// Exemplo: ImageURLBuilder — a URL nunca é persistida, só gerada em tempo de resposta.
type ImageURLBuilder interface {
    // Build retorna a URL pública ou assinada para uma página em uma variante.
    // variant ∈ {"thumb", "medium", "full"}; page contém storage_key e flags de acesso.
    Build(ctx context.Context, page Page, variant string) (string, error)
}

// Implementação imgproxy: HMAC(key/salt) sobre path+resize+expiração.
// Implementação static: URL direta do CDN usando a chave pré-renderizada no upload.
```

### 3.3 Regras importantes

- **O banco nunca guarda URL, só `storage_key`.** O `ImageURLBuilder` gera a URL em tempo de resposta.
- **imgproxy:** URLs assinadas com HMAC (key/salt) para evitar abuso de CPU; cache na frente (Cloudflare ou Caddy), pois o imgproxy não tem cache próprio.
- **Modo static:** o worker gera as variantes (tamanhos e WebP/AVIF) no upload, e o front serve direto do storage/CDN.
- **Meilisearch:** a sincronização acontece por outbox ou jobs, nunca no request. Se o Meilisearch cair, nada quebra. Comando `cli search reindex` para quem ativar o driver com dados existentes.
- **Capabilities:** se um recurso não existe em um driver (ex.: facetas avançadas no FTS), a API expõe via `Capabilities()` e o front esconde o recurso.
- **Docker Compose profiles:** `--profile imgproxy`, `--profile meilisearch`, `--profile redis`. O serviço só sobe se for usado.
- **Validação de config no boot:** combinações inválidas falham com mensagem clara.

### 3.4 Módulos internos do Go

`auth`, `authz`, `catalog` (séries e capítulos), `media` (upload e processamento), `library` (follows e progresso), `moderation`, `search`, `notifications`.

Também existem `vip` (planos, assinaturas e regras de acesso) e `admin` (endpoints agregadores do painel administrativo).

### 3.5 Fluxo de upload

1. Usuário envia ZIP ou imagens (preferencialmente por URL pré-assinada).
2. A API valida tamanho, magic bytes e limite de páginas, e cria um job.
3. O worker extrai, remove metadados, normaliza, ordena as páginas e grava no storage com nome baseado em hash do conteúdo.
4. Se `IMAGE_PIPELINE=static`, gera também as variantes.
5. O capítulo passa de `processing` para `published`.

```go
// Esqueleto do worker de mídia (internal/media/worker.go)
func (w *Worker) ProcessChapter(ctx context.Context, job river.Job) error {
    ch, err := w.repo.Chapter(ctx, job.ChapterID)
    if err != nil {
        return fmt.Errorf("load chapter: %w", err)
    }
    files, err := w.zip.SafeExtract(ctx, job.StorageKey) // protege contra zip bomb
    if err != nil {
        return w.fail(ch, err)
    }
    files = media.StripMetadata(files)                    // remove EXIF
    media.NormalizeOrder(files)                           // ordena páginas
    for _, f := range files {
        key := media.ContentKey(f)                        // sha256 → chave deduplicável
        if err := w.storage.Put(ctx, key, f); err != nil {
            return w.fail(ch, err)
        }
    }
    if w.cfg.ImagePipeline == "static" {
        if err := w.variants.Generate(ctx, files); err != nil { // WebP/AVIF + tamanhos
            return w.fail(ch, err)
        }
    }
    return w.publish(ctx, ch) // processing → published
}
```

### 3.6 Fluxo de leitura

O front busca a lista de páginas na API e carrega as imagens direto do imgproxy/CDN com cache `immutable`. Os bytes das imagens nunca passam pelo Go.

```
Browser ──GET /api/chapters/{id}/pages──▶ API Go
            (valida CanRead, devolve [{page, url_assinada}])
Browser ──GET https://img.example.com/...──▶ imgproxy / CDN
            (bytes da imagem; nunca passam pela API)
```

---

## 4. Autenticação e autorização

### 4.1 Autenticação

- **`builtin` (padrão):** `scs` para sessões em cookie httpOnly, `argon2id`, `goth` para OAuth social, `go-webauthn` para passkeys, `pquerna/otp` para TOTP. Você escreve a cola, não a criptografia.
- **`oidc` (opcional):** aceita Keycloak, Authentik, Zitadel e outros IdPs.
- **Alternativa:** trocar o builtin por Ory Kratos (headless), se você não quiser manter recuperação de senha e MFA. A UI continua sendo sua.
- A auth vive no Go, não no Next, para existir uma única fonte de verdade.

### 4.2 Autorização por permissões atômicas

O código **nunca checa role**:

```go
if !authz.Can(ctx, user, "chapter.publish", authz.Group(groupID)) {
    return ErrForbidden
}
```

```sql
permissions      (key text primary key)
roles            (id, name, position int)
role_permissions (role_id, permission_key)
user_roles       (user_id, role_id)                 -- escopo global
group_members    (group_id, user_id, role_id)       -- escopo por grupo
user_overrides   (user_id, permission_key, effect)  -- allow/deny pontual
```

Regras de design:

- Role é só um nome para um pacote de permissões, editável pelo dono da instância.
- A mesma permissão pode valer globalmente ou só dentro de um grupo.
- **Anti-escalação:** ninguém concede o que não tem nem edita cargo acima do seu (`position`).
- **Deny vence allow.**
- O primeiro dono é criado via `cli bootstrap-owner`, sem role mágico no código.
- Toda ação sensível vai para o `audit_log`.

```go
// Ordem de avaliação do resolver (internal/authz/resolve.go)
// 1. user_overrides (deny vence allow) → 2. group_members → 3. user_roles → 4. negação
func (r *Resolver) Can(ctx context.Context, u User, perm string, scope Scope) bool {
    if eff, ok := r.overrides.Effect(u.ID, perm); ok {
        return eff == EffectAllow // override pontual tem precedência
    }
    if r.groupAllows(ctx, u, perm, scope) { return true }
    if r.roleAllows(ctx, u, perm) { return true }
    return false
}
```

---

## 5. VIP e controle de acesso ao conteúdo

O VIP é um cargo com a permissão `content.read_vip` e prazo de validade; o código nunca checa "é VIP" diretamente, e toda decisão de leitura passa por um único resolver.

### 5.1 Regras definidas pelo publicador

| Configuração | Quem lê | Quando |
| --- | --- | --- |
| Série pública, capítulo público | Todos | Ao publicar |
| Capítulo com acesso antecipado (`early` + `public_at`) | VIPs de imediato; demais usuários só a partir de `public_at` | Demais só a partir de `public_at` |
| Capítulo só VIP (`vip_only`) | Apenas VIPs | Sem abertura pública, ou com `public_at` opcional |
| Série só VIP (`vip_only`) | Apenas VIPs | Todos os capítulos herdam a regra |

### 5.2 Como funciona

- **Resolver único:** `access.CanRead(user, chapter)` é usado pela API, pelo leitor, pela busca e pelos feeds. Nenhuma outra parte do código decide acesso.

```go
// Resolver único de acesso (internal/vip/access.go)
func CanRead(ctx context.Context, u User, ch Chapter) bool {
    if isStaffOrEditor(ctx, u, ch.GroupID) { return true }        // equipe vê sem ser VIP
    switch ch.Access {
    case "public":
        return true
    case "early":
        if u.HasActiveVIP(ctx) { return true }                    // VIP não espera
        return ch.PublicAt != nil && !time.Now().Before(*ch.PublicAt)
    case "vip_only":
        if ch.Series.Access == "vip_only" { return u.HasActiveVIP(ctx) }
        return u.HasActiveVIP(ctx) || (ch.PublicAt != nil && !time.Now().Before(*ch.PublicAt))
    }
    return false
}
```

- **Liberação sem depender de job:** o acesso público é calculado na leitura (`now >= public_at`). Um job agendado em `public_at` só revalida o cache do Next, envia notificações e atualiza sitemap e índice de busca.
- **Expiração:** `vip_memberships.expires_at` vale na hora; um job avisa o usuário antes de vencer.
- **Concessão:** manual pelo admin, por voucher ou, opcionalmente, por provedor de pagamento (`PaymentProvider`, com `PAYMENTS_DRIVER=none` por padrão).
- **Escopo do publicador:** definir acesso e `public_at` é a permissão `chapter.set_access`, com escopo de grupo, então cada grupo controla só os próprios projetos.
- **Equipe:** quem tem permissão de editar ou moderar o conteúdo também o vê, sem ser VIP.

### 5.3 Proteção das imagens

- A API só devolve a lista de páginas a quem pode ler; sem ela, as chaves do storage não vazam.
- Chaves baseadas em hash de conteúdo, não adivinháveis, mais URLs assinadas de curta duração para conteúdo VIP e antecipado.
- Conteúdo restrito sai com `Cache-Control: private` ou com assinatura validada na borda, para a CDN nunca servir a imagem a quem não pode ler.
- Ao se tornar público, o capítulo passa a usar URLs públicas e cacheáveis.
- **Limite honesto:** quem pode ler consegue salvar as imagens. O objetivo é barrar o acesso casual, não é DRM. Mitigações: rate limit por conta, limite de sessões simultâneas e marca d'água opcional por flag.

### 5.4 Modelo de dados

```sql
series           add: access text                -- 'public' | 'vip_only'
chapters         add: access text                -- 'public' | 'early' | 'vip_only'
chapters         add: public_at timestamptz null
vip_plans        (id, name, duration, perks jsonb)
vip_memberships  (user_id, plan_id, starts_at, expires_at, source)  -- admin | voucher | payment
vip_vouchers     (code_hash, plan_id, max_uses, expires_at)

-- VIP ativo na hora da leitura (sem depender de job):
-- expires_at > now() é avaliado no resolver CanRead.
```

---

## 6. Painel administrativo

O painel é um módulo de primeira classe do MangaShelf: um único app em `/admin`, com design, navegação e padrões iguais em todas as telas.

### 6.1 Módulos

| Módulo | O que resolve | Permissão-base |
| --- | --- | --- |
| Visão geral | Métricas de leitura, uploads, filas e alertas | `admin.view` |
| Catálogo | Séries, capítulos, tags, autores, grupos, acesso e agendamento | `series.edit`, `chapter.publish` |
| Mídia | Fila de processamento, falhas, reprocessamento, uso de storage | `media.manage` |
| Usuários | Busca, perfil, sessões, banimento, concessão de VIP | `user.view`, `user.ban` |
| Cargos e permissões | Editor de cargos, matriz de permissões, overrides | `role.manage` |
| VIP | Planos, vouchers, assinaturas e expirações | `vip.manage` |
| Moderação | Denúncias, fila e remoção de conteúdo | `report.handle` |
| Auditoria | Log filtrável por usuário, ação e período | `audit.view` |
| Instância | Configurações em runtime, estado dos drivers, reindex de busca | `instance.configure` |
| Sistema | Saúde, jobs, versões e backups | `system.view` |

### 6.2 Princípios de coesão

- **Um app só:** rota `/admin` no mesmo Next.js, com o mesmo design system e o mesmo client gerado do OpenAPI; sem painel paralelo.
- **O servidor decide:** a UI esconde o que o usuário não pode fazer, mas toda ação é validada pelo `Authorizer` nos endpoints `/admin/*`.
- **Padrão único de telas:** tabela com busca, filtros, paginação por cursor, ordenação e ações em lote; estado na URL; atalhos de teclado.
- **Navegação por permissão:** cada pessoa vê só os módulos que pode usar.
- **Auditoria total:** toda mutação registra autor, alvo e valores antes e depois.
- **Drivers do `.env` só leitura:** o painel mostra o driver ativo e a saúde de cada um; o que for editável em runtime fica no banco.
- **Acessível e responsivo:** usável em tablet e celular para moderação rápida.

---

## 7. Segurança e conformidade (OWASP)

O OWASP ASVS **nível 2** é o mínimo obrigatório e o build segue essas regras como gate de CI bloqueante. Segurança não é uma fase: upload isolado, moderação e auditoria são núcleo, e os requisitos abaixo se aplicam desde o MVP.

### 7.1 Postura de segurança

- **Defense in depth:** proxy (Caddy) → API (validação + autorização) → banco (RLS/constraints) → storage (chaves não adivinháveis). Nenhuma camada confia na anterior.
- **Fail-safe defaults:** qualquer erro de autorização nega; qualquer config ambígua falha no boot.
- **Least privilege:** o worker não tem credencial de admin do banco; a API não tem credencial de escrita no storage de variantes estáticas; o painel só expõe o que a permissão permite.
- **Superfície mínima:** sem endpoint debug, sem admin "escondido" em porta separada — o `/admin/*` vive atrás do mesmo Authorizer com permissões atômicas.

### 7.2 Gates de CI de segurança (bloqueantes)

| Gate | Ferramenta | Critério de bloqueio |
| --- | --- | --- |
| SAST | `gosec` + `semgrep` | Qualquer achado crítico/alto bloqueia o merge |
| SCA Go | `govulncheck` | Vulnerabilidade afetando symbol em uso bloqueia |
| SCA TS | `npm audit` / `pnpm audit --prod` | CVE alto/crítico em dependência de produção bloqueia |
| Container scan | `Trivy` nas imagens docker | CVE crítico com fix disponível bloqueia |
| Secret scanning | `gitleaks` | Qualquer segredo detectado bloqueia |
| Dependências | Dependabot/Renovate | PR automático; janela de patch de 7 dias para críticos |

### 7.3 Headers HTTP e configuração do proxy

O Caddy aplica, no mínimo:

```caddyfile
header {
    Strict-Transport-Security "max-age=63072000; includeSubDomains; preload"
    Content-Security-Policy   "default-src 'self'; img-src 'self' https://img.{dominio} data:; object-src 'none'; frame-ancestors 'none'; base-uri 'self'"
    X-Content-Type-Options    "nosniff"
    X-Frame-Options           "DENY"
    Referrer-Policy           "strict-origin-when-cross-origin"
    Permissions-Policy        "camera=(), microphone=(), geolocation=()"
    Cross-Origin-Opener-Policy "same-origin"
    -Server
}
```

- CSP sem `unsafe-inline` em produção; nonces/hashes gerados pelo Next.
- Cookies: `Secure`, `HttpOnly`, `SameSite=Lax` (sessão); `SameSite=Strict` para ações do `/admin`.
- Domínio de mídia separado (`img.exemplo.com`): sem cookies, sem JS — limita impacto de XSS.

### 7.4 Validação de entrada

- Contrato OpenAPI como única fonte: `oapi-codegen` valida corpo, query e path contra schemas estritos (sem campos extras).
- Uploads: tamanho máximo por arquivo e por capítulo, whitelist de magic bytes (ZIP + imagens), contagem máxima de páginas, nomes de arquivo sanitizados.
- Textos: normalização UTF-8, limites de tamanho, sanitização na renderização (sem HTML cru em título/sinopse; Markdown com whitelist de tags).
- IDs sempre validados por escopo: 404 em vez de 403 para recursos de outros grupos, evitando enumeração.

### 7.5 Isolamento do pipeline de mídia

- Worker roda em container próprio, sem credencial de admin, com timeout por job e limite de memória/CPU.
- Extração de ZIP protegida contra zip bomb (limite de descompressão, ex.: 200×, e limite de arquivos).
- Remoção de metadados (EXIF/GPS) em todas as páginas no processamento.
- Re-encode de imagem no modo static elimina payloads maliciosos escondidos.
- Storage acessível apenas por chaves geradas no worker; sem listagem pública de bucket.

### 7.6 Sessões, CSRF e rate limiting

- `scs` com rotação de session ID no login e no upgrade de privilégio.
- CSRF: double-submit token em todas as mutations do browser; endpoints `/admin/*` exigem header customizado.
- Rate limit por IP **e** por conta (login, cadastro, busca, API pública), com backoff progressivo; limites reforçados em `cli` e webhooks.
- Verificação de e-mail obrigatória antes de ações sensíveis; recuperação de senha com token de uso único e curta duração.
- MFA (TOTP/passkeys) disponível no builtin e obrigatório para contas com `role.manage` ou `instance.configure`.

### 7.7 Criptografia em trânsito e em repouso

- TLS 1.2+ (preferencial 1.3) obrigatório via Caddy/Let's Encrypt; HSTS com preload.
- Senhas: `argon2id` com parâmetros revisados por release (memória/tempo publicados em ADR).
- Tokens de sessão e URLs assinadas: aleatórios criptográficos (`crypto/rand`), TTL curto (URLs VIP/early: 5–15 min).
- Em repouso: Postgres e storage em disco com criptografia a critério do operador (LUKS/dm-crypt ou SSE no provedor S3); o projeto documenta o caminho recomendado.
- Segredos nunca em código nem em log: `.env` fora do repositório, `gitleaks` no CI, rotação documentada.

### 7.8 Auditoria e resposta a incidentes

- `audit_log` append-only: autor, ação, alvo, valores antes/depois, IP e user-agent. Retenção mínima de 12 meses, exportável.
- Runbooks versionados em `docs/`: recuperação de conta, vazamento de chave imgproxy, incidente de conteúdo, restore de backup.
- `SECURITY.md` com política de divulgação responsável e SLA de resposta.
- Alertas (Prometheus + Uptime Kuma): pico de 4xx/5xx, fila de jobs travada, falha de storage, volume anômalo de downloads de um único usuário.

### 7.9 Posicionamento legal

- **O projeto distribui software; não opera serviço.** Nada no código hospeda conteúdo dos mantenedores.
- `LICENSE` (AGPL-3.0) + aviso de ausência de garantia + `README` com aviso explícito de responsabilidade do operador.
- Modelos de termos de uso, política de privacidade e política de conteúdo (seção 3 da Fase 3) são fornecidos como templates editáveis — cada operador responde pelo próprio uso.
- Ferramentas de moderação, takedown e auditoria fazem parte do produto exatamente para dar ao operador meios de cumprir suas obrigações.

---

## 8. Storage e compatibilidade S3

O armazenamento é abstraído pela interface `Storage`, e qualquer backend S3-compatível deve funcionar sem código novo: AWS S3, Cloudflare R2, MinIO, Garage, SeaweedFS e Backblaze B2 são os alvos de teste na matriz de CI "completa".

### 8.1 Interface

```go
type Storage interface {
    Put(ctx context.Context, key string, r io.Reader, size int64) error
    Get(ctx context.Context, key string) (io.ReadSeekCloser, error)
    Delete(ctx context.Context, key string) error
    // Presign gera URL de upload (PUT) ou leitura (GET) com expiração curta.
    Presign(ctx context.Context, key string, op Op, ttl time.Duration) (string, error)
    List(ctx context.Context, prefix string) ([]ObjectInfo, error)
}
```

### 8.2 Drivers e compatibilidade

| Driver | Alvos | Observações |
| --- | --- | --- |
| `local` | Disco local/NFS | Path canônico sob `DATA_DIR`; chmod restritivo |
| `s3` | AWS S3, R2, MinIO, Garage, SeaweedFS, B2 | Via `aws-sdk-go-v2` com `endpoint` customizável |

Compatibilidade garantida por:

- `S3_FORCE_PATH_STYLE=true` por padrão (necessário para MinIO, Garage, SeaweedFS; opcional para AWS).
- `S3_REGION` configurável (R2 usa `auto`; B2 usa região própria via endpoint S3).
- Sem uso de recursos exclusivos da AWS (sem RRS, sem Object Lock, sem S3 Select).
- Testes de contrato da seção 10 rodam a mesma suíte contra MinIO em Docker e local — qualquer divergência de comportamento quebra o CI.
- Verificação de capabilities no boot (multipart suportado? presign suportado?) com mensagem clara.

### 8.3 Modelo de chaves

- **O banco nunca guarda URL, só `storage_key`**, composta por hash de conteúdo: `pages/{series_id}/{chapter_id}/{sha256}.webp`.
- Hash de conteúdo ⇒ deduplicação automática (mesma página não é gravada duas vezes) e chave não adivinhável (sem ID sequencial vazável).
- A URL final (domínio, assinatura, variante) é calculada pelo `ImageURLBuilder` em tempo de resposta — trocar de driver ou de CDN não exige migrar dados.

### 8.4 Presign e expiração

| Conteúdo | URL | TTL |
| --- | --- | --- |
| Público | CDN/imgproxy com `Cache-Control: public, max-age=31536000, immutable` | — |
| VIP / early | Presign `GET` com assinatura | 5–15 min |
| Upload | Presign `PUT` por arquivo | 15 min |

- Uploads grandes usam multipart presigned (partes de 5–50 MB), essencial em R2/MinIO.
- Assinatura validada na borda quando possível (Cloudflare Worker ou Caddy `forward_auth`) para conteúdo restrito, impedindo que a CDN sirva imagem a quem não pode ler.

### 8.5 Migração entre drivers

```bash
# Copia tudo do driver ativo para um destino, em background, com retomada.
cli storage migrate --from local --to s3 --parallel 4 --resume
# 1. Lista chaves no driver origem (prefixo por série para particionar).
# 2. Compara existence/ETag no destino; copia só o que falta (stream, sem carregar em RAM).
# 3. Reescreve apenas variáveis de ambiente; storage_key no banco não muda.
```

Migração é indolor por design: como o banco só guarda chave, trocar `STORAGE_DRIVER` após a cópia concluída é suficiente.

### 8.6 Configuração

```env
STORAGE_DRIVER=s3
S3_ENDPOINT=https://<accountid>.r2.cloudflarestorage.com   # ou http://minio:9000
S3_REGION=auto          # auto (R2) | us-east-1 (AWS) | us-east-1 (MinIO padrão)
S3_BUCKET=mangashelf
S3_ACCESS_KEY=...       # sem valor ⇒ usa IAM role / instance profile quando disponível
S3_SECRET_KEY=...
S3_FORCE_PATH_STYLE=true
S3_PUBLIC_BASE_URL=https://cdn.exemplo.com   # opcional: base de CDN pública
```

### 8.7 Boas práticas por instância

- **Bucket privado por padrão:** todo acesso público passa por CDN/imgproxy com URL assinada; nunca bucket público de leitura.
- **Lifecycle:** expirar uploads multipart incompletos (7 dias) e `processing` órfão (14 dias).
- **Versionamento opcional:** recomendado em produção para recuperação rápida de deleção acidental.
- **Particionamento por prefixo** (`series_id`/`chapter_id`) mantém listagem eficiente em B2/Garage, que penalizam list amplas.
- **Métricas de uso** (bytes por série/capítulo) expostas no painel de Mídia para o operador dimensionar custo de egress.

---

## 9. Roadmap por fases

### Fase 0: Fundação (1 a 2 semanas)

**Entregas**

- Licença AGPL-3.0 (protege contra forks fechados oferecidos como serviço), com `LICENSE`, aviso de ausência de garantia e política de contribuição (DCO).
- Monorepo: `apps/api`, `apps/worker`, `apps/web`, `deploy/`, `docs/`.
- `docker-compose` de desenvolvimento com profiles.
- CI: lint, testes, build e verificações de segurança bloqueantes (seção 7), com matriz "mínimo" e "completo".
- `README`, `CONTRIBUTING`, `CODE_OF_CONDUCT`, `SECURITY.md` e ADRs curtos com as decisões deste documento.
- Definição das interfaces (seção 3.2) e loader de config com validação.
- Modelagem inicial do banco e esqueleto do OpenAPI.
- Geração de client TS a partir do OpenAPI integrada ao build.

**Pronto quando:** `docker compose up` sobe API, worker, web e Postgres vazios, e a CI passa.

### Fase 1: MVP (6 a 8 semanas)

**Auth e autorização**

- Cadastro, login, logout, verificação de e-mail e recuperação de senha (`builtin`).
- Modelo completo de permissões, roles, escopo por grupo e overrides.
- `cli bootstrap-owner`, audit log básico.

**Catálogo**

- CRUD de séries, títulos alternativos, autores, tags e grupos.
- CRUD de capítulos com estados (`draft`, `processing`, `published`, `removed`).

**Mídia**

- Upload por URL pré-assinada, validação, extração de ZIP e job no worker.
- Pipeline de imagem com os dois modos (`imgproxy` e `static`) e `ImageURLBuilder`.
- Storage `local` e `s3`.

**Leitor e site**

- Leitor com modo paginado e scroll vertical, atalhos de teclado, layout mobile e pré-carregamento das próximas páginas.
- Home, catálogo, página da série, página do capítulo, SEO (metadados, Open Graph, sitemap).
- Busca com Postgres FTS (`unaccent` + `pg_trgm`).

**Operação**

- Deploy em VPS com Caddy e HTTPS, backups automatizados e restauração testada.
- Logs estruturados e healthchecks.

**VIP e acesso**

- Campos de acesso e agendamento em séries e capítulos, resolver único `CanRead` e proteção das imagens (seção 5).
- VIP concedido manualmente pelo admin e por vouchers, com expiração.
- Job de liberação em `public_at` para revalidar cache, notificar e atualizar índices.

**Painel admin v1**

- Shell em `/admin` com layout, navegação por permissão e padrão de tabelas (seção 6).
- Módulos: visão geral, catálogo, fila de mídia, usuários e auditoria básica.

**Segurança**

- Gates de segurança da CI ativos e headers de segurança no proxy (seção 7).

**Pronto quando:** um uploader autorizado publica um capítulo, um visitante o lê em qualquer dispositivo, um capítulo com acesso antecipado abre para o público só na data marcada, e o ambiente restaura de backup.

### Fase 2: Comunidade e UX (4 a 6 semanas)

- Biblioteca: follows, listas, histórico e progresso de leitura sincronizado.
- Filtros por tags, status, ano e demografia.
- Driver **Meilisearch**, `cli search reindex` e outbox de sincronização.
- Driver **OIDC**, OAuth social, passkeys e TOTP.
- Comentários e notificações de novos capítulos (in-app e e-mail).
- i18n, tema claro/escuro e PWA.
- Driver de cache Redis (opcional).
- VIP com planos e níveis (antecedência diferente por plano), aviso por e-mail antes de expirar.
- Interface `PaymentProvider` com `PAYMENTS_DRIVER=none` por padrão e um primeiro adaptador opcional.
- Painel admin v2: VIP, moderação, configurações da instância e saúde do sistema.

**Pronto quando:** trocar `SEARCH_DRIVER` e `AUTH_DRIVER` funciona sem mudar código, com testes cobrindo ambos os caminhos.

### Fase 3: Moderação e robustez (4 a 6 semanas)

- UI de gestão de cargos e permissões com regras anti-escalação.
- Audit log navegável e filtrável.
- Sistema de denúncias, fila de moderação e banimentos (com expiração e motivo).
- Ferramentas de remoção de conteúdo para o operador: formulário público opcional, remoção rápida e registro. São recursos da instância, não obrigações do projeto.
- Rate limiting, proteção contra scraping e abuso, captcha opcional.
- Modelos de termos de uso, privacidade e política de conteúdo, editáveis por instância (o operador responde por eles).
- Testes de carga, métricas (Prometheus), dashboards e alertas.
- Revisão completa contra o OWASP ASVS nível 2 e fechamento dos itens abertos (seção 7).

**Pronto quando:** uma denúncia ou takedown percorre o fluxo completo e fica auditado.

### Fase 4: Escala e ecossistema (contínuo)

- CDN/R2 em produção e cache em camadas.
- API pública documentada com tokens e escopos.
- Webhooks, RSS/Atom e sitemap dinâmico.
- Painel admin: relatórios avançados, ações em lote e dashboards configuráveis.
- Importação e exportação de dados da instância.
- Otimização de queries e partições para tabelas grandes (páginas, comentários, progresso).

### Fase 5: Comunidade open source (contínuo)

- Documentação de self-host (VPS mínima, upgrade, backup e restore, troubleshooting).
- Issues `good first issue`, templates de PR e roadmap público.
- Releases versionadas (SemVer), changelog e imagens Docker publicadas.
- Sistema de temas e possivelmente plugins.
- Avaliação futura de federação (ActivityPub).

---

## 10. Estratégia de testes

- **Unitários** para regras de negócio e para o `Authorizer` (matriz de permissões, escopo, deny e anti-escalação).
- **Integração** com Postgres real via testcontainers.
- **Contrato:** testes que rodam a mesma suíte contra cada implementação de `SearchEngine`, `Storage` e `ImageURLBuilder` (inclui MinIO vs. local na prática).
- **E2E** com Playwright nos fluxos críticos: cadastro, upload, leitura, moderação.
- **CI com duas matrizes:** "mínimo" (postgres, static, local, builtin) e "completo" (meilisearch, imgproxy, s3, redis). Evite testar todas as combinações; cada flag multiplica o custo.

---

## 11. Riscos e mitigação

| Risco | Mitigação |
| --- | --- |
| Uso indevido da plataforma por operadores de instâncias | O projeto distribui só o software, com aviso de responsabilidade no README, no LICENSE e nos termos modelo; ferramentas de moderação à disposição do operador |
| Banda de imagens estoura a VPS | CDN na frente, cache `immutable`, URLs com hash de conteúdo |
| Upload malicioso | Validação de magic bytes, limites, worker isolado, domínio de mídia separado, remoção de metadados |
| CPU alta no imgproxy | URLs assinadas, cache, conjunto fixo de tamanhos permitidos |
| Explosão de flags e combinações | Duas matrizes de CI, defaults fortes, validação no boot |
| Escalação de privilégio | Anti-escalação por `position`, deny vence allow, auditoria |
| Perda de dados | Backups automáticos, restore testado periodicamente |
| Scraping agressivo | Rate limit, Cloudflare, limites por IP e por sessão |

---

## 12. Estrutura sugerida do monorepo

```
/
├── apps/
│   ├── api/        # cmd/api, internal/{auth,authz,catalog,media,library,moderation,search}
│   ├── worker/     # cmd/worker (mesmo módulo Go, outro entrypoint)
│   └── web/        # Next.js
├── packages/
│   └── api-client/ # client TS gerado do OpenAPI
├── deploy/
│   ├── docker-compose.yml
│   ├── Caddyfile
│   └── .env.example
├── api/openapi.yaml
├── migrations/
└── docs/           # ADRs, self-host, contribuição
```

---

## 13. Próximos passos imediatos

1. Registrar em ADRs a licença (AGPL-3.0) e o nome (MangaShelf).
2. Criar o repositório e a estrutura da seção 12.
3. Escrever o schema SQL inicial, incluindo o sistema de permissões.
4. Definir as interfaces da seção 3.2 e o loader de config.
5. Subir o `docker-compose` de desenvolvimento e a CI.

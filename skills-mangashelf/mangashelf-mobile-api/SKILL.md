---
name: mangashelf-mobile-api
description: "Padrões de API para app mobile MangaShelf (futuro React Native / Expo). Garante que o backend seja mobile-ready desde o MVP: versionamento, auth com refresh token, payloads compactos, ETag, push notifications, rate limit por device."
---

# MangaShelf Mobile API

Skill que define os padrões de API que o backend MangaShelf deve seguir desde o MVP para que o app mobile (futuro React Native + Expo) consuma a API sem retrabalho.

## Quando ativar

Toda sessão de IA que vai:
- Adicionar/modificar endpoint em `apps/api/`
- Modificar `api/openapi.yaml`
- Mudar autenticação (seção 4 do roadmap)
- Mudar storage de mídia (seção 5.3 e 8)

## Princípios

1. **API é mobile-first.** Mobile é o cliente principal de consumo; web é o painel admin + leitor web. Toda decisão de API considera mobile antes.
2. **Versionamento explícito.** `/api/v1/...` obrigatório; bump só com breaking change.
3. **Payloads mínimos.** Mobile paga por byte; nunca mandar campo que o client não pediu (sparse fieldsets).
4. **Cache é rei.** ETag, `Last-Modified`, `Cache-Control` em todo GET. Mobile em 3G agradece.
5. **Offline-first ready.** Toda lista paginada suporta cursor + `If-Modified-Since` para sync incremental.

## Endpoints mobile (prefixo `/api/v1/mobile/`)

Endpoints otimizados para mobile são prefixados com `/mobile/` e podem ter:
- Campos extras (`device_id`, `app_version`, `platform`).
- Payloads mais compactos (sem campos admin).
- Rate limit por device (não só por user).

```yaml
# api/openapi.yaml
paths:
  /api/v1/mobile/chapters/{id}:
    get:
      tags: [mobile]
      parameters:
        - name: X-Device-Id
          in: header
          required: true
          schema: { type: string }
        - name: X-App-Version
          in: header
          required: true
          schema: { type: string, pattern: '^\d+\.\d+\.\d+$' }
        - name: If-None-Match
          in: header
          schema: { type: string }
      responses:
        '200':
          description: Chapter
          headers:
            ETag: { schema: { type: string } }
            Cache-Control: { schema: { type: string } }
        '304':
          description: Not Modified
```

## Autenticação mobile

### Modelo: access token curto + refresh token longo

```go
// Login (POST /api/v1/auth/login)
// Devolve:
type TokenPair struct {
    AccessToken  string `json:"access_token"`   // JWT, 15 min
    RefreshToken string `json:"refresh_token"`  // opaque, 30 dias, rotativo
    TokenType    string `json:"token_type"`     // "Bearer"
    ExpiresIn    int    `json:"expires_in"`     // 900 (15 min em segundos)
}
```

```go
// Refresh (POST /api/v1/auth/refresh)
// Body: { "refresh_token": "..." }
// Devolve novo par; refresh token antigo é invalidado (rotação).
// Se o refresh token antigo for usado novamente = possível roubo = revoga TODOS os tokens do user.
```

**Regras:**
- Access token: **JWT** com `sub` (user_id), `scope` (permissões), `exp` (15 min). Assinado com `Ed25519` (chave privada no server, pública verificável).
- Refresh token: **opaque random** (32 bytes base64url), guardado no banco com `user_id`, `device_id`, `expires_at`, `rotated_at`. **Nunca** JWT.
- Refresh token é **rotativo**: cada uso gera um novo e invalida o anterior. Se um antigo for reusado, todos os tokens do usuário são revogados (suspeita de roubo).
- Refresh token pode ser revogado pelo admin (logout remoto) e pelo usuário (logout do device).
- `device_id` no refresh token permite listagem de sessões ativas no painel admin.

### Headers obrigatórios em toda request mobile

```http
Authorization: Bearer <access_token>
X-Device-Id: <uuid-do-device>           // gerado no app, persiste entre sessões
X-App-Version: 1.2.3                    // semver do app
X-Platform: ios|android                 // para variantes de imagem
If-None-Match: <etag>                   // para cache condicional (opcional)
```

### Logout

```http
POST /api/v1/auth/logout
Authorization: Bearer <access>
Body: { "device_id": "..." }
```
Revoga o refresh token daquele device. Access token continua válido até expirar (15 min) — para revogação imediata, manter denylist em Redis (opcional).

## Push notifications

### Modelo: FCM (Android) + APNs (iOS), admin por web

```go
// Tabela
type DevicePushToken struct {
    UserID    string    `json:"user_id"`
    DeviceID  string    `json:"device_id"`
    Platform  string    `json:"platform"`   // "ios" | "android"
    Token     string    `json:"token"`      // FCM token ou APNs device token
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    LastSeen  time.Time `json:"last_seen"`  // para expirar tokens inativos
}
```

- **Registro:** `POST /api/v1/mobile/devices/push` com `{ platform, token }`. Idempotente por `(user_id, device_id)`.
- **Unregister:** `DELETE /api/v1/mobile/devices/push` no logout.
- **Eventos que disparam push:**
  - Novo capítulo de série seguida.
  - Capítulo `early` virou `public` (para não-VIP).
  - VIP expira em 3 dias.
  - Resposta a denúncia/comentário do usuário.
- **Não dispara push:** atualizações administrativas, mudança de senha (nao é notificável), logs.
- **Payload** máximo 4KB; usar `data` (não `notification`) para controle total no app. `notification` só se o app estiver em background.
- **Deep link:** `mangashelf://chapter/{id}` ou `mangashelf://series/{id}`.

### Driver de push (interface)

```go
type PushProvider interface {
    Send(ctx context.Context, token string, platform string, payload PushPayload) error
    SendBatch(ctx context.Context, tokens []string, platform string, payload PushPayload) error
}

// Implementações:
// - pushFCM: via Firebase Admin SDK
// - pushAPNs: via github.com/sideshow/apns2
// - pushNone: no-op (padrão em dev)
```

Config: `PUSH_DRIVER=none|fcm|apns|both`. Default `none` em dev; `both` em produção.

## Image variants para mobile

O `ImageURLBuilder` (seção 3.2 do roadmap) já suporta variantes. Mobile pede variantes específicas:

```http
GET /api/v1/mobile/chapters/{id}/pages
X-Platform: ios
Accept: image/avif, image/webp, image/jpeg
```

Resposta inclui URLs já otimizadas por plataforma:

```json
{
  "pages": [
    {
      "id": "p_001",
      "number": 1,
      "url": "https://img.example.com/.../800w.avif",
      "width": 800,
      "height": 1200,
      "bytes": 142000
    }
  ]
}
```

| Variante | Largura | Uso |
| --- | --- | --- |
| `thumb` | 240w | Catálogo mobile |
| `medium` | 480w | Card de série mobile |
| `mobile-full` | 800w | Leitor mobile |
| `desktop-full` | 1200w | Leitor web/desktop |
| `original` | original | Download (VIP) |

iOS 14+ suporta AVIF; Android 12+ também. Para versões antigas, fallback WebP. imgproxy negocia via `Accept` header.

## Rate limiting mobile

| Endpoint | Limite | Escopo |
| --- | --- | --- |
| `POST /auth/login` | 5/min | por IP |
| `POST /auth/refresh` | 30/min | por user + device |
| `GET /chapters/{id}/pages` | 60/min | por user |
| `GET /mobile/feed` | 30/min | por user |
| `POST /devices/push` | 5/min | por device |

Headers sempre presentes:

```http
X-RateLimit-Limit: 60
X-RateLimit-Remaining: 58
X-RateLimit-Reset: 1698765432
```

## Cache condicional (ETag)

Toda listagem mobile suporta `If-None-Match`:

```http
GET /api/v1/mobile/feed
If-None-Match: "abc123"
```

Se nada mudou desde `abc123`:

```http
HTTP/1.1 304 Not Modified
ETag: "abc123"
Cache-Control: private, max-age=60
```

Para feed (chapítulos novos), o ETag muda quando há novo capítulo de série seguida. Implementado via `pgUpdatedAt` (timestamp da última mudança na lista de follows do user + ultima publicação).

## Sync incremental (offline-first)

Endpoint para o app sincronizar após volta online:

```http
GET /api/v1/mobile/sync?since=2024-01-01T00:00:00Z
```

Resposta:

```json
{
  "server_time": "2024-01-15T12:00:00Z",
  "progress": [
    { "chapter_id": "...", "page": 5, "updated_at": "2024-01-14T10:00:00Z", "deleted": false }
  ],
  "follows_added": ["series-1", "series-2"],
  "follows_removed": ["series-3"]
}
```

O app mantém SQLite local (WatermelonDB ou expo-sqlite) com mesma estrutura; sync é baseado em `updated_at` desde último sync. Conflitos (progresso diferente em device e server) resolvem por "último escrita vence" para progresso; follows são union.

## Erros mobile-friendly

```json
HTTP/1.1 404 Not Found
Content-Type: application/problem+json

{
  "type": "https://docs.mangashelf.example/errors/chapter.not_found",
  "title": "Capítulo não encontrado",
  "status": 404,
  "code": "chapter.not_found",
  "detail": "O capítulo foi removido ou nunca existiu.",
  "instance": "/api/v1/mobile/chapters/abc"
}
```

- `code` é estável e usado pelo app para branching (não usar `status` para isso — pode haver 404 para reasons diferentes).
- `detail` é humano e pode ser mostrado.
- Sem stack trace em produção.

## Versionamento e breaking changes

- URL: `/api/v1/...`. Bump para `/v2/` só com breaking change.
- App antigo continua funcionando com `/v1/` por 6 meses após `/v2/` lançado.
- Header `X-App-Version` permite ao server decidir comportamento por versão (deprecatar gradual).
- Antes de remover campo da resposta, marcar `deprecated: true` no OpenAPI por 1 release.

## Checklist de PR que toca API

- [ ] Endpoint versionado (`/api/v1/...`).
- [ ] Headers mobile aceitos (`X-Device-Id`, `X-App-Version`, `X-Platform`).
- [ ] Rate limit configurado.
- [ ] ETag em GETs de listagem.
- [ ] Erros seguem RFC 9457 com `code` estável.
- [ ] OpenAPI atualizado e `oasdiff` não acusa breaking change sem bump.
- [ ] Testes de contrato cobrem o novo endpoint.
- [ ] Se é mutação: idempotência via `Idempotency-Key` suportada.
- [ ] Se retorna imagem: variante mobile (`mobile-full`) disponível.

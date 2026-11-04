---
name: mangashelf-perf-budget
description: "Roda Lighthouse nas 5 rotas críticas do MangaShelf (home, catálogo, série, capítulo, admin) e bloqueia regressão > 10% em LCP, CLS ou TBT. Roda em CI por PR que toca apps/web/."
---

# MangaShelf Performance Budget

Skill de **gate de CI** que mantém performance das 5 rotas críticas. Bloqueia regressão > 10% em métricas Core Web Vitals.

## Quando ativar

Em todo PR que modifica arquivos em `apps/web/`. Roda em CI job paralelo; não bloqueia PRs de documentação ou backend-only.

## Rotas críticas

| Rota | URL (staging) | Métricas prioritárias |
| --- | --- | --- |
| Home | `/` | LCP, CLS, JS bundle |
| Catálogo | `/catalog` | LCP, TBT |
| Série | `/series/sample-series` | LCP, CLS |
| Capítulo (leitor) | `/chapters/sample-chapter` | LCP, TBT, JS bundle |
| Admin | `/admin/catalog` | TBT, JS bundle |

As rotas usam fixtures estáveis em staging (séries/capítulos de teste); não devem mudar entre runs.

## Budgets

| Métrica | Alvo | Limite (bloqueia) |
| --- | --- | --- |
| LCP (Largest Contentful Paint) | < 2.5s em 3G rápido | < 3.0s |
| CLS (Cumulative Layout Shift) | < 0.05 | < 0.1 |
| TBT (Total Blocking Time) | < 200ms | < 400ms |
| FCP (First Contentful Paint) | < 1.5s | < 2.0s |
| Speed Index | < 3.0s | < 4.0s |
| JS bundle por rota (gzip) | < 200 KB | < 250 KB |
| Image weight (total) | < 500 KB | < 800 KB |

### Desktop vs Mobile

Rodar em ambas as configurações:

- **Mobile**: Moto G Power (Lighthouse default), 3G rápido, 4x CPU slowdown.
- **Desktop**: Desktop preset, sem throttle.

Budget bloqueia em mobile; desktop é informativo.

## Como rodar

### Local

```bash
make perf-budget
# Sobe staging, roda Lighthouse 5x por rota, compara com main
```

### CI

```yaml
# .github/workflows/perf-budget.yml
jobs:
  perf-budget:
    runs-on: ubuntu-latest
    if: contains(join(github.event.pull_request.files.*.path, ','), 'apps/web/')
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-node@v4
      - run: npm ci
      - name: Build web
        run: npm run build --workspace=apps/web
      - name: Start staging
        run: |
          docker compose -f deploy/docker-compose.staging.yml up -d
          npx wait-on http://localhost:3000
      - name: Run Lighthouse
        run: npx @lhci/cli autorun --config=lighthouserc.json
      - name: Compare with main
        run: node scripts/perf_compare.js
```

### Configuração LHCI

`lighthouserc.json`:

```json
{
  "ci": {
    "collect": {
      "url": [
        "http://localhost:3000/",
        "http://localhost:3000/catalog",
        "http://localhost:3000/series/sample-series",
        "http://localhost:3000/chapters/sample-chapter",
        "http://localhost:3000/admin/catalog"
      ],
      "numberOfRuns": 5,
      "settings": {
        "preset": "mobile",
        "throttling": {
          "rttMs": 150,
          "throughputKbps": 1638.4,
          "cpuSlowdownMultiplier": 4
        }
      }
    },
    "assert": {
      "assertions": {
        "categories:performance": ["warn", { "minScore": 0.85 }],
        "first-contentful-paint": ["error", { "maxNumericValue": 2000 }],
        "largest-contentful-paint": ["error", { "maxNumericValue": 3000 }],
        "cumulative-layout-shift": ["error", { "maxNumericValue": 0.1 }],
        "total-blocking-time": ["error", { "maxNumericValue": 400 }],
        "resource-summary:script:size": ["error", { "maxNumericValue": 256000 }]
      }
    },
    "upload": {
      "target": "filesystem",
      "outputDir": ".lighthouse"
    }
  }
}
```

### Comparação com main

`scripts/perf_compare.js`:

```javascript
#!/usr/bin/env node
const fs = require('fs');

// Carrega resultados do PR
const prResults = JSON.parse(fs.readFileSync('.lighthouse/last-run/lhr.json'));

// Carrega resultados da main (cacheado em CI)
const mainResults = JSON.parse(fs.readFileSync('.lighthouse/main/lhr.json'));

const metrics = [
  'largest-contentful-paint',
  'cumulative-layout-shift',
  'total-blocking-time',
  'first-contentful-paint'
];

const regressions = [];
for (const m of metrics) {
  const prValue = prResults.audits[m].numericValue;
  const mainValue = mainResults.audits[m].numericValue;
  const regression = (prValue - mainValue) / mainValue;
  if (regression > 0.10) {
    regressions.push({
      metric: m,
      main: mainValue,
      pr: prValue,
      regression: (regression * 100).toFixed(1) + '%'
    });
  }
}

if (regressions.length > 0) {
  console.error('❌ Performance regression detectada:');
  for (const r of regressions) {
    console.error(`  ${r.metric}: main=${r.main.toFixed(0)}ms, pr=${r.pr.toFixed(0)}ms (+${r.regression})`);
  }
  process.exit(1);
}
console.log('✅ Sem regressão de performance');
```

## Bundle analysis

Além do Lighthouse, gerar relatório de bundle por rota:

```bash
ANALYZE=true npm run build --workspace=apps/web
# Gera apps/web/.next/analyze/{client,server}.html
```

Bloquear se:
- Chunk não-previsto aparece (ex.: lodash inteiro em vez de `lodash/get`).
- Bundle de uma rota cresce > 20 KB sem justificativa no PR.
- Dependência nova adiciona > 50 KB ao bundle total.

## Imagens

Auditar weight de imagens:

```bash
# Lista imagens > 100KB carregadas na rota
npx lighthouse http://localhost:3000/chapters/sample-chapter \
  --output=json --output-path=.lighthouse/images.json
jq '.audits["network-requests"].details.items[] | select(.resourceType == "Image") | {url, transferSize}' .lighthouse/images.json
```

Bloquear se:
- Imagem > 200 KB sem variante mobile.
- Imagem sem `next/image` (regra da skill `mangashelf-frontend-optimized`).
- Mais de 5 imagens sem `loading="lazy"`.

## Saída de erro

```
❌ Performance regression detectada:

  Rota: /chapters/sample-chapter (mobile)
  
  largest-contentful-paint: main=1850ms, pr=2310ms (+24.9%)
    → provável causa: imagem do leitor sem `priority` na primeira página
    → ver commit: abc1234 (apps/web/app/chapters/[id]/page.tsx)
  
  total-blocking-time: main=180ms, pr=320ms (+77.8%)
    → provável causa: biblioteca adicionada
    → ver chunk novo: node_modules_pdfjs-dist.js (180 KB)
  
  resource-summary:script:size: main=180000, pr=290000 (+61.1%)
    → budget excedido (250 KB limite)

Resolva as regressões ou justifique no PR com ADR.
```

## Quando regression é aceitável

- Feature nova que adiciona > 50 KB de JS (ex.: leitor PDF). Exigir ADR justificando.
- Refactor que melhora métrica A mas piora B em < 10%. OK se A é prioridade.
- Mudança de design system que afeta múltiplas rotas. Exigir ADR e bump de versão.

## Integração com Vercel Analytics (opcional)

Em produção, Vercel Analytics + Speed Insights coleta Core Web Vitals reais. Alerta se p75 de LCP > 4s em field data (diferente de lab data). Field data é verdade; lab data é gate.

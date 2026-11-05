# scripts/dev-backend.ps1
# Inicia a API Go do MangaShelf em modo desenvolvimento.
# Requer: DATABASE_URL, SESSION_SECRET definidas (ou use .env)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$rootDir = Split-Path -Parent $scriptDir

# Carrega .env se existir
$envFile = Join-Path $rootDir ".env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            $env:$Matches[1] = $Matches[2]
        }
    }
}

# Valida variáveis obrigatórias
$missing = @()
foreach ($v in @("DATABASE_URL", "SESSION_SECRET", "SEARCH_DRIVER", "IMAGE_PIPELINE", "STORAGE_DRIVER", "CACHE_DRIVER", "AUTH_DRIVER")) {
    if (-not $env:$v) { $missing += $v }
}
if ($missing.Count -gt 0) {
    Write-Warning "Variáveis de ambiente não definidas: $($missing -join ', ')"
    Write-Host "Defina-as ou crie um arquivo .env na raiz do projeto."
    Write-Host "Exemplo: cp deploy/.env.example .env"
    exit 1
}

Write-Host "Iniciando MangaShelf API (backend) em http://localhost:$env:PORT ..." -ForegroundColor Cyan
Write-Host "DATABASE_URL: $env:DATABASE_URL" -ForegroundColor Gray
Write-Host "STORAGE_DRIVER: $env:STORAGE_DRIVER | IMAGE_PIPELINE: $env:IMAGE_PIPELINE | CACHE_DRIVER: $env:CACHE_DRIVER | AUTH_DRIVER: $env:AUTH_DRIVER" -ForegroundColor Gray

Set-Location $rootDir
go run ./cmd/api

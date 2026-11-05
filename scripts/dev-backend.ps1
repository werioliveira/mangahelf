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
            Set-Item -Path "env:$($Matches[1])" -Value $Matches[2]
        }
    }
}

# Valida variáveis obrigatórias
$requiredVars = @("DATABASE_URL", "SESSION_SECRET", "SEARCH_DRIVER", "IMAGE_PIPELINE", "STORAGE_DRIVER", "CACHE_DRIVER", "AUTH_DRIVER")
$missing = @()
foreach ($v in $requiredVars) {
    $item = Get-Item -Path "env:$v" -ErrorAction SilentlyContinue
    if (-not $item) { $missing += $v }
}
if ($missing.Count -gt 0) {
    Write-Warning "Variáveis de ambiente não definidas: $($missing -join ', ')"
    Write-Host "Defina-as ou crie um arquivo .env na raiz do projeto."
    Write-Host "Exemplo: cp deploy/.env.example .env"
    exit 1
}

$port = (Get-Item -Path "env:PORT" -ErrorAction SilentlyContinue).Value
if (-not $port) { $port = "8080" }

Write-Host "Iniciando MangaShelf API (backend) em http://localhost:$port ..." -ForegroundColor Cyan
Write-Host "DATABASE_URL: $((Get-Item -Path "env:DATABASE_URL").Value)" -ForegroundColor Gray
Write-Host "STORAGE_DRIVER: $((Get-Item -Path "env:STORAGE_DRIVER").Value) | IMAGE_PIPELINE: $((Get-Item -Path "env:IMAGE_PIPELINE").Value) | CACHE_DRIVER: $((Get-Item -Path "env:CACHE_DRIVER").Value) | AUTH_DRIVER: $((Get-Item -Path "env:AUTH_DRIVER").Value)" -ForegroundColor Gray

Set-Location $rootDir
go run ./cmd/api

# scripts/dev-frontend.ps1
# Inicia o frontend Next.js do MangaShelf em modo desenvolvimento.

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$rootDir = Split-Path -Parent $scriptDir
$webDir = Join-Path $rootDir "apps/web"

# Carrega .env se existir (variáveis do frontend)
$envFile = Join-Path $rootDir ".env"
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match "^([^=]+)=(.*)$") {
            $env:$Matches[1] = $Matches[2]
        }
    }
}

# Define NEXT_PUBLIC_API_URL se não estiver definida
if (-not $env:NEXT_PUBLIC_API_URL) {
    $env:NEXT_PUBLIC_API_URL = "http://localhost:8080"
    Write-Host "NEXT_PUBLIC_API_URL não definida, usando padrão: $env:NEXT_PUBLIC_API_URL" -ForegroundColor Yellow
}

Write-Host "Iniciando MangaShelf Frontend (Next.js) em http://localhost:3000 ..." -ForegroundColor Cyan
Write-Host "NEXT_PUBLIC_API_URL: $env:NEXT_PUBLIC_API_URL" -ForegroundColor Gray

Set-Location $webDir
npm run dev

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
            Set-Item -Path "env:$($Matches[1])" -Value $Matches[2]
        }
    }
}

# Define NEXT_PUBLIC_API_URL se não estiver definida
$apiUrlItem = Get-Item -Path "env:NEXT_PUBLIC_API_URL" -ErrorAction SilentlyContinue
if (-not $apiUrlItem) {
    Set-Item -Path "env:NEXT_PUBLIC_API_URL" -Value "http://localhost:8080"
    Write-Host "NEXT_PUBLIC_API_URL não definida, usando padrão: http://localhost:8080" -ForegroundColor Yellow
}
$apiUrl = (Get-Item -Path "env:NEXT_PUBLIC_API_URL").Value

Write-Host "Iniciando MangaShelf Frontend (Next.js) em http://localhost:3000 ..." -ForegroundColor Cyan
Write-Host "NEXT_PUBLIC_API_URL: $apiUrl" -ForegroundColor Gray

Set-Location $webDir
npm run dev

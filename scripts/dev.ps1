# scripts/dev.ps1
# Inicia backend e frontend em paralelo (mesmo que `npm run dev`).
# Requer: concurrently instalado (npm install concurrently)

$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Definition
$rootDir = Split-Path -Parent $scriptDir

Write-Host "Iniciando MangaShelf (backend + frontend) ..." -ForegroundColor Cyan

Set-Location $rootDir

# Verifica se o concurrently está disponível
if (-not (Get-Command concurrently -ErrorAction SilentlyContinue)) {
    Write-Host "Instalando concurrently..." -ForegroundColor Yellow
    npm install -g concurrently
}

Start-Process pwsh -ArgumentList "-File", (Join-Path $scriptDir "dev-backend.ps1") -WindowStyle Normal
Start-Process pwsh -ArgumentList "-File", (Join-Path $scriptDir "dev-frontend.ps1") -WindowStyle Normal

Write-Host "Backend e frontend iniciados em janelas separadas." -ForegroundColor Green
Write-Host "Backend:  http://localhost:8080" -ForegroundColor Gray
Write-Host "Frontend: http://localhost:3000" -ForegroundColor Gray

# Build AnnTodo release binaries.
#   .\build.ps1                      # current platform -> .\anntodo.exe
#   .\build.ps1 -Version 1.0.0 -All  # windows/linux amd64+arm64 -> .\dist\
param(
    [string]$Version = "dev",
    [switch]$All,
    [switch]$SkipWeb
)
$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

if (-not $SkipWeb) {
    Push-Location web
    try {
        if (-not (Test-Path node_modules)) { npm ci; if ($LASTEXITCODE) { throw "npm ci failed" } }
        npm run build; if ($LASTEXITCODE) { throw "frontend build failed" }
    } finally { Pop-Location }
}

$env:CGO_ENABLED = "0"
$ldflags = "-s -w -X anntodo/pkg/version.Version=$Version"

if (-not $All) {
    go build -trimpath -ldflags $ldflags -o anntodo.exe ./cmd/anntodo
    if ($LASTEXITCODE) { throw "go build failed" }
    Write-Host "built anntodo.exe ($Version)"
    return
}

$targets = @("windows/amd64", "windows/arm64", "linux/amd64", "linux/arm64")
foreach ($t in $targets) {
    $os, $arch = $t.Split("/")
    $out = "dist/anntodo-$Version-$os-$arch"
    New-Item -ItemType Directory -Force "$out/conf" | Out-Null
    $bin = if ($os -eq "windows") { "anntodo.exe" } else { "anntodo" }
    $env:GOOS = $os; $env:GOARCH = $arch
    go build -trimpath -ldflags $ldflags -o "$out/$bin" ./cmd/anntodo
    if ($LASTEXITCODE) { throw "go build $t failed" }
    Copy-Item conf/anntodo.ini "$out/conf/"
    Copy-Item LICENSE, README.md $out
    if ($os -eq "linux") { Copy-Item docker/anntodo.service $out }
    Write-Host "built $out"
}
Remove-Item Env:GOOS, Env:GOARCH

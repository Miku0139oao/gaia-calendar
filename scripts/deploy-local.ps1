param(
    [Parameter(Mandatory = $true)]
    [string]$Remote,
    [string]$RemoteDir = "/opt/gaia-calendar"
)

$ErrorActionPreference = "Stop"

$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$DeployDir = Join-Path $Root "dist\deploy"
$FrontendDeployDir = Join-Path $DeployDir "frontend"
$FrontendDistDeployDir = Join-Path $FrontendDeployDir "dist"

Write-Host "Building frontend with Bun..."
Push-Location (Join-Path $Root "frontend")
try {
    bun install --frozen-lockfile
    bun run build
}
finally {
    Pop-Location
}

Write-Host "Preparing deploy bundle..."
Remove-Item -LiteralPath $DeployDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $FrontendDeployDir | Out-Null
Copy-Item -LiteralPath (Join-Path $Root "frontend\dist") -Destination $FrontendDistDeployDir -Recurse -Force

Write-Host "Building Linux backend binary locally..."
$oldGoos = $env:GOOS
$oldGoarch = $env:GOARCH
$oldCgo = $env:CGO_ENABLED
try {
    $env:GOOS = "linux"
    $env:GOARCH = "amd64"
    $env:CGO_ENABLED = "0"
    Push-Location $Root
    try {
        go build -trimpath -ldflags="-s -w" -o (Join-Path $DeployDir "gaia-calendar") ./cmd/server
    }
    finally {
        Pop-Location
    }
}
finally {
    $env:GOOS = $oldGoos
    $env:GOARCH = $oldGoarch
    $env:CGO_ENABLED = $oldCgo
}

Write-Host "Uploading slim runtime context to ${Remote}:${RemoteDir}..."
$ArchivePath = Join-Path $Root "dist\gaia-calendar-runtime.tar"
Remove-Item -LiteralPath $ArchivePath -Force -ErrorAction SilentlyContinue
Push-Location $Root
try {
    tar -cf $ArchivePath Dockerfile.runtime docker-compose.yml docker-compose.local-build.yml dist/deploy
    ssh $Remote "mkdir -p '$RemoteDir'"
    scp $ArchivePath "${Remote}:/tmp/gaia-calendar-runtime.tar"
    ssh $Remote "tar -xf /tmp/gaia-calendar-runtime.tar -C '$RemoteDir' && rm -f /tmp/gaia-calendar-runtime.tar"
}
finally {
    Pop-Location
    Remove-Item -LiteralPath $ArchivePath -Force -ErrorAction SilentlyContinue
}

Write-Host "Recreating app container on remote host..."
ssh $Remote "cd '$RemoteDir' && docker-compose -f docker-compose.yml -f docker-compose.local-build.yml up -d --build app && docker builder prune -f >/dev/null && docker-compose ps && df -h / && docker logs --tail 50 gaia-calendar-app-1"

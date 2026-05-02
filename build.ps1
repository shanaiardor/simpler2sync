$ErrorActionPreference = "Stop"

if (-not (Test-Path "bin")) { New-Item -ItemType Directory "bin" | Out-Null }

$ldflags = "-s -w"
$env:CGO_ENABLED = "1"

Write-Host "Building for Windows (amd64)..."
go build -ldflags $ldflags -o "bin\simpler2sync-windows-amd64.exe" .

Write-Host "Building for Windows (arm64)..."
$env:GOARCH = "arm64"
go build -ldflags $ldflags -o "bin\simpler2sync-windows-arm64.exe" .
$env:GOARCH = "amd64"

Write-Host "Cross-compiling for macOS (amd64)..."
$env:GOOS = "darwin"
go build -ldflags $ldflags -o "bin\simpler2sync-darwin-amd64" .

Write-Host "Cross-compiling for macOS (arm64)..."
$env:GOARCH = "arm64"
go build -ldflags $ldflags -o "bin\simpler2sync-darwin-arm64" .
$env:GOARCH = "amd64"

Write-Host "Cross-compiling for Linux (amd64)..."
$env:GOOS = "linux"
go build -ldflags $ldflags -o "bin\simpler2sync-linux-amd64" .

$env:GOOS = "windows"
$env:GOARCH = "amd64"

Write-Host "Build complete. Binaries in bin/"

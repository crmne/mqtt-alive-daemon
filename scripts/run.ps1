$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$exePath = Join-Path $repoRoot "mqtt-alive-daemon.exe"

if (-not (Test-Path $exePath)) {
	Write-Host "Building local binary"
	& (Join-Path $PSScriptRoot "build.ps1")
}

Write-Host "Running $exePath"
& $exePath

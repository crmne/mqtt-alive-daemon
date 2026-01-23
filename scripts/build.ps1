$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$outputPath = Join-Path $repoRoot "mqtt-alive-daemon.exe"

Write-Host "Building $outputPath"
& go build -o $outputPath

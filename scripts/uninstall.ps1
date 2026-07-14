param(
	[ValidateSet("CurrentUser", "AllUsers")]
	[string]$InstallScope = "CurrentUser",
	[string]$InstallDir,
	[string]$ConfigDir,
	[switch]$KeepConfig,
	[switch]$UnregisterTask = $true,
	[string]$TaskName = "mqtt-alive-daemon"
)

$ErrorActionPreference = "Stop"

$exeName = "mqtt-alive-daemon.exe"

if (-not $InstallDir) {
	if ($InstallScope -eq "AllUsers") {
		$InstallDir = Join-Path $env:ProgramData "mqtt-alive-daemon"
	} else {
		$InstallDir = Join-Path $env:APPDATA "mqtt-alive-daemon"
	}
}

if (-not $ConfigDir) {
	if ($InstallScope -eq "AllUsers") {
		$ConfigDir = Join-Path $env:ProgramData "mqtt-alive-daemon"
	} else {
		$ConfigDir = Join-Path $env:APPDATA "mqtt-alive-daemon"
	}
}

$exePath = Join-Path $InstallDir $exeName
if (Test-Path $exePath) {
	Remove-Item -Force $exePath
	Write-Host "Removed $exePath"
} else {
	Write-Host "Binary not found at $exePath"
}

if (-not $KeepConfig -and (Test-Path $ConfigDir)) {
	Remove-Item -Recurse -Force $ConfigDir
	Write-Host "Removed $ConfigDir"
} elseif ($KeepConfig) {
	Write-Host "Kept config directory $ConfigDir"
}

if ($UnregisterTask) {
	try {
		Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false -ErrorAction Stop | Out-Null
		Write-Host "Removed scheduled task $TaskName"
	} catch {
		Write-Host "Scheduled task $TaskName not found or could not be removed"
	}
}

$startupDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup"
$startupScript = Join-Path $startupDir "mqtt-alive-daemon.cmd"
if (Test-Path $startupScript) {
	Remove-Item -Force $startupScript
	Write-Host "Removed Startup folder entry $startupScript"
}

Write-Host "Uninstall complete"

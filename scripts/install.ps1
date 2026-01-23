param(
	[ValidateSet("CurrentUser", "AllUsers")]
	[string]$InstallScope = "CurrentUser",
	[string]$InstallDir,
	[string]$ConfigDir,
	[switch]$RegisterTask = $true,
	[string]$TaskName = "mqtt-alive-daemon"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
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

Write-Host "Building binary"
& (Join-Path $PSScriptRoot "build.ps1")

$builtExe = Join-Path $repoRoot $exeName
if (-not (Test-Path $builtExe)) {
	throw "Build output not found: $builtExe"
}

Write-Host "Installing to $InstallDir"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Copy-Item -Force -Path $builtExe -Destination (Join-Path $InstallDir $exeName)

$configExample = Join-Path $repoRoot "config.yaml.example"
$configTarget = Join-Path $ConfigDir "config.yaml"
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null
if (-not (Test-Path $configTarget)) {
	Copy-Item -Path $configExample -Destination $configTarget
	Write-Host "Wrote example config to $configTarget"
} else {
	Write-Host "Config already exists at $configTarget"
}

if ($RegisterTask) {
	$taskExe = Join-Path $InstallDir $exeName
	if ($InstallScope -eq "AllUsers") {
		Write-Host "Registering scheduled task '$TaskName' for system startup"
		$action = New-ScheduledTaskAction -Execute $taskExe
		$trigger = New-ScheduledTaskTrigger -AtStartup
		$principal = New-ScheduledTaskPrincipal -UserId "SYSTEM" -LogonType ServiceAccount -RunLevel Highest
		$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal
		try {
			Register-ScheduledTask -TaskName $TaskName -InputObject $task -Force | Out-Null
		} catch {
			throw "Failed to register scheduled task '$TaskName'. Re-run in an elevated PowerShell session. $($_.Exception.Message)"
		}
	} else {
		Write-Host "Registering scheduled task '$TaskName' for user logon"
		$action = New-ScheduledTaskAction -Execute $taskExe
		$trigger = New-ScheduledTaskTrigger -AtLogOn -User $env:USERNAME
		$principal = New-ScheduledTaskPrincipal -UserId $env:USERNAME -LogonType Interactive -RunLevel Limited
		$task = New-ScheduledTask -Action $action -Trigger $trigger -Principal $principal
		try {
			Register-ScheduledTask -TaskName $TaskName -InputObject $task -Force | Out-Null
		} catch {
			if ($_.Exception.Message -match "Access is denied") {
				$startupDir = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\Startup"
				$startupScript = Join-Path $startupDir "mqtt-alive-daemon.cmd"
				New-Item -ItemType Directory -Force -Path $startupDir | Out-Null
				"@echo off`r`nstart `"`" `"$taskExe`"`r`n" | Set-Content -Path $startupScript -Encoding ASCII
				Write-Host "Scheduled Task access denied; created Startup folder entry at $startupScript"
			} else {
				throw "Failed to register scheduled task '$TaskName'. $($_.Exception.Message)"
			}
		}
	}
}

Write-Host "Install complete"

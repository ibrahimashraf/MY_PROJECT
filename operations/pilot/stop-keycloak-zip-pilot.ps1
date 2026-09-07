#[INTEGIN Keycloak ZIP Pilot] Stop only the recorded local Keycloak ZIP Java process; retain dedicated identity PostgreSQL and verified artifacts for review.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$metadataPath = 'C:\MY PROJECT\operations\pilot\runtime\keycloak-zip\keycloak-zip-pilot-metadata.json'
if (-not (Test-Path -LiteralPath $metadataPath)) { Write-Output 'KEYCLOAK_ZIP_PILOT_PROCESS_ALREADY_ABSENT'; exit 0 }
$metadata = Get-Content -LiteralPath $metadataPath -Raw | ConvertFrom-Json
if ($null -eq $metadata.process_id -or [int]$metadata.process_id -le 0) { throw 'Keycloak ZIP pilot metadata has no valid process identifier.' }
$process = Get-Process -Id ([int]$metadata.process_id) -ErrorAction SilentlyContinue
if ($null -ne $process) {
  Stop-Process -Id $process.Id -ErrorAction Stop
  $process.WaitForExit(20000) | Out-Null
  if (-not $process.HasExited) { Stop-Process -Id $process.Id -Force -ErrorAction Stop }
}
Remove-Item -LiteralPath $metadataPath -Force
Write-Output 'KEYCLOAK_ZIP_PILOT_PROCESS_STOPPED_IDENTITY_DATA_RETAINED'

param(
  [string]$WorkspaceRoot = 'C:\MY_PROJECT'
)

$ErrorActionPreference = 'Stop'

$inventoryPath = Join-Path $WorkspaceRoot 'integin-pilot-source\contracts\production_secret_inventory_v1.json'
if (-not (Test-Path -LiteralPath $inventoryPath)) {
  throw "Secret inventory is missing: $inventoryPath"
}

# This verifier reads only the non-secret inventory artifact, protected-directory
# metadata, Docker lifecycle/network names, and OpenBao sealed-status metadata.
# It never reads private file contents or Docker environment values.
$inventory = Get-Content -LiteralPath $inventoryPath -Raw | ConvertFrom-Json
if ($inventory.inventory_version -ne 'production-secret-inventory/v1') {
  throw "Unexpected secret inventory version: $($inventory.inventory_version)"
}
if ($inventory.status -ne 'design-baseline-not-wired') {
  throw "Secret inventory must remain design-only and unwired: $($inventory.status)"
}
if (-not $inventory.secret_classes -or $inventory.secret_classes.Count -lt 7) {
  throw 'Secret inventory has insufficient classified secret references.'
}
foreach ($class in $inventory.secret_classes) {
  if ([string]::IsNullOrWhiteSpace($class.owner) -or [string]::IsNullOrWhiteSpace($class.recovery_reference)) {
    throw "Secret inventory ownership/recovery metadata is incomplete for $($class.secret_id)."
  }
  if (-not $class.planned_openbao_path.StartsWith('kv/production/')) {
    throw "Secret inventory path is outside production namespace: $($class.planned_openbao_path)"
  }
}

$privateDirectory = Join-Path $WorkspaceRoot 'private\integin-secrets'
if (-not (Test-Path -LiteralPath $privateDirectory)) {
  throw "Protected private directory is missing: $privateDirectory"
}

$containerStatus = docker ps --filter 'name=integin-pilot-openbao' --format '{{.Names}} {{.Status}}'
if ($LASTEXITCODE -ne 0 -or $containerStatus -notmatch '^integin-pilot-openbao Up') {
  throw 'OpenBao pilot container is not running.'
}

$networkNames = docker inspect integin-pilot-openbao --format '{{range $key, $_ := .NetworkSettings.Networks}}{{$key}} {{end}}'
if ($LASTEXITCODE -ne 0) {
  throw 'OpenBao pilot network inspection failed.'
}
if ($networkNames.Trim() -ne 'integin-pilot-secrets-net') {
  throw "OpenBao pilot is connected to an unexpected network set: $($networkNames.Trim())"
}

$sealStatus = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri 'http://127.0.0.1:18200/v1/sys/seal-status'
if ($sealStatus.StatusCode -ne 200) {
  throw "OpenBao seal-status response was $($sealStatus.StatusCode)"
}
$status = $sealStatus.Content | ConvertFrom-Json
if (-not $status.initialized -or -not $status.sealed -or $status.progress -ne 0) {
  throw 'OpenBao pilot is not in the required initialized sealed baseline state.'
}

Write-Output 'PRODUCTION_SECRET_INVENTORY_REFERENCE_ONLY_VALIDATED'
Write-Output 'OPENBAO_PILOT_SEALED_UNWIRED_BOUNDARY_VALIDATED'

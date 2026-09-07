#[INTEGIN Keycloak ZIP Pilot] Create private ZIP-runtime configuration without displaying any credential values.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$secretDirectory = 'C:\INTEGIN-SECRETS'
$postgresEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-postgres.env'
$zipRuntimeEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-zip-runtime.env'

function Read-EnvironmentFile {
  param([Parameter(Mandatory = $true)][string]$Path)

  $values = @{}
  foreach ($line in Get-Content -LiteralPath $Path) {
    $trimmed = $line.Trim()
    if ($trimmed.Length -eq 0 -or $trimmed.StartsWith('#')) { continue }
    $separator = $trimmed.IndexOf('=')
    if ($separator -lt 1) { throw "Invalid private environment entry in $Path" }
    $name = $trimmed.Substring(0, $separator)
    $value = $trimmed.Substring($separator + 1)
    if ([string]::IsNullOrWhiteSpace($value)) { throw "Private environment entry $name is empty." }
    $values[$name] = $value
  }
  return $values
}

function New-PrivateSecret {
  param([Parameter(Mandatory = $true)][int]$Bytes)

  $buffer = New-Object byte[] $Bytes
  $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
  try { $rng.GetBytes($buffer) } finally { $rng.Dispose() }
  return [Convert]::ToBase64String($buffer)
}

if (-not (Test-Path -LiteralPath $postgresEnvironmentPath)) {
  throw "Required private Keycloak PostgreSQL environment is missing: $postgresEnvironmentPath"
}
if (Test-Path -LiteralPath $zipRuntimeEnvironmentPath) {
  throw "Refusing to overwrite existing private ZIP runtime environment: $zipRuntimeEnvironmentPath"
}

$postgres = Read-EnvironmentFile -Path $postgresEnvironmentPath
foreach ($required in @('POSTGRES_USER', 'POSTGRES_PASSWORD', 'POSTGRES_DB')) {
  if (-not $postgres.ContainsKey($required)) { throw "Required private PostgreSQL key is absent: $required" }
}
if ($postgres['POSTGRES_USER'] -ne 'integin_keycloak_owner' -or $postgres['POSTGRES_DB'] -ne 'keycloak_pilot') {
  throw 'The existing private Keycloak PostgreSQL environment does not match the isolated ZIP-pilot identity database contract.'
}

$lines = @(
  'KC_DB=postgres',
  'KC_DB_URL=jdbc:postgresql://127.0.0.1:15433/keycloak_pilot',
  ('KC_DB_USERNAME=' + $postgres['POSTGRES_USER']),
  ('KC_DB_PASSWORD=' + $postgres['POSTGRES_PASSWORD']),
  ('KC_BOOTSTRAP_ADMIN_USERNAME=pilot_admin_' + [Guid]::NewGuid().ToString('N').Substring(0, 12)),
  ('KC_BOOTSTRAP_ADMIN_PASSWORD=' + (New-PrivateSecret -Bytes 32)),
  'KC_HEALTH_ENABLED=true',
  'KC_HTTP_HOST=127.0.0.1',
  'KC_HTTP_PORT=18180',
  'KC_HTTP_MANAGEMENT_HOST=127.0.0.1',
  'KC_HTTP_MANAGEMENT_PORT=19090'
)

New-Item -ItemType Directory -Force -Path $secretDirectory | Out-Null
[System.IO.File]::WriteAllText($zipRuntimeEnvironmentPath, ($lines -join [Environment]::NewLine), (New-Object System.Text.UTF8Encoding($false)))
Write-Output 'KEYCLOAK_ZIP_PILOT_PRIVATE_RUNTIME_ENV_CREATED_WITHOUT_DISPLAYING_VALUES'

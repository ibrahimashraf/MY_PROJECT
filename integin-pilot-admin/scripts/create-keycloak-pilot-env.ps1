[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$secretDirectory = 'C:\INTEGIN-SECRETS'
$postgresEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-postgres.env'
$runtimeEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-runtime.env'
$markerPath = 'C:\INTEGIN-PILOT\runtime\keycloak-pilot-private-env.created.txt'

foreach ($path in @($postgresEnvironmentPath, $runtimeEnvironmentPath)) {
  if (Test-Path -LiteralPath $path) {
    throw "Refusing to overwrite existing private Keycloak pilot environment file: $path"
  }
}

function New-PilotSecret {
  param([Parameter(Mandatory = $true)][int]$Length)

  $alphabet = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  $bytes = New-Object byte[] $Length
  [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
  $builder = New-Object System.Text.StringBuilder
  foreach ($byte in $bytes) {
    [void] $builder.Append($alphabet[$byte % $alphabet.Length])
  }
  return $builder.ToString()
}

New-Item -ItemType Directory -Force -Path $secretDirectory | Out-Null
New-Item -ItemType Directory -Force -Path (Split-Path -Parent $markerPath) | Out-Null

$postgresPassword = New-PilotSecret -Length 48
$bootstrapPassword = New-PilotSecret -Length 64
$temporaryPaths = @()

try {
  $postgresLines = @(
    'POSTGRES_USER=integin_keycloak_owner',
    "POSTGRES_PASSWORD=$postgresPassword",
    'POSTGRES_DB=keycloak_pilot'
  )
  $runtimeLines = @(
    'KC_DB=postgres',
    'KC_DB_URL=jdbc:postgresql://integin-pilot-keycloak-postgres:5432/keycloak_pilot',
    'KC_DB_USERNAME=integin_keycloak_owner',
    "KC_DB_PASSWORD=$postgresPassword",
    'KC_BOOTSTRAP_ADMIN_USERNAME=security_operator_pilot',
    "KC_BOOTSTRAP_ADMIN_PASSWORD=$bootstrapPassword",
    'KC_HEALTH_ENABLED=true',
    'KC_METRICS_ENABLED=false',
    'KC_HOSTNAME=http://127.0.0.1:18180'
  )

  $postgresTemporaryPath = "$postgresEnvironmentPath.new"
  $runtimeTemporaryPath = "$runtimeEnvironmentPath.new"
  $temporaryPaths = @($postgresTemporaryPath, $runtimeTemporaryPath)

  foreach ($path in $temporaryPaths) {
    if (Test-Path -LiteralPath $path) {
      throw "Refusing to overwrite temporary private Keycloak pilot environment file: $path"
    }
  }

  [System.IO.File]::WriteAllLines($postgresTemporaryPath, [string[]]$postgresLines, [System.Text.UTF8Encoding]::new($false))
  [System.IO.File]::WriteAllLines($runtimeTemporaryPath, [string[]]$runtimeLines, [System.Text.UTF8Encoding]::new($false))
  Move-Item -LiteralPath $postgresTemporaryPath -Destination $postgresEnvironmentPath
  Move-Item -LiteralPath $runtimeTemporaryPath -Destination $runtimeEnvironmentPath

  $marker = @(
    'purpose=isolated-keycloak-pilot-private-environment',
    "created_at_utc=$([DateTime]::UtcNow.ToString('o'))",
    'secrets_displayed=false',
    'acceptance_environment_changed=false',
    'openbao_connected=false'
  )
  [System.IO.File]::WriteAllLines($markerPath, [string[]]$marker, [System.Text.UTF8Encoding]::new($false))
} finally {
  foreach ($path in $temporaryPaths) {
    if (Test-Path -LiteralPath $path) {
      Remove-Item -LiteralPath $path -Force
    }
  }
}

Write-Output 'KEYCLOAK_PILOT_PRIVATE_ENV_CREATED_WITHOUT_DISPLAYING_VALUES'

#[INTEGIN Casdoor Pilot] Start the Casdoor IdP against its isolated PostgreSQL container; drop-in 127.0.0.1:18180 replacement for Keycloak.
# Data persists across restarts; pass -Fresh for a factory reset (wipes Casdoor users/apps).
# Pinned images (NOT :latest):
#   postgres:16-alpine  digest sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685
#   casbin/casdoor:latest@sha256:1b479655bf51b1c630f2a3ea93ec1ef58388e1d5861173377269266ea321537e  (= v4.3.0, 2026-09-09; versioned v4.x tags do not resolve from all daemons, digest pin keeps it immutable)
[CmdletBinding()]
param([switch]$Fresh)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$pilotRoot = 'C:\MY_PROJECT\operations\pilot'
$secretDirectory = 'C:\MY_PROJECT\private\integin-secrets'
$postgresEnvironmentPath = Join-Path $secretDirectory 'casdoor-pilot-postgres.env'
$casdoorEnvironmentPath = Join-Path $secretDirectory 'casdoor-pilot-runtime.env'
$casdoorConfigHostPath = Join-Path $pilotRoot 'casdoor\app.conf'
$casdoorContainer = 'integin-pilot-casdoor'
$postgresContainer = 'integin-pilot-casdoor-postgres'
$networkName = 'integin-casdoor-pilot-net'
$postgresVolume = 'integin-pilot-casdoor-postgres-data'
$postgresImage = 'postgres:16-alpine@sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685'
$casdoorImage = 'casbin/casdoor:latest@sha256:1b479655bf51b1c630f2a3ea93ec1ef58388e1d5861173377269266ea321537e'

function Read-EnvironmentFile {
  param([Parameter(Mandatory = $true)][string]$Path)
  $values = @{}
  foreach ($line in Get-Content -LiteralPath $Path) {
    $trimmed = $line.Trim()
    if ($trimmed.Length -eq 0 -or $trimmed.StartsWith('#')) { continue }
    $separator = $trimmed.IndexOf('=')
    if ($separator -lt 1) { throw "Invalid private environment entry in $Path" }
    $values[$trimmed.Substring(0, $separator)] = $trimmed.Substring($separator + 1)
  }
  return $values
}

function Get-DockerExitCode {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)
  $previous = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    & docker @Arguments 1>$null 2>$null
    return $LASTEXITCODE
  } finally { $ErrorActionPreference = $previous }
}

function Invoke-DockerQuietly {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)
  $previous = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    & docker @Arguments | Out-Null
    $exitCode = $LASTEXITCODE
  } finally { $ErrorActionPreference = $previous }
  if ($exitCode -ne 0) { throw "Docker command failed: docker $($Arguments -join ' ')" }
}

function Test-DockerResourceExists {
  param([Parameter(Mandatory = $true)][ValidateSet('container', 'network', 'volume')][string]$Kind,[Parameter(Mandatory = $true)][string]$Name)
  return (Get-DockerExitCode -Arguments @($Kind, 'inspect', $Name)) -eq 0
}

function Assert-IssuerMutable {
  if (Get-NetTCPConnection -State Listen -LocalPort 18180 -ErrorAction SilentlyContinue) {
    throw 'Loopback port 18180 is already in use; fail fast — Casdoor must not fight an existing Keycloak :18180 holder.'
  }
}

function Wait-PostgresReady {
  $deadline = [DateTime]::UtcNow.AddSeconds(90)
  while ([DateTime]::UtcNow -lt $deadline) {
    if ((Get-DockerExitCode -Arguments @('exec', $postgresContainer, 'pg_isready', '-U', 'integin_casdoor_owner', '-d', 'casdoor_pilot')) -eq 0) { return }
    Start-Sleep -Seconds 2
  }
  throw 'Isolated Casdoor PostgreSQL container did not become ready.'
}

foreach ($path in @($postgresEnvironmentPath, $casdoorEnvironmentPath, $casdoorConfigHostPath)) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Required Casdoor-pilot file is missing: $path" }
}
Assert-IssuerMutable

$postgres = Read-EnvironmentFile -Path $postgresEnvironmentPath
$casdoor = Read-EnvironmentFile -Path $casdoorEnvironmentPath
foreach ($required in @('POSTGRES_USER','POSTGRES_PASSWORD','POSTGRES_DB')) { if (-not $postgres.ContainsKey($required)) { throw "Private Casdoor PostgreSQL environment is missing $required" } }
foreach ($required in @('CASDOOR_PG_USER','CASDOOR_PG_PASSWORD','CASDOOR_PG_DB','CASDOOR_PG_HOST','CASDOOR_PG_PORT')) { if (-not $casdoor.ContainsKey($required)) { throw "Private Casdoor runtime environment is missing $required" } }
if ($casdoor['CASDOOR_PG_HOST'] -ne 'integin-pilot-casdoor-postgres' -or $casdoor['CASDOOR_PG_PORT'] -ne '5432' -or $casdoor['CASDOOR_PG_DB'] -ne 'casdoor_pilot') { throw 'Private Casdoor runtime environment violates the documented isolated topology.' }

if ((Get-DockerExitCode -Arguments @('image', 'inspect', $postgresImage)) -ne 0) { Invoke-DockerQuietly -Arguments @('pull', $postgresImage) }
if ((Get-DockerExitCode -Arguments @('image', 'inspect', $casdoorImage)) -ne 0) { Invoke-DockerQuietly -Arguments @('pull', $casdoorImage) }

if (-not (Test-DockerResourceExists -Kind 'network' -Name $networkName)) { Invoke-DockerQuietly -Arguments @('network', 'create', $networkName) }
if (Test-DockerResourceExists -Kind 'container' -Name $postgresContainer) {
  Invoke-DockerQuietly -Arguments @('rm', '-f', $postgresContainer)
}
if ($Fresh -and (Test-DockerResourceExists -Kind 'volume' -Name $postgresVolume)) {
  Invoke-DockerQuietly -Arguments @('volume', 'rm', $postgresVolume)
}
if (-not (Test-DockerResourceExists -Kind 'volume' -Name $postgresVolume)) {
  Invoke-DockerQuietly -Arguments @('volume', 'create', $postgresVolume)
}
Invoke-DockerQuietly -Arguments @('run', '-d', '--name', $postgresContainer, '--network', $networkName, '--env-file', $postgresEnvironmentPath, '--volume', ("${postgresVolume}:/var/lib/postgresql/data"), $postgresImage)
Wait-PostgresReady

if (Test-DockerResourceExists -Kind 'container' -Name $casdoorContainer) {
  Invoke-DockerQuietly -Arguments @('rm', '-f', $casdoorContainer)
}
Invoke-DockerQuietly -Arguments @('run', '-d', '--name', $casdoorContainer, '--network', $networkName, '-e', 'INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK=true', '--mount', ("type=bind,source=${casdoorConfigHostPath},target=/conf/app.conf"), '--publish', '127.0.0.1:18180:8000', $casdoorImage)

function Wait-HealthyUrl {
  param([Parameter(Mandatory = $true)][string]$Url,[Parameter(Mandatory = $true)][int]$TimeoutSeconds)
  $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
  while ([DateTime]::UtcNow -lt $deadline) {
    try {
      $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri $Url -ErrorAction Stop
      if ($response.StatusCode -eq 200) { return }
    } catch { }
    Start-Sleep -Seconds 2
  }
  throw "Casdoor pilot did not return HTTP 200: $Url"
}
Wait-HealthyUrl -Url 'http://127.0.0.1:18180/.well-known/openid-configuration' -TimeoutSeconds 90
Write-Output 'CASDOOR_PILOT_ISOLATED_HEALTH_AND_DISCOVERY_VERIFIED'

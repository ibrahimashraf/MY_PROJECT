[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$networkName = 'integin-identity-pilot-net'
$postgresContainer = 'integin-pilot-keycloak-postgres'
$keycloakContainer = 'integin-pilot-keycloak'
$postgresVolume = 'integin-pilot-keycloak-postgres-data'
$postgresImage = 'postgres:18'
$keycloakImage = 'quay.io/keycloak/keycloak:26.7.1'
$postgresEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\keycloak-pilot-postgres.env'
$runtimeEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\keycloak-pilot-runtime.env'
$metadataPath = 'C:\MY_PROJECT\operations\pilot\runtime\keycloak-pilot-metadata.json'

function Invoke-DockerQuietly {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)

  & docker @Arguments | Out-Null
  if ($LASTEXITCODE -ne 0) {
    throw "Docker command failed: docker $($Arguments -join ' ')"
  }
}

function Get-DockerExitCode {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)

  $previousErrorActionPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    & docker @Arguments 1>$null 2>$null
    return $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
}

function Test-DockerResourceExists {
  param(
    [Parameter(Mandatory = $true)][ValidateSet('container', 'network', 'volume')][string]$Kind,
    [Parameter(Mandatory = $true)][string]$Name
  )

  return (Get-DockerExitCode -Arguments @($Kind, 'inspect', $Name)) -eq 0
}

function Assert-HealthyUrl {
  param([Parameter(Mandatory = $true)][string]$Url)

  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri $Url
  } catch {
    throw "Required preflight endpoint is unavailable: $Url"
  }
  if ($response.StatusCode -ne 200) {
    throw "Required preflight endpoint did not return HTTP 200: $Url"
  }
}

function Wait-HealthyUrl {
  param(
    [Parameter(Mandatory = $true)][string]$Url,
    [Parameter(Mandatory = $true)][int]$TimeoutSeconds
  )

  $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
  do {
    try {
      $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 3 -Uri $Url
      if ($response.StatusCode -eq 200) {
        return
      }
    } catch {
      # Keycloak is still starting. Do not emit response content or exception detail.
    }
    Start-Sleep -Seconds 2
  } while ([DateTime]::UtcNow -lt $deadline)

  throw "Timed out waiting for Keycloak pilot endpoint: $Url"
}

foreach ($path in @($postgresEnvironmentPath, $runtimeEnvironmentPath)) {
  if (-not (Test-Path -LiteralPath $path)) {
    throw "Required private Keycloak pilot environment file is missing: $path"
  }
}

foreach ($endpoint in @(
  'http://127.0.0.1:8080/healthz',
  'http://127.0.0.1:8080/readyz',
  'http://127.0.0.1:18080/healthz',
  'http://127.0.0.1:18080/readyz'
)) {
  Assert-HealthyUrl -Url $endpoint
}

foreach ($resource in @(
  @{ Kind = 'container'; Name = $postgresContainer },
  @{ Kind = 'container'; Name = $keycloakContainer },
  @{ Kind = 'network'; Name = $networkName },
  @{ Kind = 'volume'; Name = $postgresVolume }
)) {
  if (Test-DockerResourceExists -Kind $resource.Kind -Name $resource.Name) {
    throw "Refusing to reuse existing Keycloak pilot $($resource.Kind): $($resource.Name)"
  }
}

if ((Get-DockerExitCode -Arguments @('image', 'inspect', $postgresImage)) -ne 0) {
  Invoke-DockerQuietly -Arguments @('pull', $postgresImage)
}
if ((Get-DockerExitCode -Arguments @('image', 'inspect', $keycloakImage)) -ne 0) {
  Invoke-DockerQuietly -Arguments @('pull', $keycloakImage)
}

$keycloakDigest = (& docker image inspect $keycloakImage --format '{{index .RepoDigests 0}}').Trim()
if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace($keycloakDigest)) {
  throw 'Unable to resolve the pulled Keycloak image digest.'
}

Invoke-DockerQuietly -Arguments @('network', 'create', '--internal', '--label', 'integin.scope=identity-pilot', $networkName)
Invoke-DockerQuietly -Arguments @(
  'run', '-d', '--name', $postgresContainer,
  '--network', $networkName,
  '--label', 'integin.scope=identity-pilot',
  '--env-file', $postgresEnvironmentPath,
  '--volume', "$postgresVolume`:/var/lib/postgresql",
  $postgresImage
)

$postgresDeadline = [DateTime]::UtcNow.AddSeconds(60)
$postgresExitCode = 1
do {
  $postgresExitCode = Get-DockerExitCode -Arguments @('exec', $postgresContainer, 'pg_isready', '-U', 'integin_keycloak_owner', '-d', 'keycloak_pilot')
  if ($postgresExitCode -eq 0) {
    break
  }
  Start-Sleep -Seconds 2
} while ([DateTime]::UtcNow -lt $postgresDeadline)
if ($postgresExitCode -ne 0) {
  throw 'The isolated Keycloak PostgreSQL container did not become ready.'
}

Invoke-DockerQuietly -Arguments @(
  'run', '-d', '--name', $keycloakContainer,
  '--network', $networkName,
  '--label', 'integin.scope=identity-pilot',
  '--memory', '1g',
  '--publish', '127.0.0.1:18180:8080',
  '--publish', '127.0.0.1:19090:9000',
  '--env-file', $runtimeEnvironmentPath,
  $keycloakImage,
  'start-dev', '--http-port=8080', '--http-management-port=9000'
)

Wait-HealthyUrl -Url 'http://127.0.0.1:19090/health/ready' -TimeoutSeconds 120
Wait-HealthyUrl -Url 'http://127.0.0.1:18180/realms/master/.well-known/openid-configuration' -TimeoutSeconds 30

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $metadataPath) | Out-Null
$metadata = [ordered]@{
  purpose = 'isolated-keycloak-oidc-pilot'
  created_at_utc = [DateTime]::UtcNow.ToString('o')
  keycloak_image = $keycloakImage
  keycloak_image_digest = $keycloakDigest
  keycloak_container = $keycloakContainer
  keycloak_http = '127.0.0.1:18180'
  keycloak_management = '127.0.0.1:19090'
  postgres_container = $postgresContainer
  postgres_host_published = $false
  docker_network = $networkName
  openbao_connected = $false
  integin_oidc_enabled = $false
}
$metadata | ConvertTo-Json | Set-Content -LiteralPath $metadataPath -Encoding utf8

foreach ($endpoint in @(
  'http://127.0.0.1:8080/healthz',
  'http://127.0.0.1:8080/readyz',
  'http://127.0.0.1:18080/healthz',
  'http://127.0.0.1:18080/readyz'
)) {
  Assert-HealthyUrl -Url $endpoint
}

Write-Output 'KEYCLOAK_PILOT_ISOLATED_HEALTH_AND_DISCOVERY_VERIFIED'

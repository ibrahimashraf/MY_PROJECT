#[INTEGIN Keycloak ZIP Pilot] Start only the verified local Keycloak ZIP against its separate loopback identity PostgreSQL database.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$pilotRoot = 'C:\INTEGIN-PILOT'
$secretDirectory = 'C:\integin-secrets'
$postgresEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-postgres.env'
$zipRuntimeEnvironmentPath = Join-Path $secretDirectory 'keycloak-pilot-zip-runtime.env'
$keycloakHome = Join-Path $pilotRoot 'keycloak-zip\keycloak-26.7.1'
$runner = Join-Path $keycloakHome 'lib\quarkus-run.jar'
$java = 'C:\Program Files\Java\jdk-26.0.1\bin\java.exe'
$runtimeDirectory = Join-Path $pilotRoot 'runtime\keycloak-zip'
$metadataPath = Join-Path $runtimeDirectory 'keycloak-zip-pilot-metadata.json'
$postgresContainer = 'integin-pilot-keycloak-zip-postgres'
$networkName = 'integin-identity-zip-pilot-net'
$postgresVolume = 'integin-pilot-keycloak-zip-postgres-data'
$postgresImage = 'postgres:18'

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

function Get-ZipIdentityResourceMode {
  $containerExists = Test-DockerResourceExists -Kind 'container' -Name $postgresContainer
  $networkExists = Test-DockerResourceExists -Kind 'network' -Name $networkName
  $volumeExists = Test-DockerResourceExists -Kind 'volume' -Name $postgresVolume
  $count = @(@($containerExists, $networkExists, $volumeExists) | Where-Object { $_ }).Count
  if ($count -eq 0) { return 'absent' }
  if ($count -ne 3) { throw 'Partial ZIP-specific Keycloak identity resources exist; refusing adoption or cleanup.' }
  $identity = & docker inspect $postgresContainer --format '{{.Config.Image}}|{{.HostConfig.NetworkMode}}|{{json .HostConfig.PortBindings}}|{{range .Mounts}}{{.Name}}:{{.Destination}};{{end}}'
  if ($LASTEXITCODE -ne 0) { throw 'Unable to inspect existing ZIP-specific identity PostgreSQL resource.' }
  if ($identity -notmatch '^postgres:18\|integin-identity-zip-pilot-net\|' -or $identity -notmatch '"HostIp":"127\.0\.0\.1"' -or $identity -notmatch '"HostPort":"15433"' -or $identity -notmatch 'integin-pilot-keycloak-zip-postgres-data:/var/lib/postgresql') { throw 'Existing ZIP-specific identity resource does not match the documented isolated topology.' }
  return 'matching'
}

function Assert-HealthyUrl {
  param([Parameter(Mandatory = $true)][string]$Url)
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 -Uri $Url
    if ($response.StatusCode -ne 200) { throw "Unexpected HTTP status $($response.StatusCode)" }
  } catch { throw "Required preflight endpoint is unavailable: $Url" }
}

function Assert-OpenBaoSealed {
  $previous = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    $raw = & docker exec integin-pilot-openbao env BAO_ADDR='http://127.0.0.1:8200' bao status -format=json 2>$null
    $exitCode = $LASTEXITCODE
  } finally { $ErrorActionPreference = $previous }
  if ($exitCode -ne 0 -and $exitCode -ne 2) { throw 'Unable to inspect isolated OpenBao sealed status.' }
  $status = $raw | ConvertFrom-Json
  if (-not $status.sealed) { throw 'Isolated OpenBao must remain sealed before Keycloak ZIP pilot startup.' }
}

function Wait-PostgresReady {
  $deadline = [DateTime]::UtcNow.AddSeconds(90)
  while ([DateTime]::UtcNow -lt $deadline) {
    if ((Get-DockerExitCode -Arguments @('exec', $postgresContainer, 'pg_isready', '-U', 'integin_keycloak_owner', '-d', 'keycloak_pilot')) -eq 0) { return }
    Start-Sleep -Seconds 2
  }
  throw 'Dedicated Keycloak PostgreSQL container did not become ready.'
}

function Wait-HealthyUrl {
  param([Parameter(Mandatory = $true)][string]$Url,[Parameter(Mandatory = $true)][int]$TimeoutSeconds)
  $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
  while ([DateTime]::UtcNow -lt $deadline) {
    try {
      $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri $Url
      if ($response.StatusCode -eq 200) { return }
    } catch { }
    Start-Sleep -Seconds 2
  }
  throw "Keycloak ZIP pilot did not return HTTP 200: $Url"
}

foreach ($path in @($postgresEnvironmentPath, $zipRuntimeEnvironmentPath, $runner, $java)) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Required ZIP-pilot file is missing: $path" }
}
foreach ($url in @('http://127.0.0.1:8080/healthz','http://127.0.0.1:8080/readyz','http://127.0.0.1:18080/healthz','http://127.0.0.1:18080/readyz')) { Assert-HealthyUrl -Url $url }
Assert-OpenBaoSealed
if (Test-Path -LiteralPath $metadataPath) { throw 'A Keycloak ZIP pilot metadata file already exists; refusing a second process.' }
foreach ($port in @(15433, 18180, 19090)) {
  if ($port -eq 15433 -and (Get-ZipIdentityResourceMode) -eq 'matching') { continue }
  if (Get-NetTCPConnection -State Listen -LocalPort $port -ErrorAction SilentlyContinue) { throw "Required loopback port is already in use: $port" }
}
$resourceMode = Get-ZipIdentityResourceMode

$postgres = Read-EnvironmentFile -Path $postgresEnvironmentPath
$runtime = Read-EnvironmentFile -Path $zipRuntimeEnvironmentPath
foreach ($required in @('POSTGRES_USER','POSTGRES_PASSWORD','POSTGRES_DB')) { if (-not $postgres.ContainsKey($required)) { throw "Private PostgreSQL environment is missing $required" } }
foreach ($required in @('KC_DB','KC_DB_URL','KC_DB_USERNAME','KC_DB_PASSWORD','KC_BOOTSTRAP_ADMIN_USERNAME','KC_BOOTSTRAP_ADMIN_PASSWORD','KC_HEALTH_ENABLED','KC_HTTP_HOST','KC_HTTP_PORT','KC_HTTP_MANAGEMENT_HOST','KC_HTTP_MANAGEMENT_PORT')) { if (-not $runtime.ContainsKey($required)) { throw "Private ZIP runtime environment is missing $required" } }
if ($runtime['KC_DB_URL'] -ne 'jdbc:postgresql://127.0.0.1:15433/keycloak_pilot' -or $runtime['KC_HTTP_HOST'] -ne '127.0.0.1' -or $runtime['KC_HTTP_PORT'] -ne '18180' -or $runtime['KC_HTTP_MANAGEMENT_HOST'] -ne '127.0.0.1' -or $runtime['KC_HTTP_MANAGEMENT_PORT'] -ne '19090') { throw 'Private ZIP runtime environment violates the documented local-only topology.' }

if ((Get-DockerExitCode -Arguments @('image', 'inspect', $postgresImage)) -ne 0) { Invoke-DockerQuietly -Arguments @('pull', $postgresImage) }
if ($resourceMode -eq 'absent') {
  Invoke-DockerQuietly -Arguments @('network', 'create', $networkName)
  Invoke-DockerQuietly -Arguments @('volume', 'create', $postgresVolume)
  Invoke-DockerQuietly -Arguments @('run', '-d', '--name', $postgresContainer, '--network', $networkName, '--env-file', $postgresEnvironmentPath, '--publish', '127.0.0.1:15433:5432', '--volume', ("${postgresVolume}:/var/lib/postgresql"), $postgresImage)
}
Wait-PostgresReady

New-Item -ItemType Directory -Force -Path $runtimeDirectory | Out-Null
$stdoutPath = Join-Path $runtimeDirectory 'keycloak-zip.stdout.log'
$stderrPath = Join-Path $runtimeDirectory 'keycloak-zip.stderr.log'
foreach ($path in @($stdoutPath, $stderrPath)) { if (Test-Path -LiteralPath $path) { Move-Item -LiteralPath $path -Destination ($path + '.before-retry-' + [DateTime]::UtcNow.Ticks) } }
$originalEnvironment = @{}
foreach ($key in $runtime.Keys) { $originalEnvironment[$key] = [Environment]::GetEnvironmentVariable($key, 'Process'); [Environment]::SetEnvironmentVariable($key, $runtime[$key], 'Process') }
$originalHome = [Environment]::GetEnvironmentVariable('KC_HOME_DIR', 'Process')
[Environment]::SetEnvironmentVariable('KC_HOME_DIR', $keycloakHome.Replace('\','/'), 'Process')
try {
  $process = Start-Process -FilePath $java -ArgumentList @('-Dkc.config.built=true', '-jar', $runner, 'start-dev', '--http-host=127.0.0.1', '--http-port=18180', '--http-management-host=127.0.0.1', '--http-management-port=19090', '--health-enabled=true') -WorkingDirectory $keycloakHome -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -PassThru
} finally {
  foreach ($key in $runtime.Keys) { [Environment]::SetEnvironmentVariable($key, $originalEnvironment[$key], 'Process') }
  [Environment]::SetEnvironmentVariable('KC_HOME_DIR', $originalHome, 'Process')
}

try {
  Wait-HealthyUrl -Url 'http://127.0.0.1:19090/health/ready' -TimeoutSeconds 150
  Wait-HealthyUrl -Url 'http://127.0.0.1:18180/realms/master/.well-known/openid-configuration' -TimeoutSeconds 30
} catch {
  if (-not $process.HasExited) { Stop-Process -Id $process.Id -ErrorAction SilentlyContinue }
  throw "Keycloak ZIP pilot failed startup validation. Review $stdoutPath and $stderrPath."
}

$metadata = [ordered]@{ process_id=$process.Id; started_at_utc=[DateTime]::UtcNow.ToString('o'); java=$java; distribution_version='26.7.1'; http='127.0.0.1:18180'; management='127.0.0.1:19090'; postgres_container=$postgresContainer } | ConvertTo-Json
[System.IO.File]::WriteAllText($metadataPath, $metadata, (New-Object System.Text.UTF8Encoding($false)))
Write-Output 'KEYCLOAK_ZIP_PILOT_ISOLATED_HEALTH_AND_DISCOVERY_VERIFIED'

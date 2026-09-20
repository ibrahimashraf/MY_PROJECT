#[INTEGIN RustFS Pilot] Start the isolated pilot object store (S3 on 127.0.0.1:19000,
# console on 127.0.0.1:19001) with its own data/log volumes. Credentials come
# from the private pilot env (INTEGIN_S3_ACCESS_KEY/INTEGIN_S3_SECRET_KEY) so the
# server and the store always agree. Data persists across restarts; pass
# -Fresh for a factory reset (wipes buckets).
# Pinned image (NOT :latest): rustfs/rustfs:1.0.0-rc.1 (matches
# deploy/compose/integin-infrastructure.compose.yaml and the Stage-A drill docs).
[CmdletBinding()]
param([switch]$Fresh)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$pilotRoot = 'C:\MY_PROJECT\operations\pilot'
$secretPath = 'C:\MY_PROJECT\private\integin-secrets\integin-pilot.env'
$containerName = 'integin-pilot-rustfs'
$dataVolume = 'integin-pilot-rustfs-data'
$logsVolume = 'integin-pilot-rustfs-logs'
$image = 'rustfs/rustfs:1.0.0-rc.1'
$s3Port = '19000'
$consolePort = '19001'

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

if (-not (Test-Path -LiteralPath $secretPath)) { throw "Pilot secrets file is missing: $secretPath" }
$secrets = Read-EnvironmentFile -Path $secretPath
foreach ($required in @('INTEGIN_S3_ACCESS_KEY', 'INTEGIN_S3_SECRET_KEY')) {
  if ([string]::IsNullOrWhiteSpace($secrets[$required])) { throw "Required pilot setting is missing: $required" }
}

if ((Get-DockerExitCode -Arguments @('image', 'inspect', $image)) -ne 0) { Invoke-DockerQuietly -Arguments @('pull', $image) }

foreach ($name in @($containerName)) {
  if ((Get-DockerExitCode -Arguments @('container', 'inspect', $name)) -eq 0) {
    Invoke-DockerQuietly -Arguments @('rm', '-f', $name)
  }
}
if ($Fresh) {
  foreach ($volume in @($dataVolume, $logsVolume)) {
    if ((Get-DockerExitCode -Arguments @('volume', 'inspect', $volume)) -eq 0) {
      Invoke-DockerQuietly -Arguments @('volume', 'rm', $volume)
    }
  }
}
foreach ($volume in @($dataVolume, $logsVolume)) {
  if ((Get-DockerExitCode -Arguments @('volume', 'inspect', $volume)) -ne 0) {
    Invoke-DockerQuietly -Arguments @('volume', 'create', $volume)
  }
}

Invoke-DockerQuietly -Arguments @('run', '-d', '--name', $containerName, '--restart', 'unless-stopped',
  '--publish', "127.0.0.1:${s3Port}:9000", '--publish', "127.0.0.1:${consolePort}:9001",
  '-e', "RUSTFS_ACCESS_KEY=$($secrets['INTEGIN_S3_ACCESS_KEY'])",
  '-e', "RUSTFS_SECRET_KEY=$($secrets['INTEGIN_S3_SECRET_KEY'])",
  '-e', 'RUSTFS_ADDRESS=:9000', '-e', 'RUSTFS_CONSOLE_ADDRESS=:9001', '-e', 'RUSTFS_CONSOLE_ENABLE=true',
  '-e', 'RUSTFS_OBS_LOG_DIRECTORY=/var/log/rustfs',
  '--volume', "${dataVolume}:/data", '--volume', "${logsVolume}:/var/log/rustfs",
  $image, '/data')

$deadline = [DateTime]::UtcNow.AddSeconds(90)
while ([DateTime]::UtcNow -lt $deadline) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri "http://127.0.0.1:${s3Port}/minio/health/live" -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
      Write-Output 'RUSTFS_PILOT_S3_LIVE'
      exit 0
    }
  } catch { }
  Start-Sleep -Seconds 2
}
throw "RustFS pilot did not return HTTP 200 on :$s3Port/minio/health/live"

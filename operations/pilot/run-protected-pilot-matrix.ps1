$ErrorActionPreference = 'Stop'

# INTEGIN controlled pilot acceptance runner.
# Safety contract: this runner targets only the isolated loopback pilot on port 18080,
# never prints protected values or fixture contents, does not modify acceptance, and
# removes its temporary fixture in all exit paths.

$root = 'C:\MY PROJECT'
$sourceRoot = Join-Path $root 'integin-pilot-source'
$secretsPath = Join-Path $root 'private\integin-secrets\integin-pilot.env'
$fixtureDirectory = Join-Path $root 'private\integin-secrets\pilot-live-fixtures'
$pilotLauncher = Join-Path $root 'operations\pilot\launch-pilot-server.ps1'
$pilotURL = 'http://127.0.0.1:18080'

foreach ($path in @($sourceRoot, $secretsPath, $pilotLauncher)) {
  if (-not (Test-Path -LiteralPath $path)) {
    throw "Required controlled-pilot resource is unavailable."
  }
}

$target = [uri]$pilotURL
if ($target.Scheme -ne 'http' -or $target.Host -ne '127.0.0.1' -or $target.Port -ne 18080) {
  throw 'Controlled pilot target is invalid.'
}

$acceptanceListener = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction Stop | Select-Object -First 1
$pilotListener = Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction Stop | Select-Object -First 1
if ($acceptanceListener.OwningProcess -eq $pilotLilotener.OwningProcess) {
    throw 'Acceptance and pilot unexpectedly share a process; refusing to run.'
}

$values = @{}
foreach ($line in Get-Content -LiteralPath $secretsPath) {
  if ($line -match '^([^#=]+)=(.*)$') {
    $values[$matches[1].Trim()] = $matches[2]
  }
}

foreach ($required in @('INTEGIN_TENANT_ID', 'INTEGIN_DB_URL', 'PILOT_SYNC_SECRET')) {
  if ([string]::IsNullOrWhiteSpace($values[$required])) {
    throw 'Required controlled-pilot configuration is unavailable.'
  }
}

New-Item -ItemType Directory -Force -Path $fixtureDirectory | Out-Null
$fixturePath = Join-Path $fixtureDirectory ('pilot-matrix-' + [Guid]::NewGuid().ToString('N') + '.json')
$runnerEnvironment = @{
  INTEGIN_TENANT_ID = $values['INTEGIN_TENANT_ID']
  INTEGIN_DB_URL = $values['INTEGIN_DB_URL']
  INTEGIN_SYNC_SECRET = $values['PILOT_SYNC_SECRET']
  INTEGIN_SERVER_URL = $pilotURL
  INTEGIN_LIVE_FIXTURE_FILE = $fixturePath
}
$priorEnvironment = @{}

try {
  foreach ($entry in $runnerEnvironment.GetEnumerator()) {
    $priorEnvironment[$entry.Key] = [Environment]::GetEnvironmentVariable($entry.Key, 'Process')
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
  }

  Push-Location $sourceRoot
  try {
    & go run .\cmd\integin-live-matrix seed *> $null
    if ($LASTEXITCODE -ne 0) {
      throw 'Pilot matrix seed failed.'
    }
  } finally {
    Pop-Location
  }

  $activePilot = Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction Stop | Select-Object -First 1
  if ($activePilot.OwningProcess -eq $acceptanceListener.OwningProcess) {
    throw 'Pilot listener changed to the acceptance process; refusing restart.'
  }
  Stop-Process -Id $activePilot.OwningProcess -Force
  Start-Sleep -Seconds 2
  if (Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction SilentlyContinue) {
    throw 'Pilot listener did not stop cleanly.'
  }

  & powershell -NoProfile -NonInteractive -ExecutionPolicy Bypass -File $pilotLauncher *> $null
  if ($LASTEXITCODE -ne 0) {
    throw 'Pilot launcher failed.'
  }

  Push-Location $sourceRoot
  try {
    & go run .\cmd\integin-live-matrix exercise *> $null
    if ($LASTEXITCODE -ne 0) {
      throw 'Pilot matrix exercise failed.'
    }
  } finally {
    Pop-Location
  }

  Write-Output 'PROTECTED_PILOT_MATRIX_PASSED'
} finally {
  if (Test-Path -LiteralPath $fixturePath) {
    Remove-Item -LiteralPath $fixturePath -Force
  }
  foreach ($entry in $priorEnvironment.GetEnumerator()) {
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
  }
}

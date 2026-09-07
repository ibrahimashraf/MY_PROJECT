$ErrorActionPreference = 'Stop'

# Stage-bounded INTEGIN pilot matrix runner.
# It runs only against 127.0.0.1:18080, keeps protected values in memory,
# writes generic stage markers only, and deletes its fixture and transient logs
# on success, failure, or a bounded subprocess timeout.

$root = 'C:\MY PROJECT'
$sourceRoot = Join-Path $root 'integin-pilot-source'
$secretsPath = Join-Path $root 'private\integin-secrets\integin-pilot.env'
$fixtureDirectory = Join-Path $root 'private\integin-secrets\pilot-live-fixtures'
$pilotLauncher = Join-Path $root 'operations\pilot\launch-pilot-server.ps1'
$runtimeRoot = Join-Path $root 'operations\pilot\runtime'
$statusPath = Join-Path $runtimeRoot 'protected-pilot-matrix-v2.status'
$pilotURL = 'http://127.0.0.1:18080'

function Write-Stage([string]$stage, [string]$state) {
  [System.IO.File]::AppendAllText($statusPath, ("{0:o} {1} {2}`r`n" -f [DateTime]::UtcNow, $stage, $state), (New-Object System.Text.UTF8Encoding($false)))
}

function Invoke-BoundedProcess([string]$stage, [string]$filePath, [string]$arguments, [string]$workingDirectory, [int]$timeoutSeconds) {
  $stdout = Join-Path $runtimeRoot ("protected-pilot-matrix-v2-{0}.stdout.log" -f $stage)
  $stderr = Join-Path $runtimeRoot ("protected-pilot-matrix-v2-{0}.stderr.log" -f $stage)
  Remove-Item -LiteralPath $stdout, $stderr -Force -ErrorAction SilentlyContinue
  Write-Stage $stage 'STARTED'
  $process = Start-Process -FilePath $filePath -ArgumentList $arguments -WorkingDirectory $workingDirectory -RedirectStandardOutput $stdout -RedirectStandardError $stderr -PassThru
  $deadline = [DateTime]::UtcNow.AddSeconds($timeoutSeconds)
  while (-not $process.HasExited -and [DateTime]::UtcNow -lt $deadline) {
    Start-Sleep -Milliseconds 500
    $process.Refresh()
  }
  if (-not $process.HasExited) {
    Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
    Write-Stage $stage 'TIMEOUT'
    throw "Controlled pilot stage timed out: $stage"
  }
  $process.WaitForExit()
  if ($process.ExitCode -ne 0) {
    Write-Stage $stage 'FAILED'
    throw "Controlled pilot stage failed: $stage"
  }
  Remove-Item -LiteralPath $stdout, $stderr -Force -ErrorAction SilentlyContinue
  Write-Stage $stage 'PASSED'
}

foreach ($path in @($sourceRoot, $secretsPath, $pilotLauncher)) {
  if (-not (Test-Path -LiteralPath $path)) {
    throw 'Required controlled-pilot resource is unavailable.'
  }
}

$target = [uri]$pilotURL
if ($target.Scheme -ne 'http' -or $target.Host -ne '127.0.0.1' -or $target.Port -ne 18080) {
  throw 'Controlled pilot target is invalid.'
}

$acceptanceListener = Get-NetTCPConnection -LocalPort 8080 -State Listen -ErrorAction Stop | Select-Object -First 1
$pilotListener = Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction Stop | Select-Object -First 1
if ($acceptanceListener.OwningProcess -eq $pilotListener.OwningProcess) {
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

New-Item -ItemType Directory -Force -Path $fixtureDirectory, $runtimeRoot | Out-Null
[System.IO.File]::WriteAllText($statusPath, '', (New-Object System.Text.UTF8Encoding($false)))
$fixturePath = Join-Path $fixtureDirectory ('pilot-matrix-v2-' + [Guid]::NewGuid().ToString('N') + '.json')
$environmentValues = @{
  INTEGIN_TENANT_ID = $values['INTEGIN_TENANT_ID']
  INTEGIN_DB_URL = $values['INTEGIN_DB_URL']
  INTEGIN_SYNC_SECRET = $values['PILOT_SYNC_SECRET']
  INTEGIN_SERVER_URL = $pilotURL
  INTEGIN_LIVE_FIXTURE_FILE = $fixturePath
}
$previousEnvironment = @{}

try {
  foreach ($entry in $environmentValues.GetEnumerator()) {
    $previousEnvironment[$entry.Key] = [Environment]::GetEnvironmentVariable($entry.Key, 'Process')
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
  }
  Write-Stage 'PRECHECK' 'PASSED'

  Invoke-BoundedProcess 'SEED' 'powershell.exe' '-NoProfile -NonInteractive -Command "& go run .\cmd\integin-live-matrix seed; exit $LASTEXITCODE"' $sourceRoot 90

  $activePilot = Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction Stop | Select-Object -First 1
  if ($activePilot.OwningProcess -eq $acceptanceListener.OwningProcess) {
    throw 'Pilot listener changed to the acceptance process; refusing restart.'
  }
  Write-Stage 'PILOT_STOP' 'STARTED'
  Stop-Process -Id $activePilot.OwningProcess -Force
  Start-Sleep -Seconds 2
  if (Get-NetTCPConnection -LocalPort 18080 -State Listen -ErrorAction SilentlyContinue) {
    Write-Stage 'PILOT_STOP' 'FAILED'
    throw 'Pilot listener did not stop cleanly.'
  }
  Write-Stage 'PILOT_STOP' 'PASSED'

  $launcherArgs = '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + $pilotLauncher + '"'
  Invoke-BoundedProcess 'PILOT_START' 'powershell.exe' $launcherArgs $root 90
  Invoke-BoundedProcess 'EXERCISE' 'powershell.exe' '-NoProfile -NonInteractive -Command "& go run .\cmd\integin-live-matrix exercise; exit $LASTEXITCODE"' $sourceRoot 90

  Write-Stage 'MATRIX' 'PASSED'
  Write-Output 'PROTECTED_PILOT_MATRIX_V2_PASSED'
} finally {
  if (Test-Path -LiteralPath $fixturePath) {
    Remove-Item -LiteralPath $fixturePath -Force
  }
  foreach ($entry in $previousEnvironment.GetEnumerator()) {
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
  }
  Write-Stage 'CLEANUP' 'COMPLETED'
}

$ErrorActionPreference = 'Stop'

# Stage-bounded INTEGIN pilot matrix runner.
# It runs only against 127.0.0.1:18080, keeps protected values in memory,
# writes generic stage markers only, and deletes its fixture and transient logs
# on success, failure, or a bounded subprocess timeout.

$root = 'C:\MY_PROJECT'
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

function Invoke-BoundedProcess([string]$stage, [string]$message, [string]$workingDirectory, [int]$timeoutSeconds) {
  $stdout = Join-Path $runtimeRoot ("protected-pilot-matrix-v2-{0}.stdout.log" -f $stage)
  $stderr = Join-Path $runtimeRoot ("protected-pilot-matrix-v2-{0}.stderr.log" -f $stage)
  Remove-Item -LiteralPath $stdout, $stderr -Force -ErrorAction SilentlyContinue
  Write-Stage $stage 'STARTED'
  $innerCommand = "& {{ try {{ {0} 1>'{1}' 2>'{2}'; exit `$LASTEXITCODE }} catch {{ exit 1 }} }}" -f $message, $stdout, $stderr
  $arguments = '-NoProfile -NonInteractive -ExecutionPolicy Bypass -Command "' + $innerCommand + '"'
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = 'powershell.exe'
  $psi.Arguments = $arguments
  $psi.WorkingDirectory = $workingDirectory
  $psi.UseShellExecute = $false
  $psi.CreateNoWindow = $true
  $process = New-Object System.Diagnostics.Process
  $process.StartInfo = $psi
  [void]$process.Start()
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
  [void]$process.WaitForExit()
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

# INTEGIN_ rename: forward-fill new keys from pre-rename INTEGIN_ keys so
# existing private env files keep working untouched.
foreach ($entry in @($values.GetEnumerator())) {
  if ($entry.Key -is [string] -and $entry.Key.StartsWith('INTEGIN_')) {
    $newKey = 'INTEGIN_' + $entry.Key.Substring(6)
    if ([string]::IsNullOrWhiteSpace([string]$values[$newKey])) {
      $values[$newKey] = $entry.Value
    }
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

  Invoke-BoundedProcess 'SEED' '& go run .\cmd\integin-live-matrix seed' $sourceRoot 90

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

  $launcherMessage = "& '$pilotLauncher'"
  Invoke-BoundedProcess 'PILOT_START' $launcherMessage $root 120
  Invoke-BoundedProcess 'EXERCISE' '& go run .\cmd\integin-live-matrix exercise' $sourceRoot 90

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

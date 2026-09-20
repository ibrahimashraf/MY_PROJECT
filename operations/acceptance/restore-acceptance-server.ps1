$runtimeRoot = 'C:\MY_PROJECT\operations\acceptance'
$environmentPath = 'C:\MY_PROJECT\private\integin-secrets\integin-server.env'
$binaryPath = Join-Path $runtimeRoot 'integin-server-provision.exe'
$stdoutPath = Join-Path $runtimeRoot 'integin-server-provision.stdout.log'
$stderrPath = Join-Path $runtimeRoot 'integin-server-provision.stderr.log'
$pidPath = Join-Path $runtimeRoot 'integin-server-provision.pid'

if (-not (Test-Path -LiteralPath $binaryPath)) { throw "Known-good acceptance binary is missing: $binaryPath" }
if (-not (Test-Path -LiteralPath $environmentPath)) { throw "Acceptance private environment is missing: $environmentPath" }
if (Get-Process -Name 'integin-server-provision' -ErrorAction SilentlyContinue) {
  throw 'Acceptance server process already exists. Refusing to start a duplicate.'
}

$values = @{}
Get-Content -LiteralPath $environmentPath | ForEach-Object {
  if ($_ -match '^([^#=][^=]*)=(.*)$') { $values[$matches[1]] = $matches[2] }
}

# INTEGIN_ rename: accept the pre-rename INTEGIN_ key from existing private env
# files (which stay untouched).
$bindAddr = $values['INTEGIN_HTTP_ADDR']
if ([string]::IsNullOrWhiteSpace($bindAddr)) { $bindAddr = $values['INTEGIN_HTTP_ADDR'] }
if ($bindAddr -ne '127.0.0.1:8080') {
  throw 'Acceptance environment must explicitly bind INTEGIN_HTTP_ADDR=127.0.0.1:8080.'
}

Remove-Item -LiteralPath $stdoutPath, $stderrPath, $pidPath -Force -ErrorAction SilentlyContinue
$savedEnvironment = @{}
foreach ($entry in $values.GetEnumerator()) {
  $savedEnvironment[$entry.Key] = [Environment]::GetEnvironmentVariable($entry.Key, 'Process')
  [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
}

try {
  $process = Start-Process -FilePath $binaryPath -WorkingDirectory $runtimeRoot -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -PassThru
} finally {
  foreach ($entry in $savedEnvironment.GetEnumerator()) {
    [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
  }
}

$process.Id | Set-Content -LiteralPath $pidPath -Encoding ascii
Start-Sleep -Seconds 3

try {
  $health = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 http://127.0.0.1:8080/healthz
  $ready = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 http://127.0.0.1:8080/readyz
  if ($health.StatusCode -ne 200 -or $ready.StatusCode -ne 200) { throw 'Acceptance health or readiness returned a non-200 response.' }
} catch {
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  throw "Acceptance server did not restore safely. Review $stdoutPath and $stderrPath."
}

Write-Output 'ACCEPTANCE_SERVER_HEALTH_AND_READINESS_RESTORED'

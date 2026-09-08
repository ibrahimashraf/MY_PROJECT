$sourceRoot = 'C:\MY_PROJECT\integin-pilot-source'
$secretsPath = 'C:\MY_PROJECT\private\integin-secrets\integin-pilot.env'
$runtimeRoot = 'C:\MY_PROJECT\operations\pilot\runtime'
$binaryPath = Join-Path $runtimeRoot 'integin-server-pilot.exe'
$stdoutPath = Join-Path $runtimeRoot 'server.stdout.log'
$stderrPath = Join-Path $runtimeRoot 'server.stderr.log'
$pidPath = Join-Path $runtimeRoot 'server.pid'

if (-not (Test-Path -LiteralPath $sourceRoot)) { throw "Pilot source copy is missing: $sourceRoot" }
if (-not (Test-Path -LiteralPath $secretsPath)) { throw "Pilot secrets file is missing: $secretsPath" }

$values = @{}
Get-Content -LiteralPath $secretsPath | ForEach-Object {
  if ($_ -match '^([^#=][^=]*)=(.*)$') { $values[$matches[1]] = $matches[2] }
}

foreach ($required in @('INTEGIN_TENANT_ID', 'INTEGIN_DB_URL', 'INTEGIN_S3_ENDPOINT', 'INTEGIN_S3_BUCKET', 'INTEGIN_S3_ACCESS_KEY', 'INTEGIN_S3_SECRET_KEY', 'INTEGIN_S3_REGION', 'PILOT_SYNC_SECRET')) {
  if ([string]::IsNullOrWhiteSpace($values[$required])) { throw "Required pilot setting is missing: $required" }
}

$existing = Get-Process -Name 'integin-server-pilot' -ErrorAction SilentlyContinue
if ($existing) { throw 'Pilot server process already exists. Refusing to start a second instance.' }

New-Item -ItemType Directory -Force -Path $runtimeRoot | Out-Null
Remove-Item -LiteralPath $stdoutPath, $stderrPath, $pidPath -Force -ErrorAction SilentlyContinue

Push-Location $sourceRoot
try {
  & go build -o $binaryPath .\cmd\integin-server
  if ($LASTEXITCODE -ne 0) { throw 'Pilot Go build failed.' }
} finally {
  Pop-Location
}

$processEnvironment = @{
  INTEGIN_HTTP_ADDR = '127.0.0.1:18080'
  INTEGIN_TENANT_ID = $values['INTEGIN_TENANT_ID']
  INTEGIN_DB_URL = $values['INTEGIN_DB_URL']
  INTEGIN_EVIDENCE_STORE = 'rustfs'
  INTEGIN_S3_ENDPOINT = $values['INTEGIN_S3_ENDPOINT']
  INTEGIN_S3_BUCKET = $values['INTEGIN_S3_BUCKET']
  INTEGIN_S3_ACCESS_KEY = $values['INTEGIN_S3_ACCESS_KEY']
  INTEGIN_S3_SECRET_KEY = $values['INTEGIN_S3_SECRET_KEY']
  INTEGIN_S3_REGION = $values['INTEGIN_S3_REGION']
  INTEGIN_SYNC_SECRET = $values['PILOT_SYNC_SECRET']
  INTEGIN_LOCAL_PROVISIONING_ENABLED = 'false'
}

$savedEnvironment = @{}
foreach ($entry in $processEnvironment.GetEnumerator()) {
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
  $health = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 http://127.0.0.1:18080/healthz
  $ready = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 http://127.0.0.1:18080/readyz
  if ($health.StatusCode -ne 200 -or $ready.StatusCode -ne 200) { throw 'Pilot health or readiness returned a non-200 response.' }
} catch {
  Stop-Process -Id $process.Id -Force -ErrorAction SilentlyContinue
  throw "Pilot server failed health/readiness validation. Review $stdoutPath and $stderrPath."
}

Write-Output 'PILOT_GO_SERVER_HEALTH_AND_READINESS_VERIFIED'

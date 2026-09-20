#[INTEGIN TSA Pilot] Start the pilot-only RFC 3161 timestamp responder on
# 127.0.0.1:18280. The signer keypair must already exist (integin-cli gen-tsa
# writes it to the private secrets directory); this script never generates
# keys, only starts the responder. Restart-safe: replaces any existing
# pilot-tsa process.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$sourceRoot = 'C:\MY_PROJECT\integin-pilot-source'
$runtimeRoot = 'C:\MY_PROJECT\operations\pilot\runtime'
$binaryPath = Join-Path $runtimeRoot 'pilot-tsa.exe'
$stdoutPath = Join-Path $runtimeRoot 'tsa.stdout.log'
$stderrPath = Join-Path $runtimeRoot 'tsa.stderr.log'
$pidPath = Join-Path $runtimeRoot 'tsa.pid'
$keyPath = 'C:\MY_PROJECT\private\integin-secrets\tsa-pilot-signer.key'
$rootsPath = 'C:\MY_PROJECT\private\integin-secrets\tsa-pilot-roots.pem'
$addr = '127.0.0.1:18280'

foreach ($path in @($keyPath, $rootsPath)) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Pilot TSA keypair is missing: $path (run: go run ./cmd/integin-cli gen-tsa from integin-pilot-source)" }
}

Get-Process -Name 'pilot-tsa' -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 1

New-Item -ItemType Directory -Force -Path $runtimeRoot | Out-Null
Remove-Item -LiteralPath $stdoutPath, $stderrPath, $pidPath -Force -ErrorAction SilentlyContinue

Push-Location $sourceRoot
try {
  & go build -o $binaryPath .\cmd\pilot-tsa
  if ($LASTEXITCODE -ne 0) { throw 'Pilot TSA build failed.' }
} finally {
  Pop-Location
}

# The responder provisions from the process environment, so export first:
# the child inherits these on Windows at creation time.
[Environment]::SetEnvironmentVariable('INTEGIN_TSA_ADDR', $addr, 'Process')
[Environment]::SetEnvironmentVariable('INTEGIN_TSA_SIGNER_KEY_FILE', $keyPath, 'Process')
[Environment]::SetEnvironmentVariable('INTEGIN_TSA_ROOTS_FILE', $rootsPath, 'Process')

$process = Start-Process -FilePath $binaryPath -WorkingDirectory $runtimeRoot -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -PassThru
$process.Id | Set-Content -LiteralPath $pidPath -Encoding ascii

$deadline = [DateTime]::UtcNow.AddSeconds(30)
while ([DateTime]::UtcNow -lt $deadline) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri "http://${addr}/healthz" -ErrorAction Stop
    if ($response.StatusCode -eq 200) {
      Write-Output 'TSA_PILOT_RESPONDER_HEALTHY'
      exit 0
    }
  } catch { }
  if ($process.HasExited) { throw "Pilot TSA exited early (code $($process.ExitCode)). Review $stderrPath." }
  Start-Sleep -Seconds 1
}
throw "Pilot TSA did not answer /healthz on $addr. Review $stderrPath."

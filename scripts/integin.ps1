[CmdletBinding()]
param(
  [ValidateSet('help', 'go-test', 'go-vet', 'contract-generate', 'flutter-contract', 'pilot-health', 'acceptance-health', 'python-test', 'verify-core')]
  [string]$Task = 'help'
)

$repositoryRoot = Split-Path -Parent $PSScriptRoot
$flutterRunner = Join-Path $repositoryRoot 'field_app\run-contract-vector-parity.ps1'

function Invoke-Checked {
  param([scriptblock]$Action, [string]$Name)
  & $Action
  if ($LASTEXITCODE -and $LASTEXITCODE -ne 0) { throw "$Name failed with exit code $LASTEXITCODE." }
}

function Invoke-Health {
  param([string]$BaseUrl, [string]$Name)
  foreach ($path in @('healthz', 'readyz')) {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 "$BaseUrl/$path"
    if ($response.StatusCode -ne 200) { throw "$Name $path returned HTTP $($response.StatusCode)." }
    Write-Output "$Name $path -> $($response.StatusCode) $($response.Content)"
  }
}

function Invoke-GoTest {
  Push-Location $repositoryRoot
  try { Invoke-Checked { go test ./... } 'Go test' } finally { Pop-Location }
}

function Invoke-GoVet {
  Push-Location $repositoryRoot
  try { Invoke-Checked { go vet ./... } 'Go vet' } finally { Pop-Location }
}

function Invoke-ContractGeneration {
  Push-Location $repositoryRoot
  try { Invoke-Checked { go run .\cmd\contract-vector-fixture } 'Contract vector generation' } finally { Pop-Location }
}

function Invoke-FlutterContract {
  if (-not (Test-Path -LiteralPath $flutterRunner)) { throw "Missing approved Flutter contract runner: $flutterRunner" }
  & powershell -ExecutionPolicy Bypass -File $flutterRunner
  if ($LASTEXITCODE -ne 0) { throw "Flutter contract parity failed with exit code $LASTEXITCODE." }
}

switch ($Task) {
  'help' {
    @'
INTEGIN safe pilot tasks
  go-test            Run all Go tests.
  go-vet             Run Go vet.
  contract-generate  Regenerate deterministic test-only signed vectors.
  flutter-contract   Run direct-Dart Flutter contract parity test.
  pilot-health       Read pilot health/readiness at 127.0.0.1:18080.
  acceptance-health  Read acceptance health/readiness at 127.0.0.1:8080.
  python-test        Run python -m pytest ai_service -q if pytest is installed.
  verify-core        Run Go tests/vet, contract generation, Flutter parity, and both health checks.

This runner never starts, stops, migrates, initializes, unseals, or reconfigures any service.
'@ | Write-Output
  }
  'go-test' { Invoke-GoTest }
  'go-vet' { Invoke-GoVet }
  'contract-generate' { Invoke-ContractGeneration }
  'flutter-contract' { Invoke-FlutterContract }
  'pilot-health' { Invoke-Health 'http://127.0.0.1:18080' 'pilot' }
  'acceptance-health' { Invoke-Health 'http://127.0.0.1:8080' 'acceptance' }
  'python-test' {
    Push-Location $repositoryRoot
    try { Invoke-Checked { python -m pytest ai_service -q } 'Python advisory tests' } finally { Pop-Location }
  }
  'verify-core' {
    Invoke-GoTest
    Invoke-GoVet
    Invoke-ContractGeneration
    Invoke-FlutterContract
    Invoke-Health 'http://127.0.0.1:18080' 'pilot'
    Invoke-Health 'http://127.0.0.1:8080' 'acceptance'
    Write-Output 'INTEGIN_PILOT_CORE_VERIFICATION_PASSED'
  }
}

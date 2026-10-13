# Stable verification for the field app test suite.
#
# Problem: on constrained machines the Dart test isolate occasionally dies at
# spawn time ("Failed to load ... Connection closed before test suite loaded")
# for a random file. That is infrastructure flake, not a test failure — but
# `flutter test` exits 1 either way.
#
# Policy (deliberately narrow):
# - Files that fail ONLY with "Failed to load" are retried once, standalone.
# - Any assertion failure, timeout ("did not complete"), or compile error
#   fails the run immediately with no retry. Retries never mask real failures.
# - Both phases are printed; exit 0 iff everything passes.

$ErrorActionPreference = 'Stop'

function Invoke-FlutterTest {
  param([string[]]$Targets)
  $logFile = [System.IO.Path]::GetTempFileName()
  if ($Targets.Count -eq 0) {
    flutter test --reporter compact 2>&1 | Tee-Object -FilePath $logFile
  } else {
    flutter test @Targets --reporter compact 2>&1 | Tee-Object -FilePath $logFile
  }
  return @{ ExitCode = $LASTEXITCODE; Log = (Get-Content -Raw $logFile) }
}

function Get-LoadFailures {
  param([string]$Log)
  $files = @()
  foreach ($m in [regex]::Matches($Log, 'Failed to load "([^"]+)"')) {
    $files += $m.Groups[1].Value
  }
  return ($files | Select-Object -Unique)
}

function Has-RealFailures {
  param([string]$Log)
  foreach ($line in $Log -split "`r?`n") {
    if ($line -match '\[E\]' -and $line -notmatch 'loading ' -and $line -notmatch 'Some tests failed') {
      return $true
    }
  }
  return $false
}

Write-Host '=== Phase 1: full suite ==='
$phase1 = Invoke-FlutterTest @()
if ($phase1.ExitCode -eq 0) {
  Write-Host '=== PASS: full suite green on first attempt ==='
  exit 0
}

$loadFailures = Get-LoadFailures $phase1.Log
$realFailures = Has-RealFailures $phase1.Log
if ($realFailures) {
  Write-Host '=== FAIL: assertion/timeout failures present — no retry ==='
  exit 1
}
if ($loadFailures.Count -eq 0) {
  Write-Host '=== FAIL: suite failed without identifiable spawn flakes ==='
  exit 1
}

Write-Host "=== Phase 2: retrying $($loadFailures.Count) spawn-flaked file(s) ==="
$phase2 = Invoke-FlutterTest $loadFailures
if ($phase2.ExitCode -eq 0) {
  Write-Host '=== PASS: flakes cleared on retry; no assertion failures ==='
  exit 0
}
Write-Host '=== FAIL: retry did not clear the failures ==='
exit 1

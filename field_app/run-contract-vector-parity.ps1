$projectRoot = 'C:\MY_PROJECT\integin-pilot-source\field_app'
$testPath = 'test\contract_vector_parity_test.dart'
$dartPath = 'C:\flutter\bin\cache\dart-sdk\bin\dart.exe'
$toolsSnapshot = 'C:\flutter\bin\cache\flutter_tools.snapshot'

foreach ($path in @($projectRoot, $dartPath, $toolsSnapshot, (Join-Path $projectRoot $testPath))) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Required pilot Flutter test path is missing: $path" }
}

$previousProgramFilesX86 = [Environment]::GetEnvironmentVariable('PROGRAMFILES(X86)', 'Process')
[Environment]::SetEnvironmentVariable('PROGRAMFILES(X86)', 'C:\Program Files (x86)', 'Process')
Push-Location $projectRoot
try {
  Write-Output 'RUNNING_PILOT_FLUTTER_CONTRACT_VECTOR_TEST'
  & $dartPath $toolsSnapshot test $testPath
  $exitCode = $LASTEXITCODE
} finally {
  Pop-Location
  [Environment]::SetEnvironmentVariable('PROGRAMFILES(X86)', $previousProgramFilesX86, 'Process')
}

if ($exitCode -ne 0) { throw "Pilot Flutter contract-vector test failed with exit code $exitCode." }
Write-Output 'PILOT_FLUTTER_CONTRACT_VECTOR_TEST_PASSED'

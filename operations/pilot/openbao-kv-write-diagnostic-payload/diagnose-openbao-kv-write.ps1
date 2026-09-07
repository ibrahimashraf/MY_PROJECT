$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$containerName = 'integin-pilot-openbao'
$recoveryPath = 'C:\MY PROJECT\private\integin-secrets\openbao-pilot-recovery.json'
$baoAddress = 'http://127.0.0.1:8200'
$secretMount = 'secret'
$secretPath = 'integin-pilot/rehearsal'
$disposableValue = 'pilot-disposable'

if (-not (Test-Path -LiteralPath $recoveryPath)) { throw 'Private recovery material is unavailable.' }
$recovery = Get-Content -LiteralPath $recoveryPath -Raw | ConvertFrom-Json
$rootToken = [string]$recovery.root_token
if ([string]::IsNullOrWhiteSpace($rootToken)) { throw 'Private recovery material is incomplete.' }

$output = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv put "-mount=$secretMount" $secretPath "value=$disposableValue" 2>&1
if ($LASTEXITCODE -eq 0) {
  Write-Output 'OPENBAO_DISPOSABLE_KV_WRITE_PASSED'
} else {
  $text = ($output -join "`n")
  if ($text -match 'permission denied') {
    Write-Output 'OPENBAO_DISPOSABLE_KV_WRITE_DENIED'
  } elseif ($text -match 'sealed') {
    Write-Output 'OPENBAO_DISPOSABLE_KV_WRITE_SEALED'
  } elseif ($text -match 'connection refused|connection reset|EOF') {
    Write-Output 'OPENBAO_DISPOSABLE_KV_WRITE_TRANSPORT_FAILURE'
  } else {
    Write-Output 'OPENBAO_DISPOSABLE_KV_WRITE_OTHER_FAILURE'
  }
}

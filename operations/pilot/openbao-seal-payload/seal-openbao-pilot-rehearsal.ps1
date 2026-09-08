$ErrorActionPreference = 'Stop'
$containerName = 'integin-pilot-openbao'
$recoveryPath = 'C:\MY_PROJECT\private\integin-secrets\openbao-pilot-recovery.json'
$baoAddress = 'http://127.0.0.1:8200'

if (-not (Test-Path -LiteralPath $recoveryPath)) { throw 'Private recovery material is unavailable.' }
$recovery = Get-Content -LiteralPath $recoveryPath -Raw | ConvertFrom-Json
$rootToken = [string]$recovery.root_token
if ([string]::IsNullOrWhiteSpace($rootToken)) { throw 'Private recovery material is incomplete.' }

$ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao operator seal 2>&1
if ($LASTEXITCODE -ne 0) { throw 'The isolated OpenBao service could not be sealed.' }

$statusOutput = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao status -format=json 2>&1
if ($LASTEXITCODE -ne 2) { throw 'OpenBao did not return the expected sealed-status exit code.' }
$status = (($statusOutput -join "`n") | ConvertFrom-Json)
if (-not [bool]$status.sealed -or [int]$status.progress -ne 0) { throw 'OpenBao did not return to the sealed baseline.' }

Write-Output 'OPENBAO_REHEARSAL_SEALED_BASELINE_RESTORED'

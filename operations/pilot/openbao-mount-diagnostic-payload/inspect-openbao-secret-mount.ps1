$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$containerName = 'integin-pilot-openbao'
$recoveryPath = 'C:\MY_PROJECT\private\integin-secrets\openbao-pilot-recovery.json'
$baoAddress = 'http://127.0.0.1:8200'

if (-not (Test-Path -LiteralPath $recoveryPath)) { throw 'Private recovery material is unavailable.' }
$recovery = Get-Content -LiteralPath $recoveryPath -Raw | ConvertFrom-Json
$rootToken = [string]$recovery.root_token
if ([string]::IsNullOrWhiteSpace($rootToken)) { throw 'Private recovery material is incomplete.' }

$output = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao secrets list -format=json 2>&1
if ($LASTEXITCODE -ne 0) { throw 'The read-only OpenBao mount diagnostic failed.' }
$mounts = (($output -join "`n") | ConvertFrom-Json)
if ($mounts.PSObject.Properties.Name -contains 'secret/') {
  $mountType = [string]$mounts.'secret/'.type
  Write-Output "OPENBAO_SECRET_MOUNT_PRESENT_TYPE_$mountType"
} else {
  Write-Output 'OPENBAO_SECRET_MOUNT_ABSENT'
}

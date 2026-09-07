$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false
$containerName = 'integin-pilot-openbao'
$recoveryPath = 'C:\MY PROJECT\private\integin-secrets\openbao-pilot-recovery.json'
$baoAddress = 'http://127.0.0.1:8200'

if (-not (Test-Path -LiteralPath $recoveryPath)) { throw 'Private recovery material is unavailable.' }
$recovery = Get-Content -LiteralPath $recoveryPath -Raw | ConvertFrom-Json
$rootToken = [string]$recovery.root_token
if ([string]::IsNullOrWhiteSpace($rootToken)) { throw 'Private recovery material is incomplete.' }

$output = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao secrets list -detailed -format=json 2>&1
if ($LASTEXITCODE -ne 0) { throw 'The non-mutating OpenBao engine-options diagnostic failed.' }
$mounts = (($output -join "`n") | ConvertFrom-Json)
if (-not ($mounts.PSObject.Properties.Name -contains 'secret/')) { throw 'The expected secret mount is absent.' }
$mount = $mounts.'secret/'
$version = [string]$mount.options.version
if ([string]::IsNullOrWhiteSpace($version)) { $version = '1' }
Write-Output "OPENBAO_SECRET_MOUNT_TYPE_$($mount.type)_VERSION_$version"

$probeOutput = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv get -mount=secret integin-pilot/nonexistent -format=json 2>&1
if ($LASTEXITCODE -eq 0) {
  Write-Output 'OPENBAO_NONMUTATING_ROOT_READ_RESPONSE_SUCCESS'
} else {
  Write-Output 'OPENBAO_NONMUTATING_ROOT_READ_RESPONSE_EXPECTED_NOT_FOUND'
}

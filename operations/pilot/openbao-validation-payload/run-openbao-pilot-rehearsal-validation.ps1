$ErrorActionPreference = 'Stop'
$PSNativeCommandUseErrorActionPreference = $false

$containerName = 'integin-pilot-openbao'
$recoveryPath = 'C:\MY_PROJECT\private\integin-secrets\openbao-pilot-recovery.json'
$policyName = 'integin-pilot-read-rehearsal'
$policyPath = 'C:\MY_PROJECT\operations\pilot\openbao\integin-pilot-read-rehearsal.hcl'
$secretMount = 'secret'
$secretPath = 'integin-pilot/rehearsal'
$disposableValue = 'pilot-disposable'
$baoAddress = 'http://127.0.0.1:8200'

function Invoke-BaoJson {
  param(
    [string[]]$Arguments,
    [string]$Token = ''
  )

  $dockerArguments = @('exec', $containerName, 'env', "BAO_ADDR=$baoAddress")
  if ($Token) { $dockerArguments += "BAO_TOKEN=$Token" }
  $dockerArguments += 'bao'
  $dockerArguments += $Arguments
  $output = & docker @dockerArguments 2>&1
  if ($LASTEXITCODE -ne 0 -and $LASTEXITCODE -ne 2) { throw 'A private OpenBao command failed.' }
  return (($output -join "`n") | ConvertFrom-Json)
}

function Assert-SealedState {
  param([bool]$Expected)
  $status = Invoke-BaoJson -Arguments @('status', '-format=json')
  if ([bool]$status.sealed -ne $Expected) { throw 'OpenBao seal state did not match the expected rehearsal state.' }
}

function Assert-CommandDenied {
  param(
    [string[]]$Arguments,
    [string]$Token
  )

  $dockerArguments = @('exec', $containerName, 'env', "BAO_ADDR=$baoAddress", "BAO_TOKEN=$Token", 'bao')
  $dockerArguments += $Arguments
  $previousErrorActionPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    $ignored = & docker @dockerArguments 2>$null
    $exitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
  if ($exitCode -eq 0) { throw 'A least-privilege denial check unexpectedly succeeded.' }
}

function Assert-RuntimeHealth {
  foreach ($endpoint in @(
    'http://127.0.0.1:8080/healthz',
    'http://127.0.0.1:8080/readyz',
    'http://127.0.0.1:18080/healthz',
    'http://127.0.0.1:18080/readyz'
  )) {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 $endpoint
    if ($response.StatusCode -ne 200) { throw 'A protected INTEGIN health endpoint was not healthy during the OpenBao rehearsal.' }
  }
}

if (-not (Test-Path -LiteralPath $recoveryPath)) { throw 'The private OpenBao recovery file is not present.' }
if (-not (docker ps --format '{{.Names}}' | Select-String -SimpleMatch -Quiet $containerName)) { throw 'The isolated OpenBao container is not running.' }

Assert-RuntimeHealth

$recovery = Get-Content -LiteralPath $recoveryPath -Raw | ConvertFrom-Json
$shares = @($recovery.unseal_keys_b64)
$rootToken = [string]$recovery.root_token
if ($shares.Count -ne 3 -or [string]::IsNullOrWhiteSpace($rootToken)) { throw 'The private recovery material does not meet the expected Shamir rehearsal shape.' }

try {
  Assert-SealedState -Expected $true
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[0] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The first private Shamir share submission failed.' }
  Assert-SealedState -Expected $true
  Write-Output 'OPENBAO_ONE_SHARE_REMAINS_SEALED'

  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[1] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The second private Shamir share submission failed.' }
  Assert-SealedState -Expected $false
  Start-Sleep -Seconds 2
  Write-Output 'OPENBAO_TWO_SHARE_UNSEAL_PASSED'

  $mounts = Invoke-BaoJson -Arguments @('secrets', 'list', '-format=json') -Token $rootToken
  if (-not ($mounts.PSObject.Properties.Name -contains "$secretMount/")) {
    $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao secrets enable "-path=$secretMount" kv-v2 2>&1
    if ($LASTEXITCODE -ne 0) { throw 'The disposable KV v2 secret engine could not be enabled.' }
  }

  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv put "-mount=$secretMount" $secretPath "value=$disposableValue" 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The disposable rehearsal secret could not be written.' }

  $policy = @'
path "secret/data/integin-pilot/rehearsal" {
  capabilities = ["read"]
}
'@
  [System.IO.File]::WriteAllText($policyPath, $policy, [System.Text.UTF8Encoding]::new($false))
  & docker cp $policyPath "${containerName}:/tmp/integin-pilot-read-rehearsal.hcl" | Out-Null
  if ($LASTEXITCODE -ne 0) { throw 'The disposable least-privilege policy file could not be placed in the isolated container.' }
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao policy write $policyName /tmp/integin-pilot-read-rehearsal.hcl 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The disposable least-privilege policy could not be written.' }

  $tokenJson = Invoke-BaoJson -Arguments @('token', 'create', "-policy=$policyName", '-ttl=3m', '-format=json') -Token $rootToken
  $testToken = [string]$tokenJson.auth.client_token
  $testAccessor = [string]$tokenJson.auth.accessor
  if ([string]::IsNullOrWhiteSpace($testToken) -or [string]::IsNullOrWhiteSpace($testAccessor)) { throw 'The disposable least-privilege token was not created.' }

  $readValue = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$testToken" bao kv get "-mount=$secretMount" -field=value $secretPath 2>&1
  if ($LASTEXITCODE -ne 0 -or (($readValue -join '') -ne $disposableValue)) { throw 'The least-privilege token could not read its permitted path.' }
  Assert-CommandDenied -Arguments @('kv', 'put', "-mount=$secretMount", $secretPath, 'value=not-permitted') -Token $testToken
  Assert-CommandDenied -Arguments @('kv', 'get', "-mount=$secretMount", 'integin-pilot/other') -Token $testToken
  Assert-CommandDenied -Arguments @('kv', 'list', "-mount=$secretMount", 'integin-pilot') -Token $testToken
  Assert-CommandDenied -Arguments @('read', 'sys/mounts') -Token $testToken
  Write-Output 'OPENBAO_LEAST_PRIVILEGE_PASSED'

  $auditCheck = & docker exec $containerName sh -c 'test -s /openbao/logs/audit.log' 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The declarative audit file was not written after audited rehearsal operations.' }
  $leakCheck = & docker exec $containerName env "PROBE_ROOT=$rootToken" "PROBE_SHARE_ONE=$($shares[0])" "PROBE_SHARE_TWO=$($shares[1])" "PROBE_VALUE=$disposableValue" sh -c 'for v in "$PROBE_ROOT" "$PROBE_SHARE_ONE" "$PROBE_SHARE_TWO" "$PROBE_VALUE"; do if grep -Fq "$v" /openbao/logs/audit.log; then exit 1; fi; done' 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The audit file contained a prohibited secret or recovery value.' }
  Write-Output 'OPENBAO_AUDIT_NON_SECRET_RECORDING_PASSED'

  & docker restart $containerName | Out-Null
  Start-Sleep -Seconds 2
  Assert-SealedState -Expected $true
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[0] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The first restart-recovery share submission failed.' }
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[1] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The second restart-recovery share submission failed.' }
  Assert-SealedState -Expected $false
  Start-Sleep -Seconds 2
  $persistedValue = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv get "-mount=$secretMount" -field=value $secretPath 2>&1
  if ($LASTEXITCODE -ne 0 -or (($persistedValue -join '') -ne $disposableValue)) { throw 'The disposable rehearsal secret did not survive sealed restart and recovery.' }
  Write-Output 'OPENBAO_RESTART_PERSISTENCE_PASSED'

  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao operator seal 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The deliberate rehearsal seal action failed.' }
  Assert-SealedState -Expected $true
  $previousErrorActionPreference = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    $sealedRead = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv get "-mount=$secretMount" -field=value $secretPath 2>$null
    $sealedReadExitCode = $LASTEXITCODE
  } finally {
    $ErrorActionPreference = $previousErrorActionPreference
  }
  if ($sealedReadExitCode -eq 0) { throw 'A secret read unexpectedly succeeded while OpenBao was sealed.' }
  Assert-RuntimeHealth
  Write-Output 'OPENBAO_SEALED_OUTAGE_INTEGIN_HEALTH_PASSED'

  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[0] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The first cleanup unseal share submission failed.' }
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" bao operator unseal $shares[1] 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The second cleanup unseal share submission failed.' }
  Assert-SealedState -Expected $false

  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao token revoke -accessor $testAccessor 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The disposable least-privilege token could not be revoked.' }
  Assert-CommandDenied -Arguments @('kv', 'get', "-mount=$secretMount", $secretPath) -Token $testToken
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao kv delete "-mount=$secretMount" $secretPath 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The disposable rehearsal secret could not be deleted.' }
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao policy delete $policyName 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The disposable least-privilege policy could not be deleted.' }
  $ignored = & docker exec $containerName env "BAO_ADDR=$baoAddress" "BAO_TOKEN=$rootToken" bao operator seal 2>&1
  if ($LASTEXITCODE -ne 0) { throw 'The final sealed state could not be restored.' }
  Assert-SealedState -Expected $true
  Assert-RuntimeHealth
  Write-Output 'OPENBAO_REHEARSAL_CLEANUP_AND_FINAL_SEAL_PASSED'
} finally {
  if (Test-Path -LiteralPath $policyPath) { Remove-Item -LiteralPath $policyPath -Force }
  try {
    $ignored = & docker exec --user 0 $containerName rm -f /tmp/integin-pilot-read-rehearsal.hcl 2>&1
  } catch {
    # Temporary container-file removal must not mask the actual rehearsal result.
  }
}

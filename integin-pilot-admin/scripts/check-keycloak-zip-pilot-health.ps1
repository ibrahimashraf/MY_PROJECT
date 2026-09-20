#[INTEGIN Keycloak ZIP Pilot] Verify only local health, discovery, identity PostgreSQL readiness, and protected-runtime continuity.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$checks = @(
  @{ Name='acceptance-health'; Url='http://127.0.0.1:8080/healthz' },
  @{ Name='acceptance-readiness'; Url='http://127.0.0.1:8080/readyz' },
  @{ Name='pilot-health'; Url='http://127.0.0.1:18080/healthz' },
  @{ Name='pilot-readiness'; Url='http://127.0.0.1:18080/readyz' },
  @{ Name='keycloak-readiness'; Url='http://127.0.0.1:19090/health/ready' },
  @{ Name='keycloak-openid-configuration'; Url='http://127.0.0.1:18180/realms/master/.well-known/openid-configuration' }
)

$result = foreach ($check in $checks) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 8 -Uri $check.Url
    [pscustomobject]@{ Check=$check.Name; Status=($response.StatusCode -eq 200); HttpStatus=$response.StatusCode }
  } catch { [pscustomobject]@{ Check=$check.Name; Status=$false; HttpStatus='UNAVAILABLE' } }
}
$result | Format-Table -AutoSize
if (@($result | Where-Object { -not $_.Status }).Count -gt 0) { throw 'One or more isolated Keycloak ZIP pilot health checks failed.' }

$previous = $ErrorActionPreference
try {
  $ErrorActionPreference = 'Continue'
  & docker exec integin-pilot-keycloak-zip-postgres pg_isready -U INTEGIN_keycloak_owner -d keycloak_pilot 1>$null 2>$null
  $postgresExit = $LASTEXITCODE
  $rawBao = & docker exec integin-pilot-openbao env BAO_ADDR='http://127.0.0.1:8200' bao status -format=json 2>$null
  $baoExit = $LASTEXITCODE
} finally { $ErrorActionPreference = $previous }
if ($postgresExit -ne 0) { throw 'Dedicated Keycloak ZIP pilot PostgreSQL is not ready.' }
if ($baoExit -ne 0 -and $baoExit -ne 2) { throw 'Unable to inspect isolated OpenBao sealed status.' }
if (-not (($rawBao | ConvertFrom-Json).sealed)) { throw 'Isolated OpenBao is not sealed.' }
Write-Output 'KEYCLOAK_ZIP_PILOT_CONCURRENT_HEALTH_VERIFIED'



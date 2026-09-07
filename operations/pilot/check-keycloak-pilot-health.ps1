[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$checks = @(
  @{ Name = 'acceptance-health'; Url = 'http://127.0.0.1:8080/healthz' },
  @{ Name = 'acceptance-readiness'; Url = 'http://127.0.0.1:8080/readyz' },
  @{ Name = 'pilot-health'; Url = 'http://127.0.0.1:18080/healthz' },
  @{ Name = 'pilot-readiness'; Url = 'http://127.0.0.1:18080/readyz' },
  @{ Name = 'keycloak-readiness'; Url = 'http://127.0.0.1:19090/health/ready' },
  @{ Name = 'keycloak-discovery'; Url = 'http://127.0.0.1:18180/realms/master/.well-known/openid-configuration' }
)

$result = foreach ($check in $checks) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 -Uri $check.Url
    [pscustomobject]@{ Check = $check.Name; Status = $response.StatusCode -eq 200; HttpStatus = $response.StatusCode }
  } catch {
    [pscustomobject]@{ Check = $check.Name; Status = $false; HttpStatus = 'UNAVAILABLE' }
  }
}

$result | Format-Table -AutoSize
if (($result | Where-Object { -not $_.Status }).Count -gt 0) {
  throw 'One or more isolated Keycloak pilot health checks failed.'
}

Write-Output 'KEYCLOAK_PILOT_CONCURRENT_HEALTH_VERIFIED'

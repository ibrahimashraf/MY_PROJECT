#[INTEGIN Keycloak ZIP Pilot] Validate local bootstrap Admin REST authentication without displaying credentials, token, or realm content and without changing Keycloak.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$runtimeEnvironmentPath = 'C:\integin-secrets\keycloak-pilot-zip-runtime.env'
$baseUrl = 'http://127.0.0.1:18180'

function Read-EnvironmentFile {
  param([Parameter(Mandatory = $true)][string]$Path)
  $values = @{}
  foreach ($line in Get-Content -LiteralPath $Path) {
    $trimmed = $line.Trim()
    if ($trimmed.Length -eq 0 -or $trimmed.StartsWith('#')) { continue }
    $separator = $trimmed.IndexOf('=')
    if ($separator -lt 1) { throw "Invalid private environment entry in $Path" }
    $values[$trimmed.Substring(0, $separator)] = $trimmed.Substring($separator + 1)
  }
  return $values
}

if (-not (Test-Path -LiteralPath $runtimeEnvironmentPath)) { throw 'Private Keycloak ZIP runtime environment is missing.' }
$runtime = Read-EnvironmentFile -Path $runtimeEnvironmentPath
foreach ($required in @('KC_BOOTSTRAP_ADMIN_USERNAME','KC_BOOTSTRAP_ADMIN_PASSWORD')) {
  if (-not $runtime.ContainsKey($required) -or [string]::IsNullOrWhiteSpace($runtime[$required])) { throw "Required private Keycloak bootstrap key is absent: $required" }
}

$token = $null
$headers = $null
try {
  $tokenResponse = Invoke-RestMethod -Method Post -Uri "$baseUrl/realms/master/protocol/openid-connect/token" -ContentType 'application/x-www-form-urlencoded' -Body @{
    grant_type = 'password'
    client_id = 'admin-cli'
    username = $runtime['KC_BOOTSTRAP_ADMIN_USERNAME']
    password = $runtime['KC_BOOTSTRAP_ADMIN_PASSWORD']
  }
  $token = [string]$tokenResponse.access_token
  if ([string]::IsNullOrWhiteSpace($token)) { throw 'Keycloak did not issue a bootstrap access token.' }
  $headers = @{ Authorization = "Bearer $token" }
  $response = Invoke-WebRequest -UseBasicParsing -Method Get -Uri "$baseUrl/admin/realms/master" -Headers $headers -TimeoutSec 10
  if ($response.StatusCode -ne 200) { throw 'Keycloak Admin REST master-realm read did not return HTTP 200.' }
  Write-Output 'KEYCLOAK_ZIP_ADMIN_REST_BOOTSTRAP_AUTH_AND_MASTER_READ_VERIFIED'
} finally {
  $token = $null
  $headers = $null
}


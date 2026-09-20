#[INTEGIN Keycloak ZIP Pilot] Finalize verified existing synthetic realm/client; no Keycloak mutation and no secret/token output.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$runtimeEnvironmentPath = 'C:\integin-secrets\keycloak-pilot-zip-runtime.env'
$runtimeDirectory = 'C:\INTEGIN-PILOT\runtime\keycloak-zip'
$releaseRecordPath = Join-Path $runtimeDirectory 'integin-pilot-realm-release.json'
$baseUrl = 'http://127.0.0.1:18180'
$managementBaseUrl = 'http://127.0.0.1:19090'
$realmName = 'integin-pilot'
$clientId = 'integin-field-pilot'

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

function Get-HttpStatus {
  param([Parameter(Mandatory = $true)][string]$Uri, [hashtable]$Headers = $null)
  try {
    if ($null -eq $Headers) { $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri $Uri -ErrorAction Stop }
    else { $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri $Uri -Headers $Headers -ErrorAction Stop }
    return [int]$response.StatusCode
  } catch [System.Net.WebException] {
    if ($null -eq $_.Exception.Response) { throw }
    return [int]$_.Exception.Response.StatusCode
  }
}

function Get-FalseWhenOmitted {
  param([Parameter(Mandatory = $true)][object]$Object, [Parameter(Mandatory = $true)][string]$PropertyName)
  $property = $Object.PSObject.Properties[$PropertyName]
  if ($null -eq $property) { return $false }
  return [bool]$property.Value
}

function Invoke-KeycloakGet {
  param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][hashtable]$Headers)
  return Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri "$baseUrl$Path" -Headers $Headers -ErrorAction Stop
}

function Get-NoRedirectResponse {
  param([Parameter(Mandatory = $true)][string]$Uri)
  $request = [System.Net.HttpWebRequest]::Create($Uri)
  $request.AllowAutoRedirect = $false
  $request.Method = 'GET'
  $response = $null
  try { $response = $request.GetResponse() } catch [System.Net.WebException] { $response = $_.Exception.Response }
  if ($null -eq $response) { throw 'Implicit-flow negative request did not return an HTTP response.' }
  $headersProperty = $response.PSObject.Properties['Headers']
  $location = if ($null -eq $headersProperty) { '' } else { [string]$headersProperty.Value['Location'] }
  try { return [pscustomobject]@{ StatusCode = [int]$response.StatusCode; Location = $location } } finally { $response.Close() }
}

if (-not (Test-Path -LiteralPath $runtimeEnvironmentPath)) { throw 'Private Keycloak ZIP runtime environment is missing.' }
if (Test-Path -LiteralPath $releaseRecordPath) { throw 'Refusing to overwrite an existing non-secret Keycloak pilot release record.' }
if ((Get-HttpStatus -Uri "$managementBaseUrl/health/ready") -ne 200) { throw 'Keycloak ZIP readiness preflight failed.' }
if ((Get-HttpStatus -Uri 'http://127.0.0.1:8080/healthz') -ne 200 -or (Get-HttpStatus -Uri 'http://127.0.0.1:18080/healthz') -ne 200) { throw 'Protected INTEGIN runtime preflight failed.' }

$runtime = Read-EnvironmentFile -Path $runtimeEnvironmentPath
foreach ($required in @('KC_BOOTSTRAP_ADMIN_USERNAME','KC_BOOTSTRAP_ADMIN_PASSWORD')) {
  if (-not $runtime.ContainsKey($required) -or [string]::IsNullOrWhiteSpace($runtime[$required])) { throw "Required private Keycloak bootstrap key is absent: $required" }
}

$token = $null
$headers = $null
try {
  $tokenResponse = Invoke-RestMethod -Method Post -Uri "$baseUrl/realms/master/protocol/openid-connect/token" -ContentType 'application/x-www-form-urlencoded' -Body @{
    grant_type = 'password'; client_id = 'admin-cli'; username = $runtime['KC_BOOTSTRAP_ADMIN_USERNAME']; password = $runtime['KC_BOOTSTRAP_ADMIN_PASSWORD']
  }
  $token = [string]$tokenResponse.access_token
  if ([string]::IsNullOrWhiteSpace($token)) { throw 'Keycloak did not issue a bootstrap access token.' }
  $headers = @{ Authorization = "Bearer $token" }

  $realmRead = Invoke-KeycloakGet -Path "/admin/realms/$realmName" -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json
  $clientSearch = Invoke-KeycloakGet -Path "/admin/realms/$realmName/clients?clientId=$clientId" -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json
  $matches = @($clientSearch | Where-Object { $_.clientId -eq $clientId })
  if ($matches.Count -ne 1) { throw 'Synthetic client lookup did not return exactly one client.' }
  $clientRead = Invoke-KeycloakGet -Path ("/admin/realms/{0}/clients/{1}" -f $realmName, $matches[0].id) -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json

  if ($realmRead.realm -ne $realmName -or -not $realmRead.enabled -or $realmRead.sslRequired -ne 'external' -or -not $realmRead.bruteForceProtected -or -not $realmRead.eventsEnabled -or -not $realmRead.adminEventsEnabled -or $realmRead.adminEventsDetailsEnabled) { throw 'Synthetic realm allowlist verification failed.' }
  $pkce = $clientRead.attributes.PSObject.Properties['pkce.code.challenge.method']
  $deviceGrant = $clientRead.attributes.PSObject.Properties['oauth2.device.authorization.grant.enabled']
  if ($clientRead.clientId -ne $clientId -or -not $clientRead.publicClient -or -not $clientRead.standardFlowEnabled -or $clientRead.implicitFlowEnabled -or $clientRead.directAccessGrantsEnabled -or $clientRead.serviceAccountsEnabled -or (Get-FalseWhenOmitted -Object $clientRead -PropertyName 'authorizationServicesEnabled') -or $clientRead.fullScopeAllowed -or @($clientRead.redirectUris).Count -ne 1 -or @($clientRead.redirectUris)[0] -ne 'http://127.0.0.1' -or $null -eq $pkce -or $pkce.Value -ne 'S256' -or $null -eq $deviceGrant -or $deviceGrant.Value -ne 'false') { throw 'Synthetic public-client allowlist verification failed.' }

  $issuer = "$baseUrl/realms/$realmName"
  if ((Get-HttpStatus -Uri "$issuer/.well-known/openid-configuration") -ne 200 -or (Get-HttpStatus -Uri "$issuer/protocol/openid-connect/certs") -ne 200) { throw 'Synthetic realm discovery/JWKS verification failed.' }
  $directStatus = $null
  try {
    $directResponse = Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$issuer/protocol/openid-connect/token" -ContentType 'application/x-www-form-urlencoded' -Body @{ grant_type = 'password'; client_id = $clientId; username = 'synthetic-no-user'; password = 'synthetic-no-password' } -ErrorAction Stop
    if ($null -ne $directResponse) { throw 'Direct/password grant unexpectedly returned success.' }
  } catch [System.Net.WebException] {
    $directStatus = [int]$_.Exception.Response.StatusCode
  }
  if ($directStatus -ne 400) { throw 'Direct/password grant rejection returned an unexpected status.' }
  $implicitResponse = Get-NoRedirectResponse -Uri "$issuer/protocol/openid-connect/auth?client_id=$clientId&redirect_uri=http%3A%2F%2F127.0.0.1&response_type=token&scope=openid&nonce=synthetic"
  $implicitLocation = $implicitResponse.Location
  if (([int]$implicitResponse.StatusCode -ne 302) -or [string]::IsNullOrWhiteSpace($implicitLocation) -or $implicitLocation -notmatch 'error=(unsupported_response_type|unauthorized_client)') { throw 'Implicit-flow rejection verification failed.' }

  $releaseRecord = [ordered]@{
    realm = $realmName; client_id = $clientId; issuer = $issuer; jwks_uri = "$issuer/protocol/openid-connect/certs"; finalized_at_utc = [DateTime]::UtcNow.ToString('o')
    realm_policy = [ordered]@{ ssl_required = 'external'; brute_force_protected = $true; events_enabled = $true; admin_events_enabled = $true; admin_event_details_enabled = $false }
    client_policy = [ordered]@{ public_client = $true; standard_flow = $true; pkce_method = 'S256'; redirect_uri = 'http://127.0.0.1'; direct_access_grant = $false; implicit_flow = $false; service_accounts = $false; authorization_services = $false; device_grant = $false }
    negative_policy = [ordered]@{ direct_password_grant_rejected = $true; implicit_flow_rejected = $true }
    authorization_boundary = 'Keycloak identity-only; INTEGIN Go/PostgreSQL remains authoritative for tenant, organization, device, workflow, capability, and RLS.'
  }
  [System.IO.File]::WriteAllText($releaseRecordPath, ($releaseRecord | ConvertTo-Json -Depth 8), (New-Object System.Text.UTF8Encoding($false)))
  Write-Output 'KEYCLOAK_ZIP_SYNTHETIC_REALM_AND_PUBLIC_PKCE_CLIENT_FINALIZED'
} finally {
  $token = $null
  $headers = $null
}

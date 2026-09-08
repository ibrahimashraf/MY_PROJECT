#[INTEGIN Keycloak ZIP Pilot] Create exactly one synthetic realm and one public PKCE-S256 client; never grant INTEGIN authorization or emit secrets.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$runtimeEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\keycloak-pilot-zip-runtime.env'
$runtimeDirectory = 'C:\MY_PROJECT\operations\pilot\runtime\keycloak-zip'
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
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri $Uri -Headers $Headers -ErrorAction Stop
    return [int]$response.StatusCode
  } catch [System.Net.WebException] {
    if ($null -eq $_.Exception.Response) { throw }
    return [int]$_.Exception.Response.StatusCode
  }
}

function Invoke-KeycloakGet {
  param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][hashtable]$Headers)
  return Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri "$baseUrl$Path" -Headers $Headers -ErrorAction Stop
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
    grant_type = 'password'
    client_id = 'admin-cli'
    username = $runtime['KC_BOOTSTRAP_ADMIN_USERNAME']
    password = $runtime['KC_BOOTSTRAP_ADMIN_PASSWORD']
  }
  $token = [string]$tokenResponse.access_token
  if ([string]::IsNullOrWhiteSpace($token)) { throw 'Keycloak did not issue a bootstrap access token.' }
  $headers = @{ Authorization = "Bearer $token" }

  $existingRealmStatus = Get-HttpStatus -Uri "$baseUrl/admin/realms/$realmName" -Headers $headers
  if ($existingRealmStatus -eq 200) { throw 'Refusing to mutate a pre-existing synthetic Keycloak pilot realm.' }
  if ($existingRealmStatus -ne 404) { throw 'Keycloak realm-absence preflight returned an unexpected status.' }

  $realmRepresentation = [ordered]@{
    realm = $realmName
    displayName = 'INTEGIN Synthetic Pilot'
    enabled = $true
    sslRequired = 'external'
    registrationAllowed = $false
    resetPasswordAllowed = $false
    rememberMe = $false
    loginWithEmailAllowed = $false
    duplicateEmailsAllowed = $false
    verifyEmail = $false
    bruteForceProtected = $true
    eventsEnabled = $true
    adminEventsEnabled = $true
    adminEventsDetailsEnabled = $false
  }
  $realmBody = $realmRepresentation | ConvertTo-Json -Depth 6
  $realmCreate = Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$baseUrl/admin/realms" -Headers $headers -ContentType 'application/json' -Body $realmBody -ErrorAction Stop
  if (@(201,204) -notcontains [int]$realmCreate.StatusCode) { throw 'Keycloak synthetic realm creation returned an unexpected status.' }

  $clientRepresentation = [ordered]@{
    clientId = $clientId
    name = 'INTEGIN Field Synthetic Pilot'
    description = 'Synthetic non-production INTEGIN PKCE client; no tenant or authorization claims.'
    protocol = 'openid-connect'
    enabled = $true
    publicClient = $true
    standardFlowEnabled = $true
    implicitFlowEnabled = $false
    directAccessGrantsEnabled = $false
    serviceAccountsEnabled = $false
    authorizationServicesEnabled = $false
    bearerOnly = $false
    fullScopeAllowed = $false
    redirectUris = @('http://127.0.0.1')
    webOrigins = @()
    rootUrl = ''
    baseUrl = ''
    adminUrl = ''
    attributes = @{ 'pkce.code.challenge.method' = 'S256'; 'oauth2.device.authorization.grant.enabled' = 'false' }
  }
  $clientBody = $clientRepresentation | ConvertTo-Json -Depth 8
  $clientCreate = Invoke-WebRequest -UseBasicParsing -Method Post -Uri "$baseUrl/admin/realms/$realmName/clients" -Headers $headers -ContentType 'application/json' -Body $clientBody -ErrorAction Stop
  if (@(201,204) -notcontains [int]$clientCreate.StatusCode) { throw 'Keycloak synthetic public-client creation returned an unexpected status.' }

  $realmRead = Invoke-KeycloakGet -Path "/admin/realms/$realmName" -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json
  $clientSearch = Invoke-KeycloakGet -Path "/admin/realms/$realmName/clients?clientId=$clientId" -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json
  $matches = @($clientSearch | Where-Object { $_.clientId -eq $clientId })
  if ($matches.Count -ne 1) { throw 'Keycloak synthetic client lookup did not return exactly one client.' }
  $clientRead = Invoke-KeycloakGet -Path ("/admin/realms/{0}/clients/{1}" -f $realmName, $matches[0].id) -Headers $headers | Select-Object -ExpandProperty Content | ConvertFrom-Json

  if ($realmRead.realm -ne $realmName -or -not $realmRead.enabled -or $realmRead.sslRequired -ne 'external' -or -not $realmRead.bruteForceProtected -or -not $realmRead.eventsEnabled -or -not $realmRead.adminEventsEnabled -or $realmRead.adminEventsDetailsEnabled) { throw 'Synthetic realm post-create allowlist verification failed.' }
  if ($clientRead.clientId -ne $clientId -or -not $clientRead.publicClient -or -not $clientRead.standardFlowEnabled -or $clientRead.implicitFlowEnabled -or $clientRead.directAccessGrantsEnabled -or $clientRead.serviceAccountsEnabled -or $clientRead.authorizationServicesEnabled -or $clientRead.fullScopeAllowed -or @($clientRead.redirectUris).Count -ne 1 -or @($clientRead.redirectUris)[0] -ne 'http://127.0.0.1' -or $clientRead.attributes.'pkce.code.challenge.method' -ne 'S256' -or $clientRead.attributes.'oauth2.device.authorization.grant.enabled' -ne 'false') { throw 'Synthetic public-client post-create allowlist verification failed.' }

  $issuer = "$baseUrl/realms/$realmName"
  $discoveryStatus = Get-HttpStatus -Uri "$issuer/.well-known/openid-configuration"
  $jwksStatus = Get-HttpStatus -Uri "$issuer/protocol/openid-connect/certs"
  if ($discoveryStatus -ne 200 -or $jwksStatus -ne 200) { throw 'Synthetic realm discovery/JWKS verification failed.' }
  $directGrantStatus = Get-HttpStatus -Uri "$issuer/protocol/openid-connect/token"
  if ($directGrantStatus -ne 405) { throw 'Synthetic realm token endpoint method preflight was unexpected.' }

  $releaseRecord = [ordered]@{
    realm = $realmName
    client_id = $clientId
    issuer = $issuer
    jwks_uri = "$issuer/protocol/openid-connect/certs"
    created_at_utc = [DateTime]::UtcNow.ToString('o')
    realm_policy = [ordered]@{ ssl_required = 'external'; brute_force_protected = $true; events_enabled = $true; admin_events_enabled = $true; admin_event_details_enabled = $false }
    client_policy = [ordered]@{ public_client = $true; standard_flow = $true; pkce_method = 'S256'; redirect_uri = 'http://127.0.0.1'; direct_access_grant = $false; implicit_flow = $false; service_accounts = $false; authorization_services = $false; device_grant = $false }
    authorization_boundary = 'Keycloak identity-only; INTEGIN Go/PostgreSQL remains authoritative for tenant, organization, device, workflow, capability, and RLS.'
  }
  $recordJson = $releaseRecord | ConvertTo-Json -Depth 8
  [System.IO.File]::WriteAllText($releaseRecordPath, $recordJson, (New-Object System.Text.UTF8Encoding($false)))
  Write-Output 'KEYCLOAK_ZIP_SYNTHETIC_REALM_AND_PUBLIC_PKCE_CLIENT_VERIFIED'
} finally {
  $token = $null
  $headers = $null
}

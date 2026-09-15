#[INTEGIN Casdoor Pilot] Seed exactly one synthetic pilot organization and one RS256 application; seed the pilot inspector user. Identity-only, no tenant or authorization claims.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$casdoorEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\casdoor-pilot-runtime.env'
$releaseRecordPath = 'C:\MY_PROJECT\operations\pilot\runtime\casdoor\integin-pilot-realm-release.json'
$baseUrl = 'http://127.0.0.1:18180'
$organizationName = 'integin-pilot'
$applicationName = 'integin-live-matrix'
$applicationId = 'integin-live-matrix'
$applicationSecret = 'integin-live-matrix-secret-pilot'
$insecureLoopback = 'true'
$redirectUri = 'http://127.0.0.1:18080/callback'

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
  param([Parameter(Mandatory = $true)][string]$Uri)
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri $Uri -ErrorAction Stop
    return [int]$response.StatusCode
  } catch [System.Net.WebException] {
    if ($null -eq $_.Exception.Response) { throw }
    return [int]$_.Exception.Response.StatusCode
  }
}

function Invoke-CasdoorPost {
  param([Parameter(Mandatory = $true)][string]$Path,[Parameter(Mandatory = $true)][Microsoft.PowerShell.Commands.WebRequestSession]$WebSession,[Parameter(Mandatory = $true)][string]$Body)
  $response = Invoke-WebRequest -UseBasicParsing -Method Post -TimeoutSec 10 -Uri "$baseUrl$Path" -WebSession $WebSession -ContentType 'application/json' -Body $Body -ErrorAction Stop
  $parsed = $response.Content | ConvertFrom-Json
  if ($parsed.status -ne 'ok') { throw "Casdoor $Path failed: $($parsed.msg)" }
  return $parsed
}

function Get-CasdoorObject {
  param([Parameter(Mandatory = $true)][string]$Path,[Parameter(Mandatory = $true)][Microsoft.PowerShell.Commands.WebRequestSession]$WebSession)
  $raw = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri "$baseUrl$Path" -WebSession $WebSession -Headers @{'Accept'='application/json'} -ErrorAction Stop
  $parsed = $raw.Content | ConvertFrom-Json
  if ($null -eq $parsed.PSObject.Properties['data']) { return $null }
  return $parsed.data
}

if (-not (Test-Path -LiteralPath $casdoorEnvironmentPath)) { throw 'Private Casdoor runtime environment is missing.' }
if ((Get-HttpStatus -Uri "$baseUrl/.well-known/openid-configuration") -ne 200) { throw 'Casdoor discovery preflight failed.' }
$runtime = Read-EnvironmentFile -Path $casdoorEnvironmentPath
foreach ($required in @('CASDOOR_BOOTSTRAP_ADMIN_USER','CASDOOR_BOOTSTRAP_ADMIN_PASSWORD')) {
  if (-not $runtime.ContainsKey($required) -or [string]::IsNullOrWhiteSpace($runtime[$required])) { throw "Required private Casdoor bootstrap key is absent: $required" }
}

$session = $null
try {
  $loginBody = @{ type = 'login'; username = $runtime['CASDOOR_BOOTSTRAP_ADMIN_USER']; password = $runtime['CASDOOR_BOOTSTRAP_ADMIN_PASSWORD']; application = 'app-built-in' } | ConvertTo-Json
  $login = Invoke-WebRequest -UseBasicParsing -Method Post -TimeoutSec 10 -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $loginBody -SessionVariable session -ErrorAction Stop
  $loginJson = $login.Content | ConvertFrom-Json
  if ($loginJson.status -ne 'ok') { throw 'Casdoor bootstrap admin login failed.' }

  # Idempotent reseed: reuse the synthetic org/app/user when a previous run created them.
  if ($null -eq (Get-CasdoorObject -Path "/api/get-organization?id=admin/$organizationName" -WebSession $session)) {
    $organizationBody = [ordered]@{
      owner = 'admin'
      name = $organizationName
      displayName = 'INTEGIN Synthetic Pilot'
      isDemo = $false
      passwordType = 'plain'
      passwordSalt = ''
    } | ConvertTo-Json -Depth 6
    Invoke-CasdoorPost -Path '/api/add-organization' -WebSession $session -Body $organizationBody | Out-Null
  }

  # Use the default built-in RS256 cert (cert-built-in); do not mint a custom one.
  if ($null -eq (Get-CasdoorObject -Path "/api/get-application?id=admin/$applicationName" -WebSession $session)) {

  $applicationBody = [ordered]@{
    owner = 'admin'
    name = $applicationName
    displayName = 'INTEGIN Live Matrix'
    organization = $organizationName
    clientId = $applicationId
    clientSecret = $applicationSecret
    redirectUris = @($redirectUri)
    homeUrl = $redirectUri
    isPublic = $false
    tokenFormat = 'JWT'
    signinMethods = @(@{ name = 'Password'; displayName = 'Password'; rule = 'All' })
    signupItems = @()
    enableSignUp = $false
    enableSignIn = $true
    enableCodeSignin = $false
    forceSignin = $true
    isOnboardApplication = $false
    cert = 'cert-built-in'
    grantTypes = @('authorization_code', 'password', 'client_credentials')
    enableIdp = $false
  } | ConvertTo-Json -Depth 8
  Invoke-CasdoorPost -Path '/api/add-application' -WebSession $session -Body $applicationBody | Out-Null
  }

  if ($null -eq (Get-CasdoorObject -Path "/api/get-user?id=$organizationName/pilot-inspector" -WebSession $session)) {

  $inspectorBody = [ordered]@{
    owner = $organizationName
    name = 'pilot-inspector'
    displayName = 'Pilot Inspector'
    email = 'pilot-inspector@integin.invalid'
    phone = ''
    password = 'pilot-inspector-pass-2026'
    isGlobalAdmin = $false
    isForbidden = $false
  } | ConvertTo-Json -Depth 6
  Invoke-CasdoorPost -Path '/api/add-user' -WebSession $session -Body $inspectorBody | Out-Null
  }

  $issuer = "$baseUrl"
  $discovery = Invoke-RestMethod -TimeoutSec 10 -Uri "$issuer/.well-known/openid-configuration"
  if ([string]::IsNullOrWhiteSpace($discovery.issuer) -or [string]::IsNullOrWhiteSpace($discovery.jwks_uri)) { throw 'Casdoor discovery issuer/jwks post-create verification failed.' }

  $releaseRecord = [ordered]@{
    organization = $organizationName
    application = $applicationName
    client_id = $applicationId
    client_secret = $applicationSecret
    issuer = $issuer
    jwks_uri = $discovery.jwks_uri
    signing_certificate = 'cert-built-in'
    redirect_uri = $redirectUri
    created_at_utc = [DateTime]::UtcNow.ToString('o')
    authorization_boundary = 'Casdoor identity-only; INTEGIN Go/PostgreSQL remains authoritative for tenant, organization, device, workflow, capability, and RLS.'
  } | ConvertTo-Json -Depth 6
  $releaseDir = Split-Path -Parent $releaseRecordPath
  if (-not (Test-Path -LiteralPath $releaseDir)) { New-Item -ItemType Directory -Path $releaseDir | Out-Null }
  [System.IO.File]::WriteAllText($releaseRecordPath, $releaseRecord, (New-Object System.Text.UTF8Encoding($false)))
  Write-Output 'CASDOOR_SYNTHETIC_REALM_AND_RS256_APPLICATION_VERIFIED'
} finally {
  $token = $null
  $headers = $null
}

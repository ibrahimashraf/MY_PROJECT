#[INTEGIN Casdoor Pilot] Rotate the bootstrap admin password via the dedicated
# set-password API. The generic update-user endpoint silently drops password
# changes (verified 2026-09-19: displayName round-trips, password hash never
# changes), so rotation must go through set-password with form fields —
# oldPassword/newPassword as form values alongside userOwner/userName.
# Reads the target password from the private runtime env. Idempotent: exits
# ALREADY_ROTATED when the target password already logs in.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$secretPath = 'C:\MY_PROJECT\private\integin-secrets\casdoor-pilot-runtime.env'
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

function Test-AdminLogin {
  param([Parameter(Mandatory = $true)][string]$Password)
  $loginBody = @{ type = 'login'; username = 'admin'; password = $Password; application = 'app-built-in' } | ConvertTo-Json -Compress
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $loginBody -SessionVariable sess -ErrorAction Stop
    return (($response.Content | ConvertFrom-Json).status -eq 'ok')
  } catch {
    return $false
  }
}

$runtime = Read-EnvironmentFile -Path $secretPath
$newPassword = $runtime['CASDOOR_BOOTSTRAP_ADMIN_PASSWORD']
if ([string]::IsNullOrWhiteSpace($newPassword)) { throw 'CASDOOR_BOOTSTRAP_ADMIN_PASSWORD is absent in the private runtime env.' }

if (Test-AdminLogin -Password $newPassword) {
  Write-Output 'CASDOOR_ADMIN_ALREADY_ROTATED'
  exit 0
}

# Log in with the CURRENT password (factory default 123 until rotation succeeds).
$oldPassword = '123'
$loginBody = @{ type = 'login'; username = 'admin'; password = $oldPassword; application = 'app-built-in' } | ConvertTo-Json -Compress
try {
  $null = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $loginBody -SessionVariable sess -ErrorAction Stop
} catch {
  throw 'Neither target nor factory bootstrap password logs in; manual recovery required.'
}

$setBody = @{ userOwner = 'built-in'; userName = 'admin'; oldPassword = $oldPassword; newPassword = $newPassword }
$result = (Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/set-password" -WebSession $sess -Body $setBody -ErrorAction Stop).Content | ConvertFrom-Json
if ($result.status -ne 'ok') { throw ('Casdoor set-password failed: ' + $result.msg) }

if (-not (Test-AdminLogin -Password $newPassword)) { throw 'New bootstrap password does not log in after update.' }
Write-Output 'CASDOOR_ADMIN_ROTATED_AND_VERIFIED'

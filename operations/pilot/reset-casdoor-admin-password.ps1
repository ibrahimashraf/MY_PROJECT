#[INTEGIN Casdoor Pilot] Rotate the bootstrap admin password via API (the web UI
# profile save is blocked when the built-in admin record carries a non-CIDR
# allowedIp value). Reads the target password from the private runtime env.
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

$runtime = Read-EnvironmentFile -Path $secretPath
$newPassword = $runtime['CASDOOR_BOOTSTRAP_ADMIN_PASSWORD']
if ([string]::IsNullOrWhiteSpace($newPassword)) { throw 'CASDOOR_BOOTSTRAP_ADMIN_PASSWORD is absent in the private runtime env.' }

# Log in with the CURRENT password (old 123 until rotation succeeds).
$oldPassword = '123'
$loginBody = '{"type":"login","username":"admin","password":"' + $oldPassword + '","application":"app-built-in"}'
try {
  $null = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $loginBody -SessionVariable sess -ErrorAction Stop
} catch {
  # Old password rejected: rotation may already have applied; verify by login with the new one.
  $verifyBody = '{"type":"login","username":"admin","password":"' + $newPassword + '","application":"app-built-in"}'
  $verify = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $verifyBody -SessionVariable sess2 -ErrorAction Stop
  if (($verify.Content | ConvertFrom-Json).status -ne 'ok') { throw 'Neither old nor new bootstrap password logs in.' }
  Write-Output 'CASDOOR_ADMIN_ALREADY_ROTATED'
  exit 0
}

$admin = (Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Uri "$baseUrl/api/get-user?id=built-in/admin" -WebSession $sess -Headers @{'Accept'='application/json'} -ErrorAction Stop).Content | ConvertFrom-Json
if ($admin.data.PSObject.Properties['allowedIp']) {
  Write-Output ('current allowedIp=' + $admin.data.allowedIp)
  $admin.data.allowedIp = ''
}
$admin.data.password = $newPassword
$upd = $admin.data | ConvertTo-Json -Depth 8 -Compress
$result = (Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/update-user?id=built-in/admin" -WebSession $sess -ContentType 'application/json' -Body $upd -ErrorAction Stop).Content | ConvertFrom-Json
if ($result.status -ne 'ok') { throw ('Casdoor admin update failed: ' + $result.msg) }

# Verify the new password logs in on a fresh session.
$checkBody = '{"type":"login","username":"admin","password":"' + $newPassword + '","application":"app-built-in"}'
$check = Invoke-WebRequest -UseBasicParsing -TimeoutSec 10 -Method Post -Uri "$baseUrl/api/login" -ContentType 'application/json' -Body $checkBody -SessionVariable sess3 -ErrorAction Stop
if (($check.Content | ConvertFrom-Json).status -ne 'ok') { throw 'New bootstrap password does not log in after update.' }
Write-Output 'CASDOOR_ADMIN_ROTATED_AND_VERIFIED'

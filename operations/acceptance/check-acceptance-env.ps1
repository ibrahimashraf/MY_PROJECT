$environmentPath = 'C:\MY PROJECT\private\integin-secrets\integin-server.env'
if (-not (Test-Path -LiteralPath $environmentPath)) { throw "Acceptance private environment is missing: $environmentPath" }

$present = @{}
Get-Content -LiteralPath $environmentPath | ForEach-Object {
  if ($_ -match '^([^#=][^=]*)=(.*)$') { $present[$matches[1]] = $true }
}

$required = @(
  'INTEGIN_HTTP_ADDR',
  'INTEGIN_DB_URL',
  'INTEGIN_SYNC_SECRET',
  'INTEGIN_EVIDENCE_STORE',
  'INTEGIN_S3_ENDPOINT',
  'INTEGIN_S3_BUCKET',
  'INTEGIN_S3_ACCESS_KEY',
  'INTEGIN_S3_SECRET_KEY',
  'INTEGIN_S3_REGION',
  'INTEGIN_TENANT_ID',
  'INTEGIN_TENANT_IDS',
  'INTEGIN_LOCAL_PROVISIONING_ENABLED'
)

$required | ForEach-Object {
  [PSCustomObject]@{
    Setting = $_
    Status = if ($present.ContainsKey($_)) { 'PRESENT' } else { 'MISSING' }
  }
} | Format-Table -AutoSize

$serverEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\integin-server.env'
$rustfsEnvironmentPath = 'C:\MY_PROJECT\private\integin-secrets\rustfs.env'

foreach ($path in @($serverEnvironmentPath, $rustfsEnvironmentPath)) {
  if (-not (Test-Path -LiteralPath $path)) { throw "Required private environment file is missing: $path" }
}

function Read-EnvironmentFile {
  param([string]$Path)
  $values = @{}
  Get-Content -LiteralPath $Path | ForEach-Object {
    if ($_ -match '^([^#=][^=]*)=(.*)$') { $values[$matches[1]] = $matches[2] }
  }
  return $values
}

$rustfs = Read-EnvironmentFile -Path $rustfsEnvironmentPath
$accessKey = $rustfs['RUSTFS_ACCESS_KEY']
$secretKey = $rustfs['RUSTFS_SECRET_KEY']
if ([string]::IsNullOrWhiteSpace($accessKey) -or [string]::IsNullOrWhiteSpace($secretKey)) {
  throw 'The private RustFS environment does not contain both required credential keys.'
}

$backupPath = "$serverEnvironmentPath.before-s3-recovery-$(Get-Date -Format 'yyyyMMdd-HHmmss').bak"
Copy-Item -LiteralPath $serverEnvironmentPath -Destination $backupPath -ErrorAction Stop

$retained = Get-Content -LiteralPath $serverEnvironmentPath | Where-Object {
  $_ -notmatch '^INTEGIN_S3_ACCESS_KEY=' -and $_ -notmatch '^INTEGIN_S3_SECRET_KEY='
}
$retained += "INTEGIN_S3_ACCESS_KEY=$accessKey"
$retained += "INTEGIN_S3_SECRET_KEY=$secretKey"
[System.IO.File]::WriteAllLines($serverEnvironmentPath, $retained, [System.Text.UTF8Encoding]::new($false))

Write-Output "ACCEPTANCE_S3_CREDENTIALS_RESTORED_FROM_PRIVATE_RUSTFS_ENV; backup created at $backupPath"

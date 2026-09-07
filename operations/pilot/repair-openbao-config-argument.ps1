$setupPath = 'C:\MY PROJECT\operations\pilot\setup-openbao-pilot.ps1'
$content = [System.IO.File]::ReadAllText($setupPath)
$oldRunSuffix = '$image server | Out-Null'
$newRunSuffix = '$image server -config=/bao/config | Out-Null'

if (-not $content.Contains($oldRunSuffix)) {
  throw 'Expected isolated OpenBao server launch suffix was not found; setup script was not modified.'
}

[System.IO.File]::WriteAllText(
  $setupPath,
  $content.Replace($oldRunSuffix, $newRunSuffix),
  [System.Text.UTF8Encoding]::new($false)
)

$verified = Select-String -LiteralPath $setupPath -SimpleMatch -Pattern '$image server -config=/bao/config | Out-Null'
if ($verified.Count -ne 1) {
  throw 'OpenBao explicit configuration-directory argument verification failed.'
}

Write-Output 'PILOT_OPENBAO_CONFIG_DIRECTORY_ARGUMENT_CORRECTED_AND_VERIFIED'

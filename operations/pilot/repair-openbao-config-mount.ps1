$setupPath = 'C:\MY PROJECT\operations\pilot\setup-openbao-pilot.ps1'
$content = [System.IO.File]::ReadAllText($setupPath)
$oldRun = '  docker run -d --name $containerName --network $networkName --publish "127.0.0.1:${hostPort}:8200" --volume "${dataVolume}:/openbao/file" --volume "${logsVolume}:/openbao/logs" --mount "type=bind,source=$configPath,target=/bao/config/openbao.hcl,readonly" $image server -config=/bao/config/openbao.hcl | Out-Null'
$newRun = '  docker run -d --name $containerName --network $networkName --publish "127.0.0.1:${hostPort}:8200" --volume "${dataVolume}:/openbao/file" --volume "${logsVolume}:/openbao/logs" --mount "type=bind,source=$configDirectory,target=/bao/config,readonly" $image server | Out-Null'

if (-not $content.Contains($oldRun)) {
  throw 'Expected isolated OpenBao file-mount launch command was not found; setup script was not modified.'
}

[System.IO.File]::WriteAllText(
  $setupPath,
  $content.Replace($oldRun, $newRun),
  [System.Text.UTF8Encoding]::new($false)
)

$verified = Select-String -LiteralPath $setupPath -Pattern 'source=\$configDirectory,target=/bao/config,readonly', '\$image server \| Out-Null'
if ($verified.Count -ne 2) {
  throw 'OpenBao configuration-directory mount verification failed.'
}

Write-Output 'PILOT_OPENBAO_CONFIG_DIRECTORY_MOUNT_CORRECTED_AND_VERIFIED'

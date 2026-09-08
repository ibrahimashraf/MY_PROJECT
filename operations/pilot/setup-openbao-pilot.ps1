$containerName = 'integin-pilot-openbao'
$networkName = 'integin-pilot-secrets-net'
$dataVolume = 'integin-pilot-openbao-data'
$logsVolume = 'integin-pilot-openbao-logs'
$hostPort = 18200
$image = 'openbao/openbao:2.6.0'
$pilotRoot = 'C:\MY_PROJECT\operations\pilot\openbao'
$configDirectory = Join-Path $pilotRoot 'config'
$configPath = Join-Path $configDirectory 'openbao.hcl'
$recoveryPath = 'C:\MY_PROJECT\private\integin-secrets\openbao-pilot-recovery.json'

if (Get-Process -Name 'integin-server-provision','integin-server-pilot' -ErrorAction SilentlyContinue) {
  # A presence check only: existing INTEGIN processes are expected and are never stopped or reconfigured here.
}
if (docker ps -a --format '{{.Names}}' | Select-String -SimpleMatch -Quiet $containerName) {
  throw "Refusing to overwrite existing container: $containerName"
}
if (Test-Path -LiteralPath $recoveryPath) {
  throw "Refusing to overwrite existing private recovery file: $recoveryPath"
}
if (Get-NetTCPConnection -LocalPort $hostPort -State Listen -ErrorAction SilentlyContinue) {
  throw "Loopback port $hostPort is already in use. No resource was created."
}

New-Item -ItemType Directory -Force -Path $configDirectory | Out-Null
$config = @'
ui           = false
cluster_name = "integin-pilot-openbao"
api_addr     = "http://127.0.0.1:18200"

storage "file" {
  path = "/openbao/file"
}

listener "tcp" {
  address     = "0.0.0.0:8200"
  tls_disable = true
}

default_lease_ttl = "1h"
max_lease_ttl     = "4h"

audit "file" "integin-pilot-audit" {
  description = "Pilot rehearsal audit trail"
  options {
    file_path = "/openbao/logs/audit.log"
  }
}
'@
[System.IO.File]::WriteAllText($configPath, $config, [System.Text.UTF8Encoding]::new($false))

$createdNetwork = $false
$createdDataVolume = $false
$createdLogsVolume = $false
try {
  if (-not (docker network ls --format '{{.Name}}' | Select-String -SimpleMatch -Quiet $networkName)) {
    docker network create $networkName | Out-Null
    $createdNetwork = $true
  }
  if (-not (docker volume ls --format '{{.Name}}' | Select-String -SimpleMatch -Quiet $dataVolume)) {
    docker volume create $dataVolume | Out-Null
    $createdDataVolume = $true
  }
  if (-not (docker volume ls --format '{{.Name}}' | Select-String -SimpleMatch -Quiet $logsVolume)) {
    docker volume create $logsVolume | Out-Null
    $createdLogsVolume = $true
  }

  docker run -d --name $containerName --network $networkName --publish "127.0.0.1:${hostPort}:8200" --volume "${dataVolume}:/openbao/file" --volume "${logsVolume}:/openbao/logs" --mount "type=bind,source=$configDirectory,target=/bao/config,readonly" $image server -config=/bao/config | Out-Null

  $deadline = (Get-Date).AddSeconds(30)
  do {
    Start-Sleep -Milliseconds 750
    $isRunning = docker ps --format '{{.Names}}' | Select-String -SimpleMatch -Quiet $containerName
  } until ($isRunning -or (Get-Date) -gt $deadline)
  if (-not $isRunning) { throw 'OpenBao container did not remain running after startup.' }

  $initialization = & docker exec $containerName bao operator init -address=http://127.0.0.1:8200 -key-shares=3 -key-threshold=2 -format=json
  if ($LASTEXITCODE -ne 0 -or [string]::IsNullOrWhiteSpace(($initialization -join ''))) { throw 'OpenBao initialization did not return private recovery material.' }
  [System.IO.File]::WriteAllLines($recoveryPath, [string[]]$initialization, [System.Text.UTF8Encoding]::new($false))
  & icacls $recoveryPath /inheritance:r /grant:r "${env:USERNAME}:(R,W)" | Out-Null

  Write-Output 'PILOT_OPENBAO_SEALED_INITIALIZED_WITH_PRIVATE_3_OF_2_RECOVERY'
} catch {
  if (docker ps -a --format '{{.Names}}' | Select-String -SimpleMatch -Quiet $containerName) { docker rm -f $containerName | Out-Null }
  if (Test-Path -LiteralPath $recoveryPath) { Remove-Item -LiteralPath $recoveryPath -Force }
  if ($createdLogsVolume) { docker volume rm $logsVolume | Out-Null }
  if ($createdDataVolume) { docker volume rm $dataVolume | Out-Null }
  if ($createdNetwork) { docker network rm $networkName | Out-Null }
  throw
}

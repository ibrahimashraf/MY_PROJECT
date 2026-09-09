#[INTEGIN Casdoor Pilot] Gracefully stop and remove the disposable Casdoor and isolated PostgreSQL containers and their pilot network.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

$casdoorContainer = 'integin-pilot-casdoor'
$postgresContainer = 'integin-pilot-casdoor-postgres'
$networkName = 'integin-casdoor-pilot-net'
$postgresVolume = 'integin-pilot-casdoor-postgres-data'

function Get-DockerExitCode {
  param([Parameter(Mandatory = $true)][string[]]$Arguments)
  $previous = $ErrorActionPreference
  try {
    $ErrorActionPreference = 'Continue'
    & docker @Arguments 1>$null 2>$null
    return $LASTEXITCODE
  } finally { $ErrorActionPreference = $previous }
}

foreach ($name in @($casdoorContainer, $postgresContainer)) {
  if ((Get-DockerExitCode -Arguments @('container', 'inspect', $name)) -eq 0) {
    & docker rm -f $name 1>$null
    if ($LASTEXITCODE -ne 0) { throw "Unable to remove Casdoor pilot container $name" }
  }
}
if ((Get-DockerExitCode -Arguments @('volume', 'inspect', $postgresVolume)) -eq 0) {
  & docker volume rm $postgresVolume 1>$null
}
if ((Get-DockerExitCode -Arguments @('network', 'inspect', $networkName)) -eq 0) {
  & docker network rm $networkName 1>$null
}
Write-Output 'CASDOOR_PILOT_STOPPED_AND_RESOURCES_REMOVED'

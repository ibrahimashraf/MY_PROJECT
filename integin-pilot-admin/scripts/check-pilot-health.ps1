$listeners = Get-NetTCPConnection -State Listen -LocalPort 8080,18080 -ErrorAction SilentlyContinue |
  Select-Object LocalAddress, LocalPort, OwningProcess

'LISTENERS'
$listeners | Format-Table -AutoSize

'PROCESSES'
foreach ($entry in $listeners) {
  Get-Process -Id $entry.OwningProcess -ErrorAction SilentlyContinue |
    Select-Object Id, ProcessName, StartTime |
    Format-Table -AutoSize
}

'HEALTH'
foreach ($url in @(
  'http://127.0.0.1:8080/healthz',
  'http://127.0.0.1:8080/readyz',
  'http://127.0.0.1:18080/healthz',
  'http://127.0.0.1:18080/readyz'
)) {
  try {
    $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 5 $url
    "$url -> $($response.StatusCode)"
  } catch {
    "$url -> UNAVAILABLE"
  }
}

'PILOT_LOGS_PRESENT'
Get-ChildItem 'C:\integin-pilot\runtime' -Filter 'server.*.log' -ErrorAction SilentlyContinue |
  Select-Object Name, Length, LastWriteTime |
  Format-Table -AutoSize


# fire-and-forget delegate wrapper: detaches relay, writes marker, zero LLM polls.
# Usage: .\delegate-forget.ps1 -Brief C:\MY_PROJECT\brief-x.txt -Model opencode/big-pickle
param([string]$Brief, [string]$Model = "opencode/big-pickle", [string]$Cd = "C:\MY_PROJECT")
$ts = Get-Date -Format "yyyy-MM-ddTHH-mm-ss-fffZ"
$out = Join-Path $env:TEMP "delegate-relay\$ts"
New-Item -ItemType Directory -Force -Path $out | Out-Null
$idFile = Join-Path $out "job.info"
"$Brief|$Model|$Cd" | Set-Content $idFile
Start-Job -ScriptBlock {
  param($b,$m,$c,$o)
  node "C:\MY_PROJECT\.agents\skills\opencode-delegate\scripts\relay.mjs" --brief $b --model $m --cd $c --timeout 30m --out-dir $o > (Join-Path $o "stdout.txt") 2>&1
  "DONE $(Get-Date -Format o)" | Set-Content (Join-Path $o "DONE.marker")
  try { [console]::beep(880,300) } catch {}
  try { New-BurntToastNotification -Text "Delegate done", $o } catch {}
  try { msg * /TIME:30 "Delegate done: $o" } catch {}
} -ArgumentList $Brief,$Model,$Cd,$out | Out-Null
Write-Output $out
$reg = Join-Path $env:TEMP "delegate-relay\outstanding.txt"
$out | Add-Content $reg

$setupPath = 'C:\MY PROJECT\operations\pilot\setup-openbao-pilot.ps1'
$content = [System.IO.File]::ReadAllText($setupPath)
$oldAudit = @'
audit "file" {
  path = "/openbao/logs/audit.log"
}
'@
$newAudit = @'
audit "file" "integin-pilot-audit" {
  description = "Pilot rehearsal audit trail"
  options {
    file_path = "/openbao/logs/audit.log"
  }
}
'@

if (-not $content.Contains($oldAudit)) {
  throw 'Expected isolated OpenBao audit stanza was not found; setup script was not modified.'
}

[System.IO.File]::WriteAllText(
  $setupPath,
  $content.Replace($oldAudit, $newAudit),
  [System.Text.UTF8Encoding]::new($false)
)

$verified = Select-String -LiteralPath $setupPath -Pattern 'audit "file" "integin-pilot-audit"', 'file_path = "/openbao/logs/audit.log"'
if ($verified.Count -ne 2) {
  throw 'OpenBao audit correction verification failed.'
}

Write-Output 'PILOT_OPENBAO_AUDIT_STANZA_CORRECTED_AND_VERIFIED'

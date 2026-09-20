[CmdletBinding()]
param(
    [string]$MigrationDir = "",
    [switch]$Replay,
    [string]$VerifyDb = "integin_migration_verify"
)

$ErrorActionPreference = "Stop"
$scriptRoot = if ($PSScriptRoot) { $PSScriptRoot } else { Split-Path -Parent $MyInvocation.MyCommand.Path }
if (-not $MigrationDir) { $MigrationDir = Join-Path $scriptRoot "migrations" }
$errors = @()
$warnings = @()

$all = Get-ChildItem -Path $MigrationDir -Filter "*.sql"
$up = @($all | Where-Object { $_.Name -notmatch '\.down\.' -and $_.Name -notmatch '\.draft\.' -and $_.Name -notmatch '\.candidate\.' })

# E1: filenames must be unique (ledger key + deterministic sort order depend on it)
$dups = $all.Name | Group-Object | Where-Object { $_.Count -gt 1 }
foreach ($d in $dups) { $errors += "duplicate filename: $($d.Name)" }

# E2: every replayed file must be sequence-numbered (ordering assumption)
foreach ($f in $up) {
    if ($f.Name -notmatch '^\d+_.*\.sql$') { $errors += "unnumbered up-file: $($f.Name)" }
}

# E3: orphan rollback (non-draft .down. with no up partner)
foreach ($f in ($all | Where-Object { $_.Name -match '\.down\.' -and $_.Name -notmatch '\.draft\.' })) {
    $partner = $f.Name -replace '\.down\.sql$', '.sql'
    if ($partner -notin $up.Name) { $errors += "orphan rollback, no up-file: $($f.Name)" }
}

# W1/W2: grandfathered smells — down-pair required only for NEW files above current max seq
$maxSeq = ($up.Name | ForEach-Object { if ($_ -match '^(\d+)_') { $Matches[1] } } | Sort-Object -Descending | Select-Object -First 1)
foreach ($f in $up) {
    if (-not (Test-Path (Join-Path $MigrationDir ($f.BaseName + ".down.sql")))) {
        $seq = if ($f.Name -match '^(\d+)_') { $Matches[1] } else { "" }
        if ($seq -gt $maxSeq) { $errors += "new up-file missing .down.sql pair: $($f.Name)" }
        else { $warnings += "no .down.sql pair (grandfathered): $($f.Name)" }
    }
}
$dupSeqs = $up.Name | ForEach-Object { if ($_ -match '^(\d+)_') { $Matches[1] } } | Group-Object | Where-Object { $_.Count -gt 1 }
foreach ($g in $dupSeqs) { $warnings += "sequence $($g.Name) shared by $($g.Count) files (order falls back to full-name sort)" }

if ($Replay) {
    & (Join-Path $scriptRoot "apply_migrations.ps1") -Database $VerifyDb -Fresh
    if ($LASTEXITCODE -ne 0) { $errors += "fresh replay failed (see output above)" }
    else {
        $ledger = docker exec -i integin-dev-postgres psql -U integin_runtime -d $VerifyDb -t -A -c "SELECT name FROM schema_migrations ORDER BY name;"
        $ledgerNames = @($ledger -split "`r?`n" | ForEach-Object { $_.Trim() } | Where-Object { $_ })
        $missing = @($up.Name | Where-Object { $_ -notin $ledgerNames })
        $extra = @($ledgerNames | Where-Object { $_ -notin $up.Name })
        foreach ($m in $missing) { $errors += "replayed ledger missing file: $m" }
        foreach ($m in $extra) { $errors += "ledger has unknown entry: $m" }
        # checksum drift: file edited after it was recorded (one round-trip, not 93)
        $rows = docker exec -i integin-dev-postgres psql -U integin_runtime -d $VerifyDb -t -A -F "|" -c "SELECT name, checksum FROM schema_migrations;"
        $recordedMap = @{}
        foreach ($line in ($rows -split "`r?`n")) {
            $parts = $line.Trim() -split "\|"
            if ($parts.Count -eq 2) { $recordedMap[$parts[0]] = $parts[1] }
        }
        foreach ($f in $up) {
            $actual = (Get-FileHash -LiteralPath $f.FullName -Algorithm SHA256).Hash.ToLower()
            if ($recordedMap[$f.Name] -ne $actual) { $errors += "checksum drift (edited post-apply): $($f.Name)" }
        }
        docker exec -i integin-dev-postgres psql -U integin_runtime -d integin_dev -c "DROP DATABASE $VerifyDb;" | Out-Null
    }
}

Write-Host "UP: $($up.Count) / DOWN: $(@($all | Where-Object { $_.Name -match '\.down\.' }).Count) / MAXSEQ: $maxSeq" -ForegroundColor Cyan
foreach ($w in $warnings) { Write-Host "[warn] $w" -ForegroundColor Yellow }
foreach ($e in $errors) { Write-Host "[FAIL] $e" -ForegroundColor Red }
if ($errors.Count -gt 0) { exit 1 }
Write-Host "[+] migration gate clean ($($warnings.Count) grandfathered warnings)." -ForegroundColor Green
exit 0

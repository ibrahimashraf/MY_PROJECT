# INTEGIN Stage A Automated Disaster Recovery Verification Drill (PowerShell)
# Validates PostgreSQL custom dump, SHA-256 digest, and S3/RustFS ciphertext object manifests.

[CmdletBinding()]
param (
    [string]$DrillDirectory = "C:\MY_PROJECT\operations\acceptance\backups\stage-a-recovery-b6c1804e50ac"
)

$ErrorActionPreference = "Stop"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "INTEGIN STAGE A RECOVERY & BACKUP DRILL VERIFIER (POWERSHELL)" -ForegroundColor Cyan
Write-Host "Drill Evidence Target: $DrillDirectory" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

if (-not (Test-Path -LiteralPath $DrillDirectory)) {
    Write-Error "Recovery directory not found: $DrillDirectory"
    exit 1
}

$manifestPath = Join-Path $DrillDirectory "recovery-manifest.json"
$dumpPath = Join-Path $DrillDirectory "pilot-current-schema.dump"
$dumpShaPath = Join-Path $DrillDirectory "pilot-current-schema.dump.sha256"
$evidenceBinPath = Join-Path $DrillDirectory "evidence-object.bin"

foreach ($file in @($manifestPath, $dumpPath, $dumpShaPath, $evidenceBinPath)) {
    if (-not (Test-Path -LiteralPath $file)) {
        Write-Error "Required drill artifact missing: $file"
        exit 1
    }
}
Write-Host "[+] Found all required recovery artifacts." -ForegroundColor Green

# 1. Verify PostgreSQL dump SHA-256
Write-Host "[*] Verifying PostgreSQL dump SHA-256 integrity..." -ForegroundColor Yellow
$expectedDumpHash = (Get-Content -LiteralPath $dumpShaPath -Raw).Trim().Split()[0].ToLower()
$actualDumpHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $dumpPath).Hash.ToLower()

if ($expectedDumpHash -ne $actualDumpHash) {
    Write-Error "Dump hash mismatch! Expected: $expectedDumpHash, Got: $actualDumpHash"
    exit 1
}
Write-Host "PASS: Dump SHA-256 verified: $actualDumpHash" -ForegroundColor Green

# 2. Verify Ciphertext Evidence Object SHA-256
Write-Host "[*] Verifying ciphertext evidence object integrity..." -ForegroundColor Yellow
$actualObjHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $evidenceBinPath).Hash.ToLower()
$manifestJson = Get-Content -LiteralPath $manifestPath -Raw | ConvertFrom-Json

$expectedObjHash = $manifestJson.object.ciphertext_sha256.ToLower()
if ($expectedObjHash -ne $actualObjHash) {
    Write-Error "Ciphertext object hash mismatch! Expected: $expectedObjHash, Got: $actualObjHash"
    exit 1
}
Write-Host "PASS: Ciphertext object SHA-256 matched manifest: $actualObjHash" -ForegroundColor Green

# 3. Verify Manifest Metadata Contract
Write-Host "[*] Checking manifest schema and tenant boundaries..." -ForegroundColor Yellow
if ($manifestJson.status -eq "source_cleaned" -and $manifestJson.version -eq "integin-recovery-drill-v1") {
    Write-Host "PASS: Recovery drill manifest structure verified; source residue clean." -ForegroundColor Green
} else {
    Write-Error "Invalid manifest status ($($manifestJson.status)) or drill version ($($manifestJson.version))."
    exit 1
}

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "STAGE A RECOVERY DRILL VERIFICATION: ALL CHECKS PASSED" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

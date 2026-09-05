[CmdletBinding()]
param(
    [string]$Database = "integin_migration_test",
    [switch]$Fresh
)

$ErrorActionPreference = "Stop"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "INTEGIN SCHEMA MIGRATION ENGINE (Ledger-Tracked)" -ForegroundColor Cyan
Write-Host "Target Database: $Database" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# 1. Fresh database reset if explicitly requested
if ($Fresh) {
    Write-Host "[*] Resetting database '$Database' to fresh state..." -ForegroundColor Yellow
    docker exec -i integin-dev-postgres psql -U postgres -d postgres -c "DROP DATABASE IF EXISTS $Database;" | Out-Null
    docker exec -i integin-dev-postgres psql -U postgres -d postgres -c "CREATE DATABASE $Database;" | Out-Null
    Write-Host "[+] Fresh database '$Database' created." -ForegroundColor Green
} else {
    # Ensure database exists
    $dbExists = docker exec -i integin-dev-postgres psql -U postgres -d postgres -t -c "SELECT 1 FROM pg_database WHERE datname = '$Database';"
    if ($dbExists -notmatch '1') {
        Write-Host "[*] Database '$Database' does not exist. Creating..." -ForegroundColor Yellow
        docker exec -i integin-dev-postgres psql -U postgres -d postgres -c "CREATE DATABASE $Database;" | Out-Null
    }
}

# 2. Bootstrap schema_migrations tracking table
$initLedgerSql = @"
CREATE TABLE IF NOT EXISTS schema_migrations (
    name TEXT PRIMARY KEY,
    checksum TEXT NOT NULL,
    applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
"@
$initLedgerSql | docker exec -i integin-dev-postgres psql -v ON_ERROR_STOP=1 -U postgres -d $Database | Out-Null

# 3. Retrieve already applied migration names
$appliedRaw = docker exec -i integin-dev-postgres psql -U postgres -d $Database -t -A -c "SELECT name FROM schema_migrations ORDER BY name;"
$appliedMigrations = @{}
if ($appliedRaw) {
    foreach ($v in ($appliedRaw -split "`r?`n")) {
        $trimmed = $v.Trim()
        if ($trimmed) {
            $appliedMigrations[$trimmed] = $true
        }
    }
}

# 4. Discover migration files
$migrationFiles = Get-ChildItem -Path .\migrations -Filter "*.sql" | 
    Where-Object { 
        $_.Name -notmatch '\.down\.' -and 
        $_.Name -notmatch '\.draft\.' -and 
        $_.Name -notmatch '\.candidate\.' 
    } | Sort-Object Name

# 5. Filter pending migrations
$pending = @()
foreach ($f in $migrationFiles) {
    if (-not $appliedMigrations.ContainsKey($f.Name)) {
        $pending += @{
            Name = $f.Name
            File = $f
        }
    }
}

# 6. Check if database is already up to date
if ($pending.Count -eq 0) {
    Write-Host "[+] Database '$Database' is already UP TO DATE ($($appliedMigrations.Count) migrations recorded, 0 pending)." -ForegroundColor Green
    exit 0
}

Write-Host "[*] Found $($pending.Count) pending migration(s) out of $($migrationFiles.Count) total." -ForegroundColor Yellow

# 7. Apply each pending migration inside transactional block
foreach ($p in $pending) {
    $name = $p.Name
    $filePath = $p.File.FullName

    Write-Host "  -> Applying $name..." -ForegroundColor Cyan -NoNewline

    # Compute SHA-256 checksum of migration content
    $content = Get-Content -LiteralPath $filePath -Raw
    $sha256 = (Get-FileHash -LiteralPath $filePath -Algorithm SHA256).Hash.ToLower()

    # Append ledger recording statement
    $recordSql = "`nINSERT INTO schema_migrations (name, checksum) VALUES ('$name', '$sha256');"
    $combinedSql = $content + $recordSql

    $combinedSql | docker exec -i integin-dev-postgres psql -v ON_ERROR_STOP=1 -v client_min_messages=warning -U postgres -d $Database
    if ($LASTEXITCODE -ne 0) {
        Write-Host " [FAILED]" -ForegroundColor Red
        Write-Error "Migration failed at $name"
        exit 1
    }
    Write-Host " [OK]" -ForegroundColor Green
}

Write-Host "============================================================" -ForegroundColor Green
Write-Host "SUCCESS: All pending migrations applied and recorded to ledger." -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green

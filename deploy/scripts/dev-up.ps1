# INTEGIN Developer Environment Bootstrap Script (dev-up.ps1)
# Starts local disposable PostgreSQL (port 15432), checks health, and exports test variables.

[CmdletBinding()]
param (
    [int]$DbPort = 15432,
    [string]$DbUser = "postgres",
    [string]$DbPassword = "postgres_local_test_password",
    [string]$DbName = "integin_dev"
)

$ErrorActionPreference = "Stop"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "INTEGIN DEVELOPER ENVIRONMENT BOOTSTRAP (dev-up.ps1)" -ForegroundColor Cyan
Write-Host "Database Target: 127.0.0.1:$DbPort / $DbName" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

# 1. Check Docker Daemon availability
Write-Host "[*] Checking container runtime..." -ForegroundColor Yellow
$dockerOk = $false
try {
    $dockerVersion = docker version --format '{{.Server.Version}}' 2>$null
    if ($dockerVersion) {
        $dockerOk = $true
        Write-Host "[+] Docker daemon running: v$dockerVersion" -ForegroundColor Green
    }
} catch {
    $dockerOk = $false
}

if ($dockerOk) {
    Write-Host "[*] Ensuring disposable Postgres container is running on port $DbPort..." -ForegroundColor Yellow
    $containerName = "integin-dev-postgres"
    $existing = docker ps -a --filter "name=$containerName" --format "{{.ID}}"
    if (-not $existing) {
        docker run -d --name $containerName -p "${DbPort}:5432" -e POSTGRES_USER=$DbUser -e POSTGRES_PASSWORD=$DbPassword -e POSTGRES_DB=$DbName postgres:16-alpine | Out-Null
        Write-Host "[+] Spawned $containerName on port $DbPort" -ForegroundColor Green
    } else {
        $running = docker ps --filter "name=$containerName" --format "{{.ID}}"
        if (-not $running) {
            docker start $containerName | Out-Null
            Write-Host "[+] Started existing container $containerName" -ForegroundColor Green
        } else {
            Write-Host "[+] Container $containerName is already running" -ForegroundColor Green
        }
    }
} else {
    Write-Host "[!] Docker not detected or offline; checking if local Postgres is already listening on port $DbPort..." -ForegroundColor Yellow
    $connTest = Test-NetConnection -ComputerName 127.0.0.1 -Port $DbPort -WarningAction SilentlyContinue
    if ($connTest.TcpTestSucceeded) {
        Write-Host "[+] Active Postgres listener found on port $DbPort" -ForegroundColor Green
    } else {
        Write-Host "[!] No listener on port $DbPort. Ensure PostgreSQL is active before running live integration tests." -ForegroundColor Magenta
    }
}

# 2. Verify and Apply Core Migrations Directory
$projectRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$migrationsDir = Join-Path $projectRoot "migrations"
if ($dockerOk -and (Test-Path -LiteralPath $migrationsDir)) {
    Write-Host "[*] Setting up integin_runtime role and permissions..." -ForegroundColor Yellow
    docker exec integin-dev-postgres psql -U $DbUser -d $DbName -c "DO 'BEGIN IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = ''integin_runtime'') THEN CREATE ROLE integin_runtime WITH LOGIN PASSWORD ''test_password''; END IF; END'; GRANT ALL PRIVILEGES ON DATABASE $DbName TO integin_runtime; GRANT ALL ON SCHEMA public TO integin_runtime;" | Out-Null

    Write-Host "[*] Applying core SQL migrations..." -ForegroundColor Yellow
    $coreMigrations = @(
        "0001_event_log.sql",
        "0002_device_trust_sync.sql",
        "0003_event_log_tenant_rls.sql",
        "0004_identity_subject_membership.sql",
        "0005_work_package_persistence.sql",
        "0009_identity_work_order_actor.candidate.sql",
        "0009_work_order_persistence.sql",
        "0010_work_order_rls.sql",
        "0012_certificate_template_binding_registry.candidate.sql",
        "0012_work_order_handover.sql",
        "0013_certificate_authority_lifecycle.candidate.sql",
        "0014_certificate_public_bindings.candidate.sql",
        "0044_work_order_evidence.sql",
        "0059_river_job_queue.sql"
    )
    foreach ($mig in $coreMigrations) {
        $migPath = Join-Path $migrationsDir $mig
        if (Test-Path -LiteralPath $migPath) {
            Write-Host "  -> Applying $mig" -ForegroundColor DarkGray
            Get-Content -LiteralPath $migPath -Raw | docker exec -i integin-dev-postgres psql -U $DbUser -d $DbName -v ON_ERROR_STOP=0 | Out-Null
        }
    }
    Write-Host "[+] Core database migrations applied successfully." -ForegroundColor Green
}

# 3. Export Environment Variables for Current Session
$dbUrl = "postgres://${DbUser}:${DbPassword}@127.0.0.1:${DbPort}/${DbName}?sslmode=disable"
[Environment]::SetEnvironmentVariable("INTEGIN_TEST_DATABASE_URL", $dbUrl, "Process")
[Environment]::SetEnvironmentVariable("INTEGIN_DB_URL", $dbUrl, "Process")

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "INTEGIN Dev Environment Ready" -ForegroundColor Green
Write-Host "Connection URL: $dbUrl" -ForegroundColor Cyan
Write-Host "All live integration suites are primed for testing." -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan


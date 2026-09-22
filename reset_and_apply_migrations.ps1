$ErrorActionPreference = "Stop"

Write-Host "Stopping acceptance server..."
Stop-Process -Name 'integin-server-provision' -Force -ErrorAction SilentlyContinue
Start-Sleep -Seconds 2

Write-Host "Dropping and recreating database integin_appliance..."
docker exec -i appliance-postgres psql -U integin_owner -d postgres -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = 'integin_appliance' AND pid <> pg_backend_pid();"
docker exec -i appliance-postgres psql -U integin_owner -d postgres -c "DROP DATABASE IF EXISTS integin_appliance;"
docker exec -i appliance-postgres psql -U integin_owner -d postgres -c "CREATE DATABASE integin_appliance;"

$migrationFiles = Get-ChildItem -Path "C:\MY_PROJECT\integin-pilot-source\migrations" -Filter "*.sql" | 
    Where-Object { 
        $_.Name -notmatch '\.down\.' -and 
        $_.Name -notmatch '\.draft\.' -and 
        $_.Name -notmatch '\.candidate\.' 
    } | Sort-Object Name

foreach ($f in $migrationFiles) {
    Write-Host "Applying $($f.Name)..."
    Get-Content -LiteralPath $f.FullName -Raw | docker exec -i appliance-postgres psql -v ON_ERROR_STOP=1 -U integin_owner -d integin_appliance
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Migration failed at $($f.Name)"
        exit 1
    }
}
Write-Host "All migrations applied successfully on clean database!" -ForegroundColor Green

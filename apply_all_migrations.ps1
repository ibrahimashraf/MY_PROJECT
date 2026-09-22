$ErrorActionPreference = "Stop"
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
Write-Host "All migrations applied successfully!" -ForegroundColor Green

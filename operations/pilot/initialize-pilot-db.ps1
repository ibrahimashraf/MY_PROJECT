$sourceRoot = 'C:\MY PROJECT\integin-pilot-source'
$secretsPath = 'C:\MY PROJECT\private\integin-secrets\integin-pilot.env'
$container = 'integin-pilot-postgres'

if (-not (Test-Path -LiteralPath $sourceRoot)) { throw "Pilot source copy is missing: $sourceRoot" }
if (-not (Test-Path -LiteralPath $secretsPath)) { throw "Pilot secrets file is missing: $secretsPath" }

$values = @{}
Get-Content -LiteralPath $secretsPath | ForEach-Object {
  if ($_ -match '^([^#=][^=]*)=(.*)$') { $values[$matches[1]] = $matches[2] }
}

$ownerPassword = $values['PILOT_POSTGRES_PASSWORD']
if ([string]::IsNullOrWhiteSpace($ownerPassword)) { throw 'PILOT_POSTGRES_PASSWORD is missing.' }

$runtimePassword = ([guid]::NewGuid().ToString('N') + [guid]::NewGuid().ToString('N'))
$values['PILOT_RUNTIME_PASSWORD'] = $runtimePassword
$values['INTEGIN_DB_URL'] = "postgres://integin_pilot_runtime:$runtimePassword@127.0.0.1:15432/integin_pilot?sslmode=disable"

$orderedLines = Get-Content -LiteralPath $secretsPath | Where-Object {
  $_ -notmatch '^PILOT_RUNTIME_PASSWORD=' -and $_ -notmatch '^INTEGIN_DB_URL='
}
$orderedLines += "PILOT_RUNTIME_PASSWORD=$runtimePassword"
$orderedLines += "INTEGIN_DB_URL=$($values['INTEGIN_DB_URL'])"
[System.IO.File]::WriteAllLines($secretsPath, $orderedLines, [System.Text.UTF8Encoding]::new($false))

& docker container inspect $container *> $null
if ($LASTEXITCODE -ne 0) { throw "Pilot PostgreSQL container is missing: $container" }

$migrations = @('0001_event_log.sql', '0002_device_trust_sync.sql', '0003_event_log_tenant_rls.sql')
foreach ($migration in $migrations) {
  $source = Join-Path $sourceRoot "migrations\$migration"
  if (-not (Test-Path -LiteralPath $source)) { throw "Missing migration: $source" }
  & docker cp $source "${container}:/tmp/$migration"
  if ($LASTEXITCODE -ne 0) { throw "Failed to copy migration: $migration" }
  & docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -f "/tmp/$migration"
  if ($LASTEXITCODE -ne 0) { throw "Migration failed: $migration" }
}

$sql = (@'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'integin_pilot_runtime') THEN
    CREATE ROLE integin_pilot_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '{0}';
  ELSE
    ALTER ROLE integin_pilot_runtime LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOBYPASSRLS PASSWORD '{0}';
  END IF;
END
$$;
GRANT CONNECT ON DATABASE integin_pilot TO integin_pilot_runtime;
GRANT USAGE ON SCHEMA public TO integin_pilot_runtime;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO integin_pilot_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO integin_pilot_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE integin_pilot_owner IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO integin_pilot_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE integin_pilot_owner IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO integin_pilot_runtime;
'@ -f $runtimePassword)

$sql | & docker exec -i -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot
if ($LASTEXITCODE -ne 0) { throw 'Pilot runtime-role setup failed.' }

$verification = @"
SELECT c.relname AS table_name, c.relrowsecurity AS rls_enabled
FROM pg_class c
JOIN pg_namespace n ON n.oid = c.relnamespace
WHERE n.nspname = 'public'
  AND c.relkind IN ('r', 'p')
  AND c.relname IN ('event_log', 'device_registry', 'authority_package', 'sync_device_state', 'sync_receipt', 'sync_held_transaction')
ORDER BY c.relname;
"@

$verification | & docker exec -i -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot
if ($LASTEXITCODE -ne 0) { throw 'Pilot database verification failed.' }

Write-Output 'PILOT_MIGRATIONS_RUNTIME_ROLE_AND_RLS_VERIFICATION_COMPLETE'

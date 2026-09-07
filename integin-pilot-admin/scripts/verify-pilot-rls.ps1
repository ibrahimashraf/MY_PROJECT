$secretsPath = 'C:\INTEGIN-SECRETS\integin-pilot.env'
$container = 'integin-pilot-postgres'

if (-not (Test-Path -LiteralPath $secretsPath)) { throw "Pilot secrets file is missing: $secretsPath" }

$values = @{}
Get-Content -LiteralPath $secretsPath | ForEach-Object {
  if ($_ -match '^([^#=][^=]*)=(.*)$') { $values[$matches[1]] = $matches[2] }
}

$ownerPassword = $values['PILOT_POSTGRES_PASSWORD']
$runtimePassword = $values['PILOT_RUNTIME_PASSWORD']
if ([string]::IsNullOrWhiteSpace($ownerPassword) -or [string]::IsNullOrWhiteSpace($runtimePassword)) {
  throw 'Pilot owner or runtime database password is missing.'
}

$ownerSql = @'
DELETE FROM event_log WHERE aggregate_type = 'PILOT_RLS';
INSERT INTO event_log (event_id, tenant_id, organization_id, environment, aggregate_type, aggregate_id, aggregate_version, event_type, schema_version, occurred_at, payload)
VALUES
  ('pilot-rls-event-a', 'pilot-tenant-a', 'pilot-organization-a', 'TESTING', 'PILOT_RLS', 'pilot-aggregate-a', 1, 'PilotRLSChecked', 1, now(), '{}'::jsonb),
  ('pilot-rls-event-b', 'pilot-tenant-b', 'pilot-organization-b', 'TESTING', 'PILOT_RLS', 'pilot-aggregate-b', 1, 'PilotRLSChecked', 1, now(), '{}'::jsonb);
'@

& docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -c $ownerSql
if ($LASTEXITCODE -ne 0) { throw 'Failed to create disposable RLS test rows.' }

$noContextCount = ((& docker exec -e "PGPASSWORD=$runtimePassword" $container psql -qAt -v ON_ERROR_STOP=1 -U integin_pilot_runtime -d integin_pilot -c "SELECT count(*) FROM event_log WHERE aggregate_type = 'PILOT_RLS';") -join '').Trim()
$tenantACount = ((& docker exec -e "PGPASSWORD=$runtimePassword" $container psql -qAt -v ON_ERROR_STOP=1 -U integin_pilot_runtime -d integin_pilot -c "BEGIN; SET LOCAL integin.tenant_id = 'pilot-tenant-a'; SELECT count(*) FROM event_log WHERE aggregate_type = 'PILOT_RLS'; COMMIT;") -join '').Trim()
$tenantBCount = ((& docker exec -e "PGPASSWORD=$runtimePassword" $container psql -qAt -v ON_ERROR_STOP=1 -U integin_pilot_runtime -d integin_pilot -c "BEGIN; SET LOCAL integin.tenant_id = 'pilot-tenant-b'; SELECT count(*) FROM event_log WHERE aggregate_type = 'PILOT_RLS'; COMMIT;") -join '').Trim()

& docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -c "DELETE FROM event_log WHERE aggregate_type = 'PILOT_RLS';"
if ($LASTEXITCODE -ne 0) { throw 'Failed to clean up disposable RLS test rows.' }

if ($noContextCount -ne '0') { throw "RLS failed closed without tenant context; observed count: $noContextCount" }
if ($tenantACount -ne '1') { throw "Tenant A context did not isolate one row; observed count: $tenantACount" }
if ($tenantBCount -ne '1') { throw "Tenant B context did not isolate one row; observed count: $tenantBCount" }

Write-Output 'PILOT_RUNTIME_ROLE_RLS_TENANT_CONTEXT_VERIFIED'

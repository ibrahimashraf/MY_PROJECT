$sourceRoot = 'C:\MY PROJECT\integin-pilot-source'
$secretsPath = 'C:\MY PROJECT\private\integin-secrets\integin-pilot.env'
$container = 'integin-pilot-postgres'
$migrationName = '0004_identity_subject_membership.sql'
$migrationPath = Join-Path $sourceRoot (Join-Path 'migrations' $migrationName)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $migrationPath)) { throw "Identity migration is missing: $migrationPath" }
if (-not (Test-Path -LiteralPath $secretsPath)) { throw "Pilot private environment is missing: $secretsPath" }

$settings = @{}
foreach ($line in Get-Content -LiteralPath $secretsPath) {
    if ($line -match '^([A-Za-z_][A-Za-z0-9_]*)=(.*)$') {
        $settings[$matches[1]] = $matches[2]
    }
}
if ([string]::IsNullOrWhiteSpace($settings['PILOT_POSTGRES_PASSWORD'])) { throw 'PILOT_POSTGRES_PASSWORD is required in the pilot private environment.' }

$containerStatus = & docker inspect --format '{{.State.Status}}' $container
if ($LASTEXITCODE -ne 0 -or $containerStatus.Trim() -ne 'running') { throw "The isolated pilot PostgreSQL container $container is not running." }

$ownerPassword = $settings['PILOT_POSTGRES_PASSWORD']
$exists = & docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -tAc "SELECT to_regclass('public.identity_subject') IS NOT NULL"
if ($LASTEXITCODE -ne 0) { throw 'Could not inspect the isolated pilot identity migration state.' }
if ($exists.Trim() -eq 't') { throw 'Identity schema is already present in the isolated pilot database; this one-shot runner will not reapply it.' }

& docker cp $migrationPath "$container`:/tmp/$migrationName"
if ($LASTEXITCODE -ne 0) { throw 'Could not copy the identity migration into the isolated pilot PostgreSQL container.' }

& docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -f "/tmp/$migrationName"
if ($LASTEXITCODE -ne 0) { throw 'Identity migration failed in the isolated pilot database.' }

$verification = @'
SELECT 'identity_subject_exists=' || (to_regclass('public.identity_subject') IS NOT NULL)::text;
SELECT 'identity_membership_exists=' || (to_regclass('public.identity_membership') IS NOT NULL)::text;
SELECT 'identity_membership_capability_exists=' || (to_regclass('public.identity_membership_capability') IS NOT NULL)::text;
SELECT 'runtime_direct_subject_select=' || has_table_privilege('integin_pilot_runtime', 'public.identity_subject', 'SELECT')::text;
SELECT 'runtime_direct_membership_select=' || has_table_privilege('integin_pilot_runtime', 'public.identity_membership', 'SELECT')::text;
SELECT 'runtime_resolver_execute=' || has_function_privilege('integin_pilot_runtime', 'public.integin_resolve_identity_membership(text, text)', 'EXECUTE')::text;
SELECT 'public_resolver_execute=' || EXISTS (
    SELECT 1
      FROM pg_proc p
 CROSS JOIN LATERAL aclexplode(COALESCE(p.proacl, acldefault('f', p.proowner))) AS acl
     WHERE p.oid = 'public.integin_resolve_identity_membership(text, text)'::regprocedure
       AND acl.grantee = 0
       AND acl.privilege_type = 'EXECUTE'
)::text;
'@
$observed = $verification | & docker exec -i -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -tA
if ($LASTEXITCODE -ne 0) { throw 'Could not verify the isolated pilot identity migration boundary.' }
$normalized = @($observed | ForEach-Object { $_.Trim() } | Where-Object { $_ })
$expected = @(
    'identity_subject_exists=true',
    'identity_membership_exists=true',
    'identity_membership_capability_exists=true',
    'runtime_direct_subject_select=false',
    'runtime_direct_membership_select=false',
    'runtime_resolver_execute=true',
    'public_resolver_execute=false'
)
foreach ($item in $expected) {
    if ($normalized -notcontains $item) { throw "Isolated pilot identity migration verification failed: expected $item" }
}

Write-Output 'PILOT_IDENTITY_SCHEMA_AND_RESOLVER_PRIVILEGES_VERIFIED'

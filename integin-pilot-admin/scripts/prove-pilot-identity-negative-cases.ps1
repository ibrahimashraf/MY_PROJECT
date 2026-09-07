$sourceRoot = 'C:\INTEGIN-PILOT\source'
$secretsPath = 'C:\INTEGIN-SECRETS\integin-pilot.env'
$container = 'integin-pilot-postgres'
$issuer = 'https://identity-pilot.invalid/realms/integin-pilot-negative-test'
$subject = 'negative-subject-001'
$tenant = 'tenant-identity-negative-a'
$organization = 'org-identity-negative-a'

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not (Test-Path -LiteralPath $sourceRoot)) { throw "Pilot source is missing: $sourceRoot" }
if (-not (Test-Path -LiteralPath $secretsPath)) { throw "Pilot private environment is missing: $secretsPath" }

$settings = @{}
foreach ($line in Get-Content -LiteralPath $secretsPath) {
    if ($line -match '^([A-Za-z_][A-Za-z0-9_]*)=(.*)$') {
        $settings[$matches[1]] = $matches[2]
    }
}
foreach ($required in @('PILOT_POSTGRES_PASSWORD', 'PILOT_RUNTIME_PASSWORD')) {
    if ([string]::IsNullOrWhiteSpace($settings[$required])) { throw "$required is required in the pilot private environment." }
}

$containerStatus = & docker inspect --format '{{.State.Status}}' $container
if ($LASTEXITCODE -ne 0 -or $containerStatus.Trim() -ne 'running') { throw "The isolated pilot PostgreSQL container $container is not running." }

$ownerPassword = $settings['PILOT_POSTGRES_PASSWORD']
$runtimePassword = $settings['PILOT_RUNTIME_PASSWORD']
$existing = & docker exec -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -tAc "SELECT EXISTS (SELECT 1 FROM identity_subject WHERE issuer = '$issuer' AND subject = '$subject')"
if ($LASTEXITCODE -ne 0) { throw 'Could not verify the pilot negative-test subject precondition.' }
if ($existing.Trim() -ne 'f') { throw 'Pilot negative-test subject already exists; refusing to reuse or modify it.' }

$seedAttempted = $false
try {
    $seedAttempted = $true
    $seed = @"
BEGIN;
WITH inserted_subject AS (
    INSERT INTO identity_subject (issuer, subject) VALUES ('$issuer', '$subject') RETURNING subject_id
), inserted_membership AS (
    INSERT INTO identity_membership (subject_id, tenant_id, organization_id) SELECT subject_id, '$tenant', '$organization' FROM inserted_subject RETURNING membership_id
)
INSERT INTO identity_membership_capability (membership_id, capability)
SELECT membership_id, 'inspection.read' FROM inserted_membership;
DO 'BEGIN
    BEGIN
        INSERT INTO identity_membership (subject_id, tenant_id, organization_id)
        SELECT subject_id, ''tenant-identity-negative-b'', ''org-identity-negative-b''
          FROM identity_subject
         WHERE issuer = ''$issuer'' AND subject = ''$subject'';
    EXCEPTION WHEN unique_violation THEN
        NULL;
    END;
    IF (SELECT count(*) FROM identity_membership m JOIN identity_subject s ON s.subject_id = m.subject_id WHERE s.issuer = ''$issuer'' AND s.subject = ''$subject'' AND m.status = ''ACTIVE'') <> 1 THEN
        RAISE EXCEPTION ''one active local membership invariant was not preserved'';
    END IF;
END';
SELECT 'wrong_issuer_rows=' || count(*) FROM integin_resolve_identity_membership('$issuer/wrong', '$subject');
SELECT 'missing_capability_present=' || EXISTS (SELECT 1 FROM integin_resolve_identity_membership('$issuer', '$subject') WHERE 'inspection.approve' = ANY(capabilities))::text;
COMMIT;
"@
    $seedOutput = $seed | & docker exec -i -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot -tA
    if ($LASTEXITCODE -ne 0) { throw 'Could not seed and verify the isolated pilot negative-test identity.' }
    $seedNormalized = @($seedOutput | ForEach-Object { $_.Trim() } | Where-Object { $_ })
    foreach ($item in @('wrong_issuer_rows=0', 'missing_capability_present=false')) {
        if ($seedNormalized -notcontains $item) { throw "Local identity negative check failed: expected $item" }
    }

    $savedErrorActionPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = 'Continue'
        $runtimeRead = & docker exec -e "PGPASSWORD=$runtimePassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_runtime -d integin_pilot -tAc "SELECT count(*) FROM identity_subject" 2>$null
        $runtimeReadExit = $LASTEXITCODE
    } finally {
        $ErrorActionPreference = $savedErrorActionPreference
    }
    if ($runtimeReadExit -eq 0) { throw 'Pilot runtime role unexpectedly read local identity tables directly.' }
    $runtimeResolution = & docker exec -e "PGPASSWORD=$runtimePassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_runtime -d integin_pilot -tAc "SELECT tenant_id || '|' || organization_id || '|' || COALESCE(array_to_string(capabilities, ','), '') FROM integin_resolve_identity_membership('$issuer', '$subject')"
    if ($LASTEXITCODE -ne 0) { throw 'Pilot runtime role could not execute the local identity resolver.' }
    if ($runtimeResolution.Trim() -ne "$tenant|$organization|inspection.read") { throw 'Pilot runtime resolver returned an unexpected local membership.' }

    Write-Output 'PILOT_IDENTITY_NEGATIVE_CASES_VERIFIED'
} finally {
    if ($seedAttempted) {
        $cleanup = "DELETE FROM identity_membership WHERE subject_id IN (SELECT subject_id FROM identity_subject WHERE issuer = '$issuer' AND subject = '$subject'); DELETE FROM identity_subject WHERE issuer = '$issuer' AND subject = '$subject';"
        $null = $cleanup | & docker exec -i -e "PGPASSWORD=$ownerPassword" $container psql -v ON_ERROR_STOP=1 -U integin_pilot_owner -d integin_pilot
        if ($LASTEXITCODE -ne 0) { throw 'Pilot negative-test cleanup failed; synthetic identity rows require immediate manual cleanup.' }
    }
}

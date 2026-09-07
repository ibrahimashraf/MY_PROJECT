<#
INTEGIN controlled OIDC session-matrix runner.

Scope: pilot only. This script creates a temporary Keycloak service-account client,
tests the enabled /identity/session boundary, removes synthetic mappings and the
client, then restores the pilot to disabled OIDC mode. It never touches acceptance
runtime, acceptance credentials, OpenBao, or production resources.
#>

[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'

$sourceRoot = 'C:\MY PROJECT\integin-pilot-source'
$operationsRoot = 'C:\MY PROJECT\operations\pilot'
$secretsRoot = 'C:\MY PROJECT\private\integin-secrets'
$pilotEnvironmentPath = Join-Path $secretsRoot 'integin-pilot.env'
$keycloakEnvironmentPath = Join-Path $secretsRoot 'keycloak-pilot-zip-runtime.env'
$runtimeRoot = Join-Path $operationsRoot 'runtime'
$binaryPath = Join-Path $runtimeRoot 'integin-server-pilot-oidc-matrix.exe'
$stdoutPath = Join-Path $runtimeRoot 'oidc-matrix.stdout.log'
$stderrPath = Join-Path $runtimeRoot 'oidc-matrix.stderr.log'
$normalLauncher = Join-Path $operationsRoot 'launch-pilot-server.ps1'
$pilotPort = 18080
$pilotBaseUrl = "http://127.0.0.1:$pilotPort"
$keycloakBaseUrl = 'http://127.0.0.1:18180'
$keycloakManagementUrl = 'http://127.0.0.1:19090'
$realm = 'integin-pilot'
$issuer = "$keycloakBaseUrl/realms/$realm"
$audience = 'integin-field-pilot'
$probeClientId = 'integin-oidc-route-probe-' + [Guid]::NewGuid().ToString('N').Substring(0, 12)
$probeTenant = 'tenant-oidc-session-matrix'
$probeOrganization = 'org-oidc-session-matrix'
$enabledProcess = $null
$probeClientInternalId = $null
$adminHeaders = $null
$databaseStopped = $false
$finalFailure = $null

function Read-PrivateEnvironment {
    param([Parameter(Mandatory = $true)][string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "Required private environment file is unavailable: $Path"
    }

    $values = @{}
    foreach ($line in Get-Content -LiteralPath $Path) {
        $trimmed = $line.Trim()
        if ($trimmed.Length -eq 0 -or $trimmed.StartsWith('#')) {
            continue
        }
        $separator = $trimmed.IndexOf('=')
        if ($separator -lt 1) {
            throw "Invalid private environment entry in $Path"
        }
        $values[$trimmed.Substring(0, $separator)] = $trimmed.Substring($separator + 1)
    }
    return $values
}

function Require-EnvironmentKeys {
    param(
        [Parameter(Mandatory = $true)][hashtable]$Values,
        [Parameter(Mandatory = $true)][string[]]$Keys
    )

    foreach ($key in $Keys) {
        if ([string]::IsNullOrWhiteSpace([string]$Values[$key])) {
            throw "Required private setting $key is absent"
        }
    }
}

function Get-HttpStatus {
    param(
        [Parameter(Mandatory = $true)][string]$Uri,
        [string]$Method = 'GET',
        [hashtable]$Headers = @{}
    )

    try {
        $response = Invoke-WebRequest -UseBasicParsing -TimeoutSec 12 -Method $Method -Uri $Uri -Headers $Headers
        return [int]$response.StatusCode
    } catch {
        # PowerShell 5 and 7 surface non-success HTTP responses through
        # different exception shapes.  Negative cases in this matrix expect
        # 401/403/404, so normalize either shape to a status code.
        $response = $_.Exception.Response
        if ($null -ne $response -and $null -ne $response.StatusCode) {
            return [int]$response.StatusCode
        }
        if ($_.Exception.PSObject.Properties.Name -contains 'StatusCode') {
            return [int]$_.Exception.StatusCode
        }
        throw
    }
}

function Assert-Status {
    param(
        [Parameter(Mandatory = $true)][string]$Name,
        [Parameter(Mandatory = $true)][int]$Actual,
        [Parameter(Mandatory = $true)][int]$Expected
    )

    if ($Actual -ne $Expected) {
        throw "$Name returned HTTP $Actual; expected HTTP $Expected"
    }
    Write-Output "$Name=HTTP_$Expected"
}

function Stop-EnabledPilotProcess {
    if ($null -eq $enabledProcess) {
        return
    }
    if (-not $enabledProcess.HasExited) {
        Stop-Process -Id $enabledProcess.Id -Force -ErrorAction Stop
        $enabledProcess.WaitForExit(10000) | Out-Null
    }
    $script:enabledProcess = $null
}

function Start-EnabledPilot {
    param([Parameter(Mandatory = $true)][hashtable]$PilotValues)

    $listener = Get-NetTCPConnection -LocalAddress '127.0.0.1' -LocalPort $pilotPort -State Listen -ErrorAction SilentlyContinue
    if ($listener) {
        $existing = Get-Process -Id $listener.OwningProcess -ErrorAction Stop
        if ($existing.ProcessName -ne 'integin-server-pilot') {
            throw "Refusing to replace unexpected process $($existing.ProcessName) on pilot port $pilotPort"
        }
        Stop-Process -Id $existing.Id -ErrorAction Stop
        Start-Sleep -Seconds 2
    }

    Push-Location $sourceRoot
    try {
        & go build -o $binaryPath .\cmd\integin-server
        if ($LASTEXITCODE -ne 0) {
            throw 'OIDC matrix pilot build failed'
        }
    } finally {
        Pop-Location
    }

    $processEnvironment = @{
        INTEGIN_HTTP_ADDR = "127.0.0.1:$pilotPort"
        INTEGIN_TENANT_ID = $PilotValues['INTEGIN_TENANT_ID']
        INTEGIN_DB_URL = $PilotValues['INTEGIN_DB_URL']
        INTEGIN_EVIDENCE_STORE = 'rustfs'
        INTEGIN_S3_ENDPOINT = $PilotValues['INTEGIN_S3_ENDPOINT']
        INTEGIN_S3_BUCKET = $PilotValues['INTEGIN_S3_BUCKET']
        INTEGIN_S3_ACCESS_KEY = $PilotValues['INTEGIN_S3_ACCESS_KEY']
        INTEGIN_S3_SECRET_KEY = $PilotValues['INTEGIN_S3_SECRET_KEY']
        INTEGIN_S3_REGION = $PilotValues['INTEGIN_S3_REGION']
        INTEGIN_SYNC_SECRET = $PilotValues['PILOT_SYNC_SECRET']
        INTEGIN_LOCAL_PROVISIONING_ENABLED = 'false'
        INTEGIN_OIDC_ENABLED = 'true'
        INTEGIN_OIDC_ISSUER = $issuer
        INTEGIN_OIDC_AUDIENCE = $audience
        INTEGIN_OIDC_AUTHORIZED_PARTY = $probeClientId
        INTEGIN_OIDC_ALLOW_INSECURE_LOOPBACK = 'true'
    }

    $savedEnvironment = @{}
    foreach ($entry in $processEnvironment.GetEnumerator()) {
        $savedEnvironment[$entry.Key] = [Environment]::GetEnvironmentVariable($entry.Key, 'Process')
        [Environment]::SetEnvironmentVariable($entry.Key, [string]$entry.Value, 'Process')
    }

    try {
        New-Item -ItemType Directory -Force -Path $runtimeRoot | Out-Null
        $script:enabledProcess = Start-Process -FilePath $binaryPath -WorkingDirectory $runtimeRoot -RedirectStandardOutput $stdoutPath -RedirectStandardError $stderrPath -PassThru
    } finally {
        foreach ($entry in $savedEnvironment.GetEnumerator()) {
            [Environment]::SetEnvironmentVariable($entry.Key, $entry.Value, 'Process')
        }
    }

    Start-Sleep -Seconds 4
    Assert-Status -Name 'enabled-pilot-health' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/healthz") -Expected 200
    Assert-Status -Name 'enabled-pilot-readiness' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/readyz") -Expected 200
}

function Get-KeycloakAdminHeaders {
    param([Parameter(Mandatory = $true)][hashtable]$KeycloakValues)

    $body = @{
        client_id = 'admin-cli'
        grant_type = 'password'
        username = $KeycloakValues['KC_BOOTSTRAP_ADMIN_USERNAME']
        password = $KeycloakValues['KC_BOOTSTRAP_ADMIN_PASSWORD']
    }
    $tokenResponse = Invoke-RestMethod -UseBasicParsing -Method Post -Uri "$keycloakBaseUrl/realms/master/protocol/openid-connect/token" -ContentType 'application/x-www-form-urlencoded' -Body $body -TimeoutSec 15
    $token = [string]$tokenResponse.access_token
    if ([string]::IsNullOrWhiteSpace($token)) {
        throw 'Keycloak admin token request did not return a token'
    }
    return @{ Authorization = "Bearer $token" }
}

function New-ProbeClient {
    param([Parameter(Mandatory = $true)][hashtable]$Headers)

    $client = @{
        clientId = $probeClientId
        name = 'INTEGIN temporary OIDC session-matrix probe'
        protocol = 'openid-connect'
        enabled = $true
        publicClient = $false
        standardFlowEnabled = $false
        directAccessGrantsEnabled = $false
        implicitFlowEnabled = $false
        serviceAccountsEnabled = $true
        authorizationServicesEnabled = $false
        protocolMappers = @(
            @{
                name = 'integin-required-audience'
                protocol = 'openid-connect'
                protocolMapper = 'oidc-audience-mapper'
                consentRequired = $false
                config = @{
                    'included.client.audience' = $audience
                    'id.token.claim' = 'false'
                    'access.token.claim' = 'true'
                    'introspection.token.claim' = 'true'
                }
            }
        )
    } | ConvertTo-Json -Depth 8

    Invoke-RestMethod -UseBasicParsing -Method Post -Uri "$keycloakBaseUrl/admin/realms/$realm/clients" -Headers $Headers -ContentType 'application/json' -Body $client -TimeoutSec 15 | Out-Null
    $clients = @(Invoke-RestMethod -UseBasicParsing -Method Get -Uri "$keycloakBaseUrl/admin/realms/$realm/clients?clientId=$probeClientId" -Headers $Headers -TimeoutSec 15)
    if ($clients.Count -ne 1 -or [string]::IsNullOrWhiteSpace([string]$clients[0].id)) {
        throw 'Temporary Keycloak probe client lookup was not unique'
    }
    $script:probeClientInternalId = [string]$clients[0].id
}

function Get-ProbeAccessToken {
    param([Parameter(Mandatory = $true)][hashtable]$Headers)

    $secretResponse = Invoke-RestMethod -UseBasicParsing -Method Post -Uri "$keycloakBaseUrl/admin/realms/$realm/clients/$probeClientInternalId/client-secret" -Headers $Headers -TimeoutSec 15
    $secret = [string]$secretResponse.value
    if ([string]::IsNullOrWhiteSpace($secret)) {
        throw 'Temporary Keycloak probe client secret was unavailable'
    }
    $body = @{
        grant_type = 'client_credentials'
        client_id = $probeClientId
        client_secret = $secret
    }
    $tokenResponse = Invoke-RestMethod -UseBasicParsing -Method Post -Uri "$issuer/protocol/openid-connect/token" -ContentType 'application/x-www-form-urlencoded' -Body $body -TimeoutSec 15
    $token = [string]$tokenResponse.access_token
    if ([string]::IsNullOrWhiteSpace($token)) {
        throw 'Temporary Keycloak probe token request did not return a token'
    }
    return $token
}

function Get-JwtSubject {
    param([Parameter(Mandatory = $true)][string]$Token)

    $segments = $Token.Split('.')
    if ($segments.Length -ne 3) {
        throw 'Probe access token is not a compact JWT'
    }
    $payload = $segments[1].Replace('-', '+').Replace('_', '/')
    switch ($payload.Length % 4) {
        2 { $payload += '==' }
        3 { $payload += '=' }
    }
    $claims = [System.Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($payload)) | ConvertFrom-Json
    $subject = [string]$claims.sub
    if ([string]::IsNullOrWhiteSpace($subject)) {
        throw 'Probe access token contains no subject'
    }
    return $subject
}

function ConvertTo-SqlLiteral {
    param([Parameter(Mandatory = $true)][string]$Value)
    return "'" + $Value.Replace("'", "''") + "'"
}

function Invoke-PilotSql {
    param([Parameter(Mandatory = $true)][string]$Sql)
    & docker exec integin-pilot-postgres psql -U integin_pilot_owner -d integin_pilot -v ON_ERROR_STOP=1 -c $Sql | Out-Null
    if ($LASTEXITCODE -ne 0) {
        throw 'Pilot identity SQL operation failed'
    }
}

function Remove-ProbeIdentityMapping {
    param([Parameter(Mandatory = $true)][string]$Subject)
    $issuerLiteral = ConvertTo-SqlLiteral $issuer
    $subjectLiteral = ConvertTo-SqlLiteral $Subject
    # Memberships reference subjects with ON DELETE RESTRICT. Remove the
    # dependent membership first (its capabilities cascade), then the subject.
    $sql = "DELETE FROM public.identity_membership WHERE subject_id IN (SELECT subject_id FROM public.identity_subject WHERE issuer=$issuerLiteral AND subject=$subjectLiteral); DELETE FROM public.identity_subject WHERE issuer=$issuerLiteral AND subject=$subjectLiteral;"
    Invoke-PilotSql -Sql $sql
}

function Add-ProbeIdentityMembership {
    param(
        [Parameter(Mandatory = $true)][string]$Subject,
        [Parameter(Mandatory = $true)][bool]$WithCapability
    )

    Remove-ProbeIdentityMapping -Subject $Subject
    $issuerLiteral = ConvertTo-SqlLiteral $issuer
    $subjectLiteral = ConvertTo-SqlLiteral $Subject
    $tenantLiteral = ConvertTo-SqlLiteral $probeTenant
    $organizationLiteral = ConvertTo-SqlLiteral $probeOrganization
    $sql = "INSERT INTO public.identity_subject (issuer, subject, status) VALUES ($issuerLiteral, $subjectLiteral, 'ACTIVE'); INSERT INTO public.identity_membership (subject_id, tenant_id, organization_id, status) SELECT subject_id, $tenantLiteral, $organizationLiteral, 'ACTIVE' FROM public.identity_subject WHERE issuer=$issuerLiteral AND subject=$subjectLiteral;"
    Invoke-PilotSql -Sql $sql
    if ($WithCapability) {
        $sql = "INSERT INTO public.identity_membership_capability (membership_id, capability, status) SELECT m.membership_id, 'identity.session.read', 'ACTIVE' FROM public.identity_membership m JOIN public.identity_subject s ON s.subject_id=m.subject_id WHERE s.issuer=$issuerLiteral AND s.subject=$subjectLiteral;"
        Invoke-PilotSql -Sql $sql
    }
}

function Restore-PilotDisabled {
    Stop-EnabledPilotProcess
    $listener = Get-NetTCPConnection -LocalAddress '127.0.0.1' -LocalPort $pilotPort -State Listen -ErrorAction SilentlyContinue
    if (-not $listener) {
        & powershell -ExecutionPolicy Bypass -File $normalLauncher
        if ($LASTEXITCODE -ne 0) {
            throw 'Disabled-mode pilot launcher failed during restoration'
        }
    } else {
        $existing = Get-Process -Id $listener.OwningProcess -ErrorAction Stop
        if ($existing.ProcessName -ne 'integin-server-pilot') {
            throw "Refusing to trust unexpected process $($existing.ProcessName) on pilot port $pilotPort during disabled restoration"
        }
    }
    Assert-Status -Name 'disabled-pilot-health' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/healthz") -Expected 200
    Assert-Status -Name 'disabled-pilot-readiness' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/readyz") -Expected 200
    Assert-Status -Name 'disabled-pilot-session-route' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session") -Expected 404
}

try {
    $pilotValues = Read-PrivateEnvironment -Path $pilotEnvironmentPath
    Require-EnvironmentKeys -Values $pilotValues -Keys @('INTEGIN_DB_URL', 'INTEGIN_TENANT_ID', 'PILOT_SYNC_SECRET', 'INTEGIN_S3_ENDPOINT', 'INTEGIN_S3_BUCKET', 'INTEGIN_S3_ACCESS_KEY', 'INTEGIN_S3_SECRET_KEY', 'INTEGIN_S3_REGION')
    $keycloakValues = Read-PrivateEnvironment -Path $keycloakEnvironmentPath
    Require-EnvironmentKeys -Values $keycloakValues -Keys @('KC_BOOTSTRAP_ADMIN_USERNAME', 'KC_BOOTSTRAP_ADMIN_PASSWORD')

    Assert-Status -Name 'acceptance-control-before-matrix' -Actual (Get-HttpStatus -Uri 'http://127.0.0.1:8080/readyz') -Expected 200
    Assert-Status -Name 'keycloak-before-matrix' -Actual (Get-HttpStatus -Uri "$keycloakManagementUrl/health/ready") -Expected 200

    $script:adminHeaders = Get-KeycloakAdminHeaders -KeycloakValues $keycloakValues
    New-ProbeClient -Headers $adminHeaders
    $token = Get-ProbeAccessToken -Headers $adminHeaders
    $subject = Get-JwtSubject -Token $token
    $bearerHeaders = @{ Authorization = "Bearer $token" }

    Start-EnabledPilot -PilotValues $pilotValues
    Assert-Status -Name 'oidc-missing-bearer' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session") -Expected 401
    Assert-Status -Name 'oidc-invalid-bearer' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers @{ Authorization = 'Bearer invalid.token.value' }) -Expected 401
    Assert-Status -Name 'oidc-unknown-local-subject' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers $bearerHeaders) -Expected 403

    Add-ProbeIdentityMembership -Subject $subject -WithCapability $false
    Assert-Status -Name 'oidc-missing-local-capability' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers $bearerHeaders) -Expected 403

    Add-ProbeIdentityMembership -Subject $subject -WithCapability $true
    Assert-Status -Name 'oidc-authorized-local-session' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers $bearerHeaders) -Expected 200

    & docker stop integin-pilot-postgres | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Pilot PostgreSQL stop failed during resolver-outage check' }
    $script:databaseStopped = $true
    Start-Sleep -Seconds 2
    Assert-Status -Name 'oidc-resolver-outage' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers $bearerHeaders) -Expected 503

    & docker start integin-pilot-postgres | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Pilot PostgreSQL restart failed after resolver-outage check' }
    $script:databaseStopped = $false
    $ready = $false
    for ($attempt = 0; $attempt -lt 12; $attempt++) {
        & docker exec integin-pilot-postgres pg_isready -U integin_pilot_owner -d integin_pilot | Out-Null
        if ($LASTEXITCODE -eq 0) { $ready = $true; break }
        Start-Sleep -Seconds 2
    }
    if (-not $ready) { throw 'Pilot PostgreSQL did not become ready after resolver-outage check' }
    Assert-Status -Name 'oidc-session-after-database-recovery' -Actual (Get-HttpStatus -Uri "$pilotBaseUrl/identity/session" -Headers $bearerHeaders) -Expected 200
    Write-Output 'OIDC_PILOT_SESSION_MATRIX_PASSED'
} catch {
    $finalFailure = $_
} finally {
    if ($databaseStopped) {
        & docker start integin-pilot-postgres | Out-Null
    }
    if ($probeClientInternalId -and $adminHeaders) {
        try {
            Invoke-RestMethod -UseBasicParsing -Method Delete -Uri "$keycloakBaseUrl/admin/realms/$realm/clients/$probeClientInternalId" -Headers $adminHeaders -TimeoutSec 15 | Out-Null
        } catch {
            Write-Output 'OIDC_MATRIX_WARNING_KEYCLOAK_PROBE_CLIENT_CLEANUP_FAILED'
            if (-not $finalFailure) { $finalFailure = $_ }
        }
    }
    if (Get-Variable -Name subject -ErrorAction SilentlyContinue) {
        try {
            Remove-ProbeIdentityMapping -Subject $subject
        } catch {
            Write-Output 'OIDC_MATRIX_WARNING_LOCAL_IDENTITY_CLEANUP_FAILED'
            if (-not $finalFailure) { $finalFailure = $_ }
        }
    }
    try {
        Restore-PilotDisabled
    } catch {
        Write-Output 'OIDC_MATRIX_WARNING_DISABLED_RESTORATION_FAILED'
        if (-not $finalFailure) { $finalFailure = $_ }
    }
    $token = $null
    $adminHeaders = $null
}

if ($finalFailure) {
    throw $finalFailure
}

Write-Output 'OIDC_PILOT_SESSION_MATRIX_CLEANUP_AND_DISABLED_RESTORATION_VERIFIED'

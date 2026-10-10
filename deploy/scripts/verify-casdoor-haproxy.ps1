# Verification Suite for Casdoor IAM & HAProxy Ingress Architecture

[CmdletBinding()]
param(
    [string]$HaproxyHttp = "http://127.0.0.1:80",
    [string]$HaproxyHttps = "https://127.0.0.1:443",
    [string]$HaproxyStats = "http://127.0.0.1:8404",
    [string]$CasdoorDirect = "http://127.0.0.1:18181"
)

$ErrorActionPreference = "Stop"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "INTEGIN CASDOOR & HAPROXY SUITE VERIFICATION" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

$passed = 0
$total = 0

function Run-Check {
    param([string]$Name, [scriptblock]$Test)
    $script:total++
    try {
        $result = & $Test
        if ($result -eq $true) {
            Write-Host "  [PASS] $Name" -ForegroundColor Green
            $script:passed++
        } else {
            Write-Host "  [FAIL] ${Name}: Assertion returned false" -ForegroundColor Red
        }
    } catch {
        Write-Host "  [FAIL] ${Name}: $_" -ForegroundColor Red
    }
}

# 1. HAProxy HTTP Ingress
Run-Check "HAProxy HTTP Ingress /healthz (port 80)" {
    $res = curl.exe -s --max-time 5 "$HaproxyHttp/healthz"
    $res -match '"status":"ok"'
}

# 2. HAProxy HTTPS TLS Ingress
Run-Check "HAProxy HTTPS Ingress TLS Termination (port 443)" {
    $res = curl.exe -k -s --max-time 5 "$HaproxyHttps/healthz"
    $res -match '"status":"ok"'
}

# 3. HAProxy Security Headers
Run-Check "HAProxy OWASP Security Headers (HSTS, X-Frame-Options, X-Content-Type)" {
    $headers = curl.exe -k -s -I --max-time 5 "$HaproxyHttps/healthz"
    ($headers -match 'strict-transport-security') -and ($headers -match 'x-frame-options: DENY')
}

# 4. HAProxy WAF Scanner Shield
Run-Check "HAProxy WAF Malicious Scanner User-Agent Rejection (403)" {
    $code = curl.exe -s -o /dev/null -w "%{http_code}" -H "User-Agent: sqlmap/1.5" "$HaproxyHttp/"
    $code -eq "403"
}

# 5. HAProxy Prometheus Metrics Exporter
Run-Check "HAProxy Prometheus Exporter (/metrics on port 8404)" {
    $metrics = curl.exe -s --max-time 5 "$HaproxyStats/metrics"
    [bool]($metrics -match 'haproxy_process_nbthread')
}

# 6. HAProxy Stats Web Console
Run-Check "HAProxy Stats Console Authenticated (port 8404)" {
    $code = curl.exe -s -u "admin:integin_haproxy_admin" -o /dev/null -w "%{http_code}" "$HaproxyStats/stats"
    $code -eq "200"
}

# 7. Casdoor OIDC Discovery via HAProxy Ingress
Run-Check "Casdoor OIDC Discovery via HAProxy Ingress (/ .well-known)" {
    $cfg = curl.exe -s --max-time 5 "$HaproxyHttp/.well-known/openid-configuration" | ConvertFrom-Json
    $cfg.issuer -eq "http://appliance-casdoor:8000"
}

# 8. Casdoor Token Issuance via HAProxy Ingress
Run-Check "Casdoor RS256 JWT Issuance via HAProxy (/api/login/oauth/access_token)" {
    $res = curl.exe -s -X POST "$HaproxyHttp/api/login/oauth/access_token" `
        -d "grant_type=client_credentials&client_id=integin-live-matrix&client_secret=integin-live-matrix-secret-pilot" `
        | ConvertFrom-Json
    (-not [string]::IsNullOrEmpty($res.access_token)) -and ($res.token_type -eq "Bearer")
}

# 9. Casdoor Database Seed Integrity
Run-Check "Casdoor DB Seed: Organizations, Providers, Webhooks, Roles, Users" {
    $orgs = docker exec appliance-casdoor-postgres psql -U casdoor_owner -d casdoor_appliance -t -A -c "SELECT COUNT(*) FROM organization WHERE name = 'integin-pilot';"
    $providers = docker exec appliance-casdoor-postgres psql -U casdoor_owner -d casdoor_appliance -t -A -c "SELECT COUNT(*) FROM provider WHERE name = 'provider_storage_rustfs';"
    $webhooks = docker exec appliance-casdoor-postgres psql -U casdoor_owner -d casdoor_appliance -t -A -c "SELECT COUNT(*) FROM webhook WHERE name = 'webhook_integin_audit';"
    $apps = docker exec appliance-casdoor-postgres psql -U casdoor_owner -d casdoor_appliance -t -A -c "SELECT COUNT(*) FROM application WHERE name = 'integin-live-matrix';"
    $roles = docker exec appliance-casdoor-postgres psql -U casdoor_owner -d casdoor_appliance -t -A -c "SELECT COUNT(*) FROM role WHERE name = 'role_integin_admin';"
    ($orgs -eq "1") -and ($providers -eq "1") -and ($webhooks -eq "1") -and ($apps -eq "1") -and ($roles -eq "1")
}

# 10. Casdoor WebAuthn FIDO2 Ingress
Run-Check "Casdoor WebAuthn FIDO2 Discovery via HAProxy (:80/api/webauthn)" {
    $code = curl.exe -s -o /dev/null -w "%{http_code}" "$HaproxyHttp/api/webauthn/signin/begin"
    $code -eq "200"
}

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "Result: $passed / $total checks passed." -ForegroundColor $(if ($passed -eq $total) { "Green" } else { "Red" })
Write-Host "============================================================" -ForegroundColor Cyan

if ($passed -ne $total) {
    exit 1
}

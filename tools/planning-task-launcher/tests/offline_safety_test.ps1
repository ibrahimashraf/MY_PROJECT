$ErrorActionPreference = 'Stop'

$launcherRoot = Split-Path -Parent $PSScriptRoot
$launcher = Join-Path $launcherRoot 'scripts\planning_task_launcher.py'
$bridge = 'C:\MY_PROJECT\tools\planning-bridge\scripts\planning_bridge.py'
$scratch = Join-Path $launcherRoot 'test-scratch'
$boundRoot = Join-Path $scratch 'bound-root'
$configPath = Join-Path $scratch 'launcher-config.json'
$receiptPath = Join-Path $launcherRoot 'receipts\offline-safety-test.json'
$eventLog = Join-Path $launcherRoot 'launcher-events.jsonl'
$oldLogPresent = Test-Path -LiteralPath $eventLog
$oldLog = if ($oldLogPresent) { [System.IO.File]::ReadAllBytes($eventLog) } else { $null }

function Invoke-Launcher {
    param([string[]]$Arguments, [int]$ExpectedExit)
    $output = & python $launcher @Arguments 2>&1
    if ($LASTEXITCODE -ne $ExpectedExit) {
        throw "Expected launcher exit $ExpectedExit but received ${LASTEXITCODE}: $output"
    }
    return ($output | Out-String).Trim()
}

try {
    Remove-Item -LiteralPath $scratch -Force -Recurse -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Path (Join-Path $boundRoot '.planning') -Force | Out-Null
    Set-Content -LiteralPath (Join-Path $boundRoot 'task_plan.md') -Value "- [x] disposable planning item" -Encoding utf8
    Set-Content -LiteralPath (Join-Path $boundRoot 'findings.md') -Value 'disposable finding' -Encoding utf8
    Set-Content -LiteralPath (Join-Path $boundRoot 'progress.md') -Value 'disposable progress' -Encoding utf8
    Set-Content -LiteralPath (Join-Path $boundRoot 'planning-request.md') -Value 'Create no task during offline test.' -Encoding utf8

    @{
        enabled = $true
        project_root = $boundRoot
        max_context_chars = 512
        completion_timeout_seconds = 3
        fail_open_completion = $true
    } | ConvertTo-Json | Set-Content -LiteralPath (Join-Path $boundRoot '.planning\automation.json') -Encoding utf8

    $config = @{
        enabled = $false
        project_root = $boundRoot
        bridge_script = $bridge
        api_base_url = 'https://api.manus.ai'
        api_key_env = 'MANUS_PLANNING_LAUNCHER_API_KEY'
        allowed_project_ids = @('offline-test-project')
        network_timeout_seconds = 3
        poll_interval_seconds = 1
        max_prompt_chars = 1000
    }
    $config | ConvertTo-Json | Set-Content -LiteralPath $configPath -Encoding utf8

    $validate = Invoke-Launcher -Arguments @('validate-config', '--config', $configPath) -ExpectedExit 0 | ConvertFrom-Json
    if ($validate.enabled -ne $false -or $validate.valid -ne $true) { throw 'Disabled launcher config did not validate correctly.' }

    $disabled = Invoke-Launcher -Arguments @('preflight', '--config', $configPath) -ExpectedExit 2 | ConvertFrom-Json
    if ($disabled.reason -ne 'launcher is disabled') { throw 'Disabled launcher preflight did not fail safely.' }

    $config.enabled = $true
    $config | ConvertTo-Json | Set-Content -LiteralPath $configPath -Encoding utf8
    $prepared = Invoke-Launcher -Arguments @('preflight', '--config', $configPath) -ExpectedExit 0 | ConvertFrom-Json
    if ($prepared.prepared -ne $true) { throw 'Disposable enabled preflight did not prepare bounded context.' }

    $unconfirmed = Invoke-Launcher -Arguments @('create', '--config', $configPath, '--project-id', 'offline-test-project', '--prompt-file', (Join-Path $boundRoot 'planning-request.md')) -ExpectedExit 2 | ConvertFrom-Json
    if ($unconfirmed.reason -ne 'refusing task creation without --confirm-create') { throw 'Task creation confirmation guard did not fail safely.' }

    New-Item -ItemType File -Path (Join-Path $boundRoot '.planning\automation-disabled') -Force | Out-Null
    $bypassed = Invoke-Launcher -Arguments @('preflight', '--config', $configPath) -ExpectedExit 2 | ConvertFrom-Json
    if ($bypassed.reason -notmatch 'local emergency bypass is active') { throw 'Emergency bypass did not prevent planning context preparation.' }
    Remove-Item -LiteralPath (Join-Path $boundRoot '.planning\automation-disabled') -Force

    New-Item -ItemType Directory -Path (Split-Path -Parent $receiptPath) -Force | Out-Null
    @{ task_id = 'offline-safety-task'; completion_review = 'pending' } | ConvertTo-Json | Set-Content -LiteralPath $receiptPath -Encoding utf8
    Remove-Item Env:MANUS_PLANNING_LAUNCHER_API_KEY -ErrorAction SilentlyContinue
    $noCredential = Invoke-Launcher -Arguments @('monitor', '--config', $configPath, '--receipt', $receiptPath) -ExpectedExit 2 | ConvertFrom-Json
    if ($noCredential.reason -ne 'API credential environment variable MANUS_PLANNING_LAUNCHER_API_KEY is not set') { throw 'Missing API credential did not fail before task-status access.' }

    [pscustomobject]@{
        compiled_launcher = $true
        disabled_preflight_declined = $true
        disposable_preflight_prepared = $true
        unconfirmed_create_declined = $true
        emergency_bypass_declined = $true
        missing_credential_declined = $true
        api_task_created = $false
        protected_runtime_accessed = $false
    } | ConvertTo-Json -Compress
    exit 0
}
finally {
    Remove-Item -LiteralPath $scratch -Force -Recurse -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $receiptPath -Force -ErrorAction SilentlyContinue
    if ($oldLogPresent) {
        [System.IO.File]::WriteAllBytes($eventLog, $oldLog)
    } else {
        Remove-Item -LiteralPath $eventLog -Force -ErrorAction SilentlyContinue
    }
}

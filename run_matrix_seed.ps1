$env:INTEGIN_DB_URL = "postgres://integin_owner:integin_sovereign_db_secret@127.0.0.1:6432/integin_appliance?sslmode=disable&default_query_exec_mode=simple_protocol"
$env:INTEGIN_LIVE_FIXTURE_FILE = "C:\MY_PROJECT\private\integin-secrets\integin-live-fixture.json"
$env:INTEGIN_TENANT_ID = "integin-integration-tenant"
$env:INTEGIN_SYNC_SECRET = "299fe914a8994ce584940439d6fb6be4412c7cef8a904ebf8f158cce3208f622"
$env:INTEGIN_OIDC_TOKEN_ENDPOINT = "http://127.0.0.1:18180/api/login/oauth/access_token"
$env:INTEGIN_OIDC_CLIENT_ID = "76a5e92c546b3fcdc2c4"
$env:INTEGIN_OIDC_CLIENT_SECRET = "a55092b9a311e7f435a38a8b73ff2b0c5851fb1a"
& "C:\Users\hima3\AppData\Local\Temp\opencode\integin-live-matrix.exe" seed
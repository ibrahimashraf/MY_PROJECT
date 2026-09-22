$body = @{
    grant_type    = "client_credentials"
    client_id     = "76a5e92c546b3fcdc2c4"
    client_secret = "a55092b9a311e7f435a38a8b73ff2b0c5851fb1a"
}
$res = Invoke-RestMethod -Uri "http://127.0.0.1:18180/api/login/oauth/access_token" -Method Post -Body $body
$token = $res.access_token
Write-Host "Token: $token"
$parts = $token.Split('.')
$p = $parts[1]
while($p.Length % 4 -ne 0) { $p += '=' }
$bytes = [Convert]::FromBase64String($p)
$json = [System.Text.Encoding]::UTF8.GetString($bytes)
Write-Host "Claims: $json"

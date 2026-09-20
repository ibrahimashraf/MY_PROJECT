$outputPath = 'C:\integin-secrets\integin-pilot.env'

if (Test-Path -LiteralPath $outputPath) {
  throw 'Pilot secret file already exists and will not be overwritten.'
}

function New-PilotSecret {
  param([int]$Length)
  $alphabet = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'
  $bytes = New-Object byte[] $Length
  [System.Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($bytes)
  return -join ($bytes | ForEach-Object { $alphabet[$_ % $alphabet.Length] })
}

$postgresPassword = New-PilotSecret -Length 64
$rustfsAccessKey = "pilot-$(New-PilotSecret -Length 24)"
$rustfsSecretKey = New-PilotSecret -Length 96
$syncSecret = New-PilotSecret -Length 96

$lines = @(
  '# Isolated INTEGIN pilot secrets. Never commit or share.',
  "PILOT_POSTGRES_PASSWORD=$postgresPassword",
  "PILOT_RUSTFS_ACCESS_KEY=$rustfsAccessKey",
  "PILOT_RUSTFS_SECRET_KEY=$rustfsSecretKey",
  "PILOT_SYNC_SECRET=$syncSecret",
  'POSTGRES_USER=integin_pilot_owner',
  "POSTGRES_PASSWORD=$postgresPassword",
  'POSTGRES_DB=integin_pilot',
  "RUSTFS_ACCESS_KEY=$rustfsAccessKey",
  "RUSTFS_SECRET_KEY=$rustfsSecretKey",
  'INTEGIN_HTTP_ADDR=127.0.0.1:18080',
  "INTEGIN_DB_URL=postgres://integin_pilot_runtime:$postgresPassword@127.0.0.1:15432/integin_pilot?sslmode=disable",
  'INTEGIN_EVIDENCE_STORE=rustfs',
  'INTEGIN_S3_ENDPOINT=http://127.0.0.1:19000',
  'INTEGIN_S3_BUCKET=integin-pilot-evidence',
  "INTEGIN_S3_ACCESS_KEY=$rustfsAccessKey",
  "INTEGIN_S3_SECRET_KEY=$rustfsSecretKey",
  'INTEGIN_S3_REGION=us-east-1',
  'INTEGIN_LOCAL_PROVISIONING_ENABLED=false'
)

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $outputPath) | Out-Null
[System.IO.File]::WriteAllLines($outputPath, $lines, [System.Text.UTF8Encoding]::new($false))
Write-Output 'PILOT_PRIVATE_ENV_CREATED_WITHOUT_DISPLAYING_VALUES'

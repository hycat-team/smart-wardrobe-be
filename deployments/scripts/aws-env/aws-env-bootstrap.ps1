# aws-env-bootstrap.ps1
# Scaffold the AWS-box env from the VPS .env.production.
#
# Reads local settings from aws-local.env (gitignored) and writes the generated
# AWS env to deployments/scripts/aws-env/outcome/.env.aws (gitignored). It copies the SaaS
# keys shared with the VPS (JWT/Brevo/Cloudinary/AI), prefills every AWS-box
# constant, and blanks the managed/box-specific values so you can fill them from
# the Supabase / Redis Cloud / CloudAMQP dashboards.
#
# Run from the repo root (PowerShell 5.1+):
#   powershell -ExecutionPolicy Bypass -File deployments/scripts/aws-env/aws-env-bootstrap.ps1
#
# Option precedence: CLI param > aws-local.env > built-in default.
# Options:
#   -EnvProd <path>   default: .env.production
#   -Template <path>  default: .env.aws.example
#   -Out <path>       default: deployments/scripts/aws-env/outcome/.env.aws
[CmdletBinding()]
param(
    [string]$EnvProd  = "",
    [string]$Template = "",
    [string]$Out      = ""
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
# scripts live in deployments/scripts/aws-env/ -> three levels up is the repo root.
$RepoRoot  = (Resolve-Path (Join-Path $ScriptDir "..\..\..")).Path

function Read-EnvMap([string]$path) {
    $map = @{}
    foreach ($raw in [IO.File]::ReadAllLines((Resolve-Path -LiteralPath $path).Path)) {
        $line = $raw.TrimEnd("`r")
        if ($line -match '^(?<k>[A-Za-z_][A-Za-z0-9_]*)=(?<v>.*)$') {
            $map[$Matches['k']] = $Matches['v']
        }
    }
    return $map
}

# Local (gitignored) settings file.
$local = @{}
$localPath = Join-Path $ScriptDir "aws-local.env"
if (Test-Path -LiteralPath $localPath) { $local = Read-EnvMap $localPath }

function Resolve-Setting([string]$cli, [string]$key, [string]$default) {
    if (-not [string]::IsNullOrWhiteSpace($cli)) { return $cli }
    if ($local.ContainsKey($key) -and -not [string]::IsNullOrWhiteSpace($local[$key])) { return $local[$key] }
    return $default
}

function Resolve-RepoPath([string]$p) {
    if ([IO.Path]::IsPathRooted($p)) { return $p }
    return (Join-Path $RepoRoot $p)
}

$EnvProd  = Resolve-RepoPath (Resolve-Setting $EnvProd  'VPS_ENV_PROD'     '.env.production')
$Template = Resolve-RepoPath (Resolve-Setting $Template 'AWS_ENV_TEMPLATE' '.env.aws.example')
$Out      = Resolve-RepoPath (Resolve-Setting $Out      'AWS_ENV_FILE'     'deployments/scripts/aws-env/outcome/.env.aws')

if (-not (Test-Path -LiteralPath $EnvProd))  { Write-Host "FAIL: VPS env not found: $EnvProd" -ForegroundColor Red;  exit 1 }
if (-not (Test-Path -LiteralPath $Template)) { Write-Host "FAIL: template not found: $Template" -ForegroundColor Red; exit 1 }

$vps = Read-EnvMap $EnvProd

# Values copied verbatim from the VPS (shared quota, same issuers).
$CopyExact = @(
    'TZ','FRONT_END_ORIGIN','SWAGGER_ACCESS_CODE',
    'JWT_ALGORITHM','JWT_SECRET','JWT_PRIVATE_KEY','JWT_PUBLIC_KEY',
    'EMAIL_SENDER_EMAIL','EMAIL_SMTP_USER','EMAIL_SMTP_KEY','EMAIL_REPLY_TO',
    'CLOUDINARY_CLOUD_NAME','CLOUDINARY_API_KEY','CLOUDINARY_API_SECRET','CLOUDINARY_API_URL',
    'GOOGLE_CLIENT_IDS','GOOGLE_CLIENT_SECRET'
)
$CopyPrefix = @('VISION_','EMBEDDING_','CHAT_TEXT_','RECOMMENDATION_TEXT_','FREE_TEXT_')

# Copied from the VPS when present; otherwise emitted as a TODO. These keys are
# required for Google login on the AWS box, but the local VPS .env.production may
# not define them yet (they were added in feature 021).
$CopyRequired = @{
    'GOOGLE_CLIENT_IDS'    = 'Google OAuth web client id (comma-separated; web client first)'
    'GOOGLE_CLIENT_SECRET' = 'Google OAuth web client secret'
}

# AWS-box constants.
$Const = @{
    'SERVER_ENV'                     = 'vps'
    'API_V2_DOMAIN'                  = 'api-v2.closy.hycat.online'
    # Session cookie shared by the FE (closy.hycat.online) and the box API.
    'COOKIE_DOMAIN'                  = 'closy.hycat.online'
    'NGINX_CLIENT_MAX_BODY_SIZE'     = '20m'
    'DB_NAME'                        = 'postgres'
    'DB_PORT'                        = '6543'
    'DB_SSLMODE'                     = 'require'
    'DB_LISTENER_PORT'               = '5432'
    'DB_LISTENER_SSLMODE'            = 'require'
    'RABBITMQ_PORT'                  = '5671'
    'RABBITMQ_TLS_ENABLED'           = 'true'
    'RABBITMQ_MANAGEMENT_TLS_ENABLED'= 'true'
    'ELASTICSEARCH_ENABLED'          = 'false'
    'PAYOS_ENDPOINT'                 = 'https://api-merchant.payos.vn'
    'CLOUDINARY_API_URL'             = 'https://api.cloudinary.com'
    # Google OAuth redirect is AWS-specific (api-v2 domain), so it MUST NOT be
    # copied from the VPS (which uses api.closy...). The web client must have this
    # URI registered in Google Cloud Console.
    'GOOGLE_REDIRECT_URI'            = 'https://api-v2.closy.hycat.online/api/v1/auth/google/callback'
    'GOOGLE_STATE_TTL_MINUTES'       = '10'
}

# Must be filled from the dashboards; script 2 refuses to run while any remain.
$TodoRequired = @(
    'BOX_IP','DB_USER','DB_PASSWORD','DB_HOST','DB_LISTENER_HOST',
    'REDIS_HOST','REDIS_PORT','REDIS_PASSWORD','REDIS_TLS_ENABLED',
    'RABBITMQ_HOST','RABBITMQ_USER','RABBITMQ_PASSWORD','RABBITMQ_VHOST',
    'PAYOS_CLIENT_ID','PAYOS_API_KEY','PAYOS_CHECKSUM_KEY'
)
$TodoHint = @{
    'BOX_IP'           = 'static IP of the AWS box (aws-runbook Buoc 3)'
    'DB_USER'          = 'postgres.<project-ref>'
    'DB_PASSWORD'      = 'database password (Supabase)'
    'DB_HOST'          = 'aws-0-ap-southeast-1.pooler.supabase.com'
    'DB_LISTENER_HOST' = 'db.<project-ref>.supabase.co'
    'REDIS_HOST'       = 'Redis Cloud public endpoint'
    'REDIS_PORT'       = 'Redis Cloud endpoint port (NOT 6379)'
    'REDIS_PASSWORD'   = 'Redis Cloud default user password'
    'REDIS_TLS_ENABLED'= 'false for the free 30MB plan (no TLS); true for paid Essentials+'
    'RABBITMQ_HOST'    = '<xxx>.rmq.cloudamqp.com'
    'RABBITMQ_USER'    = 'CloudAMQP AMQP URL user'
    'RABBITMQ_PASSWORD'= 'CloudAMQP AMQP URL password'
    'RABBITMQ_VHOST'   = 'CloudAMQP vhost (path after host in the AMQP URL)'
    'PAYOS_CLIENT_ID'  = 'separate payOS channel for the AWS box (my.payos.vn), NOT copied from VPS'
    'PAYOS_API_KEY'    = 'separate payOS channel for the AWS box (my.payos.vn), NOT copied from VPS'
    'PAYOS_CHECKSUM_KEY'='separate payOS channel for the AWS box (my.payos.vn), NOT copied from VPS'
}

# Intentionally empty; no action needed.
$EmptyOk = @(
    'PORT','INGRESS_TOKEN_AUDIENCE','DB_DRILL_SCHEMA','DATABASE_MIGRATION_TABLE',
    'ELASTICSEARCH_ADDRESSES','ELASTICSEARCH_USER','ELASTICSEARCH_PASSWORD'
)

$lines = New-Object System.Collections.Generic.List[string]
$todoCount = 1  # BOX_IP

$lines.Add('# ==============================================================================')
$lines.Add('# AWS EXTERNAL BOX env - generated by deployments/scripts/aws-env/aws-env-bootstrap.ps1')
$lines.Add('# Fill every ''# TODO'' line, then run deployments/scripts/aws-env/aws-env-to-copyblock.ps1')
$lines.Add('# NEVER commit this file (deployments/scripts/aws-env/outcome is gitignored).')
$lines.Add('# ==============================================================================')
$lines.Add('')
$lines.Add('# Helper (NOT read by app/compose): static IP of the AWS box, used for AWS_HOST.')
$lines.Add('BOX_IP= # TODO (static IP of the AWS box (aws-runbook Buoc 3))')
$lines.Add('')

foreach ($raw in [IO.File]::ReadAllLines((Resolve-Path -LiteralPath $Template).Path)) {
    $line = $raw.TrimEnd("`r")

    if ($line -eq '' -or $line.StartsWith('#')) { $lines.Add($line); continue }
    if ($line -notmatch '^[A-Za-z_][A-Za-z0-9_]*=') { $lines.Add($line); continue }

    $key = $line.Substring(0, $line.IndexOf('='))

    if ($TodoRequired -contains $key) {
        $lines.Add(("{0}= # TODO ({1})" -f $key, $TodoHint[$key]))
        $todoCount++
        continue
    }
    if ($EmptyOk -contains $key) { $lines.Add(("{0}=" -f $key)); continue }

    if ($CopyRequired.ContainsKey($key)) {
        $val = ''
        if ($vps.ContainsKey($key)) { $val = $vps[$key] }
        if ([string]::IsNullOrWhiteSpace($val)) {
            $lines.Add(("{0}= # TODO ({1})" -f $key, $CopyRequired[$key]))
            $todoCount++
        } else {
            $lines.Add(("{0}={1}" -f $key, $val))
        }
        continue
    }

    $isCopy = ($CopyExact -contains $key) -or (@($CopyPrefix | Where-Object { $key.StartsWith($_) }).Count -gt 0)
    if ($isCopy) {
        $val = ''
        if ($vps.ContainsKey($key)) { $val = $vps[$key] }
        if ([string]::IsNullOrEmpty($val)) {
            if ($Const.ContainsKey($key)) { $val = $Const[$key] } else { $val = $line.Substring($line.IndexOf('=') + 1) }
        }
        $lines.Add(("{0}={1}" -f $key, $val))
        continue
    }
    if ($Const.ContainsKey($key)) { $lines.Add(("{0}={1}" -f $key, $Const[$key])); continue }

    # Unknown key: keep the template default.
    $lines.Add($line)
}

$outDir = Split-Path -Parent $Out
if ($outDir -and -not (Test-Path -LiteralPath $outDir)) { New-Item -ItemType Directory -Path $outDir -Force | Out-Null }

$text = ($lines -join "`n") + "`n"
[IO.File]::WriteAllText($Out, $text, (New-Object System.Text.UTF8Encoding($false)))

Write-Host "OK: wrote $Out" -ForegroundColor Green
Write-Host "    $todoCount '# TODO' lines to fill (Supabase/Redis/CloudAMQP/payOS/AWS box)."
Write-Host "    When done: powershell -ExecutionPolicy Bypass -File deployments/scripts/aws-env/aws-env-to-copyblock.ps1"

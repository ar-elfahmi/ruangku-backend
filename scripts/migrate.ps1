<#
.SYNOPSIS
    Applies pending SQL migrations to the PostgreSQL container. Never drops anything.

.DESCRIPTION
    Every migrations/NNN_*.sql file that is not recorded in schema_migrations is applied
    in ascending order. Each file runs in its own transaction together with its version
    record, so a failing file leaves no partial change and is not recorded.

    A recorded file whose checksum changed stops the script: an applied migration must
    never be edited. The checksum is SHA-256 of the file content with CRLF normalized to LF.

    Baseline: when schema_migrations is empty but application tables already exist (a
    database created by the old initdb mount), the structure of 001 is verified and 001 is
    recorded without running it again. Any mismatch stops the script and changes nothing.

    Migration files must not contain BEGIN/COMMIT or statements that cannot run inside a
    transaction (for example CREATE INDEX CONCURRENTLY).

.EXAMPLE
    .\scripts\migrate.ps1
.EXAMPLE
    .\scripts\migrate.ps1 -Database ruangku_test -CreateDatabase
#>
[CmdletBinding()]
param(
    [string]$Database = 'ruangku',
    [string]$MigrationsDir = 'migrations',
    [switch]$CreateDatabase
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $root 'docker-compose.yml'
$container = 'ruangku-postgres'
$pgUser = 'ruangku'
$utf8 = New-Object System.Text.UTF8Encoding($false)
$OutputEncoding = $utf8
[Console]::OutputEncoding = $utf8

function Stop-Migration([string]$Message) {
    Write-Host "migrate: $Message" -ForegroundColor Red
    exit 1
}

# Any unexpected error ends the script with a non-zero exit code.
trap { Stop-Migration "$_" }

# All SQL is sent through stdin, so no value is ever quoted on the command line.
function Invoke-Psql {
    param(
        [Parameter(Mandatory)] [string]$Target,
        [Parameter(Mandatory)] [string]$Sql,
        [switch]$SingleTransaction
    )
    $psqlArgs = @('compose', '-f', $composeFile, 'exec', '-T', 'postgres',
        'psql', '-U', $pgUser, '-d', $Target, '-X', '-q', '-t', '-A', '-v', 'ON_ERROR_STOP=1')
    if ($SingleTransaction) { $psqlArgs += '--single-transaction' }
    $psqlArgs += @('-f', '-')
    $out = $Sql | & docker @psqlArgs
    if ($LASTEXITCODE -ne 0) { throw "psql exited with code $LASTEXITCODE" }
    # The leading comma keeps an empty or one-row result an array for the caller.
    return , @($out | Where-Object { $_ -ne '' })
}

function Get-Checksum([string]$Text) {
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hash = $sha.ComputeHash($utf8.GetBytes($Text))
    } finally {
        $sha.Dispose()
    }
    return ([BitConverter]::ToString($hash) -replace '-', '').ToLowerInvariant()
}

$ledgerSql = @'
SET client_min_messages = warning;
CREATE TABLE IF NOT EXISTS schema_migrations (
    version    text        PRIMARY KEY,
    checksum   text        NOT NULL,
    applied_at timestamptz NOT NULL DEFAULT now()
);
'@

# Structure created by 001_init.sql. Each returned row is something that is missing.
$baselineCheckSql = @'
SELECT item FROM (
    SELECT 'table ' || t AS item
    FROM unnest(ARRAY['users', 'rooms', 'bookings', 'booking_logs']) AS t
    WHERE to_regclass('public.' || t) IS NULL
  UNION ALL
    SELECT 'column ' || c.tbl || '.' || c.col
    FROM (VALUES
        ('users', 'id'), ('users', 'name'), ('users', 'email'), ('users', 'password_hash'),
        ('users', 'role'), ('users', 'created_at'),
        ('rooms', 'id'), ('rooms', 'code'), ('rooms', 'name'), ('rooms', 'building'),
        ('rooms', 'capacity'), ('rooms', 'is_active'), ('rooms', 'created_at'),
        ('bookings', 'id'), ('bookings', 'user_id'), ('bookings', 'room_id'), ('bookings', 'purpose'),
        ('bookings', 'start_time'), ('bookings', 'end_time'), ('bookings', 'status'), ('bookings', 'created_at'),
        ('booking_logs', 'id'), ('booking_logs', 'booking_id'), ('booking_logs', 'actor_id'),
        ('booking_logs', 'from_status'), ('booking_logs', 'to_status'), ('booking_logs', 'note'),
        ('booking_logs', 'created_at')
    ) AS c (tbl, col)
    WHERE NOT EXISTS (
        SELECT 1 FROM information_schema.columns ic
        WHERE ic.table_schema = 'public' AND ic.table_name = c.tbl AND ic.column_name = c.col)
  UNION ALL
    SELECT 'extension btree_gist'
    WHERE NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'btree_gist')
  UNION ALL
    SELECT 'constraint ' || k.name || ' (type ' || k.kind || ')'
    FROM (VALUES
        ('users_role_check', 'c'), ('rooms_code_key', 'u'), ('bookings_status_check', 'c'),
        ('bookings_time_check', 'c'), ('bookings_room_time_excl', 'x'),
        ('booking_logs_to_status_check', 'c')
    ) AS k (name, kind)
    WHERE NOT EXISTS (
        SELECT 1 FROM pg_constraint pc
        JOIN pg_namespace n ON n.oid = pc.connamespace
        WHERE n.nspname = 'public' AND pc.conname = k.name AND pc.contype::text = k.kind)
  UNION ALL
    SELECT 'index users_email_lower_key'
    WHERE NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public' AND indexname = 'users_email_lower_key')
) AS missing;
'@

# --- Preconditions ---------------------------------------------------------------

if ($Database -notmatch '^[a-z_][a-z0-9_]{0,62}$') {
    Stop-Migration "invalid database name '$Database' (use lowercase letters, digits, underscore)"
}
if (-not [System.IO.Path]::IsPathRooted($MigrationsDir)) {
    $MigrationsDir = Join-Path $root $MigrationsDir
}
if (-not (Test-Path -LiteralPath $MigrationsDir -PathType Container)) {
    Stop-Migration "migrations directory not found: $MigrationsDir"
}

$health = & docker inspect -f '{{.State.Health.Status}}' $container 2>$null
if ($LASTEXITCODE -ne 0 -or "$health" -ne 'healthy') {
    Stop-Migration "container $container is not healthy (run 'docker compose up -d' and wait for 'healthy')"
}

$exists = Invoke-Psql -Target 'postgres' -Sql "SELECT 1 FROM pg_database WHERE datname = '$Database';"
if ($exists.Count -eq 0) {
    if (-not $CreateDatabase) {
        Stop-Migration "database '$Database' does not exist (add -CreateDatabase to create it)"
    }
    Invoke-Psql -Target 'postgres' -Sql "CREATE DATABASE $Database;" | Out-Null
    Write-Host "migrate: created database $Database"
}

# --- Read migration files ----------------------------------------------------------

$files = @()
foreach ($f in Get-ChildItem -LiteralPath $MigrationsDir -Filter '*.sql' -File | Sort-Object Name) {
    if ($f.Name -notmatch '^(\d{3})_[a-z0-9_]+\.sql$') {
        Write-Host "migrate: skipping $($f.Name) (name must be NNN_lowercase_name.sql)" -ForegroundColor Yellow
        continue
    }
    $text = [System.IO.File]::ReadAllText($f.FullName, $utf8) -replace "`r`n", "`n"
    $files += [pscustomobject]@{
        Number   = $Matches[1]
        Version  = [System.IO.Path]::GetFileNameWithoutExtension($f.Name)
        Text     = $text
        Checksum = Get-Checksum $text
    }
}
$dupes = @($files | Group-Object Number | Where-Object Count -gt 1)
if ($dupes.Count -gt 0) {
    Stop-Migration "duplicate migration number: $(($dupes | ForEach-Object Name) -join ', ')"
}
if ($files.Count -eq 0) {
    Stop-Migration "no migration files in $MigrationsDir"
}

# --- Load the ledger ---------------------------------------------------------------

$recorded = @{}
$hasLedger = Invoke-Psql -Target $Database -Sql "SELECT to_regclass('public.schema_migrations') IS NOT NULL;"
if ($hasLedger[0] -eq 't') {
    $rows = Invoke-Psql -Target $Database -Sql "SELECT version || '|' || checksum FROM schema_migrations;"
    foreach ($row in $rows) {
        $parts = $row.Split('|')
        $recorded[$parts[0]] = $parts[1]
    }
}

# --- Baseline for databases created by the old initdb mount -------------------------

if ($recorded.Count -eq 0) {
    $appTables = Invoke-Psql -Target $Database -Sql @'
SELECT count(*) FROM pg_tables
WHERE schemaname = 'public' AND tablename IN ('users', 'rooms', 'bookings', 'booking_logs');
'@
    if ([int]$appTables[0] -gt 0) {
        $first = $files | Where-Object Number -eq '001' | Select-Object -First 1
        if ($null -eq $first) {
            Stop-Migration "tables exist but there is no 001 file to record as baseline"
        }
        $missing = Invoke-Psql -Target $Database -Sql $baselineCheckSql
        if ($missing.Count -gt 0) {
            Write-Host "migrate: existing structure does not match $($first.Version); nothing was changed. Missing:" -ForegroundColor Red
            $missing | ForEach-Object { Write-Host "  - $_" -ForegroundColor Red }
            exit 1
        }
        $sql = $ledgerSql + "`nINSERT INTO schema_migrations (version, checksum) VALUES ('$($first.Version)', '$($first.Checksum)');`n"
        Invoke-Psql -Target $Database -Sql $sql -SingleTransaction | Out-Null
        $recorded[$first.Version] = $first.Checksum
        Write-Host "migrate: baseline $($first.Version) recorded (structure verified, not re-run)"
    }
}

# --- Refuse edited migrations before applying anything -----------------------------

foreach ($file in $files) {
    if ($recorded.ContainsKey($file.Version) -and $recorded[$file.Version] -ne $file.Checksum) {
        Stop-Migration "$($file.Version) was edited after it was applied (checksum mismatch); add a new migration instead"
    }
}
foreach ($version in $recorded.Keys) {
    if (-not ($files | Where-Object Version -eq $version)) {
        Write-Host "migrate: $version is recorded but its file is missing" -ForegroundColor Yellow
    }
}

# --- Apply pending files -----------------------------------------------------------

$pending = @($files | Where-Object { -not $recorded.ContainsKey($_.Version) })
if ($pending.Count -eq 0) {
    Write-Host "migrate: $Database is up to date"
    exit 0
}
foreach ($file in $pending) {
    $sql = $ledgerSql + "`n" + $file.Text + "`n;`nINSERT INTO schema_migrations (version, checksum) VALUES ('$($file.Version)', '$($file.Checksum)');`n"
    try {
        Invoke-Psql -Target $Database -Sql $sql -SingleTransaction | Out-Null
    } catch {
        Stop-Migration "$($file.Version) failed and was rolled back; nothing from it was applied"
    }
    Write-Host "migrate: applied $($file.Version)"
}

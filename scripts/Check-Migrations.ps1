# Static DDL checks only. This does not parse all MySQL grammar or contact a server.
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
& (Join-Path $PSScriptRoot 'Generate-RecordMigrations.ps1') -Check
$files = @(
    'services/identity-service/migrations/001_initial.sql',
    'services/activity-service/migrations/001_initial.sql',
    'services/activity-service/migrations/002_records.sql',
    'services/ranking-service/migrations/001_initial.sql'
)
$counts = @{ activity_identity = 3; activity_core = 32; activity_ranking = 7 }
$tables = @{}
foreach ($relativePath in $files) {
    $sql = [IO.File]::ReadAllText((Join-Path $projectRoot $relativePath))
    if ($sql.Contains('__SHARD__')) { throw "Unrendered placeholder: $relativePath" }
    $schema = [regex]::Match($sql, '(?m)^USE (\w+);').Groups[1].Value
    if (-not $counts.ContainsKey($schema)) { throw "Unknown schema: $relativePath" }
    # Ignore strings and comments before checking balanced delimiters and unsafe DDL.
    $clean = [regex]::Replace($sql, "'(?:''|[^'])*'|--[^\r\n]*", ' ')
    if ($clean -match '(?i)\b(DROP|TRUNCATE)\b') { throw "Destructive statement: $relativePath" }
    $balance = 0
    foreach ($char in $clean.ToCharArray()) {
        if ($char -eq '(') { $balance++ }
        if ($char -eq ')') { $balance-- }
        if ($balance -lt 0) { throw "Unbalanced parentheses: $relativePath" }
    }
    if ($balance -ne 0) { throw "Unbalanced parentheses: $relativePath" }
    $definitions = [regex]::Matches($sql, '(?s)CREATE TABLE (\w+) \((.*?)\) ENGINE=InnoDB[^;]*;')
    if ($definitions.Count -ne [regex]::Matches($clean, 'CREATE TABLE ').Count) {
        throw "Unrecognized table definition: $relativePath"
    }
    foreach ($definition in $definitions) {
        $name = $definition.Groups[1].Value
        $body = $definition.Groups[2].Value
        $qualified = "$schema.$name"
        if ($tables.ContainsKey($qualified)) { throw "Duplicate table: $qualified" }
        if ($body -notmatch 'PRIMARY KEY') { throw "Missing primary key: $qualified" }
        if ($name -match '^(draw_order|draw_detail|flower_lit_log|wallet_ledger)_([0-3])$') {
            $shard = $Matches[2]
            if (-not $body.Contains("MOD(user_id, 4) = $shard")) { throw "Wrong routing CHECK: $qualified" }
        }
        $tables[$qualified] = $true
    }
}
foreach ($schema in $counts.Keys) {
    $actual = @($tables.Keys | Where-Object { $_.StartsWith("$schema.") }).Count
    if ($actual -ne $counts[$schema]) { throw "Unexpected table count for ${schema}: $actual" }
    Write-Output "OK: $schema has $actual tables."
}
foreach ($baseName in @('draw_order', 'draw_detail', 'flower_lit_log', 'wallet_ledger')) {
    foreach ($shard in 0..3) {
        if (-not $tables.ContainsKey("activity_core.${baseName}_$shard")) { throw "Missing shard: ${baseName}_$shard" }
    }
}
Write-Output "OK: $($tables.Count) tables; unique names, InnoDB, primary keys, shard checks and template consistency."
Write-Output 'MySQL execution, query plans and transactional behavior are not verified by this script.'

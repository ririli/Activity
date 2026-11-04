# Generate the initial record migration from a single template; no database access.
# Once deployed, keep 002 immutable and add new numbered migrations for changes.
[CmdletBinding()]
param([switch]$Check)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$migrationRoot = Join-Path $projectRoot 'services/activity-service/migrations'
$templatePath = Join-Path $migrationRoot 'templates/002_records.sql.tmpl'
$outputPath = Join-Path $migrationRoot '002_records.sql'
$template = [IO.File]::ReadAllText($templatePath).Replace("`r`n", "`n")
$header = @'
-- GENERATED FILE. Source: templates/002_records.sql.tmpl
-- MySQL 8.4. Initial migration; user_id % 4 routing enforced by CHECK.
USE activity_core;
SET NAMES utf8mb4;
SET time_zone = '+00:00';

'@
$sections = @($header.Replace("`r`n", "`n").TrimEnd())
foreach ($shardNumber in 0..3) {
    $sections += $template.Replace('__SHARD__', [string]$shardNumber).TrimEnd()
}
$expected = ($sections -join "`n`n") + "`n"
if ($Check) {
    if (-not (Test-Path -LiteralPath $outputPath)) { throw 'Generated migration is missing.' }
    $actual = [IO.File]::ReadAllText($outputPath).Replace("`r`n", "`n")
    if ($actual -cne $expected) { throw '002_records.sql differs from its template. Regenerate before initial deployment.' }
    Write-Output 'OK: all four shards match the source template.'
} else {
    [IO.File]::WriteAllText($outputPath, $expected, [Text.UTF8Encoding]::new($false))
    Write-Output "Generated: $outputPath"
}

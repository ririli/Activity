# Generates an opt-in load proxy configuration; never edits the daily config.
[CmdletBinding()]
param()
$ErrorActionPreference = 'Stop'
$taskRoot = Split-Path -Parent $PSScriptRoot
$source = Join-Path $taskRoot 'deploy/local/nginx/default.conf'
$output = Join-Path $taskRoot '.local/sse-load/default.conf'
$config = [IO.File]::ReadAllText($source)
$replacements = @{
    'zone=api_ip:10m rate=30r/s' = 'zone=api_ip:10m rate=10000r/s'
    'zone=api_total:1m rate=100r/s' = 'zone=api_total:1m rate=10000r/s'
    'zone=sse_ip:10m rate=1r/s' = 'zone=sse_ip:10m rate=10000r/s'
    'limit_conn sse_connections 20;' = 'limit_conn sse_connections 20000;'
}
foreach ($entry in $replacements.GetEnumerator()) {
    if (-not $config.Contains($entry.Key)) { throw "Proxy config changed; review load override: $($entry.Key)" }
    $config = $config.Replace($entry.Key, $entry.Value)
}
[IO.Directory]::CreateDirectory((Split-Path -Parent $output)) | Out-Null
[IO.File]::WriteAllText($output, $config, [Text.UTF8Encoding]::new($false))
Write-Output "Prepared isolated SSE load config: $output"

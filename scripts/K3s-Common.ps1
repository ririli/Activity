$ErrorActionPreference = 'Stop'
$Repo = Split-Path $PSScriptRoot -Parent
$State = Join-Path $Repo '.local/k3s'
$Kubeconfig = Join-Path $State 'kubeconfig.yaml'
$K3d = Join-Path $State 'tools/k3d.exe'
$Go = 'D:\Program Files\GO\go1.25.1\bin\go.exe'
if (-not (Test-Path $Go)) { $Go = (Get-Command go -ErrorAction Stop).Source }

function K {
    & kubectl --kubeconfig $Kubeconfig @args
    if ($LASTEXITCODE -ne 0) { throw "kubectl failed: $($args -join ' ')" }
}
function Apply-Object($Object) {
    $Object | ConvertTo-Json -Depth 60 | & kubectl --kubeconfig $Kubeconfig apply -f -
    if ($LASTEXITCODE -ne 0) { throw 'Kubernetes object apply failed' }
}
function Apply-Secret([string]$Name, [hashtable]$Values) {
    Apply-Object @{ apiVersion='v1'; kind='Secret'; metadata=@{name=$Name; namespace='activity'}; type='Opaque'; stringData=$Values }
}
function Random-Hex { [Convert]::ToHexString([Security.Cryptography.RandomNumberGenerator]::GetBytes(32)).ToLowerInvariant() }
function Native-Check([string]$What) { if ($LASTEXITCODE -ne 0) { throw "$What failed, exit $LASTEXITCODE" } }

function Wait-RankingRedisTopology {
    $deadline = [DateTime]::UtcNow.AddSeconds(75)
    do {
        $healthy = $true
        try {
            $primaries = @{}
            foreach ($index in 0..2) {
                foreach ($shard in 0..3) {
                    $group = "ranking-shard-$shard"
                    $check = K -n activity exec "ranking-sentinel-$index" '--' redis-cli -p 26379 --raw SENTINEL CKQUORUM $group
                    if ($check -notmatch '^OK 3 usable Sentinels') { throw "$group has not discovered all three Sentinels" }
                    $primary = @(K -n activity exec "ranking-sentinel-$index" '--' redis-cli -p 26379 --raw SENTINEL GET-MASTER-ADDR-BY-NAME $group)[0]
                    if ($primary -notmatch "^ranking-redis-$shard-[01]\.") { throw "$group reported unexpected primary $primary" }
                    if ($index -eq 0) {
                        $pod = $primary.Split('.')[0]
                        $info = K -n activity exec $pod '--' redis-cli --raw INFO replication
                        if ('role:master' -notin $info -or -not ($info -match '^slave0:.*state=online,')) {
                            throw "$group is waiting for its replica"
                        }
                        $primaries[$group] = $primary
                    } elseif ($primaries[$group] -ne $primary) {
                        throw "Sentinels disagree about the primary of $group"
                    }
                }
            }
        } catch {
            $healthy = $false
            $last = $_.Exception.Message
        }
        if ($healthy) { Write-Output 'Four ranking Redis primary/replica groups and three Sentinels are ready'; return }
        Start-Sleep -Seconds 2
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Ranking Redis topology not ready: $last"
}

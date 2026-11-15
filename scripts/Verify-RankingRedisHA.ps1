#Requires -Version 7.0
param(
    [Parameter(Mandatory)][ValidateSet('Compose','K3s')][string]$Environment,
    [Parameter(Mandatory)][ValidatePattern('^[1-9][0-9]*$')][string]$EditionId,
    [Parameter(Mandatory)][uri]$BaseUrl,
    [string]$ComposeProject = 'activity-local',
    [switch]$VerifyHourly
)
. "$PSScriptRoot/K3s-Common.ps1"

# Run against an isolated development deployment with an active seeded edition.
$base = $BaseUrl.GetLeftPart([UriPartial]::Authority)
$session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
$evidence = [ordered]@{ environment = $Environment; edition = $EditionId; mysqlBoardResponses = 0 }
$paused = @{}

function Wait-Check([scriptblock]$Check, [string]$Name, [int]$Seconds) {
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    $failure = ''
    do {
        try { if (& $Check) { return } } catch { $failure = $_.Exception.Message }
        Start-Sleep -Milliseconds 750
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out waiting for $Name. $failure"
}

function Compose-Container([string]$Name) {
    $result = @(docker ps -q --filter "label=com.docker.compose.project=$ComposeProject" --filter "label=com.docker.compose.service=$Name")
    Native-Check "locate $Name"
    if ($result.Count -ne 1) { throw "Expected one Compose container for $Name" }
    return $result[0]
}

function Remote([string]$Name, [string[]]$Command) {
    if ($Environment -eq 'K3s') { K -n activity exec $Name '--' @Command; return }
    $container = Compose-Container $Name
    & docker exec $container @Command
    Native-Check "exec in $Name"
}

function Redis([string]$Name, [string[]]$Command) {
    if ($Name -like 'ranking-sentinel-*') {
        Remote $Name (@('sh','-c','REDISCLI_AUTH="$RANKING_REDIS_SENTINEL_PASSWORD" exec redis-cli --raw -p 26379 "$@"','_') + $Command)
    } else {
        Remote $Name (@('sh','-c','REDISCLI_AUTH="$RANKING_REDIS_PASSWORD" exec redis-cli -e --raw "$@"','_') + $Command)
    }
}

function Sql([string]$Query) {
    $name = if ($Environment -eq 'K3s') { 'mysql-0' } else { 'mysql' }
    Remote $name @('sh','-c','MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysql -uroot -N -e "$1"','_',$Query)
}

function Primary([string]$Sentinel, [int]$Shard) {
    $result = @(Redis $Sentinel @('SENTINEL','GET-MASTER-ADDR-BY-NAME',"ranking-shard-$Shard"))
    if ($result.Count -ne 2 -or $result[0] -notmatch "^ranking-redis-$Shard-[01](\.|$)" -or $result[1] -ne '6379') {
        throw "Invalid primary response for shard $Shard"
    }
    return $result[0].Split('.')[0]
}

function Ranking-Instance {
    if ($Environment -eq 'Compose') {
        return (docker inspect --format '{{.Id}} {{.State.StartedAt}}' (Compose-Container ranking))
    }
    $pods = (K -n activity get pods -l app.kubernetes.io/name=ranking -o json | ConvertFrom-Json).items
    if (@($pods).Count -ne 1) { throw 'Expected one ranking Pod' }
    return $pods[0].status.containerStatuses[0].containerID
}

function Pause-Node([string]$Name) {
    if ($Environment -eq 'Compose') {
        $container = Compose-Container $Name
        & docker pause $container | Out-Null
        Native-Check "pause $Name"
        $paused[$Name] = @($container)
    } else {
        $pod = K -n activity get pod $Name -o json | ConvertFrom-Json
        $container = $pod.status.containerStatuses[0].containerID -replace '^containerd://', ''
        & docker exec $pod.spec.nodeName ctr -n k8s.io tasks pause $container | Out-Null
        Native-Check "pause $Name"
        $paused[$Name] = @($pod.spec.nodeName, $container)
    }
}

function Resume-Node([string]$Name) {
    if (-not $paused.ContainsKey($Name)) { return }
    $target = $paused[$Name]
    if ($Environment -eq 'Compose') {
        & docker unpause $target[0] | Out-Null
    } else {
        & docker exec $target[0] ctr -n k8s.io tasks resume $target[1] | Out-Null
    }
    Native-Check "resume $Name"
    $paused.Remove($Name)
}

function Restart-Node([string]$Name) {
    if ($Environment -eq 'Compose') {
        & docker restart (Compose-Container $Name) | Out-Null
        Native-Check "restart $Name"
    } else {
        K -n activity delete pod $Name --timeout=45s | Out-Null
        K -n activity wait --for=condition=Ready "pod/$Name" --timeout=60s | Out-Null
    }
}

function Api([string]$Method, [string]$Path, $Body, [string]$Token = '') {
    $headers = @{ Origin=$base; 'X-Activity-CSRF'='1' }
    if ($Token) { $headers.Authorization = "Bearer $Token" }
    $options = @{ Method=$Method; Uri=$base+$Path; Headers=$headers; WebSession=$session; TimeoutSec=5 }
    if ($Environment -eq 'K3s') { $options.SkipCertificateCheck = $true }
    if ($null -ne $Body) { $options.ContentType = 'application/json'; $options.Body = $Body | ConvertTo-Json -Compress }
    Invoke-RestMethod @options
}

function Board {
    $view = Api GET "/api/ranking/v1/board?edition_id=$EditionId" $null $token
    if ($view.source -eq 'mysql') { $evidence.mysqlBoardResponses++ }
    return $view
}

$initial = @{}
foreach ($shard in 0..3) {
    $initial[$shard] = Primary ranking-sentinel-0 $shard
    $check = Redis ranking-sentinel-0 @('SENTINEL','CKQUORUM',"ranking-shard-$shard")
    $info = Redis $initial[$shard] @('INFO','replication')
    if ($check -notmatch '^OK 3 usable Sentinels' -or 'role:master' -notin $info -or -not ($info -match '^slave0:.*state=online,')) {
        throw "Shard $shard must start with a healthy replica and three Sentinels"
    }
}
$rankingInstance = Ranking-Instance
if ($VerifyHourly) {
    $active = Sql "SELECT COUNT(*) FROM activity_core.hourly_ranking_window WHERE edition_id=$EditionId AND cutoff_sec=(FLOOR(UNIX_TIMESTAMP()/3600)+1)*3600 AND cutoff_sec-UNIX_TIMESTAMP()>=120;"
    if ($active -ne '4') { throw 'Hourly verification requires an open cutoff at least two minutes away' }
}
$auth = Api POST '/api/identity/v1/auth/register' @{
    account = 'rankha' + (Random-Hex).Substring(0,12)
    password = 'Verify-' + (Random-Hex)
}
$token = $auth.access_token
$user = [uint64]::Parse($auth.user_id)
$shard = [int]($user % 4)
$primary = $initial[$shard]
$standby = if ($primary.EndsWith('-0')) { "ranking-redis-$shard-1" } else { "ranking-redis-$shard-0" }
Wait-Check { [bool](Api POST '/api/activity/v1/join' @{ edition_id=$EditionId } $token) } 'wallet initialization' 30
$draw = @{ edition_id=$EditionId; pool_code='NIGHT'; count=1; operation_id=('ha-' + (Random-Hex).Substring(0,20)) }
Api POST '/api/activity/v1/draw' $draw $token | Out-Null
Wait-Check { $view = Board; $view.source -eq 'redis' -and $view.me.score -eq 100 } 'initial Redis score' 30

try {
    Pause-Node $primary
    $timer = [Diagnostics.Stopwatch]::StartNew()
    Wait-Check { $view = Board; $view.source -eq 'mysql' -and $view.me.score -eq 100 } 'MySQL fallback after primary loss' 8
    Wait-Check {
        (Primary ranking-sentinel-0 $shard) -eq $standby -and
            'role:master' -in (Redis $standby @('INFO','replication'))
    } 'automatic promotion' ([Math]::Max(1, 30 - $timer.Elapsed.TotalSeconds))
    $evidence.primaryPromotionSeconds = [Math]::Round($timer.Elapsed.TotalSeconds, 2)
    if ($evidence.primaryPromotionSeconds -gt 30) { throw 'Primary promotion exceeded 30 seconds' }
    Resume-Node $primary
    Wait-Check { $view = Board; $view.source -eq 'redis' -and $view.me.score -eq 100 } 'live Redis recovery' ([Math]::Max(1, 60 - $timer.Elapsed.TotalSeconds))
    $evidence.liveCacheRecoverySeconds = [Math]::Round($timer.Elapsed.TotalSeconds, 2)
    if ($evidence.liveCacheRecoverySeconds -gt 60) { throw 'Live Redis recovery exceeded 60 seconds' }
    Wait-Check { 'role:slave' -in (Redis $primary @('INFO','replication')) } 'old primary becoming replica' 40
    foreach ($other in 0..3) {
        if ($other -ne $shard -and (Primary ranking-sentinel-0 $other) -ne $initial[$other]) {
            throw "An unaffected shard changed primary: $other"
        }
    }

    Restart-Node $primary
    Restart-Node ranking-sentinel-0
    Wait-Check {
        (Primary ranking-sentinel-0 $shard) -eq $standby -and
            'master_link_status:up' -in (Redis $primary @('INFO','replication'))
    } 'persisted Redis and Sentinel roles' 40

    Pause-Node $primary
    Pause-Node $standby
    $draw.operation_id = 'ha-' + (Random-Hex).Substring(0,20)
    $committed = Api POST '/api/activity/v1/draw' $draw $token
    $repeated = Api POST '/api/activity/v1/draw' $draw $token
    if ($repeated.draw_id -ne $committed.draw_id) { throw 'Repeated draw was not idempotent' }
    Wait-Check { $view = Board; $view.source -eq 'mysql' -and $view.me.score -eq 200 } 'durable score with both data nodes down' 20
    Resume-Node $standby
    Resume-Node $primary
    Wait-Check { $view = Board; $view.source -eq 'redis' -and $view.me.score -eq 200 } 'rebuild from persisted draw facts' 60
    $facts = Sql "SELECT CONCAT(COUNT(*), '|', SUM(consumed_petals)) FROM activity_ranking.score_fact WHERE edition_id=$EditionId AND user_id=$user;"
    if ($facts -ne '2|200') { throw "Expected two persisted draw facts scoring 200, got $facts" }
    if ($VerifyHourly) {
        $drawId = [uint64]::Parse($committed.draw_id)
        $row = Sql "SELECT CONCAT_WS('|', hourly_cutoff_sec, hourly_lane, hourly_seq, payload->>'$.score_reached_at_sec', UNIX_TIMESTAMP(e.starts_at), UNIX_TIMESTAMP(e.ends_at)) FROM activity_core.outbox_event o JOIN activity_core.activity_edition e ON e.edition_id=o.hourly_edition_id WHERE o.event_type='FormalDrawCommitted.v1' AND o.aggregate_id=$drawId AND o.user_id=$user AND o.hourly_shard=$shard;"
        $cutoff, $lane, $seq, $reached, $starts, $ends = ([string]$row).Split('|')
        if (-not $seq -or -not $ends) { throw 'Missing persisted hourly draw position' }
        $member = [uint64]([decimal][uint64]::MaxValue - [decimal]$user)
        $packed = 200 * ([int64]$ends - [int64]$starts + 1) + ([int64]$ends - [int64]$reached)
        $namespace = if ($Environment -eq 'K3s') { 'k3s-learn' } else { 'local' }
        $key = "ranking:${namespace}:{${EditionId}:${shard}}:hourly:${cutoff}"
        Wait-Check {
            $position = Redis $standby @('HGET', "${key}:meta", "seq:$lane")
            $score = Redis $standby @('ZSCORE', "${key}:board", $member.ToString('D20'))
            $position -and [int64]$position -ge [int64]$seq -and [int64]$score -eq $packed
        } 'hourly window replaying the draw committed during Redis outage' 45
        $evidence.hourlyReplayCutoff = [int64]$cutoff
    }

    Pause-Node ranking-sentinel-1
    Pause-Node ranking-sentinel-2
    Pause-Node $standby
    Wait-Check {
        $state = Redis ranking-sentinel-0 @('SENTINEL','MASTER',"ranking-shard-$shard")
        $quorum = Redis ranking-sentinel-0 @('SENTINEL','CKQUORUM',"ranking-shard-$shard")
        ($state -match '^s_down,|,s_down(,|$)') -and ($quorum -match '^NOQUORUM')
    } 'failed primary observed by one Sentinel without quorum' 15
    if ((Primary ranking-sentinel-0 $shard) -ne $standby) { throw 'A lone Sentinel promoted a replica' }
    Resume-Node $standby
    Resume-Node ranking-sentinel-1
    Resume-Node ranking-sentinel-2
    Wait-Check { $view = Board; $view.source -eq 'redis' -and $view.me.score -eq 200 } 'recovery after loss of majority' 60
    if ((Ranking-Instance) -ne $rankingInstance) { throw 'ranking restarted during failover verification' }
    $evidence.primary = $standby
    $evidence.shard = $shard
    $evidence.replication = @(Redis $standby @('INFO','replication') | Where-Object { $_ -match '^role:|^connected_slaves:|^slave0:' })
    $evidence.pendingHourlyWindows = [int](Sql "SELECT COUNT(*) FROM activity_ranking.hourly_settlement WHERE edition_id=$EditionId AND status <> 'COMPLETED' AND cutoff_sec <= UNIX_TIMESTAMP();")
    $evidence.finalScore = 200
    [pscustomobject]$evidence | ConvertTo-Json -Depth 4
} finally {
    foreach ($name in @($paused.Keys)) {
        try { Resume-Node $name } catch { Write-Warning $_.Exception.Message }
    }
}

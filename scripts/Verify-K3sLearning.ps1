param([switch]$SkipRecovery)
. "$PSScriptRoot/K3s-Common.ps1"
$base = 'https://localhost:18443'
$edition = '2026110601'
$session = [Microsoft.PowerShell.Commands.WebRequestSession]::new()
$checks = [Collections.Generic.List[string]]::new()
function Api([string]$Method, [string]$Path, $Body = $null, [string]$Token = '', [int]$Expected = 200) {
    $headers = @{Origin=$base; 'X-Activity-CSRF'='1'}
    if ($Token) { $headers.Authorization = "Bearer $Token" }
    $params = @{Method=$Method;Uri=$base+$Path;Headers=$headers;WebSession=$session;SkipCertificateCheck=$true;SkipHttpErrorCheck=$true;TimeoutSec=15}
    if ($null -ne $Body) { $params.Body = $Body | ConvertTo-Json -Depth 30 -Compress; $params.ContentType='application/json' }
    $response = Invoke-WebRequest @params
    if ([int]$response.StatusCode -ne $Expected) { throw "$Path returned $($response.StatusCode), expected ${Expected}: $($response.Content)" }
    if ($response.Content) { return $response.Content | ConvertFrom-Json }
}
function Check([bool]$Success, [string]$Name) {
    if (-not $Success) { throw "Check failed: $Name" }
    $checks.Add($Name); Write-Output "PASS $Name"
}
function Sql([string]$Query) {
    $result = K -n activity exec mysql-activity-0 '--' sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -N -e "$1"' _ $Query
    return ($result -join "`n")
}
function Eventually([scriptblock]$Action, [string]$Name, [int]$Seconds=90) {
    $deadline = [DateTime]::UtcNow.AddSeconds($Seconds)
    $last = ''
    do {
        try { if (& $Action) { return } } catch { $last = $_.Exception.Message }
        Start-Sleep -Milliseconds 1000
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Timed out: $Name. $last"
}
Set-Location $Repo
$homepageResponse = Invoke-WebRequest "$base/" -SkipCertificateCheck -TimeoutSec 15
Check ($homepageResponse.StatusCode -eq 200 -and $homepageResponse.Content -match '<html') 'HTTPS frontend'
Api GET '/api/activity/v1/records/config' $null '' 401 | Out-Null
$account = 'k3s' + (Random-Hex).Substring(0,12)
$registration = @{account=$account;password=('Learn-'+(Random-Hex));nickname='K3s学习验收'}
$script:auth = $null
Eventually { $script:auth = Api POST '/api/identity/v1/auth/register' $registration; return [bool]$script:auth } 'registration admission'
$token = $auth.access_token
$user = [uint64]::Parse($auth.user_id)
Check ([bool]$token) 'registration and access token'
$cookie = $session.Cookies.GetCookies("$base/api/identity/v1/auth/refresh") | Where-Object Name -eq 'activity_refresh'
Check ([bool]$cookie -and $cookie.Secure -and $cookie.HttpOnly) 'Secure HttpOnly refresh cookie'
$script:joined = $null
Eventually { $script:joined = Api POST '/api/activity/v1/join' @{edition_id=$edition} $token; return [bool]$script:joined } 'account event initializes wallet'
Check ($script:joined.gold -eq 10000000 -and $script:joined.petals -eq 100000000) 'account event and MySQL wallet initialization'
$op = 'k3s-' + (Random-Hex).Substring(0,16)
$drawBody = @{edition_id=$edition;operation_id=$op;pool_code='NIGHT';count=1}
$draw = Api POST '/api/activity/v1/draw' $drawBody $token
$replay = Api POST '/api/activity/v1/draw' $drawBody $token
Check ($draw.draw_id -eq $replay.draw_id -and $draw.flowers -eq 3 -and $draw.consumed_petals -eq 100) 'draw, flower progress and idempotent replay'
Start-Sleep -Milliseconds 600
Api POST '/api/activity/v1/stage' @{edition_id=$edition;stage=3} $token | Out-Null
Start-Sleep -Milliseconds 600
$purchase = @{edition_id=$edition;operation_id=('buy-'+$op);product_code='52'}
$order = Api POST '/api/activity/v1/purchase' $purchase $token
Start-Sleep -Milliseconds 600
$orderAgain = Api POST '/api/activity/v1/purchase' $purchase $token
Check ($order.order_id -eq $orderAgain.order_id) 'stage reward and purchase idempotency'
Eventually {
    $board = Api GET "/api/ranking/v1/board?edition_id=$edition" $null $token
    return $board.me.score -eq 100 -and $board.me.nickname -eq 'K3s学习验收'
} 'RabbitMQ ranking projection and identity gRPC'
Check $true 'RabbitMQ ranking projection and identity gRPC'
$shard = $user % 4
$facts = Sql "SELECT COUNT(*) FROM activity_core.draw_order_${shard} WHERE user_id=$user; SELECT SUM(flowers_added) FROM activity_core.flower_lit_log_${shard} WHERE user_id=$user;"
Check ($facts -eq "1`n3") 'persistent draw and independent flower facts'

# Only this isolated local certificate is skipped; browser trust is not changed.
$sseLog = Join-Path $State 'sse-verification.txt'
$sseErr = Join-Path $State 'sse-verification-error.txt'
$curlArgs = @('-k','--silent','--no-buffer','--max-time','8','-H', ('"Authorization: Bearer {0}"' -f $token), "$base/api/notification/v1/barrage/stream")
$process = Start-Process -FilePath 'curl.exe' -ArgumentList $curlArgs -WindowStyle Hidden -PassThru -RedirectStandardOutput $sseLog -RedirectStandardError $sseErr
Eventually { return (Test-Path $sseLog) -and (Get-Content $sseLog -Raw) -match 'event: ready' } 'SSE ready' 6
$messageId = 'k3s-verify-' + (Random-Hex).Substring(0,8)
$message = @{schema_version=1;event_id=$messageId;event_type='ring_awarded';occurred_at=[DateTime]::UtcNow.ToString('o');expires_at=[DateTime]::UtcNow.AddSeconds(30).ToString('o');edition_id=$edition;source='normal_draw';display_name='K3s学习验收';item_id=1207244;item_name='烟花之恋戒指';quantity=1} | ConvertTo-Json -Compress
K -n activity exec redis-0 '--' redis-cli PUBLISH 'activity:k3s-learn:notification:ring-awarded:v1' $message | Out-Null
Eventually { return (Get-Content $sseLog -Raw) -match $messageId } 'SSE broadcast through TLS ingress' 6
$process.WaitForExit(10000) | Out-Null
Check $true 'SSE ready and live broadcast through TLS ingress'

if (-not $SkipRecovery) {
    $old = (K -n activity get pod -l app.kubernetes.io/name=identity -o json | ConvertFrom-Json).items[0]
    K -n activity delete pod $old.metadata.name
    Eventually {
        $pods = (K -n activity get pod -l app.kubernetes.io/name=identity -o json | ConvertFrom-Json).items
        if ($pods.Count -ne 1 -or $pods[0].metadata.uid -eq $old.metadata.uid) { return $false }
        $me = Api GET '/api/identity/v1/me' $null $token
        return $me.user_id -eq [string]$user
    } 'Deployment replaces deleted identity Pod'
    Check $true 'Deployment self healing with persistent session'
    $before = Sql "SELECT gold_balance,petal_balance,ledger_seq FROM activity_core.wallet WHERE user_id=$user;"
    K -n activity delete pod mysql-activity-0
    K -n activity wait --for=condition=Ready pod/mysql-activity-0 --timeout=180s
    $after = Sql "SELECT gold_balance,petal_balance,ledger_seq FROM activity_core.wallet WHERE user_id=$user;"
    Check ($before -eq $after) 'StatefulSet recreation preserves wallet on PVC'
    Eventually { return [bool](Api GET "/api/activity/v1/records/state?edition_id=$edition" $null $token) } 'API recovers after MySQL restart'
}

$canRead = & kubectl --kubeconfig $Kubeconfig -n activity auth can-i get pods --as=system:serviceaccount:activity:activity-reader
$canSecret = & kubectl --kubeconfig $Kubeconfig -n activity auth can-i get secrets --as=system:serviceaccount:activity:activity-reader
Check ($canRead -eq 'yes' -and $canSecret -eq 'no') 'read-only RBAC excludes Secrets'
$jobName = 'network-check-' + (Random-Hex).Substring(0,8)
Apply-Object @{apiVersion='batch/v1';kind='Job';metadata=@{name=$jobName;namespace='activity'};spec=@{ttlSecondsAfterFinished=120;backoffLimit=0;template=@{metadata=@{labels=@{'app.kubernetes.io/name'='untrusted-probe'}};spec=@{restartPolicy='Never';automountServiceAccountToken=$false;securityContext=@{runAsNonRoot=$true;runAsUser=999;seccompProfile=@{type='RuntimeDefault'}};containers=@(@{name='probe';image='redis:7.4-alpine';securityContext=@{allowPrivilegeEscalation=$false;readOnlyRootFilesystem=$true;capabilities=@{drop=@('ALL')}};env=@(@{name='REDISCLI_AUTH';valueFrom=@{secretKeyRef=@{name='redis-auth';key='password'}}});resources=@{requests=@{cpu='10m';memory='16Mi'};limits=@{cpu='100m';memory='32Mi'}};command=@('sh','-c','if timeout 4 redis-cli -h redis ping; then echo POLICY_BYPASSED; exit 1; else echo NETWORK_BLOCKED; fi')})}}}}
K -n activity wait --for=condition=complete "job/$jobName" --timeout=60s
$networkLog = K -n activity logs "job/$jobName"
Check (($networkLog -join "`n") -match 'NETWORK_BLOCKED') 'NetworkPolicy blocks an unauthorized Pod with valid credentials'
$renewed = Api POST '/api/identity/v1/auth/refresh' @{}
Check ([bool]$renewed.access_token) 'refresh token rotation'
Api POST '/api/identity/v1/auth/logout' @{} | Out-Null
Api GET '/api/identity/v1/me' $null $renewed.access_token 401 | Out-Null
Check $true 'logout revokes existing session'
$pods = (K -n activity get pods -o json | ConvertFrom-Json).items
$running = @($pods | Where-Object { $_.status.phase -eq 'Running' })
$rankingData = @($running | Where-Object { $_.metadata.labels.'activity-role' -eq 'ranking-data' })
$sentinels = @($running | Where-Object { $_.metadata.labels.'activity-role' -eq 'ranking-sentinel' })
Check ($running.Count -eq 19 -and $rankingData.Count -eq 8 -and $sentinels.Count -eq 3 -and @($running | Where-Object { $_.spec.containers.Count -ne 1 }).Count -eq 0) 'eight base workloads, eight ranking data Pods, and three Sentinels'
$encryption = & docker exec k3d-activity-learn-server-0 k3s secrets-encrypt status 2>$null
Check (($encryption -join "`n") -match 'Encryption Status: Enabled') 'K3s encrypts Kubernetes Secrets at rest'
@{verifiedAt=(Get-Date -Format 'yyyy-MM-dd HH:mm:ss zzz');url=$base;cluster='activity-learn';checks=$checks.ToArray();runningPods=$running.Count;imageTag=(Get-Content "$State/deployment.json" -Raw|ConvertFrom-Json).imageTag} | ConvertTo-Json -Depth 10 | Set-Content -Encoding utf8 "$State/verification.json"
Write-Output "Verification completed: $($checks.Count) checks. Evidence: $State/verification.json"

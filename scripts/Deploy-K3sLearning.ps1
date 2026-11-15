param(
    [string]$ImageTag = (Get-Date -Format 'yyyyMMddHHmmss'),
    [switch]$SkipBuild
)
. "$PSScriptRoot/K3s-Common.ps1"
if (-not (Test-Path $Kubeconfig)) { throw 'Run Install-K3sLearning.ps1 first' }
Set-Location $Repo
New-Item -ItemType Directory -Force $State | Out-Null
$secretPath = Join-Path $State 'credentials.json'
if (-not (Test-Path $secretPath)) {
    $existingRoot = & kubectl --kubeconfig $Kubeconfig -n activity get secret mysql-root --ignore-not-found -o name
    if ($existingRoot) { throw 'Cluster credentials already exist. Restore .local/k3s/credentials.json before deployment; automatic password replacement is unsafe.' }
    $values = @{}
    foreach ($key in @('mysqlRoot','identityDb','activityDb','rankingDb','redis','rabbitmq','erlang','internal')) { $values[$key] = Random-Hex }
    $values | ConvertTo-Json | Set-Content -Encoding utf8 $secretPath
    # Keep local development credentials private to the current Windows user.
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    & icacls.exe $secretPath /inheritance:r /grant:r "*${sid}:(F)" '*S-1-5-18:(F)' | Out-Null
    Native-Check 'credentials ACL'
}
$secrets = Get-Content $secretPath -Raw | ConvertFrom-Json -AsHashtable
$addedRankingSecrets = $false
foreach ($key in @('rankingRedis','rankingSentinel')) {
    if (-not $secrets.ContainsKey($key)) {
        $secrets[$key] = Random-Hex
        $addedRankingSecrets = $true
    }
}
if ($addedRankingSecrets) {
    $secrets | ConvertTo-Json | Set-Content -Encoding utf8 $secretPath
}
$keyDir = Join-Path $State 'keys'
& $Go run ./scripts/k8skeys $keyDir
Native-Check 'JWT key initialization'

# Local TLS: explicit trust is used by the verification script. No OS root is installed.
$certPath = Join-Path $State 'localhost.crt'
$keyPath = Join-Path $State 'localhost.key'
if (-not (Test-Path $certPath)) {
    $rsa = [Security.Cryptography.RSA]::Create(2048)
    $request = [Security.Cryptography.X509Certificates.CertificateRequest]::new('CN=localhost', $rsa, [Security.Cryptography.HashAlgorithmName]::SHA256, [Security.Cryptography.RSASignaturePadding]::Pkcs1)
    $san = [Security.Cryptography.X509Certificates.SubjectAlternativeNameBuilder]::new()
    $san.AddDnsName('localhost'); $san.AddIpAddress([Net.IPAddress]::Loopback)
    $request.CertificateExtensions.Add($san.Build())
    $request.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509BasicConstraintsExtension]::new($false,$false,0,$true))
    $cert = $request.CreateSelfSigned([DateTimeOffset]::Now.AddMinutes(-5), [DateTimeOffset]::Now.AddMonths(3))
    [IO.File]::WriteAllText($certPath, $cert.ExportCertificatePem())
    [IO.File]::WriteAllText($keyPath, $rsa.ExportRSAPrivateKeyPem())
    $rsa.Dispose(); $cert.Dispose()
}
K apply -f deploy/k8s/namespace.yaml
Apply-Secret mysql-root @{password=$secrets.mysqlRoot}
Apply-Secret redis-auth @{password=$secrets.redis}
Apply-Secret ranking-redis-auth @{password=$secrets.rankingRedis}
Apply-Secret ranking-sentinel-auth @{password=$secrets.rankingSentinel}
Apply-Secret rabbitmq-auth @{password=$secrets.rabbitmq; cookie=$secrets.erlang}
Apply-Secret rabbitmq-connection @{AMQP_URL="amqp://activity:$($secrets.rabbitmq)@rabbitmq:5672/"}
Apply-Secret internal-auth @{INTERNAL_TOKEN=$secrets.internal}
foreach ($name in @('identity','activity','ranking')) {
    $database = @{identity='activity_identity'; activity='activity_core'; ranking='activity_ranking'}[$name]
    $user = @{identity='identity_app'; activity='core_app'; ranking='ranking_app'}[$name]
    Apply-Secret "$name-db" @{MYSQL_DSN="${user}:$($secrets[$name+'Db'])@tcp(mysql:3306)/$database"}
}
Apply-Secret schema-core @{MYSQL_DSN="root:$($secrets.mysqlRoot)@tcp(mysql:3306)/activity_core"}
Apply-Secret jwt-private-keys @{'access.seed'=(Get-Content "$keyDir/access.seed" -Raw); 'refresh.seed'=(Get-Content "$keyDir/refresh.seed" -Raw)}
Apply-Object @{apiVersion='v1';kind='Secret';metadata=@{name='local-tls';namespace='activity'};type='kubernetes.io/tls';stringData=@{'tls.crt'=(Get-Content $certPath -Raw);'tls.key'=(Get-Content $keyPath -Raw)}}
Apply-Object @{apiVersion='v1';kind='ConfigMap';metadata=@{name='access-public-key';namespace='activity'};data=@{'access.pub'=(Get-Content "$keyDir/access.pub" -Raw)}}
$userSql = ''
foreach ($name in @('identity','activity','ranking')) {
    $database = @{identity='activity_identity'; activity='activity_core'; ranking='activity_ranking'}[$name]
    $user = @{identity='identity_app'; activity='core_app'; ranking='ranking_app'}[$name]
    $userSql += "CREATE USER '${user}'@'%' IDENTIFIED BY '$($secrets[$name+'Db'])';`nGRANT SELECT,INSERT,UPDATE,DELETE ON ${database}.* TO '${user}'@'%';`n"
}
Apply-Secret db-users-bootstrap @{'090_users.sql'=$userSql}
$migrationFiles = [ordered]@{
    '000_databases.sql'='deploy/local/mysql/000_databases.sql'
    '010_identity.sql'='services/identity-service/migrations/001_initial.sql'
    '020_activity.sql'='services/activity-service/migrations/001_initial.sql'
    '021_records.sql'='services/activity-service/migrations/002_records.sql'
    '022_projection.sql'='services/activity-service/migrations/003_ranking_projection.sql'
    '023_settlement.sql'='services/activity-service/migrations/004_settlement_scheduling.sql'
    '024_hourly_ranking.sql'='services/activity-service/migrations/005_hourly_ranking.sql'
    '030_ranking.sql'='services/ranking-service/migrations/001_initial.sql'
    '031_settlement.sql'='services/ranking-service/migrations/002_settlement_resume.sql'
    '032_hourly_ranking.sql'='services/ranking-service/migrations/003_hourly_ranking.sql'
}
$migrationData = @{}
foreach ($entry in $migrationFiles.GetEnumerator()) { $migrationData[$entry.Key] = Get-Content $entry.Value -Raw }
Apply-Object @{apiVersion='v1';kind='ConfigMap';metadata=@{name='db-migrations';namespace='activity'};data=$migrationData}
if (-not $SkipBuild) {
    foreach ($name in @('identity','activity','ranking','notification')) {
        & docker build -f deploy/k8s/Dockerfile.backend --build-arg "SERVICE=$name-service" -t "activity-k8s/${name}:$ImageTag" .
        Native-Check "$name image build"
    }
    & docker build -f deploy/k8s/Dockerfile.web -t "activity-k8s/web:$ImageTag" .
    Native-Check 'web image build'
}
$images = @('identity','activity','ranking','notification','web') | ForEach-Object { "activity-k8s/${_}:$ImageTag" }
foreach ($image in @('mysql:8.4','redis:7.4-alpine','rabbitmq:4.1-management')) {
    & docker image inspect $image *> $null
    if ($LASTEXITCODE -ne 0) { & docker pull $image; Native-Check "$image pull" }
}
$needsImport = -not $SkipBuild
if ($SkipBuild) {
    foreach ($node in @('server-0','agent-0','agent-1')) {
        $existingImages = & docker exec "k3d-activity-learn-$node" ctr images list -q
        Native-Check 'node image inventory'
        foreach ($image in $images) { if ("docker.io/$image" -notin $existingImages) { $needsImport = $true } }
    }
}
if ($needsImport) {
$archive = Join-Path $State 'images-amd64.tar'
# Docker Desktop's containerd store can hold multi-platform indexes whose
# attestation/platform content is absent locally. Export only the used platform.
& docker image save --platform linux/amd64 -o $archive @images mysql:8.4 redis:7.4-alpine rabbitmq:4.1-management
Native-Check 'single-platform image export'
foreach ($node in @('server-0','agent-0','agent-1')) {
    # Explicit copies avoid Windows tar-copy issues and false success reported
    # by k3d image import; ctr targets the same containerd used by kubelet.
    & docker cp $archive "k3d-activity-learn-${node}:/tmp/activity-images.tar"
    Native-Check 'copy image archive'
    & docker exec "k3d-activity-learn-$node" ctr -n k8s.io images import /tmp/activity-images.tar
    Native-Check 'containerd import'
    $imported = & docker exec "k3d-activity-learn-$node" ctr images list -q
    Native-Check 'node image inventory'
    foreach ($image in $images) {
        if ("docker.io/$image" -notin $imported) { throw "Image missing on ${node}: $image" }
    }
}
} else { Write-Output 'All application images are already imported on all three nodes.' }
$overlay = Join-Path $State 'overlay'
New-Item -ItemType Directory -Force $overlay | Out-Null
$kustomization = "apiVersion: kustomize.config.k8s.io/v1beta1`nkind: Kustomization`nresources:`n  - ../../../deploy/k8s`nimages:`n"
foreach ($name in @('identity','activity','ranking','notification','web')) { $kustomization += "  - name: activity-k8s/$name`n    newTag: '$ImageTag'`n" }
Set-Content -Encoding utf8 "$overlay/kustomization.yaml" $kustomization
K apply -k $overlay
foreach ($name in @('mysql','redis','rabbitmq')) { K -n activity rollout status "statefulset/$name" --timeout=300s }
foreach ($name in @('ranking-redis-0','ranking-redis-1','ranking-redis-2','ranking-redis-3','ranking-sentinel')) {
    K -n activity rollout status "statefulset/$name" --timeout=300s
}
Wait-RankingRedisTopology
$existingSchema = & kubectl --kubeconfig $Kubeconfig -n activity get job activity-schema-v1 --ignore-not-found -o json
if (-not $existingSchema) { K apply -f deploy/k8s/schema-job.yaml }
elseif (($existingSchema | ConvertFrom-Json).status.succeeded -ne 1) { throw 'Existing schema Job did not complete; inspect its logs and database state before retrying.' }
K -n activity wait --for=condition=complete job/activity-schema-v1 --timeout=240s
$seedJob = (Get-Content deploy/k8s/seed-job.yaml -Raw).Replace('activity-k8s/activity:dev',"activity-k8s/activity:$ImageTag")
Set-Content -Encoding utf8 "$State/seed-job.yaml" $seedJob
$existingSeed = & kubectl --kubeconfig $Kubeconfig -n activity get job activity-seed-v1 --ignore-not-found -o json
if (-not $existingSeed) { K apply -f "$State/seed-job.yaml" }
elseif (($existingSeed | ConvertFrom-Json).status.succeeded -ne 1) { throw 'Existing seed Job did not complete; inspect its logs before retrying.' }
K -n activity wait --for=condition=complete job/activity-seed-v1 --timeout=120s
foreach ($name in @('identity','activity','ranking','notification','web')) { K -n activity rollout status "deployment/$name" --timeout=180s }
@{imageTag=$ImageTag;url='https://localhost:18443';cluster='activity-learn';edition='2026110601'} | ConvertTo-Json | Set-Content -Encoding utf8 "$State/deployment.json"
K -n activity get 'pods,svc,ingress,pvc' -o wide
Write-Output 'Open https://localhost:18443 (local self-signed certificate).'

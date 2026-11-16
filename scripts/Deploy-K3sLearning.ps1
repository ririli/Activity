param(
    [string]$ImageTag = (Get-Date -Format 'yyyyMMddHHmmss'),
    [switch]$SkipBuild,
    [switch]$PrepareDatabasesOnly,
    [switch]$ConfirmMySQLCutover
)
. "$PSScriptRoot/K3s-Common.ps1"
if (-not (Test-Path $Kubeconfig)) { throw 'Run Install-K3sLearning.ps1 first' }
Set-Location $Repo
$legacyMySQL = & kubectl --kubeconfig $Kubeconfig -n activity get statefulset mysql --ignore-not-found -o name
Native-Check 'legacy MySQL inventory'
if ($legacyMySQL -and -not ($PrepareDatabasesOnly -or $ConfirmMySQLCutover)) {
    foreach ($name in @('identity','activity','ranking')) {
        $reference = & kubectl --kubeconfig $Kubeconfig -n activity get deployment $name --ignore-not-found -o 'jsonpath={.spec.template.spec.containers[*].env[?(@.name=="MYSQL_DSN")].valueFrom.secretKeyRef.name}'
        Native-Check "$name database connection inventory"
        if ($reference -ne "$name-split-db") {
            throw 'Shared MySQL still exists. Prepare the three databases with -PrepareDatabasesOnly, migrate data, then rerun with -ConfirmMySQLCutover. See deploy/k8s/README.md.'
        }
    }
}
New-Item -ItemType Directory -Force $State | Out-Null
$secretPath = Join-Path $State 'credentials.json'
if (-not (Test-Path $secretPath)) {
    $existingRoot = & kubectl --kubeconfig $Kubeconfig -n activity get secret mysql-root mysql-identity-root mysql-activity-root mysql-ranking-root --ignore-not-found -o name
    Native-Check 'MySQL credential inventory'
    if ($existingRoot) { throw 'Cluster credentials already exist. Restore .local/k3s/credentials.json before deployment; automatic password replacement is unsafe.' }
    $values = @{}
    foreach ($key in @('identityDb','activityDb','rankingDb','redis','rabbitmq','erlang','internal')) { $values[$key] = Random-Hex }
    $values | ConvertTo-Json | Set-Content -Encoding utf8 $secretPath
    # Keep local development credentials private to the current Windows user.
    $sid = [Security.Principal.WindowsIdentity]::GetCurrent().User.Value
    & icacls.exe $secretPath /inheritance:r /grant:r "*${sid}:(F)" '*S-1-5-18:(F)' | Out-Null
    Native-Check 'credentials ACL'
}
$secrets = Get-Content $secretPath -Raw | ConvertFrom-Json -AsHashtable
$addedSecrets = $false
foreach ($key in @('rankingRedis','rankingSentinel','identityMySQLRoot','activityMySQLRoot','rankingMySQLRoot')) {
    if (-not $secrets.ContainsKey($key)) {
        $secrets[$key] = Random-Hex
        $addedSecrets = $true
    }
}
if ($addedSecrets) {
    $secrets | ConvertTo-Json | Set-Content -Encoding utf8 $secretPath
}
$databases = [ordered]@{identity='activity_identity'; activity='activity_core'; ranking='activity_ranking'}
$users = @{identity='identity_app'; activity='core_app'; ranking='ranking_app'}
K apply -f deploy/k8s/namespace.yaml
foreach ($name in $databases.Keys) {
    $database = $databases[$name]
    $user = $users[$name]
    Apply-Secret "mysql-$name-root" @{password=$secrets[$name+'MySQLRoot']}
    $migrationData = @{'000_database.sql'="CREATE DATABASE IF NOT EXISTS $database CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_bin;"}
    $snowflakeData = @{}
    foreach ($file in Get-ChildItem "services/$name-service/migrations" -Filter '*.sql' -File) {
        $target = if ($file.Name -like '*_snowflake.sql') { $snowflakeData } else { $migrationData }
        $target[$file.Name] = Get-Content -LiteralPath $file.FullName -Raw
    }
    Apply-Object @{apiVersion='v1';kind='ConfigMap';metadata=@{name="mysql-$name-migrations-v1";namespace='activity'};data=$migrationData}
    Apply-Object @{apiVersion='v1';kind='ConfigMap';metadata=@{name="mysql-$name-migrations-v2";namespace='activity'};data=$snowflakeData}
    $userSql = "CREATE USER '${user}'@'%' IDENTIFIED BY '$($secrets[$name+'Db'])';`nGRANT SELECT,INSERT,UPDATE,DELETE ON ${database}.* TO '${user}'@'%';`n"
    Apply-Secret "mysql-$name-user-bootstrap-v1" @{'090_users.sql'=$userSql}
}

function Initialize-MySQL {
    K apply -k deploy/k8s/mysql
    foreach ($name in $databases.Keys) { K -n activity rollout status "statefulset/mysql-$name" --timeout=300s }
    foreach ($name in $databases.Keys) {
        foreach ($version in @('v1','v2')) {
            $job = "mysql-$name-schema-$version"
            $existing = & kubectl --kubeconfig $Kubeconfig -n activity get job $job --ignore-not-found -o json
            Native-Check "$job inventory"
            if (-not $existing) {
                $bootstrapOptional = if ($version -eq 'v1') { 'false' } else { 'true' }
                $manifest = (Get-Content deploy/k8s/schema-job.yaml -Raw).Replace('DB_SERVICE',$name).Replace('SCHEMA_VERSION',$version).Replace('BOOTSTRAP_OPTIONAL',$bootstrapOptional)
                $manifest | & kubectl --kubeconfig $Kubeconfig apply -f -
                Native-Check "$job creation"
            } elseif (($existing | ConvertFrom-Json).status.succeeded -ne 1) {
                throw "Existing schema Job $job did not complete. Inspect its logs and database state before retrying."
            }
            K -n activity wait --for=condition=complete "job/$job" --timeout=240s
        }
    }
}

if ($PrepareDatabasesOnly) { Initialize-MySQL; return }
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
Apply-Secret redis-auth @{password=$secrets.redis}
Apply-Secret ranking-redis-auth @{password=$secrets.rankingRedis}
Apply-Secret ranking-sentinel-auth @{password=$secrets.rankingSentinel}
Apply-Secret rabbitmq-auth @{password=$secrets.rabbitmq; cookie=$secrets.erlang}
Apply-Secret rabbitmq-connection @{AMQP_URL="amqp://activity:$($secrets.rabbitmq)@rabbitmq:5672/"}
Apply-Secret internal-auth @{INTERNAL_TOKEN=$secrets.internal}
Apply-Secret jwt-private-keys @{'access.seed'=(Get-Content "$keyDir/access.seed" -Raw); 'refresh.seed'=(Get-Content "$keyDir/refresh.seed" -Raw)}
Apply-Object @{apiVersion='v1';kind='Secret';metadata=@{name='local-tls';namespace='activity'};type='kubernetes.io/tls';stringData=@{'tls.crt'=(Get-Content $certPath -Raw);'tls.key'=(Get-Content $keyPath -Raw)}}
Apply-Object @{apiVersion='v1';kind='ConfigMap';metadata=@{name='access-public-key';namespace='activity'};data=@{'access.pub'=(Get-Content "$keyDir/access.pub" -Raw)}}
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
Initialize-MySQL
foreach ($name in $databases.Keys) {
    $database = $databases[$name]
    $user = $users[$name]
    Apply-Secret "$name-split-db" @{MYSQL_DSN="${user}:$($secrets[$name+'Db'])@tcp(mysql-${name}:3306)/$database"}
}
K apply -k $overlay
foreach ($name in @('redis','rabbitmq')) { K -n activity rollout status "statefulset/$name" --timeout=300s }
foreach ($name in @('ranking-redis-0','ranking-redis-1','ranking-redis-2','ranking-redis-3','ranking-sentinel')) {
    K -n activity rollout status "statefulset/$name" --timeout=300s
}
Wait-RankingRedisTopology
$seedEdition = K -n activity exec mysql-activity-0 '--' sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -Nse "$1"' _ 'SELECT EXISTS(SELECT 1 FROM activity_core.activity_edition WHERE edition_id=2026110601)'
if ($seedEdition -ne '1') {
    $seedJob = (Get-Content deploy/k8s/seed-job.yaml -Raw).Replace('activity-k8s/activity:dev',"activity-k8s/activity:$ImageTag")
    Set-Content -Encoding utf8 "$State/seed-job.yaml" $seedJob
    $existingSeed = & kubectl --kubeconfig $Kubeconfig -n activity get job activity-seed-v2 --ignore-not-found -o json
    Native-Check 'seed Job inventory'
    if ($existingSeed) { throw 'Seed Job exists but the seed edition is missing. Inspect its logs and database state before retrying.' }
    K apply -f "$State/seed-job.yaml"
    K -n activity wait --for=condition=complete job/activity-seed-v2 --timeout=120s
}
foreach ($name in @('identity','activity','ranking','notification','web')) { K -n activity rollout status "deployment/$name" --timeout=180s }
@{imageTag=$ImageTag;url='https://localhost:18443';cluster='activity-learn';edition='2026110601'} | ConvertTo-Json | Set-Content -Encoding utf8 "$State/deployment.json"
K -n activity get 'pods,svc,ingress,pvc' -o wide
Write-Output 'Open https://localhost:18443 (local self-signed certificate).'

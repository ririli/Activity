# Requires PowerShell 7 and a running Docker Desktop Linux engine.
param([string]$Cluster = 'activity-learn')
. "$PSScriptRoot/K3s-Common.ps1"
New-Item -ItemType Directory -Force (Join-Path $State 'tools') | Out-Null
$version = 'v5.9.0'
if (-not (Test-Path $K3d)) {
    & curl.exe -fsSL --retry 3 --connect-timeout 10 --max-time 120 "https://github.com/k3d-io/k3d/releases/download/$version/k3d-windows-amd64.exe" -o $K3d
    Native-Check 'k3d download'
}
$checksums = Join-Path $State 'tools/checksums.txt'
& curl.exe -fsSL --retry 3 --connect-timeout 10 --max-time 60 "https://github.com/k3d-io/k3d/releases/download/$version/checksums.txt" -o $checksums
Native-Check 'checksum download'
$expected = ((Get-Content $checksums | Where-Object { $_ -match 'k3d-windows-amd64.exe$' }) -split '\s+')[0]
if ((Get-FileHash $K3d -Algorithm SHA256).Hash -ne $expected) { throw 'k3d SHA256 mismatch' }
$clusters = & $K3d cluster list -o json | ConvertFrom-Json
Native-Check 'cluster inventory'
if ($Cluster -notin @($clusters.name)) {
    & $K3d cluster create $Cluster --image rancher/k3s:v1.35.5-k3s1 --servers 1 --agents 2 --api-port 127.0.0.1:6551 --port '127.0.0.1:18080:80@loadbalancer' --port '127.0.0.1:18443:443@loadbalancer' --k3s-arg '--secrets-encryption@server:*' --kubeconfig-update-default=false --kubeconfig-switch-context=false --wait --timeout 240s
    Native-Check 'cluster creation'
} else {
    & $K3d cluster start $Cluster
    Native-Check 'cluster start'
}
& $K3d kubeconfig get $Cluster | Set-Content -Encoding utf8 $Kubeconfig
Native-Check 'kubeconfig export'
K wait --for=condition=Ready nodes --all --timeout=180s
K get nodes -o wide

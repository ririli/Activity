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

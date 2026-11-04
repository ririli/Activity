param(
    [string]$Buf = "buf",
    [string]$ProtocGenGo = "protoc-gen-go",
    [string]$ProtocGenGoGrpc = "protoc-gen-go-grpc"
)

$ErrorActionPreference = "Stop"

foreach ($tool in @($Buf, $ProtocGenGo, $ProtocGenGoGrpc)) {
    if (-not (Get-Command $tool -ErrorAction SilentlyContinue)) {
        throw "Missing $tool. Install pinned versions: buf v1.75.0, protoc-gen-go v1.36.12, protoc-gen-go-grpc v1.6.2."
    }
}

$root = Split-Path -Parent $PSScriptRoot
Push-Location (Join-Path $root "contracts/rpc")
try {
    & $Buf generate
    if ($LASTEXITCODE -ne 0) {
        throw "buf generate failed with exit code $LASTEXITCODE"
    }
} finally {
    Pop-Location
}

Write-Host "Generated contracts/rpc/identity/v1/*.pb.go. Commit generated files with the proto change."

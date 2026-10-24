$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
    $env:CGO_ENABLED = '0'
    $env:GOOS = 'linux'
    $env:GOARCH = 'arm'
    $env:GOARM = '7'
    go build -trimpath -ldflags '-s -w' -o zoordns-linux-armv7 .
    if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
    Get-FileHash zoordns-linux-armv7 -Algorithm SHA256
} finally {
    Remove-Item Env:CGO_ENABLED,Env:GOOS,Env:GOARCH,Env:GOARM -ErrorAction SilentlyContinue
    Pop-Location
}

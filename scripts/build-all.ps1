# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
param(
    [string]$GoBinary = 'go',
    [string]$PythonBinary = 'python',
    [string[]]$Architectures = @()
)
$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path $PSScriptRoot -Parent
if ($Architectures.Count -eq 0) { $Architectures = Get-Content (Join-Path $projectRoot 'targets.json') -Raw | ConvertFrom-Json }
$savedEnvironment = @{}
foreach ($name in @('CGO_ENABLED','GOOS','GOARCH','GOARM','GOMIPS','GOMIPS64','GOAMD64','GO386')) { $savedEnvironment[$name] = [Environment]::GetEnvironmentVariable($name) }
$built = @{}
Push-Location $projectRoot
try {
    foreach ($name in @('GOOS','GOARCH','GOARM','GOMIPS','GOMIPS64','GOAMD64','GO386')) { [Environment]::SetEnvironmentVariable($name, $null) }
    foreach ($architecture in $Architectures) {
        foreach ($name in @('GOARM','GOMIPS','GOMIPS64','GOAMD64','GO386')) { [Environment]::SetEnvironmentVariable($name, $null) }
        $env:CGO_ENABLED = '0'; $env:GOOS = 'linux'
        switch -Regex ($architecture) {
            '^arm_arm1176' { $env:GOARCH='arm'; $env:GOARM='6'; break }
            '^arm_(xscale|fa526|arm926)' { $env:GOARCH='arm'; $env:GOARM='5'; break }
            '^arm_cortex' { $env:GOARCH='arm'; $env:GOARM='7'; break }
            '^aarch64_' { $env:GOARCH='arm64'; break }
            '^mipsel_' { $env:GOARCH='mipsle'; $env:GOMIPS='softfloat'; break }
            '^mips_' { $env:GOARCH='mips'; $env:GOMIPS='softfloat'; break }
            '^x86_64$' { $env:GOARCH='amd64'; $env:GOAMD64='v1'; break }
            '^i386_' { $env:GOARCH='386'; $env:GO386='softfloat'; break }
            '^riscv64_' { $env:GOARCH='riscv64'; break }
            default { throw "Unsupported architecture: $architecture" }
        }
        $key = "$env:GOARCH-$env:GOARM"
        $destination = Join-Path $projectRoot "prebuilt/$architecture/rozaneh"
        New-Item -ItemType Directory -Path (Split-Path $destination -Parent) -Force | Out-Null
        if ($built.ContainsKey($key)) {
            Copy-Item -LiteralPath $built[$key] -Destination $destination -Force
            $sourceDirectory = Split-Path $built[$key] -Parent
            $destinationDirectory = Split-Path $destination -Parent
            Copy-Item -LiteralPath (Join-Path $sourceDirectory 'cotten-source.json') -Destination $destinationDirectory -Force
            Copy-Item -LiteralPath (Join-Path $sourceDirectory 'engine-licenses') -Destination $destinationDirectory -Recurse -Force
        }
        else {
            $buildArguments = @('scripts/build-cotten.py','--output',$destination,'--goarch',$env:GOARCH)
            if ($env:GOARCH -eq 'arm') { $buildArguments += @('--goarm',$env:GOARM) }
            if ($built.Count -eq 0) { $buildArguments += '--test' }
            & $PythonBinary @buildArguments
            if ($LASTEXITCODE -ne 0) { throw "Build failed: $architecture" }
            $built[$key] = $destination
        }
        Write-Output "Built $architecture"
    }
} finally {
    foreach ($name in $savedEnvironment.Keys) { [Environment]::SetEnvironmentVariable($name, $savedEnvironment[$name]) }
    Pop-Location
}

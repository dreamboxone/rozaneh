# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
param([string]$PythonBinary = 'python')
$ErrorActionPreference = 'Stop'
Push-Location $PSScriptRoot
try {
    & $PythonBinary scripts/build-cotten.py --output prebuilt/arm_cortex-a7_neon-vfpv4/rozaneh --goarch arm --goarm 7 --test
    if ($LASTEXITCODE -ne 0) { throw 'CottenDNS build or tests failed' }
    Get-FileHash prebuilt/arm_cortex-a7_neon-vfpv4/rozaneh -Algorithm SHA256
} finally { Pop-Location }

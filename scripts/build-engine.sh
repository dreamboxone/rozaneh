#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
arch=${1:?Usage: build-engine.sh OPENWRT_ARCH}
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"
export GOOS=linux CGO_ENABLED=0
unset GOARM GOMIPS GOMIPS64
case "$arch" in
 arm_cortex-a*|arm_arm1176*) export GOARCH=arm GOARM=7 ;;
 arm_xscale|arm_fa526|arm_arm926ej-s) export GOARCH=arm GOARM=5 ;;
 aarch64_*) export GOARCH=arm64 ;;
 mipsel_*) export GOARCH=mipsle GOMIPS=softfloat ;;
 mips_*) export GOARCH=mips GOMIPS=softfloat ;;
 mips64el_*) export GOARCH=mips64le GOMIPS64=softfloat ;;
 mips64_*) export GOARCH=mips64 GOMIPS64=softfloat ;;
 x86_64) export GOARCH=amd64 GOAMD64=v1 ;;
 i386_*) export GOARCH=386 GO386=softfloat ;;
 riscv64_*) export GOARCH=riscv64 ;;
 *) echo "Unsupported Go target: $arch" >&2; exit 1 ;;
esac
case "$arch" in arm_arm1176*) export GOARM=6 ;; esac
mkdir -p "$root/prebuilt/$arch"
cd "$root"
python3 scripts/build-cotten.py --output "prebuilt/$arch/rozaneh" --goarch "$GOARCH" --goarm "${GOARM:-7}" --test

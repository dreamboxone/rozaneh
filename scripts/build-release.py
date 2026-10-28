#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
"""Cross-build the complete application for every published architecture."""
import json
import os
from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]


def target_environment(architecture):
    if architecture.startswith("aarch64_"):
        return {"GOARCH": "arm64"}
    if architecture.startswith("arm_arm1176"):
        return {"GOARCH": "arm", "GOARM": "6"}
    if architecture.startswith("arm_cortex"):
        return {"GOARCH": "arm", "GOARM": "7"}
    if architecture == "arm_xscale":
        return {"GOARCH": "arm", "GOARM": "5"}
    if architecture.startswith("mipsel_"):
        return {"GOARCH": "mipsle", "GOMIPS": "softfloat"}
    if architecture.startswith("mips_"):
        return {"GOARCH": "mips", "GOMIPS": "softfloat"}
    if architecture == "x86_64":
        return {"GOARCH": "amd64", "GOAMD64": "v1"}
    if architecture == "i386_pentium4":
        return {"GOARCH": "386", "GO386": "sse2"}
    if architecture == "riscv64_riscv64":
        return {"GOARCH": "riscv64"}
    raise ValueError("Unsupported architecture: " + architecture)


def main():
    environment = dict(os.environ)
    for name in ("GOOS", "GOARCH", "GOARM", "GOMIPS", "GOMIPS64", "GOAMD64", "GO386"):
        environment.pop(name, None)
    cotten = (ROOT / "cotten-upstream.json").exists()
    if not cotten:
        subprocess.run(["go", "test", "-count=1", "-timeout=120s", "./..."], cwd=ROOT, env=environment, check=True)
    built = {}
    for architecture in json.loads((ROOT / "targets.json").read_text()):
        target = target_environment(architecture)
        key = tuple(sorted(target.items()))
        destination = ROOT / "prebuilt" / architecture / "rozaneh"
        destination.parent.mkdir(parents=True, exist_ok=True)
        if key in built:
            shutil.copyfile(built[key], destination)
            if cotten:
                shutil.copyfile(built[key].parent/'cotten-source.json',destination.parent/'cotten-source.json')
                shutil.copytree(built[key].parent/'engine-licenses',destination.parent/'engine-licenses',dirs_exist_ok=True)
        else:
            if cotten:
                command=["python3","scripts/build-cotten.py","--output",str(destination),"--goarch",target["GOARCH"]]
                if "GOARM" in target: command += ["--goarm",target["GOARM"]]
                if not built: command += ["--test"]
                subprocess.run(command,cwd=ROOT,env={**environment,**target},check=True)
            else:
                subprocess.run(["go", "build", "-trimpath", "-ldflags=-s -w", "-o", str(destination), "."], cwd=ROOT, env={**environment, "CGO_ENABLED": "0", "GOOS": "linux", **target}, check=True)
            built[key] = destination
        print("Built " + architecture, flush=True)


if __name__ == "__main__":
    main()

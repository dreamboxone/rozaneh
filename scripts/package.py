#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
"""Package static Rozaneh binaries and LuCI resources as OpenWrt IPK/APK."""
import argparse
import hashlib
import io
import json
import os
import re
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parent.parent
VERSIONS = {"rozaneh": "1.0.0-r3"}


def soar_version():
    lock_file = ROOT / "cotten-upstream.json"
    if lock_file.exists():
        lock=json.loads(lock_file.read_text())
        return lock.get("release",lock["revision"])
    match = re.search(r"github\.com/getlantern/soar\s+(v[^\s]+)", (ROOT / "go.mod").read_text())
    if not match:
        raise ValueError("Missing pinned Soar dependency")
    return match.group(1)


def put(stage, source, destination, mode=0o644):
    target = stage / destination.lstrip("/")
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    target.chmod(mode)


def postinst(component):
    common = '#!/bin/sh\n[ -n "${IPKG_INSTROOT}" ] && exit 0\n'
    if component == "rozaneh":
        return common + '''. /lib/functions.sh
gid=$(group_add_next rozaneh)
user_exists rozaneh || user_add rozaneh "" "$gid" "Rozaneh" /var/run/rozaneh /bin/false
# The launcher is package code; user settings stay in /etc/config/rozaneh.
cp /usr/share/rozaneh/service.init /etc/init.d/rozaneh || exit 1
chmod 755 /etc/init.d/rozaneh || exit 1
/usr/libexec/rozaneh/migrate-paths.sh || exit 1
/etc/init.d/rozaneh enable
/etc/init.d/rpcd reload >/dev/null 2>&1 || true
rm -f /tmp/luci-indexcache.*
[ "$(uci -q get rozaneh.main.enabled)" = 1 ] && /etc/init.d/rozaneh restart
exit 0
'''
    return common + '''/etc/init.d/rpcd reload >/dev/null 2>&1 || /etc/init.d/rpcd restart >/dev/null 2>&1 || true
rm -f /tmp/luci-indexcache.*
exit 0
'''


def stage_component(stage, component, arch):
    if component == "rozaneh":
        binary = ROOT / "prebuilt" / arch / "rozaneh"
        if not binary.is_file():
            raise SystemExit(f"Missing binary for {arch}; run scripts/build-engine.sh first")
        lock=json.loads((ROOT/'cotten-upstream.json').read_text())
        metadata=json.loads((binary.parent/'cotten-source.json').read_text())
        if metadata['revision']!=lock['revision'] or metadata['sha256']!=hashlib.sha256(binary.read_bytes()).hexdigest() or metadata['application_version']!=VERSIONS['rozaneh']:
            raise SystemExit('Engine build metadata does not match this package')
        for license_file in (binary.parent/'engine-licenses').rglob('*'):
            if license_file.is_file():
                put(stage,license_file,'usr/share/licenses/rozaneh/engine/'+license_file.relative_to(binary.parent/'engine-licenses').as_posix())
        put(stage, binary, "usr/bin/rozaneh", 0o755)
        put(stage, ROOT / "files/rozaneh.init", "etc/init.d/rozaneh", 0o755)
        put(stage, ROOT / "files/rozaneh.init", "usr/share/rozaneh/service.init", 0o755)
        put(stage, ROOT / "files/rozaneh.config", "etc/config/rozaneh", 0o600)
        for helper in ("cotten-config.sh", "dependencies.sh", "lan-config.sh", "lan-run.sh", "lan-cleanup.sh", "lan-firewall.sh", "health-check.sh", "health-run.sh", "migrate-paths.sh"):
            put(stage, ROOT / "files" / helper, "usr/libexec/rozaneh/" + helper, 0o755)
        licenses = [ROOT / x for x in ("LICENSE", "SOAR-LICENSE", "NOTICE")]
        licenses += list((ROOT / "licenses").iterdir())
        put(stage, ROOT / "internal/tunnelengine/LICENSE", "usr/share/licenses/rozaneh/CottenMax-engine-LICENSE")
        put(stage, ROOT / "internal/tunnelengine/NOTICE.cottendns.txt", "usr/share/licenses/rozaneh/CottenMax-engine-NOTICE")
        for file in (ROOT / "luci/root").rglob("*"):
            if file.is_file():
                relative = file.relative_to(ROOT / "luci/root")
                put(stage, file, relative.as_posix(), 0o755 if relative.as_posix().startswith("usr/libexec/") else 0o644)
        for name in ("LICENSE", "NOTICE"):
            put(stage, ROOT / "luci" / name, f"usr/share/licenses/rozaneh-ui/{name}")
    else:
        for file in (ROOT / "luci/root").rglob("*"):
            if file.is_file():
                relative = file.relative_to(ROOT / "luci/root")
                mode = 0o755 if relative.as_posix().startswith("usr/libexec/") else 0o644
                put(stage, file, relative.as_posix(), mode)
        licenses = [ROOT / "luci" / x for x in ("LICENSE", "NOTICE")]
    for file in licenses:
        put(stage, file, f"usr/share/licenses/rozaneh/{file.name}")
    version = stage / "usr/share/rozaneh" / ("core.version" if component == "rozaneh" else "luci.version")
    version.parent.mkdir(parents=True, exist_ok=True)
    version.write_text(VERSIONS[component] + "\n", encoding="utf-8")
    if component == "rozaneh":
        (version.parent / "luci.version").write_text(VERSIONS[component] + "\n", encoding="utf-8")
        (version.parent / "engine.version").write_text(soar_version() + "\n", encoding="utf-8")


def archive(root):
    stream = io.BytesIO()
    with tarfile.open(fileobj=stream, mode="w:gz", format=tarfile.GNU_FORMAT) as tar:
        for file in sorted(root.rglob("*")):
            relative = file.relative_to(root).as_posix()
            info = tar.gettarinfo(str(file), "./" + relative)
            info.uid = info.gid = 0
            info.uname = info.gname = "root"
            if info.isdir():
                info.mode = 0o755
            elif relative == "etc/config/rozaneh":
                info.mode = 0o600
            elif relative.startswith(("usr/bin/", "usr/libexec/", "etc/init.d/")) or relative in ("preinst", "postinst", "prerm", "postrm"):
                info.mode = 0o755
            else:
                info.mode = 0o644
            if file.is_file():
                with file.open("rb") as handle:
                    tar.addfile(info, handle)
            else:
                tar.addfile(info)
    return stream.getvalue()


def ar_write(path, members):
    with path.open("wb") as output:
        output.write(b"!<arch>\n")
        for name, data in members:
            header = f"{name + '/':<16}{0:<12}{0:<6}{0:<6}{'100644':<8}{len(data):<10}`\n"
            output.write(header.encode("ascii"))
            output.write(data)
            if len(data) % 2:
                output.write(b"\n")


def build(component, arch, fmt, out, apk_tool):
    version = VERSIONS[component]
    package_arch = "all" if component != "rozaneh" else arch
    destination = out / f"{component}-{version}-{package_arch}.{fmt}"
    with tempfile.TemporaryDirectory(prefix="rozaneh-package-") as temporary:
        temp = Path(temporary)
        stage = temp / "data"
        stage.mkdir()
        stage_component(stage, component, arch)
        control = temp / "control"
        control.mkdir()
        (control / "postinst").write_text(postinst(component), encoding="utf-8")
        (control / "postinst").chmod(0o755)
        (control / "preinst").write_text('''#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
if [ -x /etc/init.d/zoordns ]; then
 /etc/init.d/zoordns stop || exit 1
 /etc/init.d/zoordns disable
fi
exit 0
''', encoding="utf-8")
        deps = ["libc", "jshn", "curl", "ca-bundle", "luci-base", "rpcd", "jsonfilter"]
        license_name = "Apache-2.0 AND AGPL-3.0-or-later AND MIT AND BSD-3-Clause AND OFL-1.1"
        if component == "rozaneh":
            (control / "conffiles").write_text("/etc/config/rozaneh\n/etc/rozaneh/traffic.json\n", encoding="utf-8")
            (control / "prerm").write_text('#!/bin/sh\n[ -n "${IPKG_INSTROOT}" ] || /etc/init.d/rozaneh stop\nexit 0\n', encoding="utf-8")
            (control / "prerm").chmod(0o755)
            (control / "postrm").write_text('''#!/bin/sh
[ -n "${IPKG_INSTROOT}" ] && exit 0
if [ ! -x /usr/bin/rozaneh ]; then
 nft delete table inet rozaneh_guard 2>/dev/null || true
 if [ "$(uci -q get firewall.rozaneh_guard.path)" = /usr/libexec/rozaneh/lan-firewall.sh ]; then
  uci -q delete firewall.rozaneh_guard
  uci commit firewall
 fi
fi
exit 0
''', encoding="utf-8")
            (control / "postrm").chmod(0o755)
        if fmt == "ipk":
            metadata = f"Package: {component}\nVersion: {version}\nArchitecture: {package_arch}\nMaintainer: dreamboxone\nSection: net\nPriority: optional\nDepends: {', '.join(deps)}\nLicense: {license_name}\nDescription: Rozaneh DNS tunnel client for OpenWrt\n"
            (control / "control").write_text(metadata, encoding="utf-8")
            with (control / "control").open("a", encoding="utf-8") as handle:
                handle.write("Replaces: zoordns, luci-app-zoordns\nConflicts: zoordns, luci-app-zoordns\nProvides: zoordns, luci-app-zoordns\n")
            ar_write(destination, [("debian-binary", b"2.0\n"), ("control.tar.gz", archive(control)), ("data.tar.gz", archive(stage))])
        else:
            if not apk_tool:
                raise SystemExit("APK_TOOL or --apk-tool must point to the OpenWrt SDK apk tool")
            if component == "rozaneh":
                metadata_dir = stage / "lib/apk/packages"
                metadata_dir.mkdir(parents=True, exist_ok=True)
                config = stage / "etc/config/rozaneh"
                digest = hashlib.sha256(config.read_bytes()).hexdigest()
                (metadata_dir / "rozaneh.conffiles_static").write_text(f"/etc/config/rozaneh {digest}\n", encoding="utf-8")
            command = [apk_tool, "mkpkg", "--files", str(stage), "--output", str(destination)]
            command += ["--info", "replaces:zoordns luci-app-zoordns"]
            apk_arch = "noarch" if package_arch == "all" else package_arch
            for key, value in {"name": component, "version": version, "arch": apk_arch, "license": license_name, "description": "Rozaneh client for OpenWrt", "url": "https://github.com/dreamboxone/rozaneh", "depends": " ".join(deps)}.items():
                command += ["--info", f"{key}:{value}"]
            command += ["--script", f"post-install:{control / 'postinst'}", "--script", f"post-upgrade:{control / 'postinst'}"]
            command += ["--script", f"pre-upgrade:{control / 'prerm'}"]
            command += ["--script", f"pre-install:{control / 'preinst'}"]
            if component == "rozaneh":
                command += ["--script", f"pre-deinstall:{control / 'prerm'}"]
                command += ["--script", f"post-deinstall:{control / 'postrm'}"]
            subprocess.run(command, check=True)
    return {"name": destination.name, "sha256": hashlib.sha256(destination.read_bytes()).hexdigest(), "size": destination.stat().st_size}


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--arch")
    parser.add_argument("--all", action="store_true", help="package every architecture in targets.json")
    parser.add_argument("--format", choices=["ipk", "apk", "both"], default="both")
    parser.add_argument("--apk-tool", default=os.environ.get("APK_TOOL"))
    parser.add_argument("--output", type=Path, default=ROOT / "dist")
    options = parser.parse_args()
    if not options.arch and not options.all:
        parser.error("provide --arch or --all")
    options.output.mkdir(parents=True, exist_ok=True)
    manifest_file = options.output / "manifest.json"
    manifest = json.loads(manifest_file.read_text()) if manifest_file.exists() else {"schema": 1, "components": {}}
    manifest["components"].pop("luci-app-rozaneh", None)
    formats = ["ipk", "apk"] if options.format == "both" else [options.format]
    architectures = json.loads((ROOT / "targets.json").read_text()) if options.all else [options.arch]
    for architecture in architectures:
      for component in VERSIONS:
        entry = manifest["components"].setdefault(component, {"version": VERSIONS[component], "assets": {}})
        entry["version"] = VERSIONS[component]
        if component == "rozaneh":
            entry["bundled_luci"] = True
            entry["engine_version"] = soar_version()
        for fmt in formats:
            asset = build(component, architecture, fmt, options.output, options.apk_tool)
            entry["assets"].setdefault(fmt, {})["all" if component != "rozaneh" else architecture] = asset
    manifest_file.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()

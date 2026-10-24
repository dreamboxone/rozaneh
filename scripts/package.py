#!/usr/bin/env python3
"""Package static zoorDNS binaries and LuCI resources as OpenWrt IPK/APK."""
import argparse
import hashlib
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile

ROOT = Path(__file__).resolve().parent.parent
VERSIONS = {"zoordns": "0.1.0-r2", "luci-app-zoordns": "0.1.0-r3"}


def put(stage, source, destination, mode=0o644):
    target = stage / destination.lstrip("/")
    target.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(source, target)
    target.chmod(mode)


def postinst(component):
    common = '#!/bin/sh\n[ -n "${IPKG_INSTROOT}" ] && exit 0\n'
    if component == "zoordns":
        return common + '''. /lib/functions.sh
gid=$(group_add_next zoordns)
user_exists zoordns || user_add zoordns "" "$gid" "zoorDNS" /var/run/zoordns /bin/false
/etc/init.d/zoordns enable
[ "$(uci -q get zoordns.main.enabled)" = 1 ] && /etc/init.d/zoordns restart
exit 0
'''
    return common + '''/etc/init.d/rpcd restart >/dev/null 2>&1 || true
rm -f /tmp/luci-indexcache.*
exit 0
'''


def stage_component(stage, component, arch):
    if component == "zoordns":
        binary = ROOT / "prebuilt" / arch / "zoordns"
        if not binary.is_file():
            raise SystemExit(f"Missing binary for {arch}; run scripts/build-engine.sh first")
        put(stage, binary, "usr/bin/zoordns", 0o755)
        put(stage, ROOT / "files/zoordns.init", "etc/init.d/zoordns", 0o755)
        put(stage, ROOT / "files/zoordns.config", "etc/config/zoordns", 0o600)
        licenses = [ROOT / x for x in ("LICENSE", "SOAR-LICENSE", "NOTICE")]
        licenses += list((ROOT / "licenses").iterdir())
    else:
        for file in (ROOT / "luci/root").rglob("*"):
            if file.is_file():
                relative = file.relative_to(ROOT / "luci/root")
                mode = 0o755 if str(relative).startswith("usr/libexec/") else 0o644
                put(stage, file, str(relative), mode)
        licenses = [ROOT / "luci" / x for x in ("LICENSE", "NOTICE")]
    for file in licenses:
        put(stage, file, f"usr/share/licenses/{component}/{file.name}")
    version = stage / "usr/share/zoordns" / ("core.version" if component == "zoordns" else "luci.version")
    version.parent.mkdir(parents=True, exist_ok=True)
    version.write_text(VERSIONS[component] + "\n", encoding="utf-8")


def archive(root):
    stream = io.BytesIO()
    with tarfile.open(fileobj=stream, mode="w:gz", format=tarfile.GNU_FORMAT) as tar:
        for file in sorted(root.rglob("*")):
            info = tar.gettarinfo(str(file), "./" + str(file.relative_to(root)))
            info.uid = info.gid = 0
            info.uname = info.gname = "root"
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
    package_arch = "all" if component != "zoordns" else arch
    destination = out / f"{component}-{version}-{package_arch}.{fmt}"
    with tempfile.TemporaryDirectory(prefix="zoordns-package-") as temporary:
        temp = Path(temporary)
        stage = temp / "data"
        stage.mkdir()
        stage_component(stage, component, arch)
        control = temp / "control"
        control.mkdir()
        (control / "postinst").write_text(postinst(component), encoding="utf-8")
        (control / "postinst").chmod(0o755)
        deps = ["libc"] if component == "zoordns" else ["zoordns", "luci-base", "rpcd", "jsonfilter"]
        license_name = "Apache-2.0" if component == "zoordns" else "AGPL-3.0-or-later"
        if component == "zoordns":
            (control / "conffiles").write_text("/etc/config/zoordns\n/etc/zoordns/traffic.json\n", encoding="utf-8")
            (control / "prerm").write_text('#!/bin/sh\n[ -n "${IPKG_INSTROOT}" ] || /etc/init.d/zoordns stop\nexit 0\n', encoding="utf-8")
            (control / "prerm").chmod(0o755)
        if fmt == "ipk":
            metadata = f"Package: {component}\nVersion: {version}\nArchitecture: {package_arch}\nMaintainer: dreamboxone\nSection: net\nPriority: optional\nDepends: {', '.join(deps)}\nLicense: {license_name}\nDescription: zoorDNS DNS tunnel client for OpenWrt\n"
            (control / "control").write_text(metadata, encoding="utf-8")
            ar_write(destination, [("debian-binary", b"2.0\n"), ("control.tar.gz", archive(control)), ("data.tar.gz", archive(stage))])
        else:
            if not apk_tool:
                raise SystemExit("APK_TOOL or --apk-tool must point to the OpenWrt SDK apk tool")
            if component == "zoordns":
                metadata_dir = stage / "lib/apk/packages"
                metadata_dir.mkdir(parents=True, exist_ok=True)
                config = stage / "etc/config/zoordns"
                digest = hashlib.sha256(config.read_bytes()).hexdigest()
                (metadata_dir / "zoordns.conffiles_static").write_text(f"/etc/config/zoordns {digest}\n", encoding="utf-8")
            command = [apk_tool, "mkpkg", "--files", str(stage), "--output", str(destination)]
            apk_arch = "noarch" if package_arch == "all" else package_arch
            for key, value in {"name": component, "version": version, "arch": apk_arch, "license": license_name, "description": "zoorDNS client for OpenWrt", "url": "https://github.com/dreamboxone/zoordns", "depends": " ".join(deps)}.items():
                command += ["--info", f"{key}:{value}"]
            command += ["--script", f"post-install:{control / 'postinst'}", "--script", f"post-upgrade:{control / 'postinst'}"]
            if component == "zoordns":
                command += ["--script", f"pre-deinstall:{control / 'prerm'}"]
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
    formats = ["ipk", "apk"] if options.format == "both" else [options.format]
    architectures = json.loads((ROOT / "targets.json").read_text()) if options.all else [options.arch]
    for architecture in architectures:
      for component in VERSIONS:
        entry = manifest["components"].setdefault(component, {"version": VERSIONS[component], "assets": {}})
        entry["version"] = VERSIONS[component]
        for fmt in formats:
            asset = build(component, architecture, fmt, options.output, options.apk_tool)
            entry["assets"].setdefault(fmt, {})["all" if component != "zoordns" else architecture] = asset
    manifest_file.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()

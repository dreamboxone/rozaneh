#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
"""Check upstream releases and prepare a versioned, testable Rozaneh update."""
import argparse
import datetime
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import urllib.request
import urllib.error

ROOT = Path(__file__).resolve().parents[1]
MODULE = "github.com/getlantern/soar"


def run(*args):
    return subprocess.check_output(args, cwd=ROOT, text=True)


def api(path):
    headers = {"Accept": "application/vnd.github+json", "User-Agent": "rozaneh-release"}
    if os.environ.get("GH_TOKEN"):
        headers["Authorization"] = "Bearer " + os.environ["GH_TOKEN"]
    request = urllib.request.Request("https://api.github.com/" + path, headers=headers)
    with urllib.request.urlopen(request, timeout=30) as response:
        return json.load(response)


def stable_version(value):
    match = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)", value)
    if not match:
        raise ValueError("Expected a stable vMAJOR.MINOR.PATCH release")
    return tuple(map(int, match.groups()))


def current_version():
    text = (ROOT / "go.mod").read_text(encoding="utf-8")
    return re.search(r"github\.com/getlantern/soar\s+(\S+)", text)[1]


def needs_update(current, latest, latest_sha, latest_time):
    stable_version(latest)
    if current == latest:
        return False
    if re.fullmatch(r"v\d+\.\d+\.\d+", current):
        return stable_version(current) < stable_version(latest)
    pseudo = re.search(r"-(\d{14})-([a-f0-9]{12,40})$", current)
    if not pseudo:
        raise ValueError("Unrecognized installed Soar version")
    if latest_sha.startswith(pseudo[2]):
        return False
    current_time = datetime.datetime.strptime(pseudo[1], "%Y%m%d%H%M%S").replace(tzinfo=datetime.timezone.utc)
    release_time = datetime.datetime.fromisoformat(latest_time.replace("Z", "+00:00"))
    return release_time > current_time


def check(state, force=False):
    release = api("repos/getlantern/soar/releases/latest")
    if release.get("draft") or release.get("prerelease"):
        raise ValueError("Upstream release must be stable and published")
    latest = release["tag_name"]
    stable_version(latest)
    ref = api("repos/getlantern/soar/git/ref/tags/" + latest)["object"]
    for _ in range(5):
        if ref["type"] == "commit":
            break
        if ref["type"] != "tag":
            raise ValueError("Release tag does not point to a commit")
        ref = api("repos/getlantern/soar/git/tags/" + ref["sha"])["object"]
    else:
        raise ValueError("Too many nested tags")
    commit = api("repos/getlantern/soar/commits/" + ref["sha"])
    current = current_version()
    changed = needs_update(current, latest, ref["sha"], commit["commit"]["committer"]["date"])
    # Retry a release interrupted after the tested source was pushed. Never leave
    # a missing or partially uploaded package permanently skipped by the timer.
    repository = os.environ.get("GITHUB_REPOSITORY")
    if repository and not changed:
        package_text = (ROOT / "scripts/package.py").read_text(encoding="utf-8")
        package = re.search(r'VERSIONS = \{"rozaneh": "([^\"]+)"\}', package_text)[1]
        try:
            published = api("repos/" + repository + "/releases/tags/v" + package)
            count = len(json.loads((ROOT / "targets.json").read_text())) * 2 + 1
            changed = published.get("draft", True) or len(published.get("assets", [])) != count
        except urllib.error.HTTPError as error:
            if error.code != 404:
                raise
            changed = True
    result = {"changed": changed or force, "previous": current, "engine": latest, "engine_sha": ref["sha"], "upstream_url": release["html_url"]}
    state.write_text(json.dumps(result, indent=2) + "\n", encoding="utf-8")
    if os.environ.get("GITHUB_OUTPUT"):
        with open(os.environ["GITHUB_OUTPUT"], "a", encoding="utf-8") as output:
            output.write("changed=" + str(result["changed"]).lower() + "\n")
    print(json.dumps(result))


def next_package_version(text):
    match = re.search(r'VERSIONS = \{"rozaneh": "(\d+\.\d+\.\d+)-r(\d+)"\}', text)
    if not match:
        raise ValueError("Missing unified package version")
    return match[1] + "-r" + str(int(match[2]) + 1)


def prepare(state):
    data = json.loads(state.read_text(encoding="utf-8"))
    stable_version(data["engine"])
    run("go", "get", MODULE + "@" + data["engine"])
    run("go", "mod", "tidy")
    downloaded = json.loads(run("go", "mod", "download", "-json", MODULE + "@" + data["engine"]))
    if downloaded["Version"] != data["engine"]:
        raise ValueError("Unexpected downloaded Soar version")
    origin = downloaded.get("Origin", {}).get("Hash")
    if origin != data["engine_sha"]:
        raise ValueError("Downloaded module does not match the checked upstream release commit")
    source = Path(downloaded["Dir"])
    shutil.copyfile(source / "LICENSE", ROOT / "SOAR-LICENSE")
    shutil.copyfile(source / "data/THIRD_PARTY.md", ROOT / "licenses/Soar-resolver-data-LICENSE")
    for module, filename in [("golang.org/x/crypto", "golang-x-crypto-LICENSE"), ("golang.org/x/sys", "golang-x-sys-LICENSE"), ("golang.org/x/net","golang-x-net-LICENSE"), ("golang.org/x/text","golang-x-text-LICENSE"), ("github.com/quic-go/quic-go","quic-go-LICENSE"), ("github.com/quic-go/qpack","qpack-LICENSE")]:
        dependency = json.loads(run("go", "mod", "download", "-json", module))
        directory=Path(dependency["Dir"])
        source_license=next((directory/name for name in ("LICENSE","LICENSE.md","LICENSE.txt") if (directory/name).is_file()),None)
        if source_license is None:raise ValueError("Missing dependency license: "+module)
        shutil.copyfile(source_license, ROOT / "licenses" / filename)
    notice = ROOT / "NOTICE"
    notice.write_text(re.sub(r"(?m)^Pinned revision: [a-f0-9]+\.$", "Pinned revision: " + data["engine_sha"] + ".", notice.read_text(encoding="utf-8")), encoding="utf-8")
    package_file = ROOT / "scripts/package.py"
    text = package_file.read_text(encoding="utf-8")
    version = next_package_version(text)
    text = re.sub(r'VERSIONS = \{"rozaneh": "[^\"]+"\}', 'VERSIONS = {"rozaneh": "' + version + '"}', text)
    package_file.write_text(text, encoding="utf-8")
    makefile = ROOT / "Makefile"
    makefile.write_text(re.sub(r"(?m)^PKG_RELEASE:=\d+$", "PKG_RELEASE:=" + version.split("-r")[1], makefile.read_text(encoding="utf-8")), encoding="utf-8")
    data.update(package_version=version, tag="v" + version, base_sha=run("git", "rev-parse", "HEAD").strip())
    state.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def verify(output, state):
    data = json.loads(state.read_text(encoding="utf-8"))
    manifest = json.loads((output / "manifest.json").read_text(encoding="utf-8"))
    if set(manifest["components"]) != {"rozaneh"}:
        raise ValueError("Release must contain only the unified app")
    app = manifest["components"]["rozaneh"]
    if app["engine_version"] != data["engine"] or app["version"] != data["package_version"] or not app["bundled_luci"]:
        raise ValueError("Release metadata does not match the tested source")
    targets = set(json.loads((ROOT / "targets.json").read_text()))
    for kind in ("ipk", "apk"):
        if set(app["assets"][kind]) != targets:
            raise ValueError("Incomplete architecture coverage: " + kind)
        for asset in app["assets"][kind].values():
            if Path(asset["name"]).name != asset["name"]:
                raise ValueError("Invalid asset path")
            content = (output / asset["name"]).read_bytes()
            if len(content) != asset["size"] or hashlib.sha256(content).hexdigest() != asset["sha256"]:
                raise ValueError("Invalid package checksum: " + asset["name"])
    for target in targets:
        info = run("go", "version", "-m", str(ROOT / "prebuilt" / target / "rozaneh"))
        if not re.search(r"\bdep\s+github\.com/getlantern/soar\s+" + re.escape(data["engine"]) + r"\s", info):
            raise ValueError("Binary has wrong embedded engine: " + target)
    print("Verified complete IPK/APK packages and embedded engines for all architectures")


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["check", "prepare", "verify"])
    parser.add_argument("--state", type=Path, required=True)
    parser.add_argument("--force", action="store_true")
    parser.add_argument("--output", type=Path, default=ROOT / "dist/release")
    args = parser.parse_args()
    if args.action == "check":
        check(args.state, args.force)
    elif args.action == "prepare":
        prepare(args.state)
    else:
        verify(args.output, args.state)

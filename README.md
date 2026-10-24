# zoorDNS

A powerful DNS client for your router, powered by [Soar](https://github.com/getlantern/soar).

[راهنمای فارسی](README.fa.md)

## Features

- TCP SOCKS5, procd/UCI integration, and a dedicated unprivileged service user.
- Persian/English LuCI, Vazirmatn throughout, light/dark mode, and the shared Zirgozar visual theme.
- Instagram, Reddit and X URL tests through the tunnel. Green below 1000 ms, amber below 2000 ms, red at 2000 ms or above. HTTP status is available in the tooltip; a restricted HTTP response does not prove access to site content.
- Upload/download/total payload usage in decimal GB, today/7-day/30-day views with UTC day boundaries. Totals survive service restarts; changed totals are checkpointed at most every five minutes and on graceful shutdown. A power failure can lose the latest checkpoint interval.
- Router model, OpenWrt version and uptime.
- Missing-dependency detection and installation inside LuCI.
- Component updates from GitHub releases, selected for the router architecture and checked against SHA-256. Only changed components are downloaded; each component is a complete package, not a binary patch.

This version provides a TCP SOCKS5 proxy on `127.0.0.1:1088`. It does not enable transparent whole-LAN routing or arbitrary application UDP. SOCKS authentication is not implemented.

## Install

Download the core package matching `DISTRIB_ARCH` in `/etc/openwrt_release` and the `all` LuCI package. Use your firmware's package manager:

```sh
# APK-based OpenWrt
apk add --allow-untrusted ./zoordns-<version>-<arch>.apk ./luci-app-zoordns-<version>-all.apk

# OPKG-based OpenWrt
opkg install ./zoordns-<version>-<arch>.ipk ./luci-app-zoordns-<version>-all.ipk
```

Declared dependencies are installed by the package manager. Open **Services → zoorDNS**. Optional HTTPS-test/update tools can be installed with **Install missing dependencies**. Enter your Soar server's delegated DNS zone, public key and resolvers, save and apply, then start. The HTTPS test verifies the tunnel's exit IP. A compatible server and delegated zone are required; no public account is bundled. Keep the private key on the server.

ARM, ARM64, MIPS, x86 and RISC-V targets are listed in `targets.json`. Go-unsupported architectures such as ARC and 32-bit PowerPC are not included. Cross-building does not establish hardware compatibility with every model.

## Build

Go 1.25+ and Python 3 are required; dependencies are pinned in `go.mod`/`go.sum`.

```sh
go test ./...
sh scripts/build-engine.sh arm_cortex-a7_neon-vfpv4
python3 scripts/package.py --arch arm_cortex-a7_neon-vfpv4 --format ipk
# Both formats, using the host apk tool from an OpenWrt 25.12 SDK:
python3 scripts/package.py --arch arm_cortex-a7_neon-vfpv4 --apk-tool /path/to/sdk/staging_dir/host/bin/apk
```

Windows: `scripts/build-all.ps1` builds all listed binaries. Linux: run `scripts/build-engine.sh` for each target. Then `scripts/package.py --all --apk-tool ...` builds packages and `dist/manifest.json`. SDK Makefiles are provided at the root and in `luci/`; the core Makefile consumes `prebuilt/<architecture>/zoordns`.

Publish packages and `manifest.json` as assets of the same GitHub release. The updater uses the latest non-prerelease release. SHA-256 protects against corruption; it is not an independent signature system. Configuration is retained and engine updates briefly interrupt the tunnel. `files/soar-server.service` and `files/server.env.example` provide server configuration examples.

## Validation and licenses

The integration test transfers 128 KiB both ways through an authenticated DNS tunnel, verifies remote hostnames, exact byte counts and counter persistence. APK installation and live HTTPS through recursive resolvers have been tested on Google WiFi (Gale), ARMv7, OpenWrt 25.12.6. IPK archives and cross-built targets require further device testing.

Core wrapper: Apache-2.0. LuCI and reused Zirgozar theme: AGPL-3.0-or-later. Vazirmatn: SIL OFL 1.1. Soar and all dependency licenses/notices are retained in `SOAR-LICENSE`, `NOTICE`, `licenses/` and `luci/`. This is an independent project, not an official Lantern distribution.

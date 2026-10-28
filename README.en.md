**This project requires a Linux VPS outside Iran.**

# Rozaneh

OpenWrt DNS tunnel client powered by the pinned original [CottenDNS](https://github.com/WhiteDNS/CottenDNS) runtime.

[راهنمای فارسی](README.fa.md)

A compatible external Linux server, delegated DNS domain and shared server key are required. Soar public keys are not compatible. CottenMax and Normal use the same server credential when the server supports both modes. Max was tested under real router traffic for three minutes; network conditions vary.

## Install

Run on the router:

```sh
wget -O /tmp/rozaneh-install.sh https://raw.githubusercontent.com/dreamboxone/rozaneh/main/scripts/install.sh && sh /tmp/rozaneh-install.sh
```

The installer detects OpenWrt package architecture and APK/OPKG, verifies the published SHA-256 and installs one package containing the engine and LuCI. Releases must contain manifest.json and the corresponding architecture asset.

Alternatively copy the matching release APK/IPK into /tmp using MobaXterm and install with apk add --allow-untrusted or opkg install. Open Services → Rozaneh, enter the server domain and shared key, choose the compatible engine mode, install missing dependencies and enable the service. Stop competing transparent proxies first.

## Features and limits

Persian/English LuCI, responsive light/dark interface, SOCKS5, LAN TCP routing, service control, connectivity checks, log copy/clear and traffic counters. Counters include engine transport traffic. Native advanced TOML settings are available; some legacy UI fields remain under migration. Normal ARQ/compression/duplication controls do not control Max. LAN routing does not support arbitrary UDP.

## Updates and builds

The displayed engine version uses the Gregorian release date, for example 2026.10.24. Full upstream release tags and immutable source revisions identify builds. The daily workflow checks published CottenDNS releases, builds and tests a complete matching Rozaneh package and publishes architecture assets together with LuCI.

Use scripts/build-release.py to cross-build and scripts/package.py to package. The source pin is cotten-upstream.json. Python, Git and a compatible Go toolchain are required. Third-party licenses are bundled.

## Remove

Use apk del rozaneh or opkg remove rozaneh. Back up /etc/config/rozaneh and /etc/rozaneh if settings are needed later.

## Acknowledgements

Thanks to the WhiteDNS team, [TaJirax](https://github.com/TaJirax) and all [CottenDNS](https://github.com/WhiteDNS/CottenDNS) contributors for developing and publishing the original engine used by Rozaneh. Upstream licenses, copyright and attribution are preserved.

## Support development

If Rozaneh is useful to you, you can support development with a donation.

**USDT — BEP20 (BSC) network only**

```
0x56daaa6b76d88ee0c8dba8042121f4b77de0a813
```

Use only USDT on BEP20 (BSC), not another asset or network.

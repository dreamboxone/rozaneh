#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
umask 077
[ "$(id -u)" = 0 ] || { echo 'Run as root.' >&2; exit 1; }
[ -r /etc/openwrt_release ] || { echo 'OpenWrt is required.' >&2; exit 1; }
. /etc/openwrt_release
arch=${DISTRIB_ARCH:-}
case "$arch" in ''|*[!a-zA-Z0-9_-]*) echo 'Unknown OpenWrt package architecture.' >&2; exit 1 ;; esac
if command -v apk >/dev/null 2>&1; then format=apk
elif command -v opkg >/dev/null 2>&1; then format=ipk
else echo 'No supported package manager found.' >&2; exit 1; fi
[ -r /usr/share/libubox/jshn.sh ] || { echo 'Install libubox shell JSON support first.' >&2; exit 1; }
. /usr/share/libubox/jshn.sh
work=$(mktemp -d /tmp/rozaneh-install.XXXXXX)
trap 'rm -rf "$work"' EXIT
fetch() {
 if command -v curl >/dev/null 2>&1; then
  curl --fail --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 15 --max-time 300 "$1" -o "$2"
 else
  wget -O "$2" "$1"
 fi
}
release=https://github.com/dreamboxone/rozaneh/releases/latest/download
fetch "$release/manifest.json" "$work/manifest.json" || { echo 'Could not download the published release manifest.' >&2; exit 1; }
json_load "$(cat "$work/manifest.json")"
json_get_var schema schema
[ "$schema" = 1 ] || { echo 'Unsupported release manifest.' >&2; exit 1; }
json_select components && json_select rozaneh && json_select assets && json_select "$format" && json_select "$arch" || {
 echo "No published $format package for $arch." >&2; exit 1;
}
json_get_var filename name
json_get_var checksum sha256
case "$filename" in ''|*[!a-zA-Z0-9._+-]*) echo 'Invalid package filename.' >&2; exit 1 ;; esac
case "$checksum" in ''|*[!a-fA-F0-9]*) echo 'Invalid package checksum.' >&2; exit 1 ;; esac
[ "${#checksum}" = 64 ] || exit 1
echo "Downloading $filename for $arch ($format)"
fetch "$release/$filename" "$work/$filename"
printf '%s  %s\n' "$checksum" "$work/$filename" | sha256sum -c -
if [ "$format" = apk ]; then
 apk add --allow-untrusted "$work/$filename"
else
 opkg update
 opkg install "$work/$filename"
fi
echo 'Installed. Open Services > Rozaneh in LuCI.'

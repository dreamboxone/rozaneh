#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
root=${1:-}
legacy="$root/etc/zoordns"
destination="$root/etc/rozaneh"
[ ! -e "$destination/legacy-migrated" ] || exit 0
umask 077
mkdir -p "$destination"
if [ -f "$root/etc/config/zoordns" ]; then
 zone=$(uci -c "$root/etc/config" -q get rozaneh.main.zone || true)
 if [ -z "$zone" ]; then
  cp -p "$root/etc/config/zoordns" "$root/etc/config/rozaneh"
 fi
fi
for name in traffic.json resolvers.json; do
 if [ -f "$legacy/$name" ] && [ ! -e "$destination/$name" ]; then
  cp -p "$legacy/$name" "$destination/$name"
 fi
done
if [ -z "$root" ] && [ "$(uci -q get firewall.zoordns_guard.path || true)" = /usr/libexec/zoordns/lan-firewall.sh ]; then
 uci -q delete firewall.zoordns_guard
 uci commit firewall
 nft delete table inet zoordns_guard 2>/dev/null || true
fi
touch "$destination/legacy-migrated"

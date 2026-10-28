#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
helper=${1:?Usage: router-migration.sh MIGRATION_HELPER}
work=$(mktemp -d /tmp/rozaneh-migration-test.XXXXXX)
cleanup() {
 rm -f "$work/etc/config/zoordns" "$work/etc/config/rozaneh" "$work/etc/zoordns/traffic.json" "$work/etc/rozaneh/traffic.json" "$work/etc/rozaneh/legacy-migrated"
 rmdir "$work/etc/config" "$work/etc/zoordns" "$work/etc/rozaneh" "$work/etc" "$work"
}
trap cleanup EXIT
mkdir -p "$work/etc/config" "$work/etc/zoordns"
printf "config client 'main'\n option zone 'saved.example'\n option enabled '1'\n" > "$work/etc/config/zoordns"
printf "config client 'main'\n option zone ''\n" > "$work/etc/config/rozaneh"
printf '{"today":{"up":123}}\n' > "$work/etc/zoordns/traffic.json"
sh "$helper" "$work"
cmp "$work/etc/config/zoordns" "$work/etc/config/rozaneh"
cmp "$work/etc/zoordns/traffic.json" "$work/etc/rozaneh/traffic.json"
uci -c "$work/etc/config" set rozaneh.main.zone=changed.example
uci -c "$work/etc/config" commit rozaneh
printf 'retained\n' > "$work/etc/rozaneh/traffic.json"
sh "$helper" "$work"
[ "$(uci -c "$work/etc/config" get rozaneh.main.zone)" = changed.example ]
[ "$(cat "$work/etc/rozaneh/traffic.json")" = retained ]
echo 'Configuration and counters migrated; repeated migration preserves new data'

#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
stage=${1:?Usage: router-startup.sh STAGED_INIT}
work=$(mktemp -d /tmp/rozaneh-start-test.XXXXXX)
cleanup() {
	rm -f "$work/config/rozaneh" "$work/dependencies.sh" "$work/init.sh" "$work/instances"
	rmdir "$work/config" "$work"
}
trap cleanup EXIT
if nft list table inet rozaneh_guard >/dev/null 2>&1; then echo 'A real firewall guard is active; fixture refused' >&2; exit 1; fi
[ "$(uci -q get firewall.rozaneh_guard.path)" = /usr/libexec/rozaneh/lan-firewall.sh ]
mkdir "$work/config"
cat > "$work/config/rozaneh" <<'EOF'
config client 'main'
 option enabled '0'
 option zone 'fixture.example'
 option pubkey 'fixture'
 option lan_enabled '0'
 option lan_killswitch '0'
EOF
printf 'rozaneh_missing_dependencies() { missing=""; }\nrozaneh_proxy_conflict() { return "${TEST_CONFLICT:-1}"; }\n' > "$work/dependencies.sh"
sed "s|/usr/libexec/rozaneh/dependencies.sh|$work/dependencies.sh|" "$stage/rozaneh.init" > "$work/init.sh"
export UCI_CONFIG_DIR="$work/config"
. /lib/functions.sh
. "$work/init.sh"
procd_open_instance() { printf '%s\n' "$1" >> "$work/instances"; }
procd_close_instance() { :; }
procd_set_param() { :; }
procd_append_param() { :; }
logger() { :; }
start_service
[ ! -e "$work/instances" ]
uci -c "$work/config" set rozaneh.main.enabled=1
uci -c "$work/config" commit rozaneh
start_service
[ "$(cat "$work/instances")" = "$(printf 'client\nlan\nhealth')" ]
echo 'Master switch starts client, LAN and automatic health without an internet prerequisite'
rm -f "$work/instances"
export TEST_CONFLICT=0
if start_service; then echo 'Conflicting proxy was accepted' >&2; exit 1; fi
[ ! -e "$work/instances" ]
echo 'Conflicting transparent proxy is rejected before any instance starts'

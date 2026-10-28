#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
stage=${1:?Usage: router-firewall.sh STAGED_HELPERS}
work=$(mktemp -d /tmp/rozaneh-firewall-test.XXXXXX)
created=0
namespace=0
cleanup() {
	[ "$created" = 0 ] || nft delete table inet rozaneh_guard 2>/dev/null || true
	[ "$namespace" = 0 ] || ip netns del zdreview 2>/dev/null || true
	ip link del zdreview0 2>/dev/null || true
	rm -f "$work/config/rozaneh" "$work/guard.sh" "$work/lan.json"
	rmdir "$work/config" "$work"
}
trap cleanup EXIT
if nft list table inet rozaneh_guard >/dev/null 2>&1 || ip link show zdreview0 >/dev/null 2>&1 || ip netns list | grep -q '^zdreview '; then
	echo 'Test fixture is already in use' >&2; exit 1
fi
mkdir "$work/config"
cat > "$work/config/rozaneh" <<'EOF'
config client 'main'
 option lan_enabled '1'
 option lan_killswitch '1'
 list lan_devices 'zdreview0'
 option listen '127.0.0.1:1089'
EOF
export UCI_CONFIG_DIR="$work/config"
sed "s|/usr/libexec/rozaneh/dependencies.sh|$stage/dependencies.sh|" "$stage/lan-firewall.sh" > "$work/guard.sh"
if ip link add zdreview0 type veth peer name zdreview1 2>/dev/null; then
	ip netns add zdreview
	namespace=1
	ip link set zdreview1 netns zdreview
	ip addr add 10.253.254.1/30 dev zdreview0
	ip netns exec zdreview ip addr add 10.253.254.2/30 dev zdreview1
	ip netns exec zdreview ip link set lo up
	ip netns exec zdreview ip link set zdreview1 up
	ip netns exec zdreview ip route add default via 10.253.254.1
else
	ip link add zdreview0 type bridge
fi
ip link set zdreview0 up
sh "$work/guard.sh"
created=1
nft list table inet rozaneh_guard | grep -q 'iifname "zdreview0"'
if [ "$namespace" = 1 ]; then
	ip netns exec zdreview ping -c 1 -W 1 1.1.1.1 >/dev/null 2>&1 || true
	nft list chain inet rozaneh_guard forward | grep 'meta nfproto ipv4' | grep -Eq 'counter packets [1-9][0-9]* '
	echo 'IPv4 direct traffic blocked from isolated network namespace'
else
	if [ -x "$stage/tun-probe" ]; then
		ip link del zdreview0
		"$stage/tun-probe" zdreview0
		nft list table inet rozaneh_guard
		nft list chain inet rozaneh_guard forward | grep 'ip daddr' | grep -Eq 'counter packets [2-9][0-9]* '
		nft list chain inet rozaneh_guard forward | grep 'ip6 daddr' | grep -Eq 'counter packets [1-9][0-9]* '
		echo 'IPv4, IPv6 and DNS direct traffic blocked on isolated TUN'
	else
		echo 'Packet injection unavailable; rules and lifecycle are tested'
	fi
fi
# Simulate a firewall table rebuild while the tunnel is stopped.
nft delete table inet rozaneh_guard
sh "$work/guard.sh"
nft list table inet rozaneh_guard | grep -q 'ip6 daddr'
# Reapplying the guard must be atomic and idempotent.
sh "$work/guard.sh"
# A selected default-route interface must be refused before replacing the guard.
wan=$(ip -4 route show default | awk '{for(i=1;i<NF;i++) if($i=="dev") {print $(i+1);exit}}')
if [ -n "$wan" ]; then
	uci -c "$work/config" -q delete rozaneh.main.lan_devices
	uci -c "$work/config" add_list rozaneh.main.lan_devices="$wan"
	uci -c "$work/config" commit rozaneh
	if sh "$work/guard.sh" >/dev/null 2>&1; then echo 'WAN device was accepted' >&2; exit 1; fi
	nft list table inet rozaneh_guard | grep -q 'iifname "zdreview0"'
fi
uci -c "$work/config" set rozaneh.main.lan_killswitch=0
uci -c "$work/config" commit rozaneh
sh "$work/guard.sh"
if nft list table inet rozaneh_guard >/dev/null 2>&1; then echo 'Disabled guard remained installed' >&2; exit 1; fi
created=0
echo 'Firewall guard startup, rebuild, WAN rejection and removal passed'

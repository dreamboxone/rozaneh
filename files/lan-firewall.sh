#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
. /lib/functions.sh
. /usr/libexec/rozaneh/dependencies.sh
config_load rozaneh
config_get_bool guard main lan_killswitch 0
if [ "$guard" != 1 ]; then
	if nft list table inet rozaneh_guard >/dev/null 2>&1; then
		nft delete table inet rozaneh_guard
	fi
	exit 0
fi
config_get devices main lan_devices 'br-lan'
rozaneh_check_lan_devices "$devices"
work=$(mktemp /tmp/rozaneh-guard.XXXXXX)
trap 'rm -f "$work"' EXIT
{
	# Replace the owned table in one transaction, without an unprotected interval.
	if nft list table inet rozaneh_guard >/dev/null 2>&1; then echo 'delete table inet rozaneh_guard'; fi
	echo 'table inet rozaneh_guard {'
	echo 'chain dns_lan { type nat hook prerouting priority -115; policy accept;'
	for device in $devices; do
		printf 'iifname "%s" meta nfproto ipv4 meta l4proto { tcp, udp } th dport 53 dnat ip to 198.18.0.2\n' "$device"
		printf 'iifname "%s" meta nfproto ipv6 meta l4proto { tcp, udp } th dport 53 dnat ip6 to 2001:db8:5a00::2\n' "$device"
	done
	echo '}'
	echo 'chain forward { type filter hook forward priority -10; policy accept;'
	for device in $devices; do
		printf 'iifname "%s" oifname "zdtun0" return\n' "$device"
		printf 'iifname "%s" ip6 daddr fd00:5a00::/48 counter drop\n' "$device"
		printf 'iifname "%s" meta nfproto ipv4 ip daddr != { 0.0.0.0/8, 10.0.0.0/8, 100.64.0.0/10, 127.0.0.0/8, 169.254.0.0/16, 172.16.0.0/12, 192.168.0.0/16, 224.0.0.0/4, 240.0.0.0/4 } counter drop\n' "$device"
		printf 'iifname "%s" meta nfproto ipv6 ip6 daddr != { ::/128, ::1/128, fc00::/7, fe80::/10, ff00::/8 } counter drop\n' "$device"
	done
	printf '}\n}\n'
} > "$work"
nft -c -f "$work"
nft -f "$work"

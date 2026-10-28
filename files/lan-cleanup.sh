#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
state=/var/run/rozaneh-lan
[ -f "$state/owned" ] || exit 0
[ -z "${1:-}" ] || [ "$(cat "$state/owned")" = "$1" ] || exit 0
if [ -f "$state/engine.pid" ]; then
	pid=$(cat "$state/engine.pid")
	case "$pid" in ''|*[!0-9]*) pid='' ;; esac
	if [ -n "$pid" ] && [ -r "/proc/$pid/cmdline" ] && tr '\000' ' ' 2>/dev/null < "/proc/$pid/cmdline" | grep -Fq "sing-box run -c $state/sing-box.json"; then
		kill "$pid" 2>/dev/null || true
  for attempt in 1 2 3; do kill -0 "$pid" 2>/dev/null || break; sleep 1; done
	fi
fi
if [ -f "$state/devices" ]; then
	while IFS= read -r device; do
		case "$device" in ''|*[!a-zA-Z0-9_.:-]*) continue ;; esac
		for family in -4 -6; do
			ip "$family" rule del pref 12801 iif "$device" lookup 28028 2>/dev/null || true
		done
	done < "$state/devices"
fi
for family in -4 -6; do ip "$family" route flush table 28028 2>/dev/null || true; done
nft delete table inet rozaneh_dns 2>/dev/null || true
for chain in input forward; do
for handle in $(nft -a list chain inet fw4 "$chain" 2>/dev/null | awk '/comment "rozaneh-(lan|tcp[46])"/ {print $NF}'); do
	case "$handle" in ''|*[!0-9]*) continue ;; esac
	nft delete rule inet fw4 "$chain" handle "$handle" 2>/dev/null || true
done
done
if [ "$(cat "$state/ipv4-forward.before" 2>/dev/null)" = 0 ]; then echo 0 > /proc/sys/net/ipv4/ip_forward; fi
if [ "$(cat "$state/ipv6-forward.before" 2>/dev/null)" = 0 ]; then echo 0 > /proc/sys/net/ipv6/conf/all/forwarding; fi
rm -f "$state/owned" "$state/ready" "$state/engine.pid" "$state/devices" "$state/ipv4-forward.before" "$state/ipv6-forward.before"
rmdir "$state/lock" 2>/dev/null || true

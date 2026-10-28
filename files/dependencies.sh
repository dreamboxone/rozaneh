#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

rozaneh_package_installed() {
	if command -v apk >/dev/null 2>&1; then apk info -e "$1" >/dev/null 2>&1
	else opkg status "$1" 2>/dev/null | grep -q 'Status:.* installed'; fi
}

rozaneh_singbox_supported() {
 local binary="$1" signature cached temporary
 [ -x "$binary" ] || return 1
 signature=$(ls -linL --full-time "$binary" 2>/dev/null)
 if [ -n "$signature" ]; then
  signature="$binary:$signature"
  cached=$(cat /var/run/rozaneh-ui/singbox-version 2>/dev/null || true)
  [ "$cached" != "$signature" ] || return 0
 fi
 "$binary" version 2>/dev/null | awk 'BEGIN { ok=0 } /^sing-box version/ { split($3,v,"."); ok=(v[1]==1 && v[2]>=12) } END { exit !ok }' || return 1
 if [ -n "$signature" ]; then
  (umask 077; mkdir -p /var/run/rozaneh-ui) 2>/dev/null
  temporary=$(mktemp /var/run/rozaneh-ui/singbox-version.XXXXXX 2>/dev/null)
  if [ -n "$temporary" ]; then
   printf '%s\n' "$signature" > "$temporary" && mv -f "$temporary" /var/run/rozaneh-ui/singbox-version
   rm -f "$temporary"
  fi
 fi
 return 0
}

rozaneh_find_singbox() {
 local binary
 for binary in /usr/libexec/zirgozar/sing-box /usr/bin/sing-box; do
  if rozaneh_singbox_supported "$binary"; then printf '%s\n' "$binary"; return 0; fi
 done
 return 1
}

rozaneh_missing_dependencies() {
	local package
	missing=''
	command -v curl >/dev/null 2>&1 || missing='curl'
	[ -s /etc/ssl/certs/ca-certificates.crt ] || missing="${missing:+$missing }ca-bundle"
	command -v fw4 >/dev/null 2>&1 || missing="${missing:+$missing }firewall4"
	rozaneh_find_singbox >/dev/null || missing="${missing:+$missing }sing-box"
	command -v nft >/dev/null 2>&1 || missing="${missing:+$missing }nftables"
	rozaneh_package_installed ip-full || missing="${missing:+$missing }ip-full"
	for package in kmod-tun kmod-nft-nat; do
		rozaneh_package_installed "$package" || missing="${missing:+$missing }$package"
	done
}

rozaneh_proxy_conflict() {
	conflict_name=''
	# The saved switch and procd state also cover startup and watchdog recovery.
	if [ "$(uci -q get zirgozar.config.enabled)" = 1 ] ||
	   ubus call service list '{"name":"zirgozar"}' 2>/dev/null |
	   jsonfilter -e '@.zirgozar.instances.*.running' 2>/dev/null | grep -q true; then
		conflict_name=zirgozar
		return 0
	fi
	nft list tables 2>/dev/null | grep -Eq 'table inet (zirgozar|passwall|passwall2|nikki|openclash|homeproxy|sing-box)([[:space:]]|$)'
}

rozaneh_check_lan_devices() {
	local devices="$1" device wan_devices
	[ -n "$devices" ] || { echo 'No LAN devices selected' >&2; return 1; }
	wan_devices=$(ip -4 route show default; ip -6 route show default)
	wan_devices=$(printf '%s\n' "$wan_devices" | awk '{for(i=1;i<NF;i++) if($i=="dev") print $(i+1)}' | sort -u)
	for device in $devices; do
		case "$device" in ''|lo|zdtun0|*[!a-zA-Z0-9_.:-]*) echo 'Invalid LAN device' >&2; return 1 ;; esac
		[ "${#device}" -le 15 ] || return 1
		if printf '%s\n' "$wan_devices" | grep -Fxq "$device"; then
			echo "Refusing to route a WAN device: $device" >&2
			return 1
		fi
	done
}

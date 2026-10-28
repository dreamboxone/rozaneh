#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
. /lib/functions.sh
. /usr/share/libubox/jshn.sh
config_load rozaneh
config_get devices main lan_devices 'br-lan'
config_get listen main listen '127.0.0.1:1088'
. /usr/libexec/rozaneh/dependencies.sh
rozaneh_check_lan_devices "$devices"
case "$listen" in 127.0.0.1:*|0.0.0.0:*) ;; *) echo 'LAN mode requires an IPv4 loopback or wildcard SOCKS listener' >&2; exit 1 ;; esac
port=${listen##*:}
case "$port" in ''|*[!0-9]*) echo 'Invalid SOCKS port' >&2; exit 1 ;; esac
[ "$port" -ge 1 ] && [ "$port" -le 65535 ] || exit 1
for device in $devices; do
	case "$device" in ''|*[!a-zA-Z0-9_.:-]*) echo 'Invalid LAN device' >&2; exit 1 ;; esac
	[ "${#device}" -le 15 ] || exit 1
done
[ -n "$devices" ] || { echo 'No LAN devices selected' >&2; exit 1; }
json_init
json_add_object log; json_add_string level info; json_close_object
json_add_object dns
	json_add_array servers
		json_add_object ''
			json_add_string type https; json_add_string tag remote-dns
			json_add_string server 1.1.1.1; json_add_int server_port 443
			json_add_string path /dns-query; json_add_string detour tunnel
			json_add_object tls; json_add_boolean enabled 1; json_add_string server_name cloudflare-dns.com; json_close_object
		json_close_object
		json_add_object ''; json_add_string type local; json_add_string tag local-dns; json_close_object
		json_add_object ''
			json_add_string type fakeip; json_add_string tag tunnel-names
			json_add_string inet4_range 198.19.0.0/16
			json_add_string inet6_range 'fd00:5a00::/48'
		json_close_object
	json_close_array
	json_add_array rules
		json_add_object ''; json_add_array domain_suffix; json_add_string '' lan; json_add_string '' local; json_close_array; json_add_string server local-dns; json_close_object
		json_add_object ''
			json_add_array query_type; json_add_string '' A; json_add_string '' AAAA; json_close_array
			json_add_string server tunnel-names
		json_close_object
	json_close_array
	json_add_string final remote-dns
json_close_object
json_add_array inbounds
	json_add_object ''
		json_add_string type tun; json_add_string tag lan-tun; json_add_string interface_name zdtun0
		json_add_array address; json_add_string '' 198.18.0.1/30; json_add_string '' '2001:db8:5a00::1/126'; json_close_array
  json_add_int mtu 1400
  binary=$(rozaneh_find_singbox)
  minor=$("$binary" version | awk '/^sing-box version/ {split($3,v,".");print v[2];exit}')
  [ "$minor" -ge 15 ] || json_add_string stack system
  json_add_boolean auto_route 0
  json_add_boolean auto_redirect 0

	json_close_object
json_close_array
json_add_array outbounds
	json_add_object ''
		json_add_string type socks; json_add_string tag tunnel; json_add_string server 127.0.0.1
		json_add_int server_port "$port"; json_add_string version 5; json_add_string network tcp; json_add_string bind_interface lo
	json_close_object
json_close_array
json_add_object route
	json_add_boolean auto_detect_interface 1
	json_add_array rules
		json_add_object ''; json_add_int port 53; json_add_string action hijack-dns; json_close_object
		json_add_object ''; json_add_string network udp; json_add_string action reject; json_close_object
	json_close_array
	json_add_string final tunnel
json_close_object
json_add_object experimental
	json_add_object cache_file
		json_add_boolean enabled 1
		json_add_string path /etc/rozaneh/lan-cache.db
		json_add_boolean store_fakeip 1
	json_close_object
json_close_object
json_dump

#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -e
umask 077
. /lib/functions.sh
. /usr/share/libubox/jshn.sh
config_load rozaneh
config_get domain main zone ''
config_get key main encryption_key ''
[ -n "$domain" ] && [ -n "$key" ] || { echo 'Missing CottenDNS domain or shared key' >&2; exit 1; }
mkdir -p /var/run/rozaneh
json_init
config_get mode main tunnel_mode legacy
json_add_string TUNNEL_MODE "$mode"
json_add_array DOMAINS
json_add_string '' "$domain"
json_close_array
json_add_string ENCRYPTION_KEY "$key"
config_get preset main config_preset speed
json_add_string CONFIG_PRESET "$preset"
config_get advanced main advanced_config ''
json_add_string _advanced_toml "$advanced"
json_add_string PROTOCOL_TYPE SOCKS5
config_get startup main startup_mode resolvers
json_add_string STARTUP_MODE "$startup"
config_get_bool fast_connect main fast_connect 1
json_add_boolean FAST_CONNECT "$fast_connect"
json_add_string LOG_DIR /var/run/rozaneh/logs
json_add_string TERMINAL_UI plain
config_get listen main listen '127.0.0.1:1088'
host=${listen%:*};port=${listen##*:}
case "$port" in ''|*[!0-9]*) echo 'Invalid SOCKS port' >&2;exit 1;;esac
json_add_string LISTEN_IP "$host"
json_add_int LISTEN_PORT "$port"
json_add_int DATA_ENCRYPTION_METHOD 3
json_add_boolean SOCKS5_AUTH 0
config_get family main resolver_ip_mode auto
json_add_string RESOLVER_IP_MODE "$family"
config_get transport main resolver_transport auto
json_add_string RESOLVER_TRANSPORT "$transport"
config_get_bool fallback main http3_fallback 1
json_add_boolean RESOLVER_HTTP3_FALLBACK "$fallback"
config_get_bool debug main debug 0
if [ "$debug" = 1 ]; then json_add_string LOG_LEVEL DEBUG; else json_add_string LOG_LEVEL INFO; fi
json_dump > /var/run/rozaneh/settings.json
config_get resolvers main resolvers '1.1.1.1:53,8.8.8.8:53'
printf '%s\n' "$resolvers" | tr ',' '\n' > /var/run/rozaneh/resolvers.txt
/usr/bin/rozaneh -prepare-config /var/run/rozaneh/settings.json -output-config /var/run/rozaneh/client.toml
rm -f /var/run/rozaneh/settings.json
chown rozaneh:rozaneh /var/run/rozaneh/client.toml /var/run/rozaneh/resolvers.txt

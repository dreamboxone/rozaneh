#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
. /lib/functions.sh
. /usr/share/libubox/jshn.sh
umask 077
output=${1:-/var/run/rozaneh/health.json}
config_load rozaneh
config_get listen main listen '127.0.0.1:1088'
case "$listen" in 0.0.0.0:*) listen="127.0.0.1:${listen##*:}" ;; esac
work=$(mktemp -d /tmp/rozaneh-health.XXXXXX) || exit 1
google_pid=''; trace_pid=''
cleanup() {
	[ -z "$google_pid" ] || kill "$google_pid" 2>/dev/null || true
	[ -z "$trace_pid" ] || kill "$trace_pid" 2>/dev/null || true
	rm -f "$work/google-code" "$work/trace-code" "$work/trace-body"
	rmdir "$work"
}
trap cleanup EXIT
trap 'exit 0' INT TERM
curl --silent --show-error --noproxy '' --proto '=https' --socks5-hostname "$listen" --connect-timeout 10 --max-time 15 --max-filesize 2048 --output /dev/null --write-out '%{http_code}' https://www.gstatic.com/generate_204 > "$work/google-code" 2>/dev/null &
google_pid=$!
curl --silent --show-error --noproxy '' --proto '=https' --socks5-hostname "$listen" --connect-timeout 10 --max-time 15 --max-filesize 2048 --output "$work/trace-body" --write-out '%{http_code}' https://www.cloudflare.com/cdn-cgi/trace > "$work/trace-code" 2>/dev/null &
trace_pid=$!
wait "$google_pid"; google_exit=$?
wait "$trace_pid"; trace_exit=$?
google_pid=''; trace_pid=''
google_code=$(cat "$work/google-code")
trace_code=$(cat "$work/trace-code")
ip=$(sed -n 's/^ip=//p' "$work/trace-body" 2>/dev/null | head -1)
connected=0
if [ "$google_exit" = 0 ] && [ "$google_code" = 204 ] && [ "$trace_exit" = 0 ] && [ "$trace_code" = 200 ] && [ -n "$ip" ]; then connected=1; fi
json_init
json_add_int updated "$(date +%s)"
json_add_boolean connected "$connected"
json_add_boolean checking 0
json_add_string exit_ip "$ip"
json_add_string google_http "$google_code"
json_add_string trace_http "$trace_code"
json_add_int google_exit "$google_exit"
json_add_int trace_exit "$trace_exit"
json_dump > "$output.tmp" && mv "$output.tmp" "$output"

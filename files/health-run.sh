#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
umask 077
output=/var/run/rozaneh/health.json
child=''
watchdog=''
cleanup() {
	[ -z "$watchdog" ] || kill "$watchdog" 2>/dev/null || true
	[ -z "$child" ] || kill "$child" 2>/dev/null || true
	[ -z "$child" ] || wait "$child" 2>/dev/null || true
	printf '{"updated":%s,"connected":false,"checking":false}\n' "$(date +%s)" > "$output.tmp"
	mv "$output.tmp" "$output"
}
trap cleanup EXIT
trap 'exit 0' INT TERM
printf '{"updated":%s,"connected":false,"checking":true}\n' "$(date +%s)" > "$output.tmp"
mv "$output.tmp" "$output"
while :; do
	started=$(date +%s)
	/usr/libexec/rozaneh/health-check.sh "$output" &
	child=$!
	(sleep 20; kill "$child" 2>/dev/null) &
	watchdog=$!
	wait "$child" || true
	kill "$watchdog" 2>/dev/null || true
	wait "$watchdog" 2>/dev/null || true
	watchdog=''
	child=''
	updated=$(jsonfilter -i "$output" -e '@.updated' 2>/dev/null)
	case "$updated" in ''|*[!0-9]*) updated=0;; esac
	if [ "$updated" -lt "$started" ]; then
		printf '{"updated":%s,"connected":false,"checking":false,"reason":"health_check_incomplete"}\n' "$(date +%s)" > "$output.tmp"
		mv "$output.tmp" "$output"
	fi
	sleep 20 &
	child=$!
	wait "$child" || true
done

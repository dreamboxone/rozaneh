#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
stage=${1:?Usage: router-health.sh STAGED_HELPERS}
work=$(mktemp -d /tmp/rozaneh-health-test.XXXXXX)
cleanup() {
	rm -f "$work/config/rozaneh" "$work/bin/curl" "$work/health.json" "$work/health.json.tmp"
	rmdir "$work/config" "$work/bin" "$work"
}
trap cleanup EXIT
mkdir "$work/config" "$work/bin"
printf "config client 'main'\n option listen '127.0.0.1:1088'\n" > "$work/config/rozaneh"
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
output=/dev/null
url=''
while [ "$#" -gt 0 ]; do
 case "$1" in --output) shift; output=$1 ;; https://*) url=$1 ;; esac
 shift
done
case "$url" in
 *gstatic*)
  case "$HEALTH_SCENARIO" in google-fail) printf 000; exit 28 ;; portal) printf 302; exit 0 ;; esac
  printf 204
  ;;
 *)
  case "$HEALTH_SCENARIO" in trace-fail) printf 500; exit 0 ;; no-ip) printf 'unexpected\n' > "$output" ;; *) printf 'ip=203.0.113.7\n' > "$output" ;; esac
  printf 200
  ;;
esac
EOF
chmod 755 "$work/bin/curl"
export UCI_CONFIG_DIR="$work/config" PATH="$work/bin:$PATH"
for scenario in success google-fail portal trace-fail no-ip; do
	export HEALTH_SCENARIO="$scenario"
	sh "$stage/health-check.sh" "$work/health.json"
	connected=$(jsonfilter -i "$work/health.json" -e '@.connected')
	if [ "$scenario" = success ]; then
		[ "$connected" = true ]
		[ "$(jsonfilter -i "$work/health.json" -e '@.exit_ip')" = 203.0.113.7 ]
	else
		[ "$connected" = false ]
	fi
	[ "$(jsonfilter -i "$work/health.json" -e '@.updated')" -gt 0 ]
	echo "Automatic connectivity scenario passed: $scenario"
done

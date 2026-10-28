#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
stage=${1:?Usage: router-update.sh STAGED_UPDATER}
work=$(mktemp -d /tmp/rozaneh-update-test.XXXXXX)
cleanup() {
	rm -f "$work/bin/apk" "$work/bin/curl" "$work/bin/uci" "$work/bin/ubus" "$work/run.sh" "$work/calls" "$work/curl-calls" "$work/state/manifest.json" "$work/state/fixture.apk"
	rmdir "$work/state/lock" 2>/dev/null || true
	rmdir "$work/bin" "$work/state" "$work"
}
trap cleanup EXIT
mkdir "$work/bin" "$work/state"
sed "s|^state=/tmp/rozaneh-update$|state=$work/state|" "$stage/update" > "$work/run.sh"
cat > "$work/bin/apk" <<'EOF'
#!/bin/sh
if [ "$1" = version ]; then printf '%s\n' "$UPDATE_COMPARE"; exit 0; fi
printf '%s\n' "$*" >> "$UPDATE_CALLS"
EOF
cat > "$work/bin/curl" <<'EOF'
#!/bin/sh
printf '%s\n' "$*" >> "$UPDATE_CURL_CALLS"
while [ "$#" -gt 0 ]; do
 if [ "$1" = -o ]; then shift; printf 'package fixture\n' > "$1"; exit 0; fi
 shift
done
exit 1
EOF
cat > "$work/bin/uci" <<'EOF'
#!/bin/sh
case "$*" in
 *rozaneh.main.enabled*) printf '%s\n' "${UPDATE_USE_PROXY:-0}" ;;
 *rozaneh.main.listen*) echo 127.0.0.1:1088 ;;
 *) exit 1 ;;
esac
EOF
cat > "$work/bin/ubus" <<'EOF'
#!/bin/sh
printf '%s\n' '{"rozaneh":{"instances":{"client":{"running":true}}}}'
EOF
chmod 755 "$work/bin/apk" "$work/bin/curl" "$work/bin/uci" "$work/bin/ubus"
export PATH="$work/bin:$PATH" UPDATE_CALLS="$work/calls" UPDATE_CURL_CALLS="$work/curl-calls"
. /etc/openwrt_release
current=$(cat /usr/share/rozaneh/core.version)
checksum=$(printf 'package fixture\n' | sha256sum | awk '{print $1}')
manifest() {
 printf '{"schema":1,"components":{"rozaneh":{"version":"%s","bundled_luci":true,"assets":{"apk":{"%s":{"name":"fixture.apk","sha256":"%s"}}}}}}\n' "$1" "$DISTRIB_ARCH" "$checksum" > "$work/state/manifest.json"
}
export UPDATE_COMPARE='='
manifest "$current"
mkdir "$work/state/lock"
sh "$work/run.sh" install all 0
[ ! -e "$work/calls" ]
mkdir "$work/state/lock"
sh "$work/run.sh" install all 1
grep -q '^add --force-reinstall --allow-untrusted ' "$work/calls"
rm -f "$work/calls"
export UPDATE_COMPARE='>'
manifest '0.0.1-r1'
mkdir "$work/state/lock"
sh "$work/run.sh" install all 1
[ ! -e "$work/calls" ]
export UPDATE_COMPARE='<'
manifest '99.0.0-r1'
mkdir "$work/state/lock"
sh "$work/run.sh" install all 0
grep -q '^add --allow-untrusted ' "$work/calls"
rm -f "$work/calls" "$work/curl-calls"
export UPDATE_USE_PROXY=1
export UPDATE_COMPARE='='
manifest "$current"
mkdir "$work/state/lock"
# The fixture must not call the real service restart after the simulated install.
sed 's|\[ "$(uci -q get rozaneh.main.enabled)" = 1 \] \&\& /etc/init.d/rozaneh restart|true|' "$work/run.sh" > "$work/run.sh.tmp"
mv "$work/run.sh.tmp" "$work/run.sh"
sh "$work/run.sh" install all 1
grep -q -- '--socks5-hostname 127.0.0.1:1088' "$work/curl-calls"
echo 'Updater unchanged, reinstall, downgrade rejection and upgrade scenarios passed'

#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export CONTROL_STATE="$work/state" CONTROL_CALLS="$work/calls" CONTROL_FAIL=0
mkdir "$work/bin"
cat > "$work/json.sh" <<'EOF'
json_load() { :; }
json_get_var() { eval "$1=\$CONTROL_ACTION"; }
json_init() { :; }
json_add_boolean() { printf '%s=%s\n' "$1" "$2"; }
json_add_string() { :; }
json_dump() { :; }
EOF
cat > "$work/bin/uci" <<'EOF'
#!/bin/sh
[ "$1" != -q ] || shift
case "$1" in
 get) cat "$CONTROL_STATE" ;;
 set) printf '%s\n' "${2#*=}" > "$CONTROL_STATE" ;;
 commit) echo commit >> "$CONTROL_CALLS" ;;
 *) exit 1 ;;
esac
EOF
cat > "$work/firewall" <<'EOF'
#!/bin/sh
echo "firewall=$(cat "$CONTROL_STATE")" >> "$CONTROL_CALLS"
[ "$CONTROL_FAIL" = 0 ]
EOF
chmod +x "$work/bin/uci" "$work/firewall"
export PATH="$work/bin:$PATH"
sed -e "s|/usr/share/libubox/jshn.sh|$work/json.sh|" -e "s|/usr/libexec/rozaneh/lan-firewall.sh|$work/firewall|g" "$root/luci/root/usr/libexec/rpcd/rozaneh" > "$work/rpc"
echo 0 > "$CONTROL_STATE"
export CONTROL_ACTION=killswitch_on
printf '{}' | sh "$work/rpc" call control > "$work/result"
grep -q 'ok=1' "$work/result"
[ "$(cat "$CONTROL_STATE")" = 1 ]
grep -q 'firewall=1' "$CONTROL_CALLS"
grep -q commit "$CONTROL_CALLS"
export CONTROL_ACTION=killswitch_off
printf '{}' | sh "$work/rpc" call control > "$work/result"
grep -q 'ok=1' "$work/result"
[ "$(cat "$CONTROL_STATE")" = 0 ]
grep -q 'firewall=0' "$CONTROL_CALLS"
export CONTROL_ACTION=killswitch_on CONTROL_FAIL=1
: > "$CONTROL_CALLS"
printf '{}' | sh "$work/rpc" call control > "$work/result"
grep -q 'ok=0' "$work/result"
[ "$(cat "$CONTROL_STATE")" = 0 ]
! grep -q commit "$CONTROL_CALLS"
grep -q 'firewall=0' "$CONTROL_CALLS"
echo 'Kill switch enable, disable and failure rollback checks passed'

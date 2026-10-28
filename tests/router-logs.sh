#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
mkdir "$work/bin"
export LOG_FIXTURE="$work/source"
cat > "$work/bin/logread" <<'EOF'
#!/bin/sh
cat "$LOG_FIXTURE"
EOF
chmod +x "$work/bin/logread"
export PATH="$work/bin:$PATH"
awk 'BEGIN{for(i=1;i<=35;i++)print "rozaneh: message "i}' > "$LOG_FIXTURE"
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/output"
[ "$(wc -l < "$work/output")" = 30 ]
[ "$(head -1 "$work/output")" = 'rozaneh: message 6' ]
cmp "$work/output" "$work/recent.log"
# Re-reading a snapshot does not duplicate messages.
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/repeated"
cmp "$work/output" "$work/repeated"
# A new helper process and an empty runtime log still return saved messages.
: > "$LOG_FIXTURE"
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/empty"
cmp "$work/output" "$work/empty"
# A new message after a cleared runtime log keeps the most recent history.
echo 'rozaneh: stopped' > "$LOG_FIXTURE"
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/stopped"
[ "$(wc -l < "$work/stopped")" = 30 ]
[ "$(head -1 "$work/stopped")" = 'rozaneh: message 7' ]
[ "$(tail -1 "$work/stopped")" = 'rozaneh: stopped' ]
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/repeated"
cmp "$work/stopped" "$work/repeated"
echo 'Persistent last 30 logs, repeat reads and empty runtime log checks passed'

# Deleted messages must not return from the system log; new messages remain visible.
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" clear
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/cleared"
[ ! -s "$work/cleared" ]
echo 'rozaneh: new after delete' >> "$LOG_FIXTURE"
sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/new"
[ "$(cat "$work/new")" = 'rozaneh: new after delete' ]
echo 'Delete and subsequent new log checks passed'
# Active sessions retain more than 300 lines; stopping preserves only the last 30.
rm -f "$work/recent.log.cleared"
: > "$work/recent.log"
awk 'BEGIN{for(i=1;i<=450;i++)print "rozaneh: active "i}' > "$LOG_FIXTURE"
tail -30 "$LOG_FIXTURE" > "$work/recent.log"
ROZANEH_LOG_ACTIVE=1 sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/live"
[ "$(wc -l < "$work/live")" = 450 ]
[ "$(wc -l < "$work/recent.log")" = 30 ]
ROZANEH_LOG_ACTIVE=1 sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/live-again"
cmp "$work/live" "$work/live-again"
: > "$LOG_FIXTURE"
ROZANEH_LOG_ACTIVE=0 sh "$root/luci/root/usr/libexec/rozaneh/read-logs" "$work/recent.log" > "$work/off"
[ "$(wc -l < "$work/off")" = 30 ]
[ "$(tail -1 "$work/off")" = 'rozaneh: active 450' ]
[ ! -f "$work/recent.log.active" ]
echo 'Unlimited active history and last 30 on stop checks passed'

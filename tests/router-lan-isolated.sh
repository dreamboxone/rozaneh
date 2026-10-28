#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
# Two namespaces emulate a LAN device and its router, without changing live LAN routes.
set -eu
umask 077
work=${1:?Usage: router-lan-isolated.sh NEW_WORK_DIRECTORY}
router=zd-lan-router
device=zd-lan-device
host=zdlan-uplink
table=zd_lan_test
for ns in "$router" "$device"; do
 if ip netns list | grep -Eq "^$ns([[:space:]]|$)"; then echo 'Test namespace already exists' >&2; exit 1; fi
done
if ip link show "$host" >/dev/null 2>&1 || nft list table inet "$table" >/dev/null 2>&1 ||
   [ -e "/etc/netns/$device" ] || [ -n "$(ip -4 route show exact 10.254.54.0/30)" ]; then
 echo 'Test network is already in use' >&2; exit 1
fi
[ ! -e "$work" ] || exit 1
mkdir -m 700 "$work"
mkdir "$work/config" "$work/bin" "$work/state"
cleanup() {
 for ns in "$device" "$router"; do
  for pid in $(ip netns pids "$ns" 2>/dev/null); do kill "$pid" 2>/dev/null || true; done
 done
 sleep 2
 for ns in "$device" "$router"; do
  for pid in $(ip netns pids "$ns" 2>/dev/null); do kill -KILL "$pid" 2>/dev/null || true; done
 done
 for handle in $(nft -a list chain inet fw4 forward | awk '/comment "zd-lan-test"/ {print $NF}'); do
  nft delete rule inet fw4 forward handle "$handle" || true
 done
 nft delete table inet "$table" 2>/dev/null || true
 ip link del "$host" 2>/dev/null || true
 ip netns del "$device" 2>/dev/null || true
 ip netns del "$router" 2>/dev/null || true
 rm -f "/etc/netns/$device/resolv.conf"
 rmdir "/etc/netns/$device" 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT TERM HUP
wan=$(ip -4 route show default | awk '{for(i=1;i<NF;i++) if($i=="dev") {print $(i+1);exit}}')
for ns in "$router" "$device"; do
 ip netns add "$ns"
 ip netns exec "$ns" ip link set lo up
done
ip netns exec "$router" sysctl -qw net.ipv4.ip_forward=1 net.ipv6.conf.all.forwarding=1
ip link add "$host" type veth peer name zd-wan
sysctl -qw "net.ipv6.conf.$host.disable_ipv6=1"
ip link set zd-wan netns "$router"
ip netns exec "$router" sysctl -qw net.ipv6.conf.zd-wan.disable_ipv6=1
ip addr add 10.254.54.1/30 dev "$host"
ip link set "$host" up
ip netns exec "$router" ip addr add 10.254.54.2/30 dev zd-wan
ip netns exec "$router" ip link set zd-wan up
ip netns exec "$router" ip route add default via 10.254.54.1
ip netns exec "$router" ip link add br-lan type veth peer name zd-client
ip netns exec "$router" ip link set zd-client netns "$device"
ip netns exec "$router" ip addr add 10.254.55.1/30 dev br-lan
ip netns exec "$router" ip link set br-lan up
ip netns exec "$device" ip addr add 10.254.55.2/30 dev zd-client
ip netns exec "$device" ip link set zd-client up
ip netns exec "$device" ip route add default via 10.254.55.1
ip netns exec "$router" ip -6 addr add fd54:55::1/64 dev br-lan
ip netns exec "$device" ip -6 addr add fd54:55::2/64 dev zd-client
ip netns exec "$device" ip -6 route add default via fd54:55::1
mkdir -p "/etc/netns/$device"
printf 'nameserver 10.254.55.1\n' > "/etc/netns/$device/resolv.conf"
nft -f - <<EOF
table inet $table {
 chain guard {
  type filter hook forward priority -300; policy accept;
  iifname "$host" ip daddr { 1.1.1.1, 1.0.0.1, 2.189.1.1 } meta l4proto { tcp, udp } th dport 53 counter accept
  iifname "$host" counter reject
 }
 chain postrouting {
  type nat hook postrouting priority srcnat; policy accept;
  ip saddr 10.254.54.2 oifname "$wan" masquerade
 }
}
EOF
nft insert rule inet fw4 forward iifname "$host" oifname "$wan" ip saddr 10.254.54.2 accept comment zd-lan-test
nft insert rule inet fw4 forward iifname "$wan" oifname "$host" ip daddr 10.254.54.2 ct state established,related accept comment zd-lan-test
ip netns exec "$router" nft -f - <<'EOF'
table inet fw4 {
 chain input {
  type filter hook input priority filter; policy drop;
  iifname { "lo", "br-lan" } accept
  ct state established,related accept
 }
 chain forward {
  type filter hook forward priority filter; policy drop;
  ct state established,related accept
 }
}
table inet only_dns {
 chain output {
  type filter hook output priority -300; policy drop;
  oifname { "lo", "br-lan", "zdtun0" } accept
  ip daddr { 1.1.1.1, 1.0.0.1, 2.189.1.1 } meta l4proto { tcp, udp } th dport 53 counter accept
  counter reject
 }
}
EOF
cp /etc/config/rozaneh "$work/config/rozaneh"
uci -c "$work/config" set rozaneh.main.enabled=1
uci -c "$work/config" set rozaneh.main.listen=127.0.0.1:1089
uci -c "$work/config" set rozaneh.main.lan_devices=br-lan
uci -c "$work/config" set rozaneh.main.lan_killswitch=0
uci -c "$work/config" commit rozaneh
printf '#!/bin/sh\nexec /sbin/uci -c "%s/config" "$@"\n' "$work" > "$work/bin/uci"
chmod 700 "$work/bin/uci"
cp /usr/libexec/rozaneh/dependencies.sh "$work/dependencies.sh"
# Live Zirgozar is outside these namespaces; no shared routes, listener or firewall.
printf '\nrozaneh_proxy_conflict() { return 1; }\n' >> "$work/dependencies.sh"
sed -e "s|/usr/libexec/rozaneh/dependencies.sh|$work/dependencies.sh|g" -e "s|/etc/rozaneh/lan-cache.db|$work/lan-cache.db|g" /usr/libexec/rozaneh/lan-config.sh > "$work/lan-config.sh"
sed "s|/var/run/rozaneh-lan|$work/state|g" /usr/libexec/rozaneh/lan-cleanup.sh > "$work/lan-cleanup.sh"
sed -e "s|/usr/libexec/rozaneh/dependencies.sh|$work/dependencies.sh|g" -e "s|/usr/libexec/rozaneh/lan-config.sh|$work/lan-config.sh|g" -e "s|/usr/libexec/rozaneh/lan-cleanup.sh|$work/lan-cleanup.sh|g" -e "s|/var/run/rozaneh-lan|$work/state|g" /usr/libexec/rozaneh/lan-run.sh > "$work/lan-run.sh"
chmod 700 "$work/lan-config.sh" "$work/lan-cleanup.sh" "$work/lan-run.sh"
zone=$(uci -q get rozaneh.main.zone)
key=$(uci -q get rozaneh.main.pubkey)
ip netns exec "$router" tcpdump -U -n -i zd-wan -w "$work/uplink.pcap" > "$work/capture.log" 2>&1 &
uplink_capture=$!
ip netns exec "$router" tcpdump -U -n -i br-lan -w "$work/lan.pcap" > "$work/lan-capture.log" 2>&1 &
ip netns exec "$router" /usr/bin/rozaneh -zone "$zone" -pubkey "$key" -resolvers 1.1.1.1:53,1.0.0.1:53,2.189.1.1:53 -tcp "${ROZANEH_TEST_TCP:-on}" -listen 127.0.0.1:1089 -diagnostics-file "$work/paths.json" > "$work/core.log" 2>&1 &
core_pid=$!
ip netns exec "$router" env PATH="$work/bin:$PATH" UCI_CONFIG_DIR="$work/config" "$work/lan-run.sh" > "$work/lan.log" 2>&1 &
ready=0
for attempt in $(seq 1 15); do
 if [ -s "$work/state/ready" ]; then ready=1; break; fi
 sleep 1
done
if [ "$ready" != 1 ]; then cat "$work/lan.log"; exit 1; fi
echo 'PRODUCTION LAN ROUTES'
ip netns exec "$router" ip -4 rule show
ip netns exec "$router" ip -4 route show table 28028
echo 'DEVICE DNS'
ip netns exec "$device" nslookup www.cloudflare.com 10.254.55.1 > "$work/dns.txt" 2>&1 || true
cat "$work/dns.txt"
echo 'DEVICE HTTPS WITHOUT PROXY SETTINGS'
ip netns exec "$device" curl --noproxy '*' --connect-timeout 30 --max-time 45 --fail -sS -w '\nHTTP %{http_code} time %{time_total}\n' https://www.cloudflare.com/cdn-cgi/trace > "$work/trace.txt" 2> "$work/trace.err" || true
cat "$work/trace.txt" "$work/trace.err"
ip netns exec "$device" curl --noproxy '*' --connect-timeout 30 --max-time 45 -sS -o /dev/null -w 'Google HTTP %{http_code} time %{time_total}\n' https://www.gstatic.com/generate_204 > "$work/google.txt" 2> "$work/google.err" || true
cat "$work/google.txt" "$work/google.err"
grep -q '^HTTP 200 ' "$work/trace.txt"
grep -q '^Google HTTP 204 ' "$work/google.txt"
if [ "${ROZANEH_TEST_QUICK:-0}" != 1 ]; then
echo 'DEVICE DOWNLOAD'
ip netns exec "$device" curl -4 --noproxy '*' --connect-timeout 15 --max-time 90 --fail -sS 'https://speed.cloudflare.com/__down?bytes=262144' -o "$work/download.bin"
[ "$(wc -c < "$work/download.bin")" = 262144 ]
echo 'download_bytes=262144'
echo 'DEVICE UPLOAD'
dd if=/dev/zero of="$work/upload.bin" bs=1024 count=64 2>/dev/null
ip netns exec "$device" curl -4 --noproxy '*' --connect-timeout 45 --max-time 90 --fail -sS --data-binary "@$work/upload.bin" -o "$work/upload-response.txt" -w 'upload_http=%{http_code} upload_bytes=%{size_upload}\n' https://speed.cloudflare.com/__up > "$work/upload.txt"
cat "$work/upload.txt"
grep -q '^upload_http=200 upload_bytes=65536$' "$work/upload.txt"
fi
echo 'DEVICE IPV6 LAN THROUGH DOMAIN TUNNEL'
ip netns exec "$device" curl -6 --noproxy '*' --connect-timeout 40 --max-time 60 --fail -sS -w '\nIPv6 HTTP %{http_code}\n' https://www.cloudflare.com/cdn-cgi/trace > "$work/ipv6.txt" 2> "$work/ipv6.err" || true
cat "$work/ipv6.txt" "$work/ipv6.err"
grep -q '^IPv6 HTTP 200$' "$work/ipv6.txt"
echo 'FIREWALL RULE RECOVERY'
for chain in input forward; do
 for handle in $(ip netns exec "$router" nft -a list chain inet fw4 "$chain" | awk '/comment "rozaneh-(lan|tcp[46])"/ {print $NF}'); do
  ip netns exec "$router" nft delete rule inet fw4 "$chain" handle "$handle"
 done
done
sleep 4
ip netns exec "$router" nft list chain inet fw4 input > "$work/input-rules.txt"
cat "$work/input-rules.txt"
grep -q 'comment "rozaneh-tcp4"' "$work/input-rules.txt"
grep -q 'comment "rozaneh-tcp6"' "$work/input-rules.txt"
ip netns exec "$device" curl -4 --noproxy '*' --connect-timeout 40 --max-time 60 --fail -sS -o /dev/null -w 'recovery_http=%{http_code}\n' https://www.gstatic.com/generate_204 > "$work/recovery.txt"
cat "$work/recovery.txt"
grep -q '^recovery_http=204$' "$work/recovery.txt"
echo 'CACHE PERSISTENCE ACROSS LAN RESTART'
cached_ip=$(awk '/Address: 198.19./ {print $2;exit}' "$work/dns.txt")
[ -n "$cached_ip" ]
owner=$(cat "$work/state/owned")
kill "$owner"
for attempt in $(seq 1 10); do [ ! -f "$work/state/owned" ] && break; sleep 1; done
[ ! -f "$work/state/owned" ]
ip netns exec "$router" env PATH="$work/bin:$PATH" UCI_CONFIG_DIR="$work/config" "$work/lan-run.sh" >> "$work/lan.log" 2>&1 &
for attempt in $(seq 1 15); do [ -s "$work/state/ready" ] && break; sleep 1; done
[ -s "$work/state/ready" ]
ip netns exec "$device" curl -4 --noproxy '*' --resolve "www.cloudflare.com:443:$cached_ip" --connect-timeout 40 --max-time 60 --fail -sS -o /dev/null -w 'cached_address_http=%{http_code}\n' https://www.cloudflare.com/cdn-cgi/trace > "$work/cache-recovery.txt"
cat "$work/cache-recovery.txt"
grep -q '^cached_address_http=200$' "$work/cache-recovery.txt"
echo 'NO DIRECT FALLBACK WHEN CORE STOPS'
cp "$work/paths.json" "$work/connected-paths.json"
kill "$core_pid"
wait "$core_pid" 2>/dev/null || true
if ip netns exec "$device" curl -4 --noproxy '*' --connect-timeout 5 --max-time 8 --fail -sS https://www.gstatic.com/generate_204 > "$work/stopped-core.txt" 2> "$work/stopped-core.err"; then
 echo 'Unexpected connection with core stopped' >&2; exit 1
fi
cat "$work/stopped-core.err"
echo 'KILL SWITCH SURVIVES LAN STOP'
uci -c "$work/config" set rozaneh.main.lan_killswitch=1
uci -c "$work/config" commit rozaneh
ip netns exec "$router" env PATH="$work/bin:$PATH" UCI_CONFIG_DIR="$work/config" /usr/libexec/rozaneh/lan-firewall.sh
echo 'UPLINK FIREWALL'
nft list table inet "$table"
echo 'CLEAN SHUTDOWN'
owner=$(cat "$work/state/owned")
kill "$owner"
for attempt in $(seq 1 10); do
 [ ! -f "$work/state/owned" ] && break
 sleep 1
done
[ ! -f "$work/state/owned" ]
for chain in input forward; do
 if ip netns exec "$router" nft list chain inet fw4 "$chain" | grep -q 'comment "rozaneh-'; then echo 'Firewall cleanup failed' >&2; exit 1; fi
done
echo 'Owned firewall rules removed on stop'
ip netns exec "$device" curl -4 --noproxy '*' --resolve www.cloudflare.com:443:104.16.123.96 --connect-timeout 2 --max-time 3 -sS https://www.cloudflare.com/cdn-cgi/trace > "$work/guard.txt" 2> "$work/guard.err" && exit 1
ip netns exec "$router" nft list chain inet rozaneh_guard forward > "$work/guard-rules.txt"
grep 'ip daddr !=' "$work/guard-rules.txt" | grep -Eq 'counter packets [1-9][0-9]* '
echo 'Kill switch blocked direct traffic after shutdown'
kill -INT "$uplink_capture"
wait "$uplink_capture" 2>/dev/null || true
echo 'UPLINK NON-DNS PACKETS'
tcpdump -n -r "$work/uplink.pcap" 'ip or ip6' 2>/dev/null | awk '!/ > (1\.1\.1\.1|1\.0\.0\.1|2\.189\.1\.1)\.53:/ && !/(1\.1\.1\.1|1\.0\.0\.1|2\.189\.1\.1)\.53 > / {print; n++} END {print "non_dns_packets=" n+0; exit(n>0)}'
grep -q '^HTTP 200 ' "$work/trace.txt" && grep -q '^Google HTTP 204 ' "$work/google.txt"

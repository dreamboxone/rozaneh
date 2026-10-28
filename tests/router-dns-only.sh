#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
# Run a temporary core in a separate network namespace; leave services untouched.
set -eu
ns=zd-dns-test
host=zdtest-host
peer=zdtest-peer
table=zd_dns_test
work=/tmp/rozaneh-dns-only
[ ! -e "$work" ] || { echo 'Test directory already exists' >&2; exit 1; }
mkdir -m 700 "$work"
client=''; capture=''
cleanup() {
 [ -z "$client" ] || kill "$client" 2>/dev/null || true
 [ -z "$capture" ] || { kill -INT "$capture" 2>/dev/null || true; wait "$capture" 2>/dev/null || true; }
 for handle in $(nft -a list chain inet fw4 forward | awk '/comment "zd-dns-test"/ {print $NF}'); do
  nft delete rule inet fw4 forward handle "$handle" || true
 done
 nft delete table ip "$table" 2>/dev/null || true
 ip link del "$host" 2>/dev/null || true
 ip netns del "$ns" 2>/dev/null || true
}
trap cleanup EXIT
trap 'exit 130' INT TERM HUP
wan=$(ip -4 route show default | awk '{for(i=1;i<NF;i++) if($i=="dev") {print $(i+1);exit}}')
[ -n "$wan" ]
ip netns add "$ns"
ip netns exec "$ns" sysctl -qw net.ipv6.conf.all.disable_ipv6=1
ip netns exec "$ns" sysctl -qw net.ipv6.conf.default.disable_ipv6=1
ip link add "$host" type veth peer name "$peer"
sysctl -qw "net.ipv6.conf.$host.disable_ipv6=1"
ip link set "$peer" netns "$ns"
ip addr add 10.254.53.1/30 dev "$host"
ip link set "$host" up
ip netns exec "$ns" ip link set lo up
ip netns exec "$ns" ip addr add 10.254.53.2/30 dev "$peer"
ip netns exec "$ns" ip link set "$peer" up
ip netns exec "$ns" ip route add default via 10.254.53.1
ip netns exec "$ns" nft -f - <<'EOF'
table inet only_dns {
 chain output {
  type filter hook output priority -300; policy drop;
  oifname "lo" accept
  ip daddr { 1.1.1.1, 1.0.0.1, 2.189.1.1 } udp dport 53 counter accept
  counter reject
 }
}
EOF
nft -f - <<EOF
table ip $table {
 chain postrouting {
  type nat hook postrouting priority srcnat; policy accept;
  ip saddr 10.254.53.2 oifname "$wan" masquerade
 }
}
EOF
nft insert rule inet fw4 forward iifname "$host" oifname "$wan" ip saddr 10.254.53.2 accept comment zd-dns-test
nft insert rule inet fw4 forward iifname "$wan" oifname "$host" ip daddr 10.254.53.2 ct state established,related accept comment zd-dns-test
ip netns exec "$ns" tcpdump -U -n -i "$peer" -w "$work/dns-only.pcap" > "$work/capture.log" 2>&1 &
capture=$!
echo 'NEGATIVE CONTROL: direct HTTPS must fail'
if ip netns exec "$ns" curl --noproxy '*' --connect-timeout 3 --max-time 5 --resolve www.cloudflare.com:443:1.1.1.1 -sS https://www.cloudflare.com/cdn-cgi/trace > "$work/direct.txt" 2> "$work/direct.err"; then
 echo 'Isolation failed: direct HTTPS succeeded' >&2; exit 1
fi
cat "$work/direct.err"
zone=$(uci -q get rozaneh.main.zone)
key=$(uci -q get rozaneh.main.pubkey)
ip netns exec "$ns" /usr/bin/rozaneh -zone "$zone" -pubkey "$key" -resolvers 1.1.1.1:53,1.0.0.1:53,2.189.1.1:53 -tcp off -listen 127.0.0.1:1089 -diagnostics-file "$work/paths.json" > "$work/core.log" 2>&1 &
client=$!
sleep 2
echo 'DNS-ONLY TUNNEL: HTTPS through temporary SOCKS'
ip netns exec "$ns" curl --noproxy '' --socks5-hostname 127.0.0.1:1089 --connect-timeout 20 --max-time 60 --fail -sS -w '\nHTTP %{http_code} time %{time_total}\n' https://www.cloudflare.com/cdn-cgi/trace > "$work/trace.txt" 2> "$work/trace.err" || true
cat "$work/trace.txt" "$work/trace.err"
grep -q '^ip=' "$work/trace.txt"
grep -q '^HTTP 200 ' "$work/trace.txt"
ip netns exec "$ns" curl --noproxy '' --socks5-hostname 127.0.0.1:1089 --connect-timeout 20 --max-time 45 -sS -o /dev/null -w 'Google HTTP %{http_code} time %{time_total}\n' https://www.gstatic.com/generate_204 > "$work/google.txt" 2> "$work/google.err" || true
cat "$work/google.txt" "$work/google.err"
grep -q '^Google HTTP 204 ' "$work/google.txt"
echo 'NEGATIVE CONTROL: block DNS too; a fresh core must fail'
kill "$client" 2>/dev/null || true
wait "$client" 2>/dev/null || true
client=''
ip netns exec "$ns" nft insert rule inet only_dns output oifname "$peer" udp dport 53 counter drop
ip netns exec "$ns" /usr/bin/rozaneh -zone "$zone" -pubkey "$key" -resolvers 1.1.1.1:53,1.0.0.1:53,2.189.1.1:53 -tcp off -listen 127.0.0.1:1089 > "$work/blocked-core.log" 2>&1 &
client=$!
sleep 1
if ip netns exec "$ns" curl --noproxy '' --socks5-hostname 127.0.0.1:1089 --connect-timeout 8 --max-time 12 --fail -sS https://www.cloudflare.com/cdn-cgi/trace > "$work/blocked.txt" 2> "$work/blocked.err"; then
 echo 'Isolation failed: tunnel worked with DNS blocked' >&2; exit 1
fi
cat "$work/blocked.err"
echo 'NAMESPACE FIREWALL COUNTERS'
ip netns exec "$ns" nft list table inet only_dns
kill "$client" 2>/dev/null || true
wait "$client" 2>/dev/null || true
client=''
kill -INT "$capture" 2>/dev/null || true
wait "$capture" 2>/dev/null || true
capture=''
echo 'NON-DNS IP PACKETS ON ISOLATED INTERFACE (must be zero)'
tcpdump -n -r "$work/dns-only.pcap" 'ip or ip6' 2>/dev/null | awk '!/ > (1\.1\.1\.1|1\.0\.0\.1|2\.189\.1\.1)\.53:/ && !/(1\.1\.1\.1|1\.0\.0\.1|2\.189\.1\.1)\.53 > / {print; n++} END {print "non_dns_packets=" n+0; exit(n>0)}'
echo 'DIAGNOSTICS'
cat "$work/paths.json" 2>/dev/null || true
echo 'CORE LOG'
tail -15 "$work/core.log"

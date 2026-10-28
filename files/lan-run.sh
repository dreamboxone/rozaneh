#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
set -eu
. /usr/libexec/rozaneh/dependencies.sh
rozaneh_missing_dependencies lan
[ -z "$missing" ] || { logger -t rozaneh "LAN mode needs: $missing"; exit 1; }
singbox=$(rozaneh_find_singbox) || exit 1
if rozaneh_proxy_conflict; then
 logger -t rozaneh 'Another transparent proxy is active; LAN mode was not started'; exit 1
fi
devices=$(uci -q get rozaneh.main.lan_devices || echo br-lan)
rozaneh_check_lan_devices "$devices"
for device in $devices; do
 case "$device" in ''|*[!a-zA-Z0-9_.:-]*) exit 1 ;; esac
 [ "${#device}" -le 15 ] || exit 1
 ip link show dev "$device" >/dev/null 2>&1 || { logger -t rozaneh "LAN device not found: $device"; exit 1; }
done
/usr/libexec/rozaneh/lan-firewall.sh
for family in -4 -6; do
 [ -z "$(ip "$family" route show table 28028 2>/dev/null)" ] || { logger -t rozaneh 'Routing table 28028 is already in use'; exit 1; }
 ! ip "$family" rule show | grep -q '^12801:' || { logger -t rozaneh 'Routing rule priority 12801 is already in use'; exit 1; }
done
nft list table inet rozaneh_dns >/dev/null 2>&1 && { logger -t rozaneh 'LAN DNS table already exists'; exit 1; }
umask 077
state=/var/run/rozaneh-lan
mkdir -p "$state"
chmod 700 "$state"
mkdir "$state/lock" || exit 1
echo $$ > "$state/owned"
cleanup() { /usr/libexec/rozaneh/lan-cleanup.sh "$$"; }
trap cleanup EXIT
trap 'exit 0' INT TERM
printf '%s\n' $devices > "$state/devices"
/usr/libexec/rozaneh/lan-config.sh > "$state/sing-box.json"
"$singbox" check -c "$state/sing-box.json"
"$singbox" run -c "$state/sing-box.json" &
pid=$!
echo "$pid" > "$state/engine.pid"
ready=0
for attempt in 1 2 3 4 5 6 7 8 9 10; do
 kill -0 "$pid" 2>/dev/null || exit 1
 if ip link show zdtun0 2>/dev/null | grep -Eq '[<,]UP[,>]'; then ready=1; break; fi
 sleep 1
done
[ "$ready" = 1 ] || { logger -t rozaneh 'LAN tunnel did not start'; exit 1; }
cat /proc/sys/net/ipv4/ip_forward > "$state/ipv4-forward.before"
cat /proc/sys/net/ipv6/conf/all/forwarding > "$state/ipv6-forward.before"
echo 1 > /proc/sys/net/ipv4/ip_forward
echo 1 > /proc/sys/net/ipv6/conf/all/forwarding
# Policy tables are fixed so their ownership and cleanup are independent of CPU word size.
ip -4 route add default dev zdtun0 table 28028
ip -6 route add default dev zdtun0 table 28028
# Synthetic IPv6 destinations must reach the tunnel before the private-LAN bypass.
ip -6 route add fd00:5a00::/48 dev zdtun0 table 28028
for prefix in 0.0.0.0/8 10.0.0.0/8 100.64.0.0/10 127.0.0.0/8 169.254.0.0/16 172.16.0.0/12 192.168.0.0/16 224.0.0.0/4 240.0.0.0/4; do
 ip -4 route add throw "$prefix" table 28028
done
for prefix in '::/128' '::1/128' 'fc00::/7' 'fe80::/10' 'ff00::/8'; do
 ip -6 route add throw "$prefix" table 28028
done
{
 echo 'table inet rozaneh_dns {'
 echo 'chain dns_lan { type nat hook prerouting priority -110; policy accept;'
 for device in $devices; do
  printf 'iifname "%s" meta nfproto ipv4 meta l4proto { tcp, udp } th dport 53 dnat ip to 198.18.0.2\n' "$device"
  printf 'iifname "%s" meta nfproto ipv6 meta l4proto { tcp, udp } th dport 53 dnat ip6 to 2001:db8:5a00::2\n' "$device"
 done
 printf '}\n}\n'
} > "$state/dns.nft"
nft -c -f "$state/dns.nft"
nft -f "$state/dns.nft"
ensure_firewall() {
 # The system TCP stack injects translated SYNs back through the TUN device
 # to its local listener. fw4 has no zone for this device and otherwise drops them.
 if ! nft list chain inet fw4 input | grep -q 'comment "rozaneh-tcp4"'; then
  nft insert rule inet fw4 input iifname zdtun0 ip saddr 198.18.0.2 ip daddr 198.18.0.1 meta l4proto tcp counter accept comment 'rozaneh-tcp4'
 fi
 if ! nft list chain inet fw4 input | grep -q 'comment "rozaneh-tcp6"'; then
  nft insert rule inet fw4 input iifname zdtun0 ip6 saddr 2001:db8:5a00::2 ip6 daddr 2001:db8:5a00::1 meta l4proto tcp counter accept comment 'rozaneh-tcp6'
 fi
 for device in $devices; do
  if ! nft list chain inet fw4 forward | grep -F "iifname \"$device\" oifname \"zdtun0\"" | grep -q 'comment "rozaneh-lan"'; then
   nft insert rule inet fw4 forward iifname "$device" oifname zdtun0 accept comment 'rozaneh-lan'
  fi
 done
}
ensure_firewall
for device in $devices; do
 ip -4 rule add pref 12801 iif "$device" lookup 28028
 ip -6 rule add pref 12801 iif "$device" lookup 28028
done
printf '%s\n' "$$" > "$state/ready"
logger -t rozaneh "LAN internet sharing started using $singbox (TCP through Soar, DNS through HTTPS)"
while kill -0 "$pid" 2>/dev/null; do ensure_firewall; sleep 3 & wait $! || true; done
wait "$pid"

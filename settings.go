// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net"
	"net/netip"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

type soarSettings struct {
	safeName, maxName, maxInflight, pathWindow, maxAnswer, maxStreams, sendBuffer, streamWindow int
	hold, sessionIdle, pace                                                                     time.Duration
	want, concurrency, samples                                                                  int
	candidates, cache, network                                                                  string
	debug                                                                                       bool
	fecDownload, qnameLossGuard                                                                 bool
	fecData, fecParity                                                                          int
}

func registerSoarSettings() *soarSettings {
	s := &soarSettings{}
	flag.BoolVar(&s.fecDownload, "fec-download", false, "request download FEC; compatible server required")
	flag.IntVar(&s.fecData, "fec-data", 4, "FEC data shards (2-16)")
	flag.IntVar(&s.fecParity, "fec-parity", 1, "FEC parity shards (1-4)")
	flag.BoolVar(&s.qnameLossGuard, "qname-loss-guard", false, "experimental authenticated QNAME loss guard")
	flag.IntVar(&s.safeName, "safe-qname", 101, "initial DNS query name length")
	flag.IntVar(&s.maxName, "max-qname", 253, "maximum probed DNS query name length")
	flag.IntVar(&s.maxInflight, "max-inflight", 128, "maximum outstanding DNS queries (1-1024)")
	flag.IntVar(&s.pathWindow, "path-window", 32, "maximum outstanding queries per path")
	flag.IntVar(&s.maxAnswer, "max-answer", 1232, "maximum probed DNS response size (512-1232)")
	flag.IntVar(&s.maxStreams, "max-streams", 128, "maximum streams per Soar session (1-256)")
	flag.IntVar(&s.sendBuffer, "send-buffer", 65536, "send buffer bytes per stream")
	flag.IntVar(&s.streamWindow, "stream-window", 262144, "receive window bytes per stream")
	flag.DurationVar(&s.hold, "hold", 500*time.Millisecond, "maximum server poll hold time (100ms-1500ms)")
	flag.DurationVar(&s.sessionIdle, "session-idle", 90*time.Second, "retire an unused session before the server idle timeout")
	flag.DurationVar(&s.pace, "discovery-pace", 20*time.Millisecond, "minimum interval between discovery candidates")
	flag.IntVar(&s.want, "discovery-want", 16, "desired healthy discovery paths")
	flag.IntVar(&s.concurrency, "discovery-concurrency", 16, "maximum concurrent unverified discovery paths")
	flag.IntVar(&s.samples, "discovery-sample-ranges", 0, "optional in-country address samples; disabled by default")
	flag.StringVar(&s.candidates, "discovery-candidates", "", "extra discovery resolver IP:ports")
	flag.StringVar(&s.cache, "discovery-cache", "", "network-specific verified resolver cache file")
	flag.StringVar(&s.network, "network-id", "", "optional network cache identity override")
	flag.BoolVar(&s.debug, "debug", false, "enable detailed Soar diagnostics")
	return s
}

func (s *soarSettings) validate() error {
	if s.safeName < 64 || s.maxName < s.safeName || s.maxName > 253 || s.maxInflight < 1 || s.maxInflight > 1024 || s.pathWindow < 1 || s.pathWindow > 1024 || s.maxAnswer < 512 || s.maxAnswer > 1232 || s.maxStreams < 1 || s.maxStreams > 256 || s.sendBuffer < 4096 || s.sendBuffer > 16<<20 || s.streamWindow < 4096 || s.streamWindow > 16<<20 || s.hold < 100*time.Millisecond || s.hold > 1500*time.Millisecond || s.sessionIdle < time.Second || s.sessionIdle > time.Hour || s.pace < time.Millisecond || s.pace > time.Minute || s.want < 1 || s.want > 128 || s.concurrency < 1 || s.concurrency > 64 || s.samples < 0 || s.samples > 4096 {
		return errors.New("invalid advanced Soar settings; check query sizes, windows and discovery limits")
	}
	for _, candidate := range splitList(s.candidates) {
		if _, err := resolverIP(candidate); err != nil {
			return err
		}
	}
	return nil
}

func splitList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' })
}

// OpenWrt's /etc/resolv.conf usually points at dnsmasq on loopback. Use the
// actual uplink resolvers too, without sending discovery through that local cache.
func upstreamResolvers(paths ...string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(file, 65537))
		file.Close()
		if err != nil || len(data) > 65536 {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			fields := strings.Fields(line)
			if len(fields) < 2 || fields[0] != "nameserver" {
				continue
			}
			address, err := resolverIP(fields[1])
			if err != nil || seen[address] {
				continue
			}
			ap, _ := netip.ParseAddrPort(address)
			if ap.Addr().IsLoopback() || ap.Addr().IsUnspecified() {
				continue
			}
			seen[address] = true
			out = append(out, address)
		}
	}
	return out
}

type resolverFileCache struct {
	path, key string
	mu        sync.Mutex
}
type cachedResolvers struct {
	Network   string   `json:"network"`
	Resolvers []string `json:"resolvers"`
}

func newResolverCache(path, identity string) soar.ResolverCache {
	if path == "" {
		return nil
	}
	hash := sha256.Sum256([]byte(identity))
	return &resolverFileCache{path: path, key: hex.EncodeToString(hash[:])}
}

func networkIdentity() string {
	var addresses []string
	interfaces, _ := net.Interfaces()
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || strings.HasPrefix(iface.Name, "zdtun") {
			continue
		}
		if addrs, err := iface.Addrs(); err == nil {
			for _, addr := range addrs {
				addresses = append(addresses, iface.Name+":"+addr.String())
			}
		}
	}
	sort.Strings(addresses)
	return strings.Join(addresses, ",")
}

func (c *resolverFileCache) Load() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	data, err := os.ReadFile(c.path)
	if err != nil || len(data) > 65536 {
		return nil
	}
	var entry cachedResolvers
	if json.Unmarshal(data, &entry) != nil || entry.Network != c.key || len(entry.Resolvers) > 256 {
		return nil
	}
	var out []string
	for _, address := range entry.Resolvers {
		if value, err := resolverIP(address); err == nil && !strings.HasPrefix(value, "127.254.") {
			out = append(out, value)
		}
	}
	return out
}

func (c *resolverFileCache) Store(addresses []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	var valid []string
	for _, address := range addresses {
		if value, err := resolverIP(address); err == nil && !strings.HasPrefix(value, "127.254.") && len(valid) < 256 {
			valid = append(valid, value)
		}
	}
	data, err := json.Marshal(cachedResolvers{c.key, valid})
	if err != nil {
		return
	}
	if old, err := os.ReadFile(c.path); err == nil && string(old) == string(data) {
		return
	}
	if os.WriteFile(c.path+".tmp", data, 0600) == nil {
		os.Rename(c.path+".tmp", c.path)
	}
}

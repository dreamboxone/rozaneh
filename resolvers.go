// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dreamboxone/rozaneh/internal/resolvertransport"
	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

type resolverEndpoint struct {
	label, scheme, address, serverName, bootstrap string
	u                                             *url.URL
	port                                          int
}

type resolverTransports struct {
	addresses             []string
	endpoints             map[string]resolverEndpoint
	strict, http3Fallback bool
	clients               sync.Map
	rootCAs               *x509.CertPool
}

func parseResolvers(spec string) (*resolverTransports, error) {
	return parseResolverPolicy(spec, "auto", false, false)
}

func parseResolverPolicy(spec, mode string, encryptedOnly, fallback bool) (*resolverTransports, error) {
	if mode != "auto" && mode != "dot" && mode != "doh" && mode != "doh3" && mode != "doq" {
		return nil, fmt.Errorf("unsupported resolver mode: %s", mode)
	}
	r := &resolverTransports{endpoints: make(map[string]resolverEndpoint), strict: encryptedOnly || mode != "auto", http3Fallback: fallback}
	seen := make(map[string]bool)
	for _, raw := range strings.FieldsFunc(spec, func(c rune) bool { return c == ',' || c == '\n' || c == '\r' }) {
		raw = strings.TrimSpace(raw)
		if raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		if len(r.addresses) >= 64 {
			return nil, fmt.Errorf("at most 64 configured resolvers are supported")
		}
		if mode != "auto" && !strings.Contains(raw, "://") {
			address, err := resolverIP(raw)
			if err != nil {
				return nil, err
			}
			ap, _ := netip.ParseAddrPort(address)
			port := ap.Port()
			if port == 53 {
				if mode == "dot" || mode == "doq" {
					port = 853
				} else {
					port = 443
				}
			}
			raw = mode + "://" + net.JoinHostPort(ap.Addr().String(), strconv.Itoa(int(port)))
		}
		if !strings.Contains(raw, "://") || strings.HasPrefix(raw, "udp://") {
			if r.strict {
				return nil, fmt.Errorf("encrypted-only mode does not allow plaintext resolvers")
			}
			address, err := resolverIP(strings.TrimPrefix(raw, "udp://"))
			if err != nil {
				return nil, err
			}
			r.addresses = append(r.addresses, address)
			continue
		}
		u, err := url.Parse(raw)
		if err != nil || u.User != nil || u.Hostname() == "" || u.Fragment != "" {
			return nil, fmt.Errorf("invalid resolver URL: %s", raw)
		}
		u.Scheme = strings.ToLower(u.Scheme)
		aliases := map[string]string{"tcp": "tcp", "tls": "dot", "dot": "dot", "https": "doh", "doh": "doh", "doh3": "doh3", "doq": "doq"}
		scheme := aliases[u.Scheme]
		if scheme == "" {
			return nil, fmt.Errorf("unsupported resolver transport: %s", u.Scheme)
		}
		if r.strict && scheme == "tcp" {
			return nil, fmt.Errorf("encrypted-only mode does not allow TCP plaintext DNS")
		}
		if mode != "auto" && scheme != mode {
			return nil, fmt.Errorf("resolver scheme does not match selected mode %s", mode)
		}
		e := resolverEndpoint{label: raw, scheme: scheme, serverName: u.Hostname(), bootstrap: u.Query().Get("bootstrap")}
		if e.bootstrap != "" {
			if _, err := netip.ParseAddr(e.bootstrap); err != nil {
				return nil, fmt.Errorf("bootstrap must be an IP address")
			}
		}
		if name := u.Query().Get("server_name"); name != "" {
			e.serverName = name
		}
		port := u.Port()
		if port == "" {
			port = map[string]string{"tcp": "53", "dot": "853", "doh": "443", "doh3": "443", "doq": "853"}[scheme]
		}
		e.port, _ = strconv.Atoi(port)
		if scheme == "doq" && e.port == 53 {
			return nil, fmt.Errorf("DoQ must not use UDP port 53")
		}
		if ap, err := netip.ParseAddrPort(net.JoinHostPort("127.0.0.1", port)); err != nil || ap.Port() == 0 {
			return nil, fmt.Errorf("invalid resolver port")
		}
		host := u.Hostname()
		if e.bootstrap != "" {
			host = e.bootstrap
		}
		e.address = net.JoinHostPort(host, port)
		if r.strict {
			if _, err := netip.ParseAddr(host); err != nil {
				return nil, fmt.Errorf("encrypted-only resolvers need a numeric IP or bootstrap=IP")
			}
		}
		if scheme == "tcp" {
			if u.Path != "" && u.Path != "/" {
				return nil, fmt.Errorf("TCP resolver cannot have a path")
			}
			if _, err := netip.ParseAddr(host); err != nil {
				return nil, fmt.Errorf("TCP resolver requires an IP or bootstrap IP")
			}
		}
		q := u.Query()
		q.Del("bootstrap")
		q.Del("server_name")
		u.RawQuery = q.Encode()
		if scheme != "doh" && scheme != "doh3" && ((u.Path != "" && u.Path != "/") || u.RawQuery != "") {
			return nil, fmt.Errorf("unexpected path or query in %s resolver", u.Scheme)
		}
		if (scheme == "doh" || scheme == "doh3") && (u.Path == "" || u.Path == "/") {
			u.Path = "/dns-query"
		}
		if scheme == "doh" || scheme == "doh3" {
			u.Scheme = "https"
		}
		e.u = u
		virtual := fmt.Sprintf("127.254.0.%d:53", len(r.addresses)+1)
		r.endpoints[virtual] = e
		r.addresses = append(r.addresses, virtual)
	}
	if r.strict && len(r.addresses) == 0 {
		return nil, fmt.Errorf("encrypted-only mode needs at least one encrypted resolver")
	}
	return r, nil
}

func resolverIP(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if ip, err := netip.ParseAddr(raw); err == nil {
		raw = netip.AddrPortFrom(ip, 53).String()
	}
	if ap, err := netip.ParseAddrPort(raw); err == nil && ap.Port() != 0 {
		if netip.MustParsePrefix("127.254.0.0/16").Contains(ap.Addr().Unmap()) {
			return "", fmt.Errorf("resolver address is reserved for encrypted DNS transports: %s", raw)
		}
		return ap.String(), nil
	}
	return "", fmt.Errorf("invalid resolver IP:port: %s", raw)
}

func (r *resolverTransports) streamOnly(address string) bool {
	_, ok := r.endpoints[address]
	return ok
}

func (r *resolverTransports) constrainDiscovery(discovery soar.Discovery) soar.Discovery {
	if r.strict {
		return soar.Discovery{}
	}
	return discovery
}

func (r *resolverTransports) dial(ctx context.Context, network, address string) (net.Conn, error) {
	d := &net.Dialer{Timeout: 8 * time.Second}
	e, ok := r.endpoints[address]
	if !ok {
		if r.strict {
			return nil, fmt.Errorf("plaintext resolver blocked by encrypted-only policy")
		}
		return d.DialContext(ctx, network, address)
	}
	if r.strict {
		host, _, err := net.SplitHostPort(e.address)
		if err != nil {
			return nil, err
		}
		if _, err := netip.ParseAddr(host); err != nil {
			return nil, fmt.Errorf("plaintext bootstrap lookup blocked by encrypted-only policy")
		}
		if e.scheme == "tcp" {
			return nil, fmt.Errorf("plaintext TCP resolver blocked by encrypted-only policy")
		}
	}
	switch e.scheme {
	case "tcp":
		return d.DialContext(ctx, "tcp", e.address)
	case "dot":
		td := &tls.Dialer{NetDialer: d, Config: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: e.serverName, RootCAs: r.rootCAs}}
		return td.DialContext(ctx, "tcp", e.address)
	case "doh", "doh3", "doq":
		host, port, _ := net.SplitHostPort(e.address)
		if _, parseErr := netip.ParseAddr(host); parseErr != nil {
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 {
				return nil, fmt.Errorf("resolver hostname has no address")
			}
			e.address = net.JoinHostPort(ips[0].IP.String(), port)
		}
		path := "/dns-query"
		if e.u != nil {
			path = e.u.RequestURI()
		}
		carrier, err := resolvertransport.New(e.scheme, resolvertransport.Options{TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: e.serverName, RootCAs: r.rootCAs}, DoHPort: e.port, DoQPort: e.port, DoHPath: path, HTTP3Fallback: r.http3Fallback, HTTPHost: e.u.Host})
		if err != nil {
			return nil, err
		}
		r.clients.Store(address, carrier)
		return newEncryptedDNSConn(e, carrier), nil
	}
	return nil, fmt.Errorf("unsupported resolver transport")
}

func (r *resolverTransports) pathStats(client tunnelStatus) []soar.PathStat {
	paths := client.PathStats()
	for i := range paths {
		if e, ok := r.endpoints[paths[i].Resolver]; ok {
			address := paths[i].Resolver
			paths[i].Resolver = e.label
			paths[i].Transport = e.scheme
			if carrier, ok := r.clients.Load(address); ok {
				paths[i].Transport = carrier.(*resolvertransport.Client).Transport()
			}
		}
	}
	return paths
}

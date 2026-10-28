// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestResolverTransportParsing(t *testing.T) {
	r, err := parseResolvers("1.1.1.1,tcp://8.8.8.8,tls://dns.example:853?bootstrap=9.9.9.9,https://dns.example/dns-query?bootstrap=1.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(r.addresses) != 4 || r.streamOnly(r.addresses[0]) || !r.streamOnly(r.addresses[1]) {
		t.Fatal("incorrect transport selection")
	}
	e := r.endpoints[r.addresses[3]]
	if e.address != "1.0.0.1:443" || e.serverName != "dns.example" || e.u.RawQuery != "" {
		t.Fatalf("incorrect bootstrap/SNI: %+v", e)
	}
	for _, input := range []string{"http://example.com/dns-query", "https://u:p@example.com/dns-query", "tls://example.com?bootstrap=not-an-ip", "1.1.1.1:0", "tcp://example.com/path", "tls://1.1.1.1:0", "127.254.0.2:53,https://1.1.1.1/dns-query"} {
		if _, err := parseResolvers(input); err == nil {
			t.Errorf("accepted invalid resolver %q", input)
		}
	}
}

func TestEncryptedOnlyPolicyAndExplicitModes(t *testing.T) {
	for _, mode := range []string{"dot", "doh", "doh3", "doq"} {
		routes, err := parseResolverPolicy("1.1.1.1:53", mode, false, true)
		if err != nil {
			t.Fatal(mode, err)
		}
		endpoint := routes.endpoints[routes.addresses[0]]
		want := 443
		if mode == "dot" || mode == "doq" {
			want = 853
		}
		if !routes.strict || endpoint.scheme != mode || endpoint.port != want {
			t.Fatalf("wrong explicit mode: %+v", endpoint)
		}
		discovery := routes.constrainDiscovery(soar.Discovery{Enabled: true, Candidates: []string{"8.8.8.8:53"}, SampleRanges: 100, Cache: newResolverCache("unused", "test")})
		if discovery.Enabled || len(discovery.Candidates) != 0 || discovery.Cache != nil || discovery.SampleRanges != 0 {
			t.Fatal("plaintext discovery survived strict mode")
		}
		if _, err := routes.dial(context.Background(), "tcp", "127.0.0.1:53"); err == nil {
			t.Fatal("strict mode accepted an unknown plaintext route")
		}
	}
	for _, spec := range []string{"", "1.1.1.1:53", "udp://1.1.1.1:53", "tcp://1.1.1.1:53", "doh://dns.example/dns-query", "doq://1.1.1.1:53", "dot://1.1.1.1,8.8.8.8:53"} {
		if _, err := parseResolverPolicy(spec, "auto", true, false); err == nil {
			t.Errorf("strict policy accepted %q", spec)
		}
	}
	if _, err := parseResolverPolicy("https://1.1.1.1/dns-query", "doq", false, false); err == nil {
		t.Fatal("mismatched explicit protocol accepted")
	}
	if _, err := parseResolverPolicy("dot://dns.example?bootstrap=1.1.1.1,doh://dns.example/dns-query?bootstrap=1.0.0.1,doh3://1.1.1.1/dns-query,doq://1.1.1.1", "auto", true, false); err != nil {
		t.Fatal(err)
	}
}

func TestDoHCloseCancelsSaturatedRequests(t *testing.T) {
	var active atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		active.Add(1)
		defer active.Add(-1)
		select {
		case <-r.Context().Done():
		case <-time.After(3 * time.Second):
		}
	}))
	defer server.Close()
	c := testDoHConnection(t, server, true)
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	writer := make(chan struct{})
	go func() {
		defer close(writer)
		for i := 0; i < 40; i++ {
			if _, err := c.Write(append([]byte{0, 12}, make([]byte, 12)...)); err != nil {
				return
			}
		}
	}()
	for active.Load() < 32 {
		select {
		case <-ctx.Done():
			t.Fatalf("only %d requests started", active.Load())
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}
	c.Close()
	deadline := time.Now().Add(500 * time.Millisecond)
	for active.Load() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("close left %d HTTP requests running", active.Load())
		}
		time.Sleep(5 * time.Millisecond)
	}
	select {
	case <-writer:
	case <-ctx.Done():
		t.Fatal("DNS writer did not stop")
	}
}

func TestDoHFramingAndClosure(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Content-Type") != "application/dns-message" {
			t.Error("incorrect DoH request")
		}
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		data[2] |= 0x80
		w.Write(data)
	}))
	defer server.Close()
	c := testDoHConnection(t, server, true)
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	query := make([]byte, 24)
	query[0] = 42
	frame := binary.BigEndian.AppendUint16(nil, uint16(len(query)))
	frame = append(frame, query...)
	if _, err := c.Write(frame); err != nil {
		t.Fatal(err)
	}
	response := make([]byte, len(frame))
	if _, err := io.ReadFull(c, response); err != nil {
		t.Fatal(err)
	}
	frame[4] |= 0x80
	if string(response) != string(frame) {
		t.Fatal("DoH changed the DNS message")
	}
}

func TestDoHRejectsUntrustedTLS(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(make([]byte, 12)) }))
	defer server.Close()
	c := testDoHConnection(t, server, false)
	defer c.Close()
	c.SetDeadline(time.Now().Add(3 * time.Second))
	frame := append([]byte{0, 12}, make([]byte, 12)...)
	if _, err := c.Write(frame); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Read(make([]byte, 32)); err == nil {
		t.Fatal("untrusted DoH certificate accepted")
	}
}

func TestDiscoveryCacheIsolation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resolvers.json")
	a := newResolverCache(path, "network-a")
	a.Store([]string{"1.1.1.1:53", "127.254.0.2:53", "invalid"})
	if got := a.Load(); len(got) != 1 || got[0] != "1.1.1.1:53" {
		t.Fatalf("invalid cache content: %v", got)
	}
	if got := newResolverCache(path, "network-b").Load(); len(got) != 0 {
		t.Fatal("resolver cache leaked across networks")
	}
}

func testDoHConnection(t *testing.T, server *httptest.Server, trusted bool) net.Conn {
	t.Helper()
	routes, err := parseResolvers(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if trusted {
		routes.rootCAs = x509.NewCertPool()
		routes.rootCAs.AddCert(server.Certificate())
	}
	conn, err := routes.dial(context.Background(), "tcp", routes.addresses[0])
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

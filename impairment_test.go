// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/x509"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

func forwardDNS(ctx context.Context, target string, query []byte) ([]byte, error) {
	conn, err := (&net.Dialer{}).DialContext(ctx, "udp", target)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err = conn.Write(query); err != nil {
		return nil, err
	}
	answer := make([]byte, 4096)
	n, err := conn.Read(answer)
	return answer[:n], err
}

func impairedResolver(t *testing.T, target string, dropEvery uint64, refuseTXT bool) string {
	t.Helper()
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var workers sync.WaitGroup
	done := make(chan struct{})
	var count atomic.Uint64
	go func() {
		defer close(done)
		defer workers.Wait()
		for {
			buf := make([]byte, 4096)
			n, addr, err := pc.ReadFrom(buf)
			if err != nil {
				return
			}
			query := buf[:n]
			if dropEvery > 0 && count.Add(1)%dropEvery == 0 {
				continue
			}
			end := 12
			for end < len(query) && query[end] != 0 {
				end += int(query[end]) + 1
			}
			if end+5 > len(query) {
				continue
			}
			if refuseTXT && binary.BigEndian.Uint16(query[end+1:]) == 16 {
				answer := append([]byte(nil), query...)
				answer[2], answer[3] = 0x81, 0x85
				pc.WriteTo(answer, addr)
				continue
			}
			// Keep the path at the conservative query-name and answer budgets.
			if end-12 > 101 {
				continue
			}
			workers.Add(1)
			go func() {
				defer workers.Done()
				answer, err := forwardDNS(ctx, target, query)
				if err != nil || len(answer) > 512 {
					return
				}
				select {
				case <-time.After(10 * time.Millisecond):
				case <-ctx.Done():
					return
				}
				pc.WriteTo(answer, addr)
			}()
		}
	}()
	t.Cleanup(func() { cancel(); pc.Close(); <-done })
	return pc.LocalAddr().String()
}

func echoTunnel(t *testing.T) (*soar.Client, soar.Config) {
	return newTestTunnel(t, func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() { defer b.Close(); io.Copy(b, b) }()
		return a, nil
	})
}

func checkTunnelPayload(t *testing.T, client *soar.Client, size int, budgets ...time.Duration) {
	t.Helper()
	budget := 20 * time.Second
	if len(budgets) > 0 {
		budget = budgets[0]
	}
	ctx, cancel := context.WithTimeout(context.Background(), budget+5*time.Second)
	defer cancel()
	conn, peer := net.Pipe()
	defer peer.Close()
	go func() {
		defer peer.Close()
		handle(ctx, peer, func(ctx context.Context, _, addr string) (net.Conn, error) { return dialConfirmed(ctx, client, addr) })
	}()
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(budget))
	if _, err := conn.Write([]byte{5, 1, 0}); err != nil {
		t.Fatal(err)
	}
	var greeting [2]byte
	if _, err := io.ReadFull(conn, greeting[:]); err != nil || greeting != [2]byte{5, 0} {
		t.Fatalf("SOCKS greeting %v: %v", greeting, err)
	}
	host := "remote.example"
	request := append([]byte{5, 1, 0, 3, byte(len(host))}, []byte(host)...)
	request = binary.BigEndian.AppendUint16(request, 443)
	if _, err := conn.Write(request); err != nil {
		t.Fatal(err)
	}
	var reply [10]byte
	if _, err := io.ReadFull(conn, reply[:]); err != nil || reply[1] != 0 {
		t.Fatalf("SOCKS connect %v: %v", reply, err)
	}
	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	written := make(chan error, 1)
	go func() { _, err := conn.Write(payload); written <- err }()
	got := make([]byte, len(payload))
	if n, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("received %d/%d bytes: %v; paths: %+v", n, len(got), err, client.PathStats())
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, got) {
		t.Fatal("impaired tunnel corrupted data")
	}
}

func TestTunnelThroughImpairedResolvers(t *testing.T) {
	serverClient, config := echoTunnel(t)
	target := serverClient.PathStats()[0].Resolver
	dead, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer dead.Close()
	resolvers := []string{dead.LocalAddr().String(), impairedResolver(t, target, 7, false), impairedResolver(t, target, 9, false)}
	config.Resolvers = resolvers
	// Use the severe-restrictions profile: avoid probing above networks' hard
	// response-size limit while data is in flight.
	config.MaxAnswer = 512
	client, err := soar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	checkTunnelPayload(t, client, 8*1024)
}

func TestConservativeProfileConcurrentLoss(t *testing.T) {
	serverClient, config := echoTunnel(t)
	target := serverClient.PathStats()[0].Resolver
	config.Resolvers = []string{impairedResolver(t, target, 3, false), impairedResolver(t, target, 4, false)}
	config.MaxAnswer = 512
	config.MaxQName = 101
	config.MaxInflight = 8
	config.MaxPathWindow = 2
	config.MaxStreams = 32
	config.FECDownloadEnabled = true
	config.FECDataShards = 4
	config.FECParityShards = 2
	config.QNameLossGuard = true
	client, err := soar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	// Parallel subtests must finish before the shared client is closed.
	slots := make(chan struct{}, 2)
	t.Run("connections", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			t.Run(fmt.Sprint(i), func(t *testing.T) {
				t.Parallel()
				slots <- struct{}{}
				defer func() { <-slots }()
				checkTunnelPayload(t, client, 4096, 25*time.Second)
			})
		}
	})
}

func TestAuthenticatedTunnelThroughDoH(t *testing.T) {
	serverClient, config := echoTunnel(t)
	target := serverClient.PathStats()[0].Resolver
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query, err := io.ReadAll(io.LimitReader(r.Body, 65536))
		if err != nil {
			http.Error(w, "bad DNS request", 400)
			return
		}
		answer, err := forwardDNS(r.Context(), target, query)
		if err != nil {
			http.Error(w, "upstream timeout", 504)
			return
		}
		w.Header().Set("Content-Type", "application/dns-message")
		w.Write(answer)
	}))
	defer server.Close()
	routes, err := parseResolverPolicy(server.URL, "auto", true, false)
	if err != nil {
		t.Fatal(err)
	}
	routes.rootCAs = x509.NewCertPool()
	routes.rootCAs.AddCert(server.Certificate())
	config.Resolvers = routes.addresses
	config.StreamOnly = routes.streamOnly
	config.DialTCP = routes.dial
	client, err := soar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	checkTunnelPayload(t, client, 64*1024)
}

func TestTunnelFallsBackFromRefusedTXT(t *testing.T) {
	serverClient, config := echoTunnel(t)
	config.Resolvers = []string{impairedResolver(t, serverClient.PathStats()[0].Resolver, 0, true)}
	// The fixture drops answers above 512 bytes; keep this test focused on
	// TXT-to-AAAA fallback instead of also probing a deliberately blocked size.
	config.MaxAnswer = 512
	client, err := soar.NewClient(config)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	checkTunnelPayload(t, client, 1024)
	for _, p := range client.PathStats() {
		if p.Verified && p.RecordType == "AAAA" {
			return
		}
	}
	t.Fatal("TXT refusal did not select an authenticated AAAA path")
}

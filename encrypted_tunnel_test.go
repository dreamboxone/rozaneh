// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func TestAuthenticatedTunnelThroughEncryptedResolvers(t *testing.T) {
	for _, mode := range []string{"dot", "doh3", "doq"} {
		t.Run(mode, func(t *testing.T) {
			baseline, config := echoTunnel(t)
			target := baseline.PathStats()[0].Resolver
			certificate := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "unused certificate fixture", 400) }))
			defer certificate.Close()
			tlsConfig := certificate.TLS.Clone()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var address string
			switch mode {
			case "doh3":
				packet, err := net.ListenPacket("udp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				address = packet.LocalAddr().String()
				server := &http3.Server{TLSConfig: tlsConfig, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					query, err := io.ReadAll(io.LimitReader(r.Body, 65536))
					if err != nil {
						http.Error(w, "query", 400)
						return
					}
					answer, err := forwardDNS(r.Context(), target, query)
					if err != nil {
						http.Error(w, "upstream", 504)
						return
					}
					w.Header().Set("Content-Type", "application/dns-message")
					w.Write(answer)
				})}
				done := make(chan struct{})
				go func() { defer close(done); server.Serve(packet) }()
				defer func() { server.Close(); packet.Close(); <-done }()
			case "dot":
				listener, err := tls.Listen("tcp", "127.0.0.1:0", tlsConfig)
				if err != nil {
					t.Fatal(err)
				}
				address = listener.Addr().String()
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						conn, err := listener.Accept()
						if err != nil {
							return
						}
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer conn.Close()
							conn.SetDeadline(time.Now().Add(25 * time.Second))
							for {
								var size [2]byte
								if _, err := io.ReadFull(conn, size[:]); err != nil {
									return
								}
								query := make([]byte, int(binary.BigEndian.Uint16(size[:])))
								if _, err := io.ReadFull(conn, query); err != nil {
									return
								}
								answer, err := forwardDNS(ctx, target, query)
								if err != nil {
									return
								}
								frame := binary.BigEndian.AppendUint16(nil, uint16(len(answer)))
								if _, err := conn.Write(append(frame, answer...)); err != nil {
									return
								}
							}
						}()
					}
				}()
				defer func() { listener.Close(); cancel(); wg.Wait() }()
			case "doq":
				tlsConfig.NextProtos = []string{"doq"}
				listener, err := quic.ListenAddr("127.0.0.1:0", tlsConfig, nil)
				if err != nil {
					t.Fatal(err)
				}
				address = listener.Addr().String()
				var wg sync.WaitGroup
				wg.Add(1)
				go func() {
					defer wg.Done()
					for {
						conn, err := listener.Accept(ctx)
						if err != nil {
							return
						}
						wg.Add(1)
						go func() {
							defer wg.Done()
							defer conn.CloseWithError(0, "")
							for {
								stream, err := conn.AcceptStream(ctx)
								if err != nil {
									return
								}
								wg.Add(1)
								go func() {
									defer wg.Done()
									defer stream.Close()
									stream.SetDeadline(time.Now().Add(10 * time.Second))
									var size [2]byte
									if _, err := io.ReadFull(stream, size[:]); err != nil {
										return
									}
									query := make([]byte, int(binary.BigEndian.Uint16(size[:])))
									if _, err := io.ReadFull(stream, query); err != nil {
										return
									}
									answer, err := forwardDNS(ctx, target, query)
									if err != nil {
										return
									}
									frame := binary.BigEndian.AppendUint16(nil, uint16(len(answer)))
									stream.Write(append(frame, answer...))
								}()
							}
						}()
					}
				}()
				defer func() { cancel(); listener.Close(); wg.Wait() }()
			}
			routes, err := parseResolverPolicy(mode+"://"+address, "auto", true, false)
			if err != nil {
				t.Fatal(err)
			}
			routes.rootCAs = x509.NewCertPool()
			routes.rootCAs.AddCert(certificate.Certificate())
			config.Resolvers = routes.addresses
			config.StreamOnly = routes.streamOnly
			config.DialTCP = routes.dial
			config.Discovery = routes.constrainDiscovery(config.Discovery)
			client, err := soar.NewClient(config)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			checkTunnelPayload(t, client, 64*1024)
			paths := routes.pathStats(client)
			verified := false
			for _, path := range paths {
				if path.Verified && path.Transport == mode {
					verified = true
				}
			}
			if !verified {
				t.Fatalf("no verified %s path: %+v", mode, paths)
			}
		})
	}
}

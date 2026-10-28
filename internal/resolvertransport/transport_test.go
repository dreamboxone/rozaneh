// SPDX-License-Identifier: MIT
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package resolvertransport

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
)

func testCertificates(t *testing.T) (*tls.Config, *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "localhost"}, DNSNames: []string{"localhost"}, IPAddresses: []net.IP{net.ParseIP("127.0.0.1")}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(cert)
	return &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}, &tls.Config{RootCAs: roots, ServerName: "localhost"}
}

func query() []byte { return []byte{0x12, 0x34, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0} }
func answer(query []byte) []byte {
	result := append([]byte(nil), query...)
	result[2] |= 0x80
	return result
}
func port(address string) int {
	_, value, _ := net.SplitHostPort(address)
	number, _ := strconv.Atoi(value)
	return number
}

func TestDoH3VerifiedTLSAndNoPlaintext(t *testing.T) {
	serverTLS, clientTLS := testCertificates(t)
	packet, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var count atomic.Int32
	server := &http3.Server{TLSConfig: serverTLS, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor != 3 || r.Method != "POST" || r.Host != "resolver.example" {
			t.Error("incorrect HTTP/3 request or authority")
		}
		count.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", ContentType)
		w.Write(answer(body))
	})}
	done := make(chan struct{})
	go func() { defer close(done); server.Serve(packet) }()
	t.Cleanup(func() { server.Close(); packet.Close(); <-done })
	options := Options{TLSConfig: clientTLS, DoHPort: port(packet.LocalAddr().String()), HTTPHost: "resolver.example"}
	client, err := New("doh3", options)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := client.Exchange(ctx, "127.0.0.1:53", query())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response, answer(query())) || client.Transport() != "doh3" {
		t.Fatal("incorrect HTTP/3 exchange")
	}
	options.TLSConfig = &tls.Config{ServerName: "localhost"}
	untrusted, _ := New("doh3", options)
	defer untrusted.Close()
	if _, err := untrusted.Exchange(ctx, "127.0.0.1:53", query()); err == nil {
		t.Fatal("untrusted HTTP/3 certificate accepted")
	}
	if count.Load() != 1 {
		t.Fatal("request sent through an untrusted connection")
	}
}

func TestHTTP3FallbackIsOptInAndRemainsHTTPS(t *testing.T) {
	var count atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", ContentType)
		w.Write(answer(body))
	}))
	defer server.Close()
	u, _ := url.Parse(server.URL)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	options := Options{TLSConfig: &tls.Config{RootCAs: roots}, DoHPort: port(u.Host)}
	client, _ := New("doh3", options)
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	if _, err := client.Exchange(ctx, "127.0.0.1:53", query()); err == nil {
		t.Fatal("HTTP/3 unexpectedly succeeded")
	}
	cancel()
	if count.Load() != 0 {
		t.Fatal("HTTP/3 silently fell back")
	}
	options.HTTP3Fallback = true
	clientWithFallback, _ := New("doh3", options)
	defer clientWithFallback.Close()
	ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	response, err := clientWithFallback.Exchange(ctx, "127.0.0.1:53", query())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(response, answer(query())) || count.Load() != 1 || clientWithFallback.Transport() != "doh" {
		t.Fatal("HTTPS fallback failed")
	}
}

func serveDoQ(t *testing.T, mutate func([]byte) []byte) (string, *tls.Config, *atomic.Int32) {
	t.Helper()
	serverTLS, clientTLS := testCertificates(t)
	serverTLS.NextProtos = []string{"doq"}
	listener, err := quic.ListenAddr("127.0.0.1:0", serverTLS, nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	var count atomic.Int32
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := listener.Accept(ctx)
			if err != nil {
				return
			}
			wg.Add(1)
			go func(conn *quic.Conn) {
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
						var size [2]byte
						if _, err := io.ReadFull(stream, size[:]); err != nil {
							return
						}
						body := make([]byte, int(binary.BigEndian.Uint16(size[:])))
						if _, err := io.ReadFull(stream, body); err != nil {
							return
						}
						if len(body) < 12 {
							return
						}
						if body[0] != 0 || body[1] != 0 {
							t.Error("DoQ query ID must be zero")
						}
						var extra [1]byte
						if n, err := stream.Read(extra[:]); n != 0 || err != io.EOF {
							t.Error("DoQ query must end with FIN")
							return
						}
						count.Add(1)
						response := mutate(answer(body))
						wire := binary.BigEndian.AppendUint16(nil, uint16(len(response)))
						io.Copy(stream, bytes.NewReader(append(wire, response...)))
					}()
				}
			}(conn)
		}
	}()
	t.Cleanup(func() { cancel(); listener.Close(); wg.Wait() })
	return listener.Addr().String(), clientTLS, &count
}

func TestDoQIDMappingMultiplexingAndTLS(t *testing.T) {
	address, tlsConfig, count := serveDoQ(t, func(value []byte) []byte { return value })
	client, err := New("doq", Options{TLSConfig: tlsConfig, DoQPort: port(address)})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			q := query()
			q[1] = byte(i + 1)
			result, err := client.Exchange(ctx, "127.0.0.1:53", q)
			if err != nil {
				t.Error(err)
				return
			}
			if !bytes.Equal(result, answer(q)) {
				t.Error("DoQ did not restore query ID")
			}
		}(i)
	}
	wg.Wait()
	if count.Load() != 8 {
		t.Fatal("queries lost")
	}
	untrusted, _ := New("doq", Options{DoQPort: port(address)})
	defer untrusted.Close()
	if _, err := untrusted.Exchange(ctx, "127.0.0.1:53", query()); err == nil {
		t.Fatal("untrusted DoQ certificate accepted")
	}
}

func TestDoQRejectsNonzeroResponseID(t *testing.T) {
	address, tlsConfig, _ := serveDoQ(t, func(value []byte) []byte { value[0] = 1; return value })
	client, _ := New("doq", Options{TLSConfig: tlsConfig, DoQPort: port(address)})
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := client.Exchange(ctx, "127.0.0.1:53", query()); err == nil {
		t.Fatal("invalid DoQ ID accepted")
	}
}

func TestEncryptedClientRejectsInsecureTLSAndPlaintextURL(t *testing.T) {
	if _, err := New("doq", Options{DoQPort: 53}); err == nil {
		t.Fatal("DoQ port 53 accepted")
	}
	if _, err := New("doh", Options{TLSConfig: &tls.Config{InsecureSkipVerify: true}}); err == nil {
		t.Fatal("TLS verification can be disabled")
	}
	if _, err := ExchangeHTTP(context.Background(), http.DefaultClient, "http://127.0.0.1/dns-query", query()); err == nil {
		t.Fatal("plaintext DoH accepted")
	}
}

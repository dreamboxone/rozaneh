// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Run explicitly with ROZANEH_TEST_ROUTER=user@host and a staged binary at
// /tmp/rozaneh-review/rozaneh. The configured router service is left untouched.
func TestRouterSOCKSIntegration(t *testing.T) {
	router := os.Getenv("ROZANEH_TEST_ROUTER")
	if router == "" {
		t.Skip("ROZANEH_TEST_ROUTER is unset")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
	defer cancel()
	remote := func(script string) string {
		t.Helper()
		out, err := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=5", router, script).CombinedOutput()
		if err != nil {
			t.Fatalf("router command: %v\n%s", err, out)
		}
		return string(out)
	}
	payload := make([]byte, 3*1024*1024)
	if _, err := rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/upload":
			hash := sha256.New()
			n, err := io.Copy(hash, io.LimitReader(r.Body, 4*1024*1024))
			if err != nil || n != int64(len(payload)) {
				http.Error(w, "incomplete upload", 400)
				return
			}
			fmt.Fprint(w, hex.EncodeToString(hash.Sum(nil)))
		case "/download":
			w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
			w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer func() { cancel(); server.CloseClientConnections(); server.Close() }()
	_, config := newTestTunnel(t, func(ctx context.Context, network, addr string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, addr)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				stop := context.AfterFunc(ctx, func() { conn.Close() })
				defer stop()
				var writes sync.Mutex
				for {
					var prefix [2]byte
					if _, err := io.ReadFull(conn, prefix[:]); err != nil {
						return
					}
					query := make([]byte, binary.BigEndian.Uint16(prefix[:]))
					if _, err := io.ReadFull(conn, query); err != nil {
						return
					}
					go func() {
						answer, err := forwardDNS(ctx, config.Resolvers[0], query)
						if err != nil {
							return
						}
						frame := append(binary.BigEndian.AppendUint16(nil, uint16(len(answer))), answer...)
						writes.Lock()
						defer writes.Unlock()
						conn.Write(frame)
					}()
				}
			}()
		}
	}()
	var sshErrors bytes.Buffer
	forward := exec.CommandContext(ctx, "ssh", "-o", "BatchMode=yes", "-o", "ExitOnForwardFailure=yes", "-R", "127.0.0.1:15353:"+listener.Addr().String(), router, "echo ready; cat")
	forward.Stderr = &sshErrors
	stdin, err := forward.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := forward.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := forward.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { stdin.Close(); forward.Process.Kill(); forward.Wait() }()
	ready := make(chan bool, 1)
	go func() { scanner := bufio.NewScanner(stdout); ready <- scanner.Scan() && scanner.Text() == "ready" }()
	select {
	case ok := <-ready:
		if !ok {
			t.Fatal("SSH reverse forwarding did not start")
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	remote("test -x /tmp/rozaneh-review/rozaneh; test ! -e /tmp/rozaneh-review/local.pid")
	remote(fmt.Sprintf("start-stop-daemon -S -b -m -p /tmp/rozaneh-review/local.pid -c rozaneh:rozaneh -x /tmp/rozaneh-review/rozaneh -- -zone t.example -pubkey %s -resolvers tcp://127.0.0.1:15353 -listen 127.0.0.1:1090 -stats-file /tmp/rozaneh-review/local-traffic.json -usage-file /tmp/rozaneh-review/local-usage.json -diagnostics-file /tmp/rozaneh-review/local-paths.json; sleep 1", base64.StdEncoding.EncodeToString(config.ServerKey)))
	defer func() {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 10*time.Second)
		defer stop()
		out, err := exec.CommandContext(cleanupCtx, "ssh", "-o", "BatchMode=yes", router, "start-stop-daemon -K -p /tmp/rozaneh-review/local.pid -x /tmp/rozaneh-review/rozaneh >/dev/null 2>&1 || true; rm -f /tmp/rozaneh-review/local.pid /tmp/rozaneh-review/local-upload.bin /tmp/rozaneh-review/local-download.bin /tmp/rozaneh-review/local-traffic.json /tmp/rozaneh-review/local-usage.json /tmp/rozaneh-review/local-paths.json").CombinedOutput()
		if err != nil {
			t.Errorf("router cleanup: %v: %s", err, out)
		}
	}()
	result := remote(fmt.Sprintf(`set -e
dd if=/dev/urandom of=/tmp/rozaneh-review/local-upload.bin bs=1024 count=3072 2>/dev/null
expected=$(sha256sum /tmp/rozaneh-review/local-upload.bin | awk '{print $1}')
actual=$(curl --fail --silent --show-error --socks5-hostname 127.0.0.1:1090 --connect-timeout 10 --max-time 110 --data-binary @/tmp/rozaneh-review/local-upload.bin %s/upload)
[ "$actual" = "$expected" ] || { echo 'Upload checksum mismatch'; exit 1; }
echo '3 MiB upload checksum verified'
curl --fail --silent --show-error --socks5-hostname 127.0.0.1:1090 --connect-timeout 10 --max-time 110 %s/download -o /tmp/rozaneh-review/local-download.bin
actual=$(sha256sum /tmp/rozaneh-review/local-download.bin | awk '{print $1}')
[ "$actual" = '%s' ] || { echo 'Download checksum mismatch'; exit 1; }
echo '3 MiB download checksum verified'
code=0
curl --silent --show-error --output /dev/null --socks5-hostname 127.0.0.1:1090 --connect-timeout 5 --max-time 10 http://127.0.0.1:1/ 2>/dev/null || code=$?
[ "$code" = 97 ] || { echo "Wrong SOCKS refusal status: $code"; exit 1; }
echo 'Failed destination rejected during SOCKS handshake'
`, server.URL, server.URL, hex.EncodeToString(digest[:])))
	if !strings.Contains(result, "download checksum verified") {
		t.Fatal(result)
	}
	t.Log(strings.TrimSpace(result))
}

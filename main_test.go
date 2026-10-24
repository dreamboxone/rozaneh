package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/getlantern/soar"
)

func TestSOCKSThroughSoar(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()
	requested := make(chan string, 1)
	dial := func(_ context.Context, network, addr string) (net.Conn, error) {
		requested <- addr
		a, b := net.Pipe()
		go func() { defer b.Close(); io.Copy(b, b) }()
		return a, nil
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- soar.Serve(ctx, pc, soar.ServerConfig{Zone: "t.example", PrivateKey: priv, Dial: dial})
	}()
	defer func() {
		cancel()
		pc.Close()
		select {
		case <-serverDone:
		case <-time.After(3 * time.Second):
			t.Error("server did not stop")
		}
	}()
	cl, err := soar.NewClient(soar.Config{Zone: "t.example", ServerKey: pub, Resolvers: []string{pc.LocalAddr().String()}, TCP: "off"})
	if err != nil {
		t.Fatal(err)
	}
	defer cl.Close()
	stateDir := t.TempDir()
	meter := newTrafficMeter(filepath.Join(stateDir, "live.json"), filepath.Join(stateDir, "saved.json"))
	a, b := net.Pipe()
	defer a.Close()
	go func() {
		defer b.Close()
		handle(ctx, b, func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := cl.DialContext(ctx, network, addr)
			if err != nil {
				return nil, err
			}
			return &meteredConn{Conn: conn, m: meter}, nil
		})
	}()
	a.SetDeadline(time.Now().Add(25 * time.Second))
	if _, err = a.Write([]byte{5, 1, 0}); err != nil {
		t.Fatal(err)
	}
	var greet [2]byte
	if _, err = io.ReadFull(a, greet[:]); err != nil {
		t.Fatal(err)
	}
	if greet != [2]byte{5, 0} {
		t.Fatalf("greeting %v", greet)
	}
	hostname := "remote.example"
	req := append([]byte{5, 1, 0, 3, byte(len(hostname))}, []byte(hostname)...)
	req = binary.BigEndian.AppendUint16(req, 443)
	if _, err = a.Write(req); err != nil {
		t.Fatal(err)
	}
	var reply [10]byte
	if _, err = io.ReadFull(a, reply[:]); err != nil {
		t.Fatal(err)
	}
	if reply[1] != 0 {
		t.Fatalf("reply %v", reply)
	}
	select {
	case dest := <-requested:
		if dest != "remote.example:443" {
			t.Fatalf("destination %q", dest)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	payload := make([]byte, 128*1024)
	if _, err = rand.Read(payload); err != nil {
		t.Fatal(err)
	}
	sent := make(chan error, 1)
	go func() { _, e := io.Copy(a, bytes.NewReader(payload)); sent <- e }()
	received := make([]byte, len(payload))
	if _, err = io.ReadFull(a, received); err != nil {
		t.Fatal(err)
	}
	if err = <-sent; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, received) {
		t.Fatal("tunnel corrupted payload")
	}
	meter.close()
	measured := meter.snapshot().Today
	if measured.Up != uint64(len(payload)) || measured.Down != uint64(len(payload)) {
		t.Fatalf("wrong traffic accounting: %+v", measured)
	}
	reloaded := newTrafficMeter(filepath.Join(stateDir, "live.json"), filepath.Join(stateDir, "saved.json"))
	defer reloaded.close()
	if reloaded.snapshot().Today != measured {
		t.Fatal("traffic accounting did not survive restart")
	}
	t.Log("128 KiB roundtrip through SOCKS5 and authenticated Soar DNS tunnel; destination hostname and traffic accounting verified")
}

// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

func testTunnel(t *testing.T, dial dialFunc) *soar.Client {
	t.Helper()
	client, _ := newTestTunnel(t, dial)
	return client
}

// Repeated short-lived connections must not exhaust a session's stream slots.
func TestStreamDialerSustainedChurn(t *testing.T) {
	client := testTunnel(t, func(ctx context.Context, network, address string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() { defer b.Close(); io.Copy(b, b) }()
		return a, nil
	})
	dial := newStreamDialer(client, newTrafficMeter("", ""), 16)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	for i := 0; i < 400; i++ {
		conn, err := dial(ctx, "tcp", "echo.example:443")
		if err != nil {
			t.Fatalf("connection %d: %v", i, err)
		}
		conn.SetDeadline(time.Now().Add(5 * time.Second))
		if _, err = conn.Write([]byte{42}); err != nil {
			conn.Close()
			t.Fatal(err)
		}
		var response [1]byte
		_, err = io.ReadFull(conn, response[:])
		conn.Close()
		if err != nil || response[0] != 42 {
			t.Fatalf("connection %d failed: %v", i, err)
		}
	}
}

func newTestTunnel(t *testing.T, dial dialFunc) (*soar.Client, soar.Config) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- soar.Serve(ctx, pc, soar.ServerConfig{Zone: "t.example", PrivateKey: priv, Dial: dial})
	}()
	config := soar.Config{Zone: "t.example", ServerKey: pub, Resolvers: []string{pc.LocalAddr().String()}, TCP: "off"}
	client, err := soar.NewClient(config)
	if err != nil {
		cancel()
		pc.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		client.Close()
		cancel()
		pc.Close()
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Error("server did not stop")
		}
	})
	return client, config
}

func TestConfirmedTunnelWaitsForDestination(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	client := testTunnel(t, func(ctx context.Context, _, _ string) (net.Conn, error) {
		close(started)
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		a, b := net.Pipe()
		go func() { defer b.Close(); b.Write([]byte("ready")) }()
		return a, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	type result struct {
		conn net.Conn
		err  error
	}
	done := make(chan result, 1)
	go func() { c, err := dialConfirmed(ctx, client, "remote.example:443"); done <- result{c, err} }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	select {
	case r := <-done:
		if r.conn != nil {
			r.conn.Close()
		}
		close(release)
		t.Fatalf("dial returned before destination connected: %v", r.err)
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		defer r.conn.Close()
		r.conn.SetReadDeadline(time.Now().Add(time.Second))
		data := make([]byte, 5)
		if _, err := io.ReadFull(r.conn, data); err != nil || string(data) != "ready" {
			t.Fatalf("server greeting %q: %v", data, err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestConfirmedTunnelRejectsFailedDestination(t *testing.T) {
	client := testTunnel(t, func(context.Context, string, string) (net.Conn, error) { return nil, errors.New("destination refused") })
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, err := dialConfirmed(ctx, client, "remote.example:443")
	if c != nil {
		c.Close()
	}
	if err == nil {
		t.Fatal("failed destination reported as connected")
	}
}

func TestConfirmedTunnelDestinationTimeout(t *testing.T) {
	client := testTunnel(t, func(ctx context.Context, _, _ string) (net.Conn, error) { <-ctx.Done(); return nil, ctx.Err() })
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()
	c, err := dialConfirmed(ctx, client, "remote.example:443")
	if c != nil {
		c.Close()
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected destination timeout, got %v", err)
	}
}

func TestStreamDialerWaitsForCapacity(t *testing.T) {
	client := testTunnel(t, func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() { defer b.Close(); io.Copy(b, b) }()
		return a, nil
	})
	meter := newTrafficMeter("", "")
	defer meter.close()
	dial := newStreamDialer(client, meter, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := dial(ctx, "tcp", "remote.example:443")
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	waitCtx, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop()
	if c, err := dial(waitCtx, "tcp", "other.example:443"); !errors.Is(err, context.DeadlineExceeded) {
		if c != nil {
			c.Close()
		}
		t.Fatalf("capacity did not wait for caller budget: %v", err)
	}
	first.Close()
	first.Close()
	next, err := dial(ctx, "tcp", "other.example:443")
	if err != nil {
		t.Fatal(err)
	}
	defer next.Close()
	next.SetDeadline(time.Now().Add(time.Second))
	if _, err := next.Write([]byte("echo")); err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4)
	if _, err := io.ReadFull(next, data); err != nil || string(data) != "echo" {
		t.Fatalf("replacement stream %q: %v", data, err)
	}
}

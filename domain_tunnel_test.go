// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
package main

import (
	"context"
	"io"
	"net"
	"testing"
	"time"
)

func TestDomainFallbackUsesAuthenticatedStandby(t *testing.T) {
	_, cfg := newTestTunnel(t, func(context.Context, string, string) (net.Conn, error) {
		a, b := net.Pipe()
		go func() { defer b.Close(); io.Copy(b, b) }()
		return a, nil
	})
	cfg.Zone = "missing.example"
	client, err := newDomainTunnel(cfg, []string{"t.example"}, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	if client.clients[1] != nil {
		t.Fatal("standby started eagerly")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	conn, err := client.DialConfirmed(ctx, "tcp", "echo.example:443")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(time.Second))
	if _, err = conn.Write([]byte{42}); err != nil {
		t.Fatal(err)
	}
	var result [1]byte
	if _, err = io.ReadFull(conn, result[:]); err != nil || result[0] != 42 {
		t.Fatalf("standby data: %v", err)
	}
	if client.preferred != 1 {
		t.Fatal("successful standby not preferred")
	}
}
func TestDomainValidation(t *testing.T) {
	for _, domain := range []string{"", "a..example", "-bad.example", "bad/example", "دامنه.example"} {
		_, cfg := echoTunnel(t)
		cfg.Zone = domain
		if c, err := newDomainTunnel(cfg, nil, time.Second); err == nil {
			c.Close()
			t.Fatalf("accepted %q", domain)
		}
	}
}

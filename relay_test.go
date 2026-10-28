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

func TestRelayCancellation(t *testing.T) {
	client, local := net.Pipe()
	upstream, remote := net.Pipe()
	defer client.Close()
	defer remote.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { relay(ctx, local, upstream, time.Minute); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("relay did not close on cancellation")
	}
}

func TestRelayIdleExpiry(t *testing.T) {
	client, local := net.Pipe()
	upstream, remote := net.Pipe()
	defer client.Close()
	defer remote.Close()
	done := make(chan struct{})
	go func() { relay(context.Background(), local, upstream, 100*time.Millisecond); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("idle relay kept its connection slot")
	}
}

func TestRelayOneWayActivity(t *testing.T) {
	client, local := net.Pipe()
	upstream, remote := net.Pipe()
	defer client.Close()
	defer remote.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { relay(ctx, local, upstream, 500*time.Millisecond); close(done) }()
	go io.Copy(io.Discard, remote)
	for i := 0; i < 8; i++ {
		client.SetWriteDeadline(time.Now().Add(time.Second))
		if _, err := client.Write([]byte{1}); err != nil {
			t.Fatalf("active one-way transfer closed: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("active relay did not stop")
	}
}

func TestRelayResponseAfterClientHalfClose(t *testing.T) {
	pair := func() (*net.TCPConn, *net.TCPConn) {
		listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		sender, err := net.DialTCP("tcp", nil, listener.Addr().(*net.TCPAddr))
		if err != nil {
			t.Fatal(err)
		}
		receiver, err := listener.AcceptTCP()
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { sender.Close(); receiver.Close() })
		return sender, receiver
	}
	client, local := pair()
	upstream, remote := pair()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { relay(ctx, local, upstream, time.Second); close(done) }()
	client.SetDeadline(time.Now().Add(3 * time.Second))
	remote.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := client.Write([]byte("request")); err != nil {
		t.Fatal(err)
	}
	client.CloseWrite()
	request, err := io.ReadAll(remote)
	if err != nil || string(request) != "request" {
		t.Fatalf("request %q: %v", request, err)
	}
	if _, err := remote.Write([]byte("response after EOF")); err != nil {
		t.Fatal(err)
	}
	remote.CloseWrite()
	response, err := io.ReadAll(client)
	if err != nil || string(response) != "response after EOF" {
		t.Fatalf("response %q: %v", response, err)
	}
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("relay did not finish")
	}
}

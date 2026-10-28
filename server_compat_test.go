// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

func TestPreviousServerCompatibility(t *testing.T) {
	server := os.Getenv("ROZANEH_BASELINE_SERVER")
	if server == "" {
		t.Skip("set ROZANEH_BASELINE_SERVER to test the previous engine's server")
	}
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	keyFile := filepath.Join(t.TempDir(), "server.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(key.Seed())), 0600); err != nil {
		t.Fatal(err)
	}
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	resolver := udp.LocalAddr().String()
	udp.Close()
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		conn, err := echo.Accept()
		if err == nil {
			defer conn.Close()
			io.Copy(conn, conn)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, server, "-zone", "compat.example", "-key-file", keyFile, "-listen", resolver, "-allow-private", "-allow-ports", "any")
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { command.Process.Kill(); command.Wait() }()
	client, err := soar.NewClient(soar.Config{Zone: "compat.example", ServerKey: pub, Resolvers: []string{resolver}, TCP: "off"})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	conn, err := dialConfirmed(ctx, client, echo.Addr().String())
	if err != nil {
		t.Fatalf("new client cannot connect through previous server: %v", err)
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(15 * time.Second))
	payload := bytes.Repeat([]byte("compatible tunnel\n"), 4096)
	written := make(chan error, 1)
	go func() { _, err := conn.Write(payload); written <- err }()
	response := make([]byte, len(payload))
	if _, err := io.ReadFull(conn, response); err != nil {
		t.Fatal(err)
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(payload, response) {
		t.Fatal("previous-server round trip corrupted data")
	}
}

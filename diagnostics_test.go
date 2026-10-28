// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiagnosticsPublishImmediatelyAndClearOnStop(t *testing.T) {
	client := testTunnel(t, func(context.Context, string, string) (net.Conn, error) { return nil, net.ErrClosed })
	routes, err := parseResolvers("1.1.1.1")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "paths.json")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { recordDiagnostics(ctx, path, client, routes); close(done) }()
	deadline := time.Now().Add(time.Second)
	var snapshot struct {
		Updated     int64             `json:"updated"`
		Established bool              `json:"established"`
		Paths       []json.RawMessage `json:"paths"`
	}
	for {
		data, err := os.ReadFile(path)
		if err == nil && json.Unmarshal(data, &snapshot) == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("no initial diagnostics within one second")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if snapshot.Updated < time.Now().Add(-time.Second).Unix() {
		t.Fatal("diagnostics have no fresh timestamp")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("diagnostics did not stop")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Established || len(snapshot.Paths) != 0 {
		t.Fatal("stopped diagnostics retained live paths")
	}
	history, err := os.ReadFile(filepath.Join(filepath.Dir(path), "diagnostics.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var events []map[string]json.RawMessage
	for _, line := range bytes.Split(bytes.TrimSpace(history), []byte("\n")) {
		var event map[string]json.RawMessage
		if err := json.Unmarshal(line, &event); err != nil {
			t.Fatal(err)
		}
		if len(event["paths"]) == 0 || len(event["heap_bytes"]) == 0 || len(event["health"]) == 0 {
			t.Fatal("incomplete diagnostic sample")
		}
		events = append(events, event)
	}
	if len(events) < 2 || string(events[len(events)-1]["event"]) != `"stopped"` {
		t.Fatal("missing retained stop event")
	}
}

// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

func recordDiagnostics(ctx context.Context, path string, client tunnelStatus, routes *resolverTransports) {
	history, historyErr := os.OpenFile(filepath.Join(filepath.Dir(path), "diagnostics.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if historyErr != nil {
		log.Printf("diagnostic history: %v", historyErr)
	} else {
		defer history.Close()
	}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	first := true
	publish := func(stopped bool) {
		snapshot := struct {
			Updated     int64           `json:"updated"`
			Established bool            `json:"established"`
			Paths       []soar.PathStat `json:"paths"`
		}{Updated: time.Now().Unix(), Paths: []soar.PathStat{}}
		if !stopped {
			snapshot.Established = client.Established()
			snapshot.Paths = routes.pathStats(client)
		}
		data, err := json.Marshal(snapshot)
		if err == nil {
			err = os.WriteFile(path+".tmp", data, 0600)
		}
		if err == nil {
			err = os.Rename(path+".tmp", path)
		}
		if err != nil {
			log.Printf("resolver diagnostics: %v", err)
		}
		if history != nil {
			var memory runtime.MemStats
			runtime.ReadMemStats(&memory)
			health, _ := os.ReadFile(filepath.Join(filepath.Dir(path), "health.json"))
			if !json.Valid(health) {
				health = []byte("null")
			}
			entry := struct {
				Time       string          `json:"time"`
				Event      string          `json:"event"`
				Paths      json.RawMessage `json:"paths"`
				Health     json.RawMessage `json:"health"`
				Heap       uint64          `json:"heap_bytes"`
				Goroutines int             `json:"goroutines"`
				PID        int             `json:"pid"`
			}{time.Now().UTC().Format(time.RFC3339Nano), "sample", data, health, memory.HeapAlloc, runtime.NumGoroutine(), os.Getpid()}
			if first {
				entry.Event = "started"
				first = false
			}
			if stopped {
				entry.Event = "stopped"
			}
			if !json.Valid(data) {
				entry.Paths = []byte("null")
			}
			if err := json.NewEncoder(history).Encode(entry); err != nil {
				log.Printf("diagnostic history: %v", err)
			}
		}
	}
	publish(false)
	for {
		select {
		case <-ctx.Done():
			publish(true)
			return
		case <-ticker.C:
			publish(false)
		}
	}
}

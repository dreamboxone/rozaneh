// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type usage struct {
	Up   uint64 `json:"up"`
	Down uint64 `json:"down"`
}
type trafficSnapshot struct {
	Days    map[string]usage `json:"days"`
	Today   usage            `json:"today"`
	Week    usage            `json:"week"`
	Month   usage            `json:"month"`
	Updated int64            `json:"updated"`
}
type trafficMeter struct {
	mu               sync.Mutex
	days             map[string]usage
	live, persistent string
	stop, done       chan struct{}
}

func newTrafficMeter(live, persistent string) *trafficMeter {
	m := &trafficMeter{days: make(map[string]usage), live: live, persistent: persistent, stop: make(chan struct{}), done: make(chan struct{})}
	var latest trafficSnapshot
	for _, p := range []string{persistent, live} {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var s trafficSnapshot
		if json.Unmarshal(data, &s) == nil && s.Updated >= latest.Updated && s.Days != nil {
			latest = s
		}
	}
	if latest.Days != nil {
		m.days = latest.Days
	}
	go func() {
		defer close(m.done)
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		lastPersist := time.Now()
		for {
			select {
			case <-ticker.C:
				m.write(live)
				if time.Since(lastPersist) >= 5*time.Minute {
					m.write(persistent)
					lastPersist = time.Now()
				}
			case <-m.stop:
				m.write(live)
				m.write(persistent)
				return
			}
		}
	}()
	return m
}
func (m *trafficMeter) close() { close(m.stop); <-m.done }
func (m *trafficMeter) add(up, down uint64) {
	day := time.Now().UTC().Format("2006-01-02")
	m.mu.Lock()
	u := m.days[day]
	u.Up += up
	u.Down += down
	m.days[day] = u
	m.mu.Unlock()
}
func (m *trafficMeter) snapshot() trafficSnapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now().UTC()
	today := now.Format("2006-01-02")
	s := trafficSnapshot{Days: make(map[string]usage), Updated: now.Unix()}
	for d, u := range m.days {
		date, err := time.Parse("2006-01-02", d)
		if err != nil {
			continue
		}
		age := int(now.Sub(date).Hours() / 24)
		if age < 0 || age >= 30 {
			continue
		}
		s.Days[d] = u
		if d == today {
			s.Today = u
		}
		if age < 7 {
			s.Week.Up += u.Up
			s.Week.Down += u.Down
		}
		s.Month.Up += u.Up
		s.Month.Down += u.Down
	}
	return s
}
func (m *trafficMeter) write(path string) {
	if path == "" {
		return
	}
	snapshot := m.snapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		return
	}
	if old, err := os.ReadFile(path); err == nil {
		var prior trafficSnapshot
		if json.Unmarshal(old, &prior) == nil && prior.Today == snapshot.Today && prior.Week == snapshot.Week && prior.Month == snapshot.Month && len(prior.Days) == len(snapshot.Days) {
			return
		}
	}
	if os.MkdirAll(filepath.Dir(path), 0700) != nil {
		return
	}
	temp := path + ".tmp"
	if os.WriteFile(temp, data, 0600) == nil {
		os.Rename(temp, path)
	}
}

type meteredConn struct {
	net.Conn
	m *trafficMeter
}

func (c *meteredConn) Read(p []byte) (int, error) {
	n, e := c.Conn.Read(p)
	if n > 0 {
		c.m.add(0, uint64(n))
	}
	return n, e
}
func (c *meteredConn) Write(p []byte) (int, error) {
	n, e := c.Conn.Write(p)
	if n > 0 {
		c.m.add(uint64(n), 0)
	}
	return n, e
}
func (c *meteredConn) CloseWrite() error {
	if half, ok := c.Conn.(interface{ CloseWrite() error }); ok {
		return half.CloseWrite()
	}
	return c.Conn.Close()
}

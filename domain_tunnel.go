// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
package main

import (
	"context"
	"errors"
	"fmt"
	soar "github.com/dreamboxone/rozaneh/internal/tunnelengine"
	"net"
	"strings"
	"sync"
	"time"
)

type tunnelStatus interface {
	PathStats() []soar.PathStat
	Established() bool
}
type domainTunnel struct {
	mu        sync.Mutex
	cfg       soar.Config
	domains   []string
	clients   []*soar.Client
	preferred int
	after     time.Duration
	closed    bool
}

func newDomainTunnel(cfg soar.Config, standbys []string, after time.Duration) (*domainTunnel, error) {
	if after < time.Second || after > time.Minute {
		return nil, errors.New("domain failover delay must be 1s-1m")
	}
	domains := []string{}
	seen := map[string]bool{}
	for _, raw := range append([]string{cfg.Zone}, standbys...) {
		domain := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
		if len(domain) == 0 || len(domain) > 253 {
			return nil, errors.New("invalid tunnel domain")
		}
		for _, label := range strings.Split(domain, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return nil, errors.New("invalid tunnel domain label")
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
					return nil, errors.New("tunnel domains must use ASCII DNS names")
				}
			}
		}
		if !seen[domain] {
			domains = append(domains, domain)
			seen[domain] = true
		}
	}
	if len(domains) > 8 {
		return nil, errors.New("at most 8 tunnel domains are supported")
	}
	cfg.Zone = domains[0]
	primary, err := soar.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	clients := make([]*soar.Client, len(domains))
	clients[0] = primary
	return &domainTunnel{cfg: cfg, domains: domains, clients: clients, after: after}, nil
}

func (d *domainTunnel) DialConfirmed(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	first := d.preferred
	d.mu.Unlock()
	var failures []error
	for step := 0; step < len(d.domains); step++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		index := (first + step) % len(d.domains)
		d.mu.Lock()
		if d.closed {
			d.mu.Unlock()
			return nil, net.ErrClosed
		}
		client := d.clients[index]
		if client == nil {
			cfg := d.cfg
			cfg.Zone = d.domains[index]
			cfg.Discovery.Cache = nil
			var err error
			client, err = soar.NewClient(cfg)
			if err != nil {
				d.mu.Unlock()
				failures = append(failures, err)
				continue
			}
			d.clients[index] = client
		}
		d.mu.Unlock()
		attempt, cancel := context.WithCancel(ctx)
		if len(d.domains) > 1 {
			cancel()
			attempt, cancel = context.WithTimeout(ctx, d.after)
		}
		conn, err := client.DialConfirmed(attempt, network, address)
		cancel()
		if err != nil && len(d.domains) == 1 {
			return nil, err
		}
		if err == nil {
			d.mu.Lock()
			closed := d.closed
			if !closed {
				d.preferred = index
			}
			d.mu.Unlock()
			if closed {
				conn.Close()
				return nil, net.ErrClosed
			}
			return conn, nil
		}
		failures = append(failures, fmt.Errorf("%s: %w", d.domains[index], err))
	}
	return nil, fmt.Errorf("all tunnel domains failed: %w", errors.Join(failures...))
}
func (d *domainTunnel) PathStats() []soar.PathStat {
	d.mu.Lock()
	c := d.clients[d.preferred]
	d.mu.Unlock()
	return c.PathStats()
}
func (d *domainTunnel) Established() bool {
	d.mu.Lock()
	c := d.clients[d.preferred]
	d.mu.Unlock()
	return c.Established()
}
func (d *domainTunnel) Close() error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	clients := append([]*soar.Client(nil), d.clients...)
	d.mu.Unlock()
	for _, c := range clients {
		if c != nil {
			c.Close()
		}
	}
	return nil
}

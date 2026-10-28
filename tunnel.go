// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"net"
	"sync"
	"time"
)

type confirmedTunnel interface {
	DialConfirmed(context.Context, string, string) (net.Conn, error)
}

func dialConfirmed(ctx context.Context, client confirmedTunnel, addr string) (net.Conn, error) {
	return client.DialConfirmed(ctx, "tcp", addr)
}

func newStreamDialer(client confirmedTunnel, meter *trafficMeter, limit int) dialFunc {
	slots := make(chan struct{}, limit)
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		select {
		case slots <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		for {
			conn, err := dialConfirmed(ctx, client, addr)
			if err == nil {
				return &streamSlotConn{Conn: &meteredConn{Conn: conn, m: meter}, release: func() { <-slots }}, nil
			}
			// Closed streams may await their final acknowledgement before the engine
			// removes them. Wait within the caller's connect budget for that capacity.
			if err.Error() != "soar: too many streams" {
				<-slots
				return nil, err
			}
			timer := time.NewTimer(25 * time.Millisecond)
			select {
			case <-timer.C:
			case <-ctx.Done():
				timer.Stop()
				<-slots
				return nil, ctx.Err()
			}
		}
	}
}

type streamSlotConn struct {
	net.Conn
	once    sync.Once
	release func()
}

func (c *streamSlotConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(c.release)
	return err
}

func (c *streamSlotConn) CloseWrite() error {
	return c.Conn.(interface{ CloseWrite() error }).CloseWrite()
}

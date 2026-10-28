// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type idleTimeoutKey struct{}
type tunnelDestinationKey struct{}

type activeConn struct {
	net.Conn
	last *atomic.Int64
}

func (c activeConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	if n > 0 {
		c.last.Store(time.Now().UnixNano())
	}
	return n, err
}

func (c activeConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	if n > 0 {
		c.last.Store(time.Now().UnixNano())
	}
	return n, err
}

// Both directions share one activity clock so an active upload keeps its download side open.
func relay(ctx context.Context, client, upstream net.Conn, idle time.Duration) {
	destination, _ := ctx.Value(tunnelDestinationKey{}).(string)
	var last atomic.Int64
	last.Store(time.Now().UnixNano())
	finished := make(chan struct{})
	stopped := make(chan struct{})
	go func() {
		defer close(stopped)
		interval := idle / 4
		if interval > 15*time.Second {
			interval = 15 * time.Second
		}
		timer := time.NewTicker(interval)
		defer timer.Stop()
		for {
			select {
			case <-finished:
				return
			case <-ctx.Done():
				client.Close()
				upstream.Close()
				return
			case <-timer.C:
				if time.Since(time.Unix(0, last.Load())) >= idle {
					log.Printf("tunnel idle timeout: destination=%s idle=%s", destination, idle)
					client.Close()
					upstream.Close()
					return
				}
			}
		}
	}()
	copySide := func(dst, src net.Conn, direction string) {
		n, err := io.Copy(activeConn{dst, &last}, activeConn{src, &last})
		if err != nil && !errors.Is(err, net.ErrClosed) && ctx.Err() == nil {
			log.Printf("tunnel transfer: destination=%s direction=%s bytes=%d error=%v", destination, direction, n, err)
		}
		if err == nil {
			if half, ok := dst.(interface{ CloseWrite() error }); ok {
				if half.CloseWrite() == nil {
					return
				}
			}
		}
		client.Close()
		upstream.Close()
	}
	done := make(chan struct{})
	go func() { copySide(upstream, client, "upload"); close(done) }()
	copySide(client, upstream, "download")
	<-done
	client.Close()
	upstream.Close()
	close(finished)
	<-stopped
}

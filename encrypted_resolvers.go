// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"sync"
	"time"

	"github.com/dreamboxone/rozaneh/internal/resolvertransport"
)

// Adapt Soar's persistent framed stream to independent encrypted DNS exchanges.
func newEncryptedDNSConn(endpoint resolverEndpoint, carrier *resolvertransport.Client) net.Conn {
	client, bridge := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		defer bridge.Close()
		defer carrier.Close()
		var writes sync.Mutex
		slots := make(chan struct{}, 32)
		for {
			var prefix [2]byte
			if _, err := io.ReadFull(bridge, prefix[:]); err != nil {
				return
			}
			size := int(binary.BigEndian.Uint16(prefix[:]))
			if size < 12 {
				return
			}
			query := make([]byte, size)
			if _, err := io.ReadFull(bridge, query); err != nil {
				return
			}
			select {
			case slots <- struct{}{}:
			case <-ctx.Done():
				return
			}
			go func(query []byte) {
				defer func() { <-slots }()
				request, cancel := context.WithTimeout(ctx, 12*time.Second)
				defer cancel()
				answer, err := carrier.Exchange(request, endpoint.address, query)
				if err != nil {
					bridge.Close()
					return
				}
				response := binary.BigEndian.AppendUint16(nil, uint16(len(answer)))
				response = append(response, answer...)
				writes.Lock()
				defer writes.Unlock()
				if _, err = bridge.Write(response); err != nil {
					bridge.Close()
				}
			}(query)
		}
	}()
	return &encryptedDNSConn{Conn: client, cancel: cancel}
}

type encryptedDNSConn struct {
	net.Conn
	cancel context.CancelFunc
}

func (c *encryptedDNSConn) Close() error { c.cancel(); return c.Conn.Close() }

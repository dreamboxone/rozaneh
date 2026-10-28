// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/dreamboxone/rozaneh/internal/tunnelengine"
)

func main() {
	settings := registerSoarSettings()
	version := flag.Bool("version", false, "print version and exit")
	standbys := flag.String("standby-domains", "", "comma-separated fallback tunnel domains with the same server key")
	failAfter := flag.Duration("failover-after", 3*time.Second, "per-domain connection timeout when standbys exist")
	zone := flag.String("zone", "", "delegated tunnel zone")
	key := flag.String("pubkey", "", "base64 server public key")
	resolvers := flag.String("resolvers", "", "comma-separated resolver IP:port addresses")
	listen := flag.String("listen", "127.0.0.1:1088", "SOCKS5 listen address (no authentication)")
	country := flag.String("country", "IR", "resolver discovery country")
	discover := flag.Bool("discover", false, "enable resolver discovery")
	tcp := flag.String("tcp", "auto", "DNS transport: auto, on, only, off")
	resolverMode := flag.String("resolver-transport", "auto", "resolver hop: auto, dot, doh, doh3, doq")
	encryptedOnly := flag.Bool("encrypted-dns-only", false, "require authenticated encrypted resolver hops; disable plaintext discovery")
	http3Fallback := flag.Bool("http3-fallback", false, "allow DoH3 to retry as TLS-protected DoH; never plaintext")
	statsFile := flag.String("stats-file", "", "live traffic JSON path")
	diagnosticsFile := flag.String("diagnostics-file", "", "live resolver diagnostics JSON path")
	maxClients := flag.Int("max-clients", 256, "maximum simultaneous SOCKS clients (16-4096)")
	idleTimeout := flag.Duration("idle-timeout", 5*time.Minute, "close connections with no traffic in either direction")
	usageFile := flag.String("usage-file", "", "persistent traffic JSON path")
	versionJSON := flag.Bool("version-json", false, "print application and embedded Soar versions as JSON")
	flag.Parse()
	if *versionJSON {
		printVersions()
		return
	}
	if *version {
		fmt.Println("Rozaneh 1.0.0-r1")
		return
	}
	var diagnosticOutput io.Writer = os.Stderr
	if *diagnosticsFile != "" {
		engineLog, err := os.OpenFile(filepath.Join(filepath.Dir(*diagnosticsFile), "engine.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			log.Printf("diagnostic log: %v", err)
		} else {
			defer engineLog.Close()
			diagnosticOutput = io.MultiWriter(os.Stderr, engineLog)
			log.SetOutput(diagnosticOutput)
			log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
		}
	}
	if *maxClients < 16 || *maxClients > 4096 || *idleTimeout < 30*time.Second {
		log.Fatal("max-clients must be 16-4096 and idle-timeout must be at least 30s")
	}
	pub, err := base64.StdEncoding.DecodeString(*key)
	if err != nil || len(pub) != ed25519.PublicKeySize || *zone == "" {
		log.Fatal("zone and a valid base64 32-byte public key are required")
	}
	if err := settings.validate(); err != nil {
		log.Fatal(err)
	}
	routes, err := parseResolverPolicy(*resolvers, *resolverMode, *encryptedOnly, *http3Fallback)
	if err != nil {
		log.Fatal(err)
	}
	if len(routes.addresses) == 0 && !*discover {
		log.Fatal("configure resolvers or enable discovery")
	}
	if *tcp != "auto" && *tcp != "on" && *tcp != "only" && *tcp != "off" {
		log.Fatal("invalid tcp mode")
	}
	meter := newTrafficMeter(*statsFile, *usageFile)
	defer meter.close()
	identity := settings.network
	if identity == "" {
		identity = networkIdentity()
	}
	level := slog.LevelWarn
	if settings.debug {
		level = slog.LevelDebug
	}
	candidates := append(splitList(settings.candidates), upstreamResolvers("/tmp/resolv.conf.d/resolv.conf.auto", "/tmp/resolv.conf.auto")...)
	discovery := soar.Discovery{Enabled: *discover, Country: *country, Candidates: candidates, SampleRanges: settings.samples, Want: settings.want, Concurrency: settings.concurrency, Pace: settings.pace, Cache: newResolverCache(settings.cache, identity+"|"+*zone+"|"+*key+"|"+*country+"|"+*resolvers+"|"+strings.Join(candidates, ","))}
	discovery = routes.constrainDiscovery(discovery)
	client, err := newDomainTunnel(soar.Config{
		Zone: *zone, ServerKey: pub, Resolvers: routes.addresses, TCP: *tcp,
		FECDownloadEnabled: settings.fecDownload, FECDataShards: settings.fecData, FECParityShards: settings.fecParity, QNameLossGuard: settings.qnameLossGuard,
		StreamOnly: routes.streamOnly, DialTCP: routes.dial,
		SafeQName: settings.safeName, MaxQName: settings.maxName, Hold: settings.hold,
		MaxInflight: settings.maxInflight, MaxPathWindow: settings.pathWindow, MaxAnswer: settings.maxAnswer,
		StreamWindow: uint64(settings.streamWindow), SendBuffer: settings.sendBuffer, MaxStreams: settings.maxStreams, SessionIdle: settings.sessionIdle,
		Discovery: discovery,
		Logger:    slog.New(slog.NewJSONHandler(diagnosticOutput, &slog.HandlerOptions{Level: level})),
	}, splitList(*standbys), *failAfter)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() { <-ctx.Done(); ln.Close() }()
	if *diagnosticsFile != "" {
		diagnosticsDone := make(chan struct{})
		go func() {
			defer close(diagnosticsDone)
			recordDiagnostics(ctx, *diagnosticsFile, client, routes)
		}()
		defer func() { <-diagnosticsDone }()
	}
	slots := make(chan struct{}, *maxClients)
	dial := newStreamDialer(client, meter, settings.maxStreams)
	var workers sync.WaitGroup
	defer workers.Wait()
	log.Printf("SOCKS5 listening on %s", ln.Addr())
	for {
		c, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Print(err)
			continue
		}
		select {
		case slots <- struct{}{}:
			workers.Add(1)
			go func() {
				defer workers.Done()
				defer func() { <-slots }()
				defer c.Close()
				stopClose := context.AfterFunc(ctx, func() { c.Close() })
				defer stopClose()
				connectionCtx := context.WithValue(ctx, idleTimeoutKey{}, *idleTimeout)
				if err := handle(connectionCtx, c, dial); err != nil {
					log.Print(err)
				}
			}()
		default:
			c.Close()
		}
	}
}

type dialFunc func(context.Context, string, string) (net.Conn, error)

func handle(ctx context.Context, c net.Conn, dial dialFunc) error {
	c.SetDeadline(time.Now().Add(15 * time.Second))
	var h [2]byte
	if _, err := io.ReadFull(c, h[:]); err != nil {
		return err
	}
	if h[0] != 5 || h[1] == 0 {
		return errors.New("invalid SOCKS greeting")
	}
	methods := make([]byte, int(h[1]))
	if _, err := io.ReadFull(c, methods); err != nil {
		return err
	}
	noauth := false
	for _, m := range methods {
		if m == 0 {
			noauth = true
		}
	}
	if !noauth {
		c.Write([]byte{5, 255})
		return errors.New("unsupported SOCKS authentication")
	}
	if _, err := c.Write([]byte{5, 0}); err != nil {
		return err
	}
	var req [4]byte
	if _, err := io.ReadFull(c, req[:]); err != nil {
		return err
	}
	if req[0] != 5 || req[2] != 0 {
		return errors.New("invalid SOCKS request")
	}
	reply := func(code byte) error {
		_, err := c.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0})
		return err
	}
	if req[1] != 1 {
		reply(7)
		return errors.New("only TCP CONNECT is supported")
	}
	var host string
	switch req[3] {
	case 1, 4:
		n := 4
		if req[3] == 4 {
			n = 16
		}
		b := make([]byte, n)
		if _, err := io.ReadFull(c, b); err != nil {
			return err
		}
		host = net.IP(b).String()
	case 3:
		var n [1]byte
		if _, err := io.ReadFull(c, n[:]); err != nil {
			return err
		}
		if n[0] == 0 {
			return errors.New("empty destination")
		}
		b := make([]byte, int(n[0]))
		if _, err := io.ReadFull(c, b); err != nil {
			return err
		}
		host = string(b)
	default:
		reply(8)
		return errors.New("unsupported address type")
	}
	var p [2]byte
	if _, err := io.ReadFull(c, p[:]); err != nil {
		return err
	}
	dialCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	c.SetDeadline(time.Now().Add(50 * time.Second))
	destination := net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(p[:]))))
	started := time.Now()
	up, err := dial(dialCtx, "tcp", destination)
	if err != nil {
		reply(5)
		return fmt.Errorf("tunnel connect: destination=%s elapsed=%s: %w", destination, time.Since(started).Round(time.Millisecond), err)
	}
	defer up.Close()
	if err := reply(0); err != nil {
		return err
	}
	c.SetDeadline(time.Time{})
	idle, _ := ctx.Value(idleTimeoutKey{}).(time.Duration)
	if idle == 0 {
		idle = 5 * time.Minute
	}
	relay(context.WithValue(ctx, tunnelDestinationKey{}, destination), c, up, idle)
	return nil
}

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
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/getlantern/soar"
)

func main() {
	version := flag.Bool("version", false, "print version and exit")
	zone := flag.String("zone", "", "delegated tunnel zone")
	key := flag.String("pubkey", "", "base64 server public key")
	resolvers := flag.String("resolvers", "", "comma-separated resolver IP:port addresses")
	listen := flag.String("listen", "127.0.0.1:1088", "SOCKS5 listen address (no authentication)")
	country := flag.String("country", "IR", "resolver discovery country")
	discover := flag.Bool("discover", false, "enable resolver discovery")
	tcp := flag.String("tcp", "auto", "DNS transport: auto, on, only, off")
	statsFile := flag.String("stats-file", "", "live traffic JSON path")
	usageFile := flag.String("usage-file", "", "persistent traffic JSON path")
	flag.Parse()
	if *version {
		fmt.Println("zoorDNS 0.1.0")
		return
	}
	pub, err := base64.StdEncoding.DecodeString(*key)
	if err != nil || len(pub) != ed25519.PublicKeySize || *zone == "" {
		log.Fatal("zone and a valid base64 32-byte public key are required")
	}
	var rs []string
	for _, r := range strings.Split(*resolvers, ",") {
		if r = strings.TrimSpace(r); r != "" {
			rs = append(rs, r)
		}
	}
	if len(rs) == 0 && !*discover {
		log.Fatal("configure resolvers or enable discovery")
	}
	if *tcp != "auto" && *tcp != "on" && *tcp != "only" && *tcp != "off" {
		log.Fatal("invalid tcp mode")
	}
	meter := newTrafficMeter(*statsFile, *usageFile)
	defer meter.close()
	client, err := soar.NewClient(soar.Config{Zone: *zone, ServerKey: pub, Resolvers: rs, TCP: *tcp, Discovery: soar.Discovery{Enabled: *discover, Country: *country}})
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
	slots := make(chan struct{}, 32)
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
			go func() {
				defer func() { <-slots }()
				defer c.Close()
				if err := handle(ctx, c, func(ctx context.Context, network, addr string) (net.Conn, error) {
					conn, err := client.DialContext(ctx, network, addr)
					if err != nil {
						return nil, err
					}
					return &meteredConn{Conn: conn, m: meter}, nil
				}); err != nil {
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
	reply := func(code byte) { c.Write([]byte{5, code, 0, 1, 0, 0, 0, 0, 0, 0}) }
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
	up, err := dial(dialCtx, "tcp", net.JoinHostPort(host, strconv.Itoa(int(binary.BigEndian.Uint16(p[:])))))
	if err != nil {
		reply(5)
		return fmt.Errorf("tunnel connect: %w", err)
	}
	defer up.Close()
	reply(0)
	c.SetDeadline(time.Time{})
	done := make(chan struct{})
	go func() { io.Copy(up, c); up.Close(); close(done) }()
	io.Copy(c, up)
	c.Close()
	<-done
	return nil
}

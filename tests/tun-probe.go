// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

//go:build ignore

package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	"golang.org/x/sys/unix"
)

func command(args ...string) {
	if output, err := exec.Command("ip", args...).CombinedOutput(); err != nil {
		panic(fmt.Sprintf("ip %v: %s: %v", args, output, err))
	}
}

func checksum(data []byte) uint16 {
	var sum uint32
	for len(data) >= 2 {
		sum += uint32(binary.BigEndian.Uint16(data))
		data = data[2:]
	}
	if len(data) > 0 {
		sum += uint32(data[0]) << 8
	}
	for sum>>16 != 0 {
		sum = (sum & 65535) + (sum >> 16)
	}
	return ^uint16(sum)
}

func main() {
	if len(os.Args) != 2 || os.Args[1] != "zdreview0" {
		panic("usage: tun-probe zdreview0")
	}
	if output, err := exec.Command("nft", "list", "table", "inet", "rozaneh_guard").CombinedOutput(); err != nil {
		panic(string(output))
	}
	fd, err := unix.Open("/dev/net/tun", unix.O_RDWR, 0)
	if err != nil {
		panic(err)
	}
	defer unix.Close(fd)
	ifr, err := unix.NewIfreq(os.Args[1])
	if err != nil {
		panic(err)
	}
	ifr.SetUint16(unix.IFF_TUN | unix.IFF_NO_PI)
	if err := unix.IoctlIfreq(fd, unix.TUNSETIFF, ifr); err != nil {
		panic(err)
	}
	command("addr", "add", "10.253.254.1/30", "dev", os.Args[1])
	command("link", "set", os.Args[1], "up")
	command("-6", "addr", "add", "fdfe:253:254::1/64", "dev", os.Args[1], "nodad")
	const destination6 = "2001:db8:ffff::123"
	command("-6", "route", "add", destination6+"/128", "dev", "br-lan")
	defer exec.Command("ip", "-6", "route", "del", destination6+"/128", "dev", "br-lan").Run()
	for _, path := range []string{"/proc/sys/net/ipv4/ip_forward", "/proc/sys/net/ipv6/conf/all/forwarding"} {
		old, err := os.ReadFile(path)
		if err != nil {
			panic(err)
		}
		if err := os.WriteFile(path, []byte("1\n"), 0600); err != nil {
			panic(err)
		}
		defer os.WriteFile(path, old, 0600)
	}
	for _, destination := range []struct {
		host string
		port uint16
	}{{"203.0.113.123", 80}, {"10.253.254.1", 53}} {
		packet := make([]byte, 40)
		packet[0], packet[8], packet[9] = 0x45, 64, 17
		binary.BigEndian.PutUint16(packet[2:], uint16(len(packet)))
		copy(packet[12:16], net.ParseIP("10.253.254.2").To4())
		copy(packet[16:20], net.ParseIP(destination.host).To4())
		binary.BigEndian.PutUint16(packet[10:], checksum(packet[:20]))
		binary.BigEndian.PutUint16(packet[20:], 12345)
		binary.BigEndian.PutUint16(packet[22:], destination.port)
		binary.BigEndian.PutUint16(packet[24:], 20)
		if _, err := unix.Write(fd, packet); err != nil {
			panic(err)
		}
	}
	packet := make([]byte, 48)
	packet[0], packet[6], packet[7] = 0x60, 17, 64
	binary.BigEndian.PutUint16(packet[4:], 8)
	copy(packet[8:24], net.ParseIP("fdfe:253:254::2").To16())
	copy(packet[24:40], net.ParseIP(destination6).To16())
	binary.BigEndian.PutUint16(packet[40:], 12345)
	binary.BigEndian.PutUint16(packet[42:], 80)
	binary.BigEndian.PutUint16(packet[44:], 8)
	pseudo := append([]byte{}, packet[8:40]...)
	pseudo = append(pseudo, 0, 0, 0, 8, 0, 0, 0, 17)
	pseudo = append(pseudo, packet[40:]...)
	binary.BigEndian.PutUint16(packet[46:], checksum(pseudo))
	if _, err := unix.Write(fd, packet); err != nil {
		panic(err)
	}
	time.Sleep(250 * time.Millisecond)
	fmt.Println("Injected IPv4, IPv6 and DNS packets on isolated TUN")
}

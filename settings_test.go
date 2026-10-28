// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestUpstreamResolverCandidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resolv.conf.auto")
	data := "# Interface wan\nnameserver 192.168.1.1\nnameserver 127.0.0.1\nnameserver ::1\nnameserver 192.168.1.1\nnameserver 2001:db8::53\nnameserver invalid\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	want := []string{"192.168.1.1:53", "[2001:db8::53]:53"}
	if got := upstreamResolvers(path, path+".missing"); !reflect.DeepEqual(got, want) {
		t.Fatalf("upstream resolvers %v, want %v", got, want)
	}
}

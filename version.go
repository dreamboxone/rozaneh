// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>

package main

import (
	"encoding/json"
	"os"
)

func embeddedSoarVersion() string {
	return "v0.2.3+cotten.7db7b73658c7"
}

func printVersions() {
	json.NewEncoder(os.Stdout).Encode(map[string]string{"application": "Rozaneh", "version": "1.0.0", "engine": "Soar", "engine_version": embeddedSoarVersion()})
}

// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
package main

import (
 "bytes"
 "encoding/json"
 "testing"
 "github.com/BurntSushi/toml"
)

func TestGeneratedSettingsPreserveIntegerTypes(t *testing.T) {
 var values map[string]any
 decoder:=json.NewDecoder(bytes.NewBufferString(`{"LISTEN_PORT":1088,"DATA_ENCRYPTION_METHOD":3,"RETRY_COUNTS":[2,4],"RATIO":0.5}`))
 decoder.UseNumber()
 if err:=decoder.Decode(&values);err!=nil{t.Fatal(err)}
 normalizeJSONNumbers(values)
 var output bytes.Buffer
 if err:=toml.NewEncoder(&output).Encode(values);err!=nil{t.Fatal(err)}
 var configuration struct {ListenPort int `toml:"LISTEN_PORT"`;Method int `toml:"DATA_ENCRYPTION_METHOD"`;Retries []int `toml:"RETRY_COUNTS"`;Ratio float64 `toml:"RATIO"`}
 if _,err:=toml.Decode(output.String(),&configuration);err!=nil{t.Fatal(err)}
 if configuration.ListenPort!=1088||configuration.Method!=3||len(configuration.Retries)!=2||configuration.Retries[1]!=4||configuration.Ratio!=0.5{t.Fatalf("unexpected configuration: %+v",configuration)}
}

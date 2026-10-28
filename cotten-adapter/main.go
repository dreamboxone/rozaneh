// SPDX-License-Identifier: Apache-2.0
// Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
package main

import (
 "context"
 "encoding/json"
 "flag"
 "fmt"
 "log"
 "os"
 "os/signal"
 "syscall"
 "time"
 "bytes"
 "strconv"
 "github.com/BurntSushi/toml"
 "cottendns-go/internal/client"
 "cottendns-go/internal/config"
)

var applicationVersion="development"
var engineRevision="unknown"
var engineRelease="unknown"

func normalizeJSONNumbers(value any) any {
 switch current:=value.(type) {
 case json.Number:
  if integer,err:=strconv.ParseInt(string(current),10,64);err==nil{return integer}
  number,err:=current.Float64();if err!=nil{log.Fatal("invalid numeric configuration value")};return number
 case map[string]any:
  for key,item:=range current{current[key]=normalizeJSONNumbers(item)}
 case []any:
  for index,item:=range current{current[index]=normalizeJSONNumbers(item)}
 }
 return value
}

func atomicJSON(path string, value any) error {
 if path=="" {return nil}
 encoded,err:=json.Marshal(value);if err!=nil{return err}
 if err=os.WriteFile(path+".tmp",encoded,0600);err!=nil{return err}
 return os.Rename(path+".tmp",path)
}

func main(){
 configPath:=flag.String("config","/var/run/rozaneh/client.toml","CottenDNS configuration")
 resolverPath:=flag.String("resolvers-file","/var/run/rozaneh/resolvers.txt","resolver list")
 output:=flag.String("runtime-file","/var/run/rozaneh/runtime.json","runtime status")
 pathsOutput:=flag.String("paths-file","/var/run/rozaneh/paths.json","resolver quality")
 statsOutput:=flag.String("stats-file","/var/run/rozaneh/traffic.json","live counters")
 usageOutput:=flag.String("usage-file","/etc/rozaneh/traffic.json","persistent counters")
 version:=flag.Bool("version",false,"print application version")
 versionJSON:=flag.Bool("version-json",false,"print component versions")
 prepare:=flag.String("prepare-config","","JSON settings to convert into a CottenDNS config")
 preparedOutput:=flag.String("output-config","","generated TOML destination")
 scanOnly:=flag.Bool("scan-only",false,"scan resolvers without opening the proxy")
 nativeFlags,err:=config.NewClientConfigFlagBinder(flag.CommandLine)
 if err!=nil{log.Fatal(err)}
 flag.Parse()
 if *prepare!="" {
  data,err:=os.ReadFile(*prepare);if err!=nil{log.Fatal(err)}
  var values map[string]any
  decoder:=json.NewDecoder(bytes.NewReader(data));decoder.UseNumber()
  if err=decoder.Decode(&values);err!=nil{log.Fatal("invalid settings JSON")}
  normalizeJSONNumbers(values)
  if advanced,ok:=values["_advanced_toml"].(string);ok&&advanced!="" {
   var merged map[string]any
   if _,err=toml.Decode(advanced,&merged);err!=nil{log.Fatal("invalid advanced TOML configuration")}
   for key,value:=range values {if key!="_advanced_toml"{merged[key]=value}}
   values=merged
  }
  delete(values,"_advanced_toml")
  var buffer bytes.Buffer
  if err=toml.NewEncoder(&buffer).Encode(values);err!=nil{log.Fatal("invalid configuration values")}
  if err=os.WriteFile(*preparedOutput,buffer.Bytes(),0600);err!=nil{log.Fatal(err)}
  return
 }
 if *versionJSON {json.NewEncoder(os.Stdout).Encode(map[string]string{"application":"Rozaneh","version":applicationVersion,"engine":"CottenDNS","engine_version":engineRelease,"engine_revision":engineRevision});return}
 if *version {fmt.Printf("Rozaneh %s; CottenDNS %s\n",applicationVersion,engineRevision);return}
 overrides:=nativeFlags.Overrides()
 overrides.ResolversFilePath=resolverPath
 preConfig:=config.PeekClientStartupConfig(*configPath)
 var app *client.Client
 if !*scanOnly && preConfig.StartupMode=="logs" {
  entries:=client.ScanResolverCacheLogs(preConfig.ResolvedLogDir(),preConfig.LogScanMaxDays,preConfig.LogScanMaxResolvers)
  if len(entries)>0{app,err=client.BootstrapFromLogs(*configPath,entries,overrides)}else{app,err=client.Bootstrap(*configPath,overrides)}
 }else{app,err=client.Bootstrap(*configPath,overrides)}
 if err!=nil{log.Fatal(err)}
 ctx,stop:=signal.NotifyContext(context.Background(),os.Interrupt,syscall.SIGTERM);defer stop()
 if *scanOnly {if _,err=app.RunResolverScan(ctx);err!=nil{log.Fatal(err)};return}
 meter:=newTrafficMeter(*statsOutput,*usageOutput);defer meter.close()
 complete:=make(chan struct{})
 go func(){
  defer close(complete)
  timer:=time.NewTicker(2*time.Second);defer timer.Stop()
  var lastTX,lastRX uint64
  for {
   snapshot:=app.StatusSnapshot()
   tx,rx:=snapshot.TXBytes,snapshot.RXBytes
   if tx>=lastTX&&rx>=lastRX {meter.add(tx-lastTX,rx-lastRX)}
   lastTX,lastRX=tx,rx
   atomicJSON(*pathsOutput,map[string]any{"updated":time.Now().Unix(),"established":snapshot.Phase=="connected","paths":app.RozanehPaths()})
   report:=struct{Updated int64 `json:"updated"`;Engine string `json:"engine"`;Revision string `json:"revision"`;Status client.StatusSnapshot `json:"status"`}{time.Now().Unix(),"CottenDNS",engineRevision,snapshot}
   if err:=atomicJSON(*output,report);err!=nil{log.Printf("runtime status: %v",err)}
   select{case <-ctx.Done():return;case <-timer.C:}
  }
 }()
 if err=app.Run(ctx);err!=nil{log.Printf("CottenDNS runtime: %v",err)}
 stop();<-complete
 if *output!="" {atomicJSON(*output,map[string]any{"updated":time.Now().Unix(),"engine":"CottenDNS","revision":engineRevision,"status":map[string]string{"Phase":"stopped"}})}
}

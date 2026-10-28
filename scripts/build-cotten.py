#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
"""Build the unmodified pinned CottenDNS runtime with the Rozaneh entrypoint."""
import argparse
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import hashlib

ROOT=Path(__file__).resolve().parents[1]

def run(*args,**kwargs):
    return subprocess.run(args,check=True,**kwargs)

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('--source',type=Path,help='existing checkout at the exact pinned revision')
    parser.add_argument('--output',type=Path,required=True)
    parser.add_argument('--goarch',default='arm')
    parser.add_argument('--goarm',default='7')
    parser.add_argument('--test',action='store_true')
    args=parser.parse_args()
    lock=json.loads((ROOT/'cotten-upstream.json').read_text())
    with tempfile.TemporaryDirectory(prefix='rozaneh-cotten-build-') as temporary:
        source=Path(temporary)/'source'
        if args.source:
            revision=subprocess.check_output(['git','-C',str(args.source),'rev-parse','HEAD'],text=True).strip()
            if revision!=lock['revision']:raise SystemExit('Source checkout does not match cotten-upstream.json')
            run('git','clone','--no-hardlinks',str(args.source),str(source))
        else:
            run('git','clone',lock['repository'],str(source))
        run('git','checkout','--detach',lock['revision'],cwd=source)
        shutil.copytree(ROOT/'cotten-adapter',source/'cmd/rozaneh',ignore=shutil.ignore_patterns('go.mod','go.sum','overlay'))
        shutil.copyfile(ROOT/'cotten-adapter/overlay/client_snapshot.go.in',source/'internal/client/rozaneh_snapshot.go')
        shutil.copyfile(ROOT/'traffic.go',source/'cmd/rozaneh/traffic.go')
        env={**os.environ,'CGO_ENABLED':'0'}
        if args.test:
            test_env={key:value for key,value in env.items() if key not in ('GOOS','GOARCH','GOARM','GOAMD64','GOMIPS','GOMIPS64')}
            run('go','test','-race','-timeout=180s','./internal/client','./internal/arq','./internal/config','./cmd/rozaneh',cwd=source,env={**test_env,'CGO_ENABLED':'1'})
        env.update(GOOS='linux',GOARCH=args.goarch)
        if args.goarch=='arm':env['GOARM']=args.goarm
        output=args.output.resolve();output.parent.mkdir(parents=True,exist_ok=True)
        flags=f"-s -w -X main.applicationVersion={lock['application_version']} -X main.engineRevision={lock['revision']} -X main.engineRelease={lock.get('release',lock['revision'])}"
        run('go','build','-trimpath','-ldflags='+flags,'-o',str(output),'./cmd/rozaneh',cwd=source,env=env)
        metadata={**lock,'sha256':hashlib.sha256(output.read_bytes()).hexdigest(),'goarch':args.goarch}
        (output.parent/'cotten-source.json').write_text(json.dumps(metadata,indent=2)+'\n')
        listing=subprocess.check_output(['go','list','-m','-json','all'],cwd=source,text=True)
        decoder=json.JSONDecoder()
        license_root=output.parent/'engine-licenses'
        license_root.mkdir(exist_ok=True)
        while listing.strip():
            listing=listing.lstrip();module,index=decoder.raw_decode(listing);listing=listing[index:]
            directory=module.get('Dir')
            if not directory:continue
            destination=license_root/module['Path'].replace('/','_')
            for item in Path(directory).iterdir():
                if item.is_file() and (item.name.upper().startswith(('LICENSE','LICENCE','NOTICE','COPYING')) or item.name in ('THIRD_PARTY_NOTICES.md','SOAR_LICENSE.txt','QPACK_LICENSE.txt','QUIC_GO_LICENSE.txt')):
                    destination.mkdir(exist_ok=True);shutil.copyfile(item,destination/item.name)

if __name__=='__main__':main()

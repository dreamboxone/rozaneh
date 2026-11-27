#!/usr/bin/env python3
# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
"""Resolve a published CottenDNS release to an immutable source revision."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
from urllib.request import Request,urlopen

ROOT=Path(__file__).resolve().parents[1]

def latest():
    headers={'Accept':'application/vnd.github+json','User-Agent':'rozaneh-release'}
    if os.environ.get('GH_TOKEN'):headers['Authorization']='Bearer '+os.environ['GH_TOKEN']
    with urlopen(Request('https://api.github.com/repos/WhiteDNS/CottenDNS/releases/latest',headers=headers),timeout=30) as response:
        release=json.load(response)
    if release.get('draft') or release.get('prerelease'):raise RuntimeError('A stable published release is required')
    tag=release['tag_name']
    if not re.fullmatch(r'[A-Za-z0-9._+-]+',tag):raise RuntimeError('Invalid release tag')
    repo=json.loads((ROOT/'cotten-upstream.json').read_text())['repository']
    refs=subprocess.check_output(['git','ls-remote',repo,'refs/tags/'+tag,'refs/tags/'+tag+'^{}'],text=True)
    revisions={line.split()[1]:line.split()[0] for line in refs.splitlines()}
    revision=revisions.get('refs/tags/'+tag+'^{}') or revisions.get('refs/tags/'+tag)
    if not revision or not re.fullmatch(r'[0-9a-f]{40}',revision):raise RuntimeError('Release tag could not be resolved')
    return {'tag':tag,'revision':revision,'url':release['html_url']}

def main():
    parser=argparse.ArgumentParser()
    parser.add_argument('command',choices=['check','prepare','verify'])
    parser.add_argument('--state',type=Path,required=True)
    parser.add_argument('--output',type=Path)
    parser.add_argument('--force',action='store_true')
    parser.add_argument('--publish-tag',help='build an already tagged Rozaneh version')
    args=parser.parse_args()
    lock_path=ROOT/'cotten-upstream.json'
    lock=json.loads(lock_path.read_text())
    if args.command=='verify':
        import hashlib
        manifest=json.loads((args.output/'manifest.json').read_text())
        component=manifest['components']['rozaneh']
        if component['engine_version']!=lock.get('release',lock['revision']) or not component.get('bundled_luci'):
            raise RuntimeError('Release engine or bundled interface mismatch')
        for fmt in ('ipk','apk'):
            for asset in component['assets'][fmt].values():
                name=asset['name']
                if Path(name).name!=name:raise RuntimeError('Invalid asset path')
                if hashlib.sha256((args.output/name).read_bytes()).hexdigest()!=asset['sha256']:raise RuntimeError('Asset checksum mismatch')
        return
    if args.command=='check':
        if args.publish_tag:
            if args.force or not re.fullmatch(r'v\d+\.\d+\.\d+-r\d+',args.publish_tag):
                raise RuntimeError('Specify a valid release tag without --force')
            ref='refs/tags/'+args.publish_tag
            sha=subprocess.check_output(['git','rev-parse',ref+'^{commit}'],text=True).strip()
            subprocess.run(['git','merge-base','--is-ancestor',sha,'HEAD'],check=True)
            pinned=json.loads(subprocess.check_output(['git','show',sha+':cotten-upstream.json'],text=True))
            if 'v'+pinned['application_version']!=args.publish_tag:
                raise RuntimeError('Tag does not match the pinned package version')
            state={'tagged':True,'changed':True,'base_sha':sha,'tag':args.publish_tag,
                   'package_version':pinned['application_version'],'engine':pinned['release'],
                   'upstream_url':'https://github.com/WhiteDNS/CottenDNS/releases/tag/'+pinned['release']}
        else:
            state=latest();state['changed']=args.force or state['revision']!=lock['revision'] or state['tag']!=lock.get('release')
            state['base_sha']=subprocess.check_output(['git','rev-parse','HEAD'],text=True).strip()
        args.state.write_text(json.dumps(state,indent=2)+'\n')
        if os.environ.get('GITHUB_OUTPUT'):
            with open(os.environ['GITHUB_OUTPUT'],'a') as output:output.write('changed='+str(state['changed']).lower()+'\n')
        print(json.dumps(state));return
    state=json.loads(args.state.read_text())
    if not state.get('changed'):return
    if not re.fullmatch(r'[0-9a-f]{40}',state['revision']):raise RuntimeError('Invalid source revision')
    package=ROOT/'scripts/package.py'
    text=package.read_text()
    match=re.search(r'"rozaneh": "(\d+\.\d+\.\d+)-r(\d+)"',text)
    if not match:raise RuntimeError('Package version not found')
    version=f'{match[1]}-r{int(match[2])+1}'
    package.write_text(text.replace(match[0],f'"rozaneh": "{version}"'))
    makefile=ROOT/'Makefile';text=makefile.read_text()
    makefile.write_text(re.sub(r'PKG_RELEASE:=\d+','PKG_RELEASE:='+str(int(match[2])+1),text))
    lock.update(revision=state['revision'],release=state['tag'],application_version=version)
    lock_path.write_text(json.dumps(lock,indent=2)+'\n')
    for name in ('README.md','README.fa.md','README.en.md'):
        guide=ROOT/name
        if guide.exists():
            text=guide.read_text(encoding='utf-8')
            text=re.sub(r'(--version\s+)v\d{4}\.\d{2}\.\d{2}\.\d{6}-[0-9a-f]+',lambda match:match[1]+state['tag'],text)
            guide.write_text(text,encoding='utf-8')
    state.update(upstream_tag=state['tag'],upstream_url=state['url'],engine=state['tag'],package_version=version,tag='v'+version)
    args.state.write_text(json.dumps(state,indent=2)+'\n')

if __name__=='__main__':main()

/* SPDX-License-Identifier: Apache-2.0
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const uiSource = fs.readFileSync(path.join(__dirname, '../luci/root/www/luci-static/resources/rozaneh/ui.js'), 'utf8');
const element = (tag, attrs, children) => ({tag, attrs, children});
const ui = new Function('baseclass', 'E', 'icons', 'localStorage', 'L', uiSource)({ extend: value => value }, element, {}, { getItem: () => 'fa' }, { resource: p => p });

assert.equal(ui.siteResult({ state: 'ok', http: '200', seconds: '0.25', exit_code: 0 }).tone, 'mk-on');
assert.equal(ui.siteResult({ state: 'ok', http: '200', seconds: '1.5', exit_code: 0 }).tone, 'mk-wait');
assert.equal(ui.siteResult({ state: 'ok', http: '200', seconds: '2', exit_code: 0 }).tone, 'mk-bad');
assert.equal(ui.siteResult({ state: 'restricted', http: '403', seconds: '0.1', exit_code: 0 }).tone, 'mk-bad');
assert.equal(ui.siteResult({ state: 'failed', http: '200', seconds: '0.1', exit_code: 28 }).tone, 'mk-bad');
assert.equal(ui.siteResult({ state: 'ok', http: '302', seconds: '0.1', exit_code: 0 }).tone, 'mk-bad');
assert.equal(ui.siteResult({ state: 'ok', http: '200', seconds: 'invalid', exit_code: 0 }).tone, 'mk-bad');
console.log('LuCI site status checks passed');
const connected = {enabled:true, running:true, lan_running:true, lan_ready:true, health_running:true, health_updated:1000, connected:true};
assert.equal(ui.connectionState(connected,1001),'connected');
assert.equal(ui.connectionState({...connected,health_updated:900},1001),'connecting');
assert.equal(ui.connectionState({...connected,connected:false},1001),'connecting');
assert.equal(ui.connectionState({...connected,lan_running:false},1001),'starting');
assert.equal(ui.connectionState({...connected,lan_ready:false},1001),'starting');
assert.equal(ui.connectionState({...connected,health_running:false},1001),'connecting');
assert.equal(ui.connectionState({...connected,enabled:false},1001),'stopped');
assert.equal(ui.connectionState({...connected,conflict:true},1001),'blocked');
assert.equal(ui.connectionState({...connected,enabled:false,conflict:true},1001),'stopped');
assert.equal(ui.connectionState({...connected,enabled:false,missing:'sing-box'},1001),'stopped');
const heading=ui.wordmark('0.1.0-r5');
assert.equal(heading.attrs.dir,'ltr');
assert.equal(heading.children[0].attrs.class,'mk-version');
assert.equal(heading.children[0].children,'0.1.0-5');
assert.equal(heading.children[1].children[0].children,'روزنه');
assert.equal(ui.engineVersion('v0.0.0-20261022160106-fea80aae6c60'),'2026.10.22 / fea80aae6c60');
console.log('Connection state, header direction and version checks passed');

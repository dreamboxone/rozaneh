/* SPDX-License-Identifier: Apache-2.0
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');

class Node {
 constructor(tag, attrs={}, children=[]) { this.tag=tag;this.attrs=attrs;this.children=[];this.checked=attrs.checked!=null&&attrs.checked!==false;this.style={};this.classList={toggle(){},add(){}};this.replaceChildren(...(Array.isArray(children)?children:[children])); }
 appendChild(node){this.children.push(node);return node;}
 replaceChildren(...nodes){this.children=nodes;}
 querySelectorAll(){return [];}
 setAttribute(name,value){this.attrs[name]=value;}
 set textContent(value){this.children=[String(value)];}
 get textContent(){return this.children.map(n=>n instanceof Node?n.textContent:String(n||'')).join(' ');}
}
const E=(tag,attrs,children)=>new Node(tag,attrs,children);
const walk=node=>node instanceof Node?[node,...node.children.flatMap(walk)]:[];
const base=path.join(__dirname,'../luci/root/www/luci-static/resources');
const storage={getItem:()=> 'fa'};
const icons={icon:()=>E('span'),brand:()=>E('span')};
const L={resource:p=>p,url:p=>p};
const zui=new Function('baseclass','E','icons','localStorage','L',fs.readFileSync(path.join(base,'rozaneh/ui.js'),'utf8'))({extend:x=>x},E,icons,storage,L);
const logs=new Function('baseclass',fs.readFileSync(path.join(base,'rozaneh/logs.js'),'utf8'))({extend:x=>x});
const fields=[];
const tabs=[];
const section={tab(name){tabs.push(name);},option(type,name,label){return this.taboption(null,type,name,label);},taboption(tab,type,name,label){const option={name,label,choices:[],depends(value){this.dependency=value;},value(v){this.choices.push(v);}};fields.push(option);return option;}};
const form={Map:class {section(){return section;}lookupOption(name){return fields.filter(f=>f.name===name);}render(){return Promise.resolve(E('form'));}},Value:class {},Flag:class {},ListValue:class {},DynamicList:class {},Button:class {}};
let state={enabled:true,running:true,lan_running:true,lan_ready:true,health_running:true,connected:true,health_updated:Math.floor(Date.now()/1000),version:'0.1.0-r5',model:'Google WiFi (Gale)',missing:''};
let updateReport={components:[{name:'rozaneh',current:'0.1.0-r5',latest:'0.1.0-r5',available:false,current_engine:'v2026.10.24.183015-7db7b73'},{name:'luci-app-rozaneh',current:'0.1.0-r6',latest:'0.1.0-r6',available:false}]};
const calls=[],polls=[],uciWrites=[];
const rpc={declare(spec){return (...args)=>{
 calls.push([spec.method,...args]);
 if(spec.method==='control'&&args[0].startsWith('killswitch_'))state={...state,lan_killswitch:args[0]==='killswitch_on'};
 else if(spec.method==='control')state={...state,enabled:args[0]!=='stop',running:args[0]!=='stop'};
 return Promise.resolve(spec.method==='status'?state:spec.method==='update_status'?updateReport:spec.method==='control'||spec.method==='install_update'||spec.method==='clear_logs'?{ok:true}:{});
};}};
const loadView=name=>new Function('view','form','rpc','uci','poll','ui','icons','zui','E','localStorage','L','document','logs',fs.readFileSync(path.join(base,'view/rozaneh',name+'.js'),'utf8'))({extend:x=>x},form,rpc,{load:()=>Promise.resolve(),get:()=>undefined,set:(...args)=>uciWrites.push(args)},{add:fn=>polls.push(fn)},{createHandlerFn:(ctx,fn)=>(...args)=>fn.apply(ctx,args),addNotification(){}},icons,zui,E,storage,L,{createTextNode:s=>s},logs);

(async()=>{
 const dashboard=loadView('dashboard');
 const root=await dashboard.render([null,state]);
 await new Promise(setImmediate);
 assert(!fields.some(f=>f.name==='enabled'||f.name==='lan_enabled'));
 assert(!tabs.includes('resolver'));
 assert(!fields.some(f=>f.name==='resolvers'));
 assert(!tabs.includes('server')&&!tabs.includes('lan'));
 await loadView('server').render();
 await loadView('network').render();
 const dns=loadView('dns');
 const dnsRoot=await dns.render();
 assert.deepEqual(fields.find(f=>f.name==='country').choices,['IR','RU','CN']);
 assert.equal(fields.find(f=>f.name==='tcp').default,'auto');
 const resolverMode=fields.find(f=>f.name==='resolver_transport');
 assert.deepEqual(resolverMode.choices,['auto','dot','doh','doh3','doq']);
 assert.equal(fields.find(f=>f.name==='http3_fallback').default,'1');
 assert.deepEqual(fields.find(f=>f.name==='discover').dependency,{resolver_transport:'auto',encrypted_dns_only:'0'});
 assert.equal(fields.find(f=>f.name==='discover').retain,true);
 const secureOnly=fields.find(f=>f.name==='encrypted_dns_only');
 secureOnly.getUIElement=()=>({getValue:()=> '1'});
 const resolversField=fields.find(f=>f.name==='resolvers');
 assert.notEqual(resolversField.validate('main','1.1.1.1:53'),true);
 assert.equal(resolversField.validate('main','doh3://1.1.1.1/dns-query'),true);
 resolverMode.getUIElement=()=>({getValue:()=> 'dot'});
 assert.equal(resolversField.validate('main','1.1.1.1:53'),true);
 delete secureOnly.getUIElement;delete resolverMode.getUIElement;
 assert.deepEqual(fields.find(f=>f.name==='network_id').choices,['','mobile','home','office']);
 assert.equal(fields.find(f=>f.name==='network_id').default,'');
 let candidateValue='8.8.8.8:53,203.0.113.8:53';
 const candidateField=fields.find(f=>f.name==='discovery_candidates');
 candidateField.getUIElement=()=>({getValue:()=>candidateValue,setValue:value=>{candidateValue=value;}});
 fields.find(f=>f.name==='_candidates_public').onclick();
 const firstCandidates=candidateValue;
 fields.find(f=>f.name==='_candidates_public').onclick();
 assert.equal(candidateValue,firstCandidates);
 assert(candidateValue.includes('203.0.113.8:53'));
 assert.equal(candidateValue.split(',').filter(v=>v==='8.8.8.8:53').length,1);
 fields.find(f=>f.name==='_candidates_iran').onclick();
 assert(candidateValue.includes('185.212.51.144:53'));
 assert(candidateValue.split(',').every(v=>v.endsWith(':53')));
 assert.equal(uciWrites.length,0);
 assert(walk(dnsRoot).some(n=>n.attrs.href==='admin/services/rozaneh/dns'&&n.attrs['aria-current']==='page'));
 const switches=walk(root).filter(n=>n.attrs.role==='switch');
 assert.equal(switches.length,2);
 const statusCard=walk(root).find(n=>n.attrs.class==='mk-card zd-status-card');
 const basicSettings=walk(root).find(n=>n.attrs.class==='mk-card zd-basic-settings');
 assert(!walk(statusCard).some(n=>n.attrs.role==='switch'));
 assert.equal(walk(basicSettings).filter(n=>n.attrs.role==='switch').length,2);
 assert(basicSettings.textContent.includes('تنظیمات پایه'));
 assert(root.textContent.includes('متصل'));
 assert(!root.textContent.includes('آزمون HTTPS'));
 assert(!root.textContent.includes('اجرای خودکار'));
 assert(!fields.some(f=>f.name==='lan_killswitch'));
 const lanDevices=fields.find(f=>f.name==='lan_devices');
 assert.equal(lanDevices.validate('main',''),true);
 assert.equal(lanDevices.validate('main','br-lan'),true);
 assert.notEqual(lanDevices.validate('main','wan;reboot'),true);
 assert.equal(lanDevices.rmempty,false);
 state={...state,connected:false};
 await polls[0]();
 assert(root.textContent.includes('در حال اتصال…'));
 assert(!walk(root).some(n=>n.attrs.class==='mk-chip ok'&&n.textContent==='متصل'));
 switches[0].checked=false;switches[0].attrs.change();
 await new Promise(setImmediate);
 await new Promise(setImmediate);
 assert(calls.some(c=>c[0]==='control'&&c[1]==='stop'));
 assert(root.textContent.includes('خاموش'));
 const killSwitch=switches.find(n=>n.attrs['aria-label']==='کلید قطع اضطراری (kill switch)');
 killSwitch.checked=true;killSwitch.attrs.change();
 await new Promise(setImmediate);
 await new Promise(setImmediate);
 assert(calls.some(c=>c[0]==='control'&&c[1]==='killswitch_on'));
 assert.equal(state.enabled,false);
 assert.equal(killSwitch.checked,true);
 killSwitch.checked=false;killSwitch.attrs.change();
 await new Promise(setImmediate);
 await new Promise(setImmediate);
 assert(calls.some(c=>c[0]==='control'&&c[1]==='killswitch_off'));

 assert(!tabs.includes('advanced'));
 assert(!fields.some(f=>f.name==='max_inflight'||f.name==='debug'));
 const settings=loadView('settings');
 const settingsRoot=await settings.render();
 assert(fields.some(f=>f.name==='listen'&&f.default==='127.0.0.1:1088'));
 assert(fields.find(f=>f.name==='config_preset').choices.includes('survival'));
 assert(fields.find(f=>f.name==='resolver_ip_mode').choices.includes('dual'));
 assert.equal(fields.find(f=>f.name==='tunnel_mode').default,'legacy');
 assert(fields.find(f=>f.name==='advanced_config').rows>=10);
 assert(!fields.some(f=>f.name==='max_inflight'));
 const menu=walk(settingsRoot).find(n=>n.tag==='nav');
 assert.equal(menu.children.length,7);
 assert(menu.children.every(n=>/^--tab-icon-colour:#[0-9a-f]{6}$/.test(n.attrs.style)));
 assert.equal(menu.children.find(n=>n.attrs['aria-current']==='page').attrs.href,'admin/services/rozaneh/settings');
 const logView=loadView('log');
 const logRoot=await logView.render([{text:'Wed Oct 28 09:12:34 2026 daemon.info rozaneh[1]: SOCKS5 listening on 127.0.0.1:1088'},state]);
 assert(fields.some(f=>f.name==='debug'&&f.label==='گزارش کامل روزنه'));
 const logOutput=walk(logRoot).find(n=>n.tag==='pre');
 assert.equal(logOutput.attrs.dir,'rtl');
 assert(!walk(logRoot).some(n=>n.tag==='input'));
 assert(!logRoot.textContent.includes('بازخوانی'));
 const originalLog=logOutput.textContent;
 await polls[polls.length-1]();
 assert.equal(logOutput.textContent,originalLog);
 const logButtons=walk(logRoot).filter(n=>n.tag==='button');
 assert(logButtons.some(n=>n.textContent.includes('کپی')));
 await logButtons.find(n=>n.textContent.includes('حذف')).attrs.click();
 assert(calls.some(c=>c[0]==='clear_logs'));
 assert.equal(logOutput.textContent,'گزارشی در این سطح وجود ندارد.');
 const updates=loadView('update');
 const updateRoot=updates.render([updateReport,state]);
 assert(updateRoot.textContent.includes('به‌روزرسانی'));
 assert(updateRoot.textContent.includes('نیازمندی‌های روتر'));
 assert.equal(walk(updateRoot).filter(n=>n.attrs.class==='zd-update-row').length,3);
 assert(updateRoot.textContent.includes('هسته CottenDNS'));
 assert(updateRoot.textContent.includes('2026.10.24'));
 assert(!updateRoot.textContent.includes('نصب مجدد همراه روزنه'));
 const checking=walk(updateRoot).find(n=>n.tag==='button'&&n.attrs['data-action']==='app:refresh');
 const checkingPromise=checking.attrs.click();
 let running=walk(updateRoot).filter(n=>n.tag==='button'&&n.attrs.class.includes('mk-running'));
 assert.deepEqual(running.map(n=>n.attrs['data-action']),['app:refresh']);
 await checkingPromise;
 const reinstall=walk(updateRoot).find(n=>n.tag==='button'&&n.textContent===' نصب مجدد');
 assert(reinstall);
 const installingPromise=reinstall.attrs.click();
 running=walk(updateRoot).filter(n=>n.tag==='button'&&n.attrs.class.includes('mk-running'));
 assert.deepEqual(running.map(n=>n.attrs['data-action']),['app:download']);
 await installingPromise;
 assert(calls.some(c=>c[0]==='install_update'&&c[1]==='all'&&c[2]===true));
 updateReport={components:[{name:'rozaneh',current:'0.1.0-r11',latest:'0.1.0-r12',available:true,current_engine:'v2026.10.24.183015-7db7b73',latest_engine:'v0.2.5'}]};
 const engineRoot=updates.render([updateReport,state]);
 assert(engineRoot.textContent.includes('v0.2.5'));
 const engineUpdate=walk(engineRoot).find(n=>n.tag==='button'&&n.textContent===' به‌روزرسانی همراه روزنه');
 assert(engineUpdate);
 assert.strictEqual(engineUpdate.attrs.disabled,null,'Available update must omit the disabled HTML attribute');
 await engineUpdate.attrs.click();
 assert(calls.some(c=>c[0]==='install_update'&&c[1]==='all'&&c[2]===false));
 console.log('Dashboard controls, automatic status and update row checks passed');
})().catch(error=>{console.error(error);process.exitCode=1;});

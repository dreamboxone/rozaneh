/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 * Update row layout adapted from Zirgozar.
 * https://github.com/dreamboxone/zirgozar
 */
'use strict';
'require view';
'require rpc';
'require poll';
'require rozaneh.ui as zui';
'require rozaneh.icons-v2 as icons';

var check = rpc.declare({object:'rozaneh',method:'check_update',expect:{'':{}}});
var install = rpc.declare({object:'rozaneh',method:'install_update',params:['scope','force'],expect:{'':{}}});
var status = rpc.declare({object:'rozaneh',method:'update_status',expect:{'':{}}});
var system = rpc.declare({object:'rozaneh',method:'status',expect:{'':{}}});
var dependencies = rpc.declare({object:'rozaneh',method:'install_dependencies',params:['scope'],expect:{'':{}}});
var fa = localStorage.getItem('rozaneh-language') !== 'en';
function tr(en,persian){return fa?persian:en;}
function translated(text){
 var messages={
  'No usable release manifest was found. Check dependencies, connectivity and published GitHub releases.':'نسخه سازگار منتشرشده پیدا نشد. دسترسی به GitHub و انتشار بسته‌های این معماری را بررسی کنید.',
  'Check for updates first.':'ابتدا نسخه جدید را بررسی کنید.',
  'An update is already running.':'به‌روزرسانی دیگری در حال اجراست.',
  'Update started. The complete app will be downloaded.':'دانلود و نصب روزنه شروع شد.',
  'Reinstallation started.':'نصب مجدد شروع شد.',
  'Dependency installation started.':'نصب نیازمندی‌های روتر شروع شد.',
  'All dependencies are installed.':'همه نیازمندی‌های روتر نصب هستند.'
 };
 return fa&&messages[text]?messages[text]:text;
}

return view.extend({
 handleSaveApply:null,handleSave:null,handleReset:null,
 load:function(){return Promise.all([status().catch(function(){return {};}),system().catch(function(){return {};})]);},
 render:function(initial){
  var report=initial[0]||{}, router=initial[1]||{}, notes={}, requesting='', installAsked=0;
  var app=E('div'),requirements=E('div'),log=E('pre',{'class':'zd-log'});
  function note(key,text,kind){notes[key]={text:translated(text||''),kind:kind||'info'};}
  function busy(){return !!requesting||report.installing||Date.now()-installAsked<8000;}
  function button(label,tone,key,fn,icon,disabled){
   var actionKey=key+':'+(icon||'refresh');
   return E('button',{type:'button','data-action':actionKey,'class':'mk-btn '+tone+' mk-small'+(requesting===actionKey?' mk-running':''),disabled:busy()||disabled,click:function(){
    if(busy())return;
    requesting=actionKey;note(key,'');draw();
    return Promise.resolve().then(fn).then(function(result){
     if(result&&result.message)note(key,result.message,result.ok?'info':'error');
     if(result&&result.components)report=result;
    }).catch(function(error){note(key,error.message||String(error),'error');}).finally(function(){requesting='';return refresh();});
   }},[icons.icon(icon||'refresh'),E('span',{},label)]);
  }
  function checkButton(key,tone,label){return button(label||tr('Check','بررسی'),tone||'primary',key,function(){return check();},'refresh');}
  function installButton(key,scope,component,label){
   var newer=!!component.available, same=!!component.latest&&component.latest===component.current;
   return button(label||(newer?tr('Update to ','به‌روزرسانی به ')+zui.versionText(component.latest):tr('Reinstall','نصب مجدد')),newer?'success':'soft-blue',key,function(){
    return install(scope,!newer).then(function(result){if(result.ok)installAsked=Date.now();return result;});
   },'download',!newer&&!same);
  }
  function chip(text,tone,title){return E('bdi',{'class':'mk-chip '+(tone||''),dir:'ltr',title:title||text},text);}
  function row(label,value,actions,key){
   var n=notes[key];
   return E('div',{'class':'zd-update-row'},[
    E('div',{'class':'zd-update-label'},label),E('div',{'class':'zd-update-value'},value),E('div',{'class':'zd-update-actions'},actions),
    E('div',{'class':'zd-update-note '+(n?n.kind:''),role:'status',hidden:!n||!n.text},n?n.text:'')
   ]);
  }
  function state(component){
   return component.available&&component.latest?chip((fa?'← ':'→ ')+zui.versionText(component.latest),'info'):E('span',{'class':'mk-muted'},component.latest?tr('Up to date','به‌روز است'):tr('Not checked','بررسی نشده'));
  }
  function engineVersion(value){
   if(!value||value==='unknown')return tr('Unknown','نامشخص');
   var dated=String(value).match(/^v?(\d{4}\.\d{2}\.\d{2})(?:\.|$)/);
   if(dated)return dated[1];
   if(/^7db7b73[0-9a-f]*$/.test(value))return '2026.10.24';
   return value;
  }
  function draw(){
   var components=report.components||[];
   var core=components.filter(function(c){return c.name==='rozaneh';})[0]||{current:router.version||'—'};
   var engineActions=[checkButton('engine')];
   if(core.latest_engine&&core.latest_engine!=='unknown')engineActions.push(installButton('engine','all',core,core.available?tr('Update with Rozaneh','به‌روزرسانی همراه روزنه'):tr('Reinstall with Rozaneh','نصب مجدد همراه روزنه')));
   app.replaceChildren(
    row(zui.name(),[chip('【 '+zui.versionText(core.current)+' 】','ok'),state(core)],[checkButton('app'),installButton('app','all',core)],'app'),
    row(tr('CottenDNS engine','موتور CottenDNS'),[
     chip(engineVersion(core.current_engine),'ok',core.current_engine),
     core.latest_engine&&core.latest_engine!=='unknown'?E('span',{},[tr('With the available package: ','همراه بسته آماده: '),chip(engineVersion(core.latest_engine),'info',core.latest_engine)]):E('span',{'class':'mk-muted'},tr('No compatible package checked','بسته سازگار بررسی نشده')),
    ],engineActions,'engine')
   );
   requirements.replaceChildren(row(tr('Packages','بسته‌ها'),[E('span',{'class':'mk-chip '+(router.missing?'warn':'ok')},router.missing?tr('Missing packages','بسته‌های موردنیاز'):tr('Everything is ready','همه‌چیز آماده است')),router.missing?E('code',{},router.missing):''],router.missing?[button(tr('Install','نصب'),'primary','deps',function(){return dependencies('lan');},'update')]:[],'deps'));
   if(router.installing)requirements.appendChild(E('p',{'class':'mk-muted'},tr('Installing required packages…','در حال نصب بسته‌های موردنیاز…')));
   log.textContent=report.log||tr('No update log yet.','هنوز گزارشی از به‌روزرسانی وجود ندارد.');
  }
  var pending;
  function refresh(){
   if(pending)return pending;
   pending=Promise.all([status(),system()]).then(function(results){report=results[0];router=results[1];draw();}).catch(function(){draw();}).finally(function(){pending=null;});
   return pending;
  }
  var root=E('div',{'class':'mk'+(localStorage.getItem('rozaneh-dark')==='1'?' mk-dark':''),dir:fa?'rtl':'ltr'},[
   E('link',{rel:'stylesheet',href:L.resource('rozaneh/theme.css')+'?v=1.0.0-r1'}),E('link',{rel:'stylesheet',href:L.resource('rozaneh/app.css')+'?v=1.0.0-r1'}),
   zui.hero(tr('Updates','به‌روزرسانی'),router.version),zui.menu('update'),
   zui.card(tr('Update','به‌روزرسانی'),'update','#1e88e5',app),
   zui.card(tr('Requirements','نیازمندی‌های روتر'),'shield','#f59e0b',requirements),
   E('details',{'class':'mk-card zd-update-log'},[E('summary',{},tr('Update log','گزارش به‌روزرسانی')),log]),zui.footer()
  ]);
  draw();poll.add(refresh,5);return root;
 }
});

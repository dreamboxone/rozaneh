/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
'use strict';
'require view';
'require rpc';
'require uci';
'require poll';
'require rozaneh.icons-v2 as icons';
'require rozaneh.ui as zui';

var status = rpc.declare({ object: 'rozaneh', method: 'status', expect: { '': {} } });
var control = rpc.declare({ object: 'rozaneh', method: 'control', params: ['action'], expect: { '': {} } });
var install = rpc.declare({ object: 'rozaneh', method: 'install_dependencies', params: ['scope'], expect: { '': {} } });
var sites = rpc.declare({ object: 'rozaneh', method: 'sites', params: ['site'], expect: { '': {} } });
var paths = rpc.declare({object:'rozaneh',method:'paths',expect:{'':{}}});
var traffic = rpc.declare({ object: 'rozaneh', method: 'traffic', expect: { '': {} } });
var fa = localStorage.getItem('rozaneh-language') !== 'en';
function tr(en, persian) { return fa ? persian : en; }
function uptimeText(value) {
 var seconds = Number(value || 0), days = Math.floor(seconds / 86400);
 return (days ? days + tr('d ', ' روز ') : '') + [Math.floor(seconds / 3600) % 24, Math.floor(seconds / 60) % 60, seconds % 60].map(function(n) { return String(n).padStart(2, '0'); }).join(':');
}
function translatedMessage(text) {
 var messages = {
  'Service started.': 'سرویس روشن شد.',
  'Service stopped.': 'سرویس خاموش شد.',
  'Save the server zone and public key first.': 'ابتدا دامنه و کلید عمومی سرور را ذخیره کنید.',
  'All dependencies are installed.': 'همه وابستگی‌ها نصب هستند.',
  'Dependencies installed successfully.': 'وابستگی‌ها با موفقیت نصب شدند.',
  'Dependency installation started.': 'نصب وابستگی‌ها شروع شد.',
  'Service could not start. Check logs.': 'سرویس روشن نشد؛ گزارش‌ها را بررسی کنید.',
  'Could not change the kill switch.': 'تغییر کلید قطع اضطراری انجام نشد.'
  ,'Another transparent proxy is active. Stop it before starting Rozaneh.': 'پراکسی دیگری روی روتر فعال است. ابتدا آن را خاموش کنید.'
  ,'Zirgozar is on. Rozaneh will not start. Turn Zirgozar off first.': 'روزنه نمی‌تواند همزمان با زیرگذر فعال باشد. لطفا زیرگذر را خاموش کنید.'
 };
 if (!fa) return text;
 if (text.indexOf('Missing packages: ') === 0) return text.replace('Missing packages: ', 'بسته‌های موردنیاز: ');
 return messages[text] || text;
}

return view.extend({
 handleSaveApply: null,
 handleSave: null,
 handleReset: null,
 load: function() { return Promise.all([uci.load('rozaneh'), status().catch(function() { return {}; })]); },
 render: function(data) {
  var self = this;
  var current = data[1];
  var root = E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' });
  var serviceValue = E('div', { 'class': 'zd-value' });
  var depValue = E('div', { 'class': 'zd-value' });
  var outcome = E('div', { 'class': 'zd-result', role: 'status', hidden: true });
  var depInfo = E('p');
  var connectionDetail = E('p', { 'class': 'zd-connection-detail' });
  var powerBusy = false;
  var powerSwitch = E('input', { type:'checkbox', role:'switch', 'class':'mk-switch', 'aria-label':tr('Main switch','کلید اصلی'), checked:current.enabled ? '' : null, change:function() {
   if (powerBusy) return;
   powerBusy=true; powerSwitch.disabled=true;
   control(powerSwitch.checked?'start':'stop').then(function(result) {
    if (result.ok) outcome.hidden=true;
    else message(result);
   }).catch(function(error){message({ok:false,message:error.message||String(error)});}).finally(function(){
    powerBusy=false;refresh().finally(function(){powerSwitch.disabled=false;});
   });
  }});
  var mainSwitch=E('label',{'class':'mk-switch-row zd-main-switch'},[E('span',{},tr('Main switch','کلید اصلی')),powerSwitch]);
  var killBusy = false;
  var killSwitch = E('input', { type:'checkbox', role:'switch', 'class':'mk-switch', 'aria-label':tr('Kill switch','کلید قطع اضطراری (kill switch)'), checked:current.lan_killswitch ? '' : null, change:function() {
   if (killBusy) return;
   killBusy=true; killSwitch.disabled=true;
   control(killSwitch.checked?'killswitch_on':'killswitch_off').then(function(result) {
    if (result.ok) outcome.hidden=true;
    else message(result);
   }).catch(function(error){message({ok:false,message:tr('Could not change the kill switch.','تغییر کلید قطع اضطراری انجام نشد.')});}).finally(function(){
    killBusy=false;refresh().finally(function(){killSwitch.disabled=false;});
   });
  }});
  var killControl=E('div',{'class':'zd-kill-control'},[
   E('div',{'class':'cbi-value-title'},tr('Kill switch','کلید قطع اضطراری (kill switch)')),
   E('div',{'class':'cbi-value-field'},E('label',{'class':'mk-switch-row'},[killSwitch])),
   E('div',{'class':'cbi-value-description',hidden:true},tr('Blocks direct public internet and DNS on selected LAN interfaces when the tunnel stops. Local networks stay reachable. This applies immediately and may disconnect internet access while Rozaneh is off.','هنگام قطع تونل، اینترنت عمومی و DNS مستقیم رابط‌های انتخاب‌شده بسته می‌شود. شبکه داخلی در دسترس می‌ماند. تغییر این کلید فوری است و هنگام خاموش بودن روزنه ممکن است اینترنت قطع شود.'))
  ]);
  var switches=E('div',{'class':'zd-switches'},[mainSwitch,killControl]);
  var uptime = E('bdi', {}, uptimeText(current.uptime));
  var installButton, dependencyCard;
  var pathRows = E('tbody');
  var pathSummary = E('p');
  var pathTable = E('div',{'class':'zd-path-table',hidden:true},E('table',{},[E('thead',{},E('tr',{},[tr('Resolver','ریزالور'),tr('Transport','روش'),tr('State','وضعیت'),tr('Latency','تاخیر'),tr('Loss','افت'),tr('Name length','طول نام'),tr('Answer bytes','اندازه پاسخ')].map(function(label){return E('th',{},label);}))),pathRows]));
  function inactivePaths() {
   pathTable.hidden=true;
   pathRows.replaceChildren();
   pathSummary.textContent=!current.enabled?tr('Turn on Rozaneh to display resolver path quality.','برای نمایش کیفیت مسیرها، روزنه را روشن کنید.'):tr('Waiting for the service to start…','در انتظار شروع سرویس…');
  }
  inactivePaths();
  var siteValues = {};
  var siteCards = ['instagram', 'reddit', 'x'].map(function(name, i) {
   var value = E('span', { 'class': 'pwp-check-value' }, tr('Check', 'آزمون اتصال'));
   var button = E('button', { type:'button', 'class': 'pwp-social-check', style: '--site-colour:' + ['#df458e', '#ec693c', '#329dc2'][i], click:function(){return runSites(name);} }, [E('span',{'class':'pwp-social-icon'},icons.brand(name)),E('span',{'class':'pwp-social-name'},['Instagram','Reddit','X'][i]),value]);
   siteValues[name] = { value: value, button: button };
   return button;
  });
  function runSites(which) {
   var names = which === 'all' ? Object.keys(siteValues) : [which];
   names.forEach(function(name){siteValues[name].button.disabled=true;siteValues[name].value.textContent=tr('Checking…','در حال بررسی…');siteValues[name].value.className='pwp-check-value mk-wait';});
   return sites(which).then(function(result){
    if(result.error)throw new Error(result.error);
    (result.sites||[]).forEach(function(item){
     var card=siteValues[item.name];if(!card)return;
     var result=zui.siteResult(item), ms=result.ms;
     card.value.textContent=ms>0?ms+' '+tr('ms','م‌ثانیه'):tr('N/A','ناموفق');
     card.value.className='pwp-check-value '+result.tone;
     card.button.title='HTTP '+item.http+(item.state==='restricted'?' · '+tr('Restricted response','پاسخ محدودشده'):'')+(Number(item.exit_code)?' · curl '+item.exit_code:'');
    });
   }).catch(function(){names.forEach(function(name){siteValues[name].value.textContent=tr('N/A','ناموفق');siteValues[name].value.className='pwp-check-value mk-bad';});}).finally(function(){names.forEach(function(name){siteValues[name].button.disabled=false;});});
  }
  var usageData={},usagePeriod=localStorage.getItem('rozaneh-usage-period')||'today',usageValues={};
  function renderUsage(){var d=usageData[usagePeriod]||{};['up','down','total'].forEach(function(key){var bytes=key==='total'?Number(d.up||0)+Number(d.down||0):Number(d[key]||0);usageValues[key].textContent=(bytes/1000000000).toFixed(2);});}
  var usageButtons=[];
  var usageCard=E('section',{'class':'mk-card pwp-usage-card',style:'--accent:#10b981'},[
   E('div',{'class':'pwp-usage-title'},tr('Usage (GB)','مصرف (گیگابایت)')),
   E('div',{'class':'pwp-usage-body'},[
    E('div',{'class':'pwp-usage-values'},['up','down','total'].map(function(key,i){var value=E('strong',{dir:'ltr'},'0.00');usageValues[key]=value;return E('div',{'class':'pwp-usage-value pwp-usage-'+key},[E('span',{},[tr('Upload','آپلود'),tr('Download','دانلود'),tr('Total','مجموع')][i]),value]);})),
    E('div',{'class':'pwp-usage-periods',role:'tablist'},['today','week','month'].map(function(period,i){var b=E('button',{type:'button',role:'tab','aria-selected':String(period===usagePeriod),title:[tr('Today (UTC)','امروز (UTC)'),tr('Last 7 days','۷ روز اخیر'),tr('Last 30 days','۳۰ روز اخیر')][i],click:function(){usagePeriod=period;localStorage.setItem('rozaneh-usage-period',period);usageButtons.forEach(function(item){item.b.setAttribute('aria-selected',String(item.p===period));});renderUsage();}},[tr('1d','۱ر'),tr('1w','۷ر'),tr('1m','۳۰ر')][i]);usageButtons.push({b:b,p:period});return b;}))
   ])
  ]);
  function message(result) {
   outcome.hidden=false;
   outcome.className = 'zd-result ' + (result.ok ? 'ok' : 'error');
   outcome.textContent = translatedMessage(result.message || tr('Request failed.', 'درخواست انجام نشد.'));
  }
  function action(label, cls, task) {
   var b = E('button', { 'class': 'mk-btn ' + cls, type: 'button', click: function() {
    b.disabled = true;
    Promise.resolve().then(task).catch(function(err) { message({ ok: false, message: err.message || String(err) }); }).finally(function() { b.disabled = false; refresh(); });
   } }, label);
   return b;
  }
  function refreshPaths() {
   if(!current.enabled||!current.running){inactivePaths();return Promise.resolve();}
   return paths().then(function(report){
    if(!current.enabled||!current.running){inactivePaths();return;}
    var items=report.paths||[], healthy=0;
    pathTable.hidden=!!report.stale||!items.length;
    pathRows.replaceChildren();
    items.forEach(function(p){
     var good=p.verified&&!p.suspended&&p.loss<0.5;
     if(good)healthy++;
     var state=p.suspended?tr('Paused','متوقف'):p.verified?tr('Verified','تایید شده'):tr('Unverified','تایید نشده');
     pathRows.appendChild(E('tr',{},[E('td',{dir:'ltr'},p.resolver),E('td',{},(p.transport||'').toUpperCase()),E('td',{style:'color:'+(good?'#059669':'#d97706')},state),E('td',{},p.rtt?Math.round(p.rtt/1000000)+' '+tr('ms','م‌ثانیه'):'—'),E('td',{},Math.round((p.loss||0)*100)+'%'),E('td',{},String(p.name_len||0)),E('td',{},String(p.answer_limit||0))]));
    });
    pathSummary.textContent=report.stale||!items.length?tr('Checking resolvers; path quality will appear automatically.','در حال بررسی ریزالورها؛ کیفیت مسیرها به‌صورت خودکار نمایش داده می‌شود.'):tr('Healthy paths: ','مسیرهای سالم: ')+healthy+' / '+items.length+' · '+(report.established?tr('Authenticated session established','نشست احراز هویت شده برقرار است'):tr('Waiting for an authenticated session','در انتظار برقراری نشست احراز هویت شده'));
   }).catch(function(){pathTable.hidden=true;pathSummary.textContent=tr('Resolver diagnostics are unavailable.','آمار مسیرها در دسترس نیست.');});
  }
  function refresh(initialStatus) {
   traffic().then(function(data){usageData=data;renderUsage();}).catch(function(){});
   return (initialStatus ? Promise.resolve(initialStatus) : status()).then(function(s) {
    current=s;
    var state=zui.connectionState(s);
    var names={stopped:tr('Off','خاموش'),blocked:tr('Needs attention','نیاز به بررسی'),starting:tr('Starting…','در حال راه‌اندازی…'),connecting:tr('Connecting…','در حال اتصال…'),connected:tr('Connected','متصل')};
    var details={stopped:'',blocked:s.conflict?(s.conflict_name==='zirgozar'?tr('Zirgozar is on. Rozaneh will not start.','روزنه نمی‌تواند همزمان با زیرگذر فعال باشد. لطفا زیرگذر را خاموش کنید.'):tr('Stop the other transparent proxy first.','ابتدا پراکسی دیگر روتر را خاموش کنید.')):tr('Install the required packages.','بسته‌های موردنیاز را نصب کنید.'),starting:tr('Starting the tunnel and LAN routing…','در حال راه‌اندازی تونل و شبکه داخلی…'),connecting:tr('Waiting for external internet access…','در حال دسترسی به اینترنت...'),connected:''};
    connectionDetail.textContent=details[state];
    connectionDetail.hidden=!details[state];
    if (!powerBusy) powerSwitch.checked=!!s.enabled;
    if (!killBusy) killSwitch.checked=!!s.lan_killswitch;
    uptime.textContent = uptimeText(s.uptime);
    serviceValue.replaceChildren(E('span', { 'class': 'mk-chip '+(state==='connected'?'ok':state==='blocked'?'error':state==='stopped'?'':'warn'), 'aria-live':'polite' }, names[state]));
    serviceValue.classList.toggle('zd-running', state==='connected');
    depValue.textContent = s.missing ? tr('Needs setup', 'نیاز به نصب') : tr('Ready', 'آماده');
    depInfo.textContent = s.missing ? tr('Missing packages: ', 'بسته‌های ناقص: ') + s.missing : tr('All required packages are installed.', 'همه بسته‌های موردنیاز نصب هستند.');
    if (s.installing) depInfo.textContent = tr('Installing packages…', 'در حال نصب بسته‌ها…');
    if (s.install_log) depInfo.textContent += '\n' + translatedMessage(s.install_log);
    installButton.disabled = !s.missing || s.installing;
    dependencyCard.hidden=!s.missing&&!s.installing;
    return refreshPaths();
   }).catch(function() { serviceValue.textContent = tr('Unavailable', 'در دسترس نیست'); });
  }
  function metric(label, value, accent) { return E('div', { 'class': 'mk-metric', style: '--accent:' + accent }, [E('div', { 'class': 'zd-label' }, label), value]); }
  installButton = action(tr('Install missing dependencies', 'نصب وابستگی‌های ناقص'), 'primary', function() {
   outcome.textContent = tr('Installing packages. This may take a moment…', 'بسته‌ها در حال نصب هستند؛ کمی صبر کنید…');
   return install().then(message);
  });
  var hero = zui.hero(tr('A powerful DNS client for your router', 'کلاینت DNS قدرتمند برای روتر شما'), current.version, uptime);
  return Promise.resolve().then(function() {
   root.appendChild(E('link', { rel: 'stylesheet', href: L.resource('rozaneh/theme.css') + '?v=1.0.0-r1' }));
   root.appendChild(E('link', { rel: 'stylesheet', href: L.resource('rozaneh/app.css') + '?v=1.0.0-r1' }));
   root.appendChild(hero);
   root.appendChild(zui.menu('dashboard'));
   root.appendChild(E('div', { 'class': 'mk-grid mk-metrics pwp-status-row zd-top-row' }, [E('div',{'class':'mk-card zd-status-card',style:'--accent:#10b981'},[E('div',{'class':'zd-status-heading'},[zui.title(tr('Status','وضعیت'),'status','#10b981'),serviceValue]),connectionDetail]),E('div', { 'class': 'mk-card', style: '--accent:#06b6d4' }, [zui.title(tr('Router specifications', 'مشخصات روتر'),'router','#06b6d4'), E('div',{'class':'zd-router-details',dir:'ltr'},[E('p', {}, current.model || '—'), E('small', {}, current.os || '—'), E('small', {}, current.architecture || '—')])]),E('section',{'class':'mk-card pwp-social-card',style:'--accent:#db2777'},siteCards),usageCard]));
   root.appendChild(outcome);
   root.appendChild(E('div', { 'class': 'mk-card zd-basic-settings', style: '--accent:#8b5cf6' }, [zui.title(tr('Basic settings','تنظیمات پایه'),'settings','#8b5cf6'),switches]));
   dependencyCard=E('div', { 'class': 'mk-card', style: '--accent:#f59e0b', hidden:!current.missing }, [zui.title(tr('Requirements','نیازمندی‌های روتر'),'shield','#f59e0b'),depValue,depInfo,installButton]);
   root.appendChild(dependencyCard);
   root.appendChild(E('div',{'class':'mk-card'},[zui.title(tr('Resolver path quality','کیفیت مسیرهای ریزالور'),'status','#06b6d4'),pathSummary,pathTable]));
   root.appendChild(zui.footer());
   root.querySelectorAll('input[type=text]').forEach(function(input) { input.dir = 'ltr'; });
   refresh(current); poll.add(function() { return refresh(); }, 5);
   return zui.optionHelp(root);
  });
 }
});

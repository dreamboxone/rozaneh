'use strict';
'require view';
'require rpc';
'require poll';

var check = rpc.declare({object:'zoordns',method:'check_update',expect:{'':{}}});
var install = rpc.declare({object:'zoordns',method:'install_update',expect:{'':{}}});
var status = rpc.declare({object:'zoordns',method:'update_status',expect:{'':{}}});
var fa = localStorage.getItem('zoordns-language') !== 'en';
function tr(en,persian){return fa?persian:en;}
return view.extend({
 handleSaveApply:null,handleSave:null,handleReset:null,
 load:function(){return status().catch(function(){return {};});},
 render:function(initial){
  var result=E('div',{'class':'zd-result',role:'status'},tr('Check GitHub for a published release.','نسخه منتشرشده در GitHub را بررسی کنید.'));
  var table=E('div',{'class':'mk-grid'}),log=E('pre',{'class':'zd-log'});
  var installButton;
  function show(s){
   if(s.message){result.textContent=s.message;return;}
   table.replaceChildren();
   var available=false;
   (s.components||[]).forEach(function(c){
    available=available||c.available;
    table.appendChild(E('div',{'class':'mk-card',style:'--accent:#6366f1'},[E('h3',{},c.name==='zoordns'?tr('Tunnel engine','موتور اتصال'):tr('LuCI interface','رابط LuCI')),E('p',{},tr('Installed: ','نصب‌شده: ')+c.current),E('p',{},tr('Latest: ','آخرین نسخه: ')+(c.latest||'—')),E('small',{},c.available?tr('Update available','نسخه جدید موجود است'):tr('No change','بدون تغییر'))]));
   });
   log.textContent=s.log||tr('No update log yet.','هنوز گزارشی از به‌روزرسانی وجود ندارد.');
   installButton.disabled=!available||s.installing;
   if(s.installing)result.textContent=tr('Updating… The interface may reconnect.','در حال به‌روزرسانی… ممکن است رابط دوباره متصل شود.');
  }
  function button(label,fn){var b=E('button',{'class':'mk-btn primary',type:'button',click:function(){b.disabled=true;Promise.resolve().then(fn).catch(function(e){result.textContent=e.message||String(e);}).finally(function(){b.disabled=false;status().then(show).catch(function(){});});}},label);return b;}
  installButton=button(tr('Install changed components','نصب اجزای تغییرکرده'),function(){return install().then(function(r){result.textContent=r.message||'';});});
  var root=E('div',{'class':'mk'+(localStorage.getItem('zoordns-dark')==='1'?' mk-dark':''),dir:fa?'rtl':'ltr'},[
   E('link',{rel:'stylesheet',href:L.resource('zoordns/theme.css') + '?v=0.1.0-r3'}),E('link',{rel:'stylesheet',href:L.resource('zoordns/app.css') + '?v=0.1.0-r3'}),
   E('div',{'class':'mk-hero'},[E('div',{},[E('h2',{'class':'zd-heading'},'zoorDNS'),E('p',{},tr('Updates','به‌روزرسانی'))]),E('a',{'class':'mk-btn glass',href:L.url('admin/services/zoordns/dashboard')},tr('Home','صفحه اصلی'))]),
   E('div',{'class':'mk-card',style:'--accent:#06b6d4'},[E('h3',{},tr('Component updates','به‌روزرسانی اجزا')),E('p',{},tr('Only changed components are downloaded. Each changed component is a complete package, verified by SHA-256 and selected for this router architecture.','فقط اجزای تغییرکرده دانلود می‌شوند. هر بخش یک بسته کامل است؛ معماری روتر و SHA-256 فایل بررسی می‌شوند.')),E('small',{},tr('Configuration is preserved. A tunnel engine update briefly interrupts the connection.','تنظیمات حفظ می‌شوند. به‌روزرسانی موتور اتصال، تونل را برای مدتی کوتاه قطع می‌کند.')),E('div',{'class':'mk-row'},[button(tr('Check for updates','بررسی نسخه جدید'),function(){result.textContent=tr('Checking…','در حال بررسی…');return check().then(show);}),installButton]),result]),table,
   E('div',{'class':'mk-card',style:'--accent:#f59e0b'},[E('h3',{},tr('Update log','گزارش به‌روزرسانی')),log])
  ]);
  show(initial);poll.add(function(){return status().then(show).catch(function(){});},5);return root;
 }
});

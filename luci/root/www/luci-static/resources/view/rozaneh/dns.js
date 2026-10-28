/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
'use strict';
'require view';
'require form';
'require uci';
'require ui';
'require rozaneh.ui as zui';
var fa = localStorage.getItem('rozaneh-language') !== 'en';
function tr(en, persian) { return fa ? persian.replace(/[A-Za-z][A-Za-z0-9:/=.?_-]*/g,'\u2066$&\u2069') : en; }
return view.extend({
 addFooter: function() {
  var self = this;
  return E('div', { 'class': 'mk zd-footer', dir: fa ? 'rtl' : 'ltr' }, [
   E('button', { 'class': 'mk-btn success', click: ui.createHandlerFn(this, function(ev) { return self.handleSaveApply(ev, '0'); }), disabled: (self.map && self.map.readonly) || null }, tr('Save & apply', 'ذخیره و اعمال')),
   E('button', { 'class': 'mk-btn primary', click: ui.createHandlerFn(this, 'handleSave'), disabled: (self.map && self.map.readonly) || null }, tr('Save', 'ذخیره')),
   E('button', { 'class': 'mk-btn soft-red', click: ui.createHandlerFn(this, 'handleReset'), disabled: (self.map && self.map.readonly) || null }, tr('Reset', 'بازنشانی'))
  ]);
 },
 load: function() { return uci.load('rozaneh'); },
 render: function() {
  var m = new form.Map('rozaneh', 'DNS');
  this.map = m;
  var s = m.section(form.NamedSection, 'main', 'client');
  s.anonymous = true;
  var o;
  var outcome = E('p', { 'class':'zd-result', role:'status', hidden:true });
  function plaintextOption(option){option.depends({resolver_transport:'auto',encrypted_dns_only:'0'});option.retain=true;}
  o=s.option(form.ListValue,'resolver_transport',tr('Resolver connection','روش اتصال به ریزالور'));
  o.value('auto',tr('As specified in each address','طبق نشانی هر ریزالور'));
  o.value('dot','DoT');o.value('doh','DoH');o.value('doh3','DoH3');o.value('doq','DoQ');o.default='auto';o.rmempty=false;
  o.description=tr('The resolver must support the selected protocol. Choose a protocol or specify it in each resolver URL. Explicit encrypted protocols disable plaintext discovery. For bare IP:53 addresses, port 53 is replaced with the protocol default: 853 for DoT/DoQ, 443 for DoH/DoH3. Use a protocol URL for a custom port.','ریزالور باید پروتکل انتخابی را پشتیبانی کند. پروتکل را انتخاب کنید یا در نشانی هر ریزالور بنویسید. انتخاب پروتکل رمزگذاری‌شده، کشف مسیر ساده را غیرفعال می‌کند. در آدرس ساده با پورت ۵۳، پورت پیش‌فرض پروتکل استفاده می‌شود: ۸۵۳ برای DoT و DoQ و ۴۴۳ برای DoH و DoH3. برای پورت دلخواه، نشانی همراه پروتکل وارد کنید.');
  o=s.option(form.Flag,'encrypted_dns_only',tr('Encrypted DNS only','فقط DNS رمزگذاری‌شده'));o.default='0';o.rmempty=false;
  o.description=tr('Rejects plain DNS and TCP/53, disables plaintext resolver discovery and cached discovery paths, and requires numeric resolver addresses or bootstrap=IP. There is no downgrade to plaintext after errors. This protects the router-to-resolver hop.','DNS ساده را رد می‌کند و کشف ریزالورهای ساده و مسیرهای ذخیره‌شده کشف را کنار می‌گذارد. ریزالور باید آدرس عددی یا bootstrap=IP داشته باشد. هنگام خطا به DNS ساده برنمی‌گردد. این حفاظت مربوط به مسیر روتر تا ریزالور است.');
  o=s.option(form.Flag,'http3_fallback',tr('Allow DoH3 to retry as DoH','جایگزینی DoH3 با DoH هنگام خطا'));o.default='1';o.rmempty=false;
  o.description=tr('Optional. If HTTP/3 fails, retry the same resolver using HTTPS with TLS verification. Plain DNS is never used. This only applies to DoH3.','اختیاری است. اگر HTTP/3 کار نکند، همان ریزالور با HTTPS و بررسی گواهی TLS امتحان می‌شود. DNS ساده استفاده نمی‌شود. این گزینه فقط برای DoH3 است.');
  o = s.option(form.Value, 'resolvers', tr('Resolvers', 'ریزالورها'), tr('Comma-separated IP:port addresses, for example 1.1.1.1:53,8.8.8.8:53.', 'آدرس‌ها را با ویرگول جدا کنید؛ نمونه: 1.1.1.1:53,8.8.8.8:53'));
  o.placeholder = '1.1.1.1:53,tls://dns.google?bootstrap=8.8.8.8,https://cloudflare-dns.com/dns-query?bootstrap=1.1.1.1';
  function code(value) { return E('bdi', {dir:'ltr',style:'white-space:nowrap'}, value); }
  o.description = fa ? E('span', {}, ['مسیرها را با ویرگول جدا کنید: ',code('IP:port'),' یا ',code('tcp://'),' یا ',code('dot://'),' یا ',code('doh://'),' یا ',code('doh3://'),' یا ',code('doq://'),'. پیشوندهای ',code('tls://'),' و ',code('https://'),' هم پذیرفته می‌شوند. برای نام ریزالور در حالت امن، ',code('bootstrap=IP'),' لازم است. گواهی TLS همیشه بررسی می‌شود.']) : 'Comma-separated IP:port, tcp://, dot://, doh://, doh3:// or doq:// endpoints. tls:// and https:// aliases are also accepted. Encrypted-only mode requires a numeric resolver IP or bootstrap=IP. TLS certificates are always verified.';
  o.validate=function(section,value){
   function selected(name){var field=m.lookupOption(name,section)[0],widget=field&&field.getUIElement&&field.getUIElement(section);return widget?widget.getValue():uci.get('rozaneh',section,name);}
   if(selected('encrypted_dns_only')==='1'&&(!selected('resolver_transport')||selected('resolver_transport')==='auto')){
    var addresses=String(value||'').split(/[\n,]/).map(function(v){return v.trim();}).filter(Boolean);
    if(!addresses.length||addresses.some(function(v){return !/^(dot|tls|doh|https|doh3|doq):\/\//i.test(v);}))return tr('Encrypted DNS only is enabled, but the Resolvers field is empty or contains a plain DNS address such as 1.1.1.1:53. In Resolver connection at the top of this page, choose DoH, DoT, DoH3 or DoQ supported by your resolver. Alternatively, keep By each resolver address and enter encrypted addresses such as https://1.1.1.1/dns-query. The DNS protocol menu with UDP and TCP options does not select encryption.','گزینه «فقط DNS رمزگذاری شده» روشن است، اما بخش «ریزالورها» خالی است یا نشانی DNS ساده مانند 1.1.1.1:53 دارد. در منوی «روش اتصال به ریزالور» در بالای همین صفحه، یکی از گزینه های DoH، DoT، DoH3 یا DoQ را که ریزالور شما پشتیبانی می کند انتخاب کنید. یا گزینه «طبق نشانی هر ریزالور» را نگه دارید و نشانی رمزگذاری شده مانند https://1.1.1.1/dns-query وارد کنید. منوی «پروتکل DNS» با گزینه های UDP و TCP برای انتخاب رمزگذاری نیست.');
   }
   return true;
  };
  o = s.option(form.Button, '_automatic_paths', tr('Severe restrictions profile', 'پروفایل محدودیت شدید'));
  plaintextOption(o);
  o.description = zui.optionDescription('_automatic_paths');
  o.inputtitle=tr('Use resilient defaults', 'استفاده از تنظیمات خودکار');o.inputstyle='apply';
  o.onclick=function(){
   var values={resolvers:'1.1.1.1:53,1.0.0.1:53,9.9.9.9:53,https://1.1.1.1/dns-query,https://dns.google/dns-query?bootstrap=8.8.8.8,tls://1.1.1.1:853',tcp:'on',discover:'1',discovery_cache:'1',safe_qname:'101',max_answer:'512'};
   Object.keys(values).forEach(function(name){var option=m.lookupOption(name,'main')[0];if(option){var widget=option.getUIElement('main');if(widget)widget.setValue(values[name]);}else{uci.set('rozaneh','main',name,values[name]);}});
   outcome.textContent=tr('Profile selected. Save and apply to activate.', 'پروفایل انتخاب شد. برای فعال شدن، ذخیره و اعمال کنید.');
   outcome.hidden=false;
  };
  o = s.option(form.ListValue, 'tcp', tr('DNS protocol', 'پروتکل DNS'));
  plaintextOption(o);
  o.description = zui.optionDescription('tcp');
  o.value('auto', tr('Automatic', 'خودکار')); o.value('on', tr('UDP and TCP', 'UDP و TCP')); o.value('only', tr('TCP only', 'فقط TCP')); o.value('off', tr('UDP only', 'فقط UDP')); o.default = 'auto';
  o = s.option(form.Flag, 'discover', tr('Discover resolvers', 'کشف ریزالورهای بیشتر')); o.rmempty = false;
  plaintextOption(o);
  o.description = zui.optionDescription('discover');
  o = s.option(form.ListValue, 'country', tr('Discovery region', 'منطقه کشف'));
  plaintextOption(o);
  o.description = zui.optionDescription('country');
  o.value('IR',tr('Iran','ایران'));o.value('RU',tr('Russia','روسیه'));o.value('CN',tr('China','چین'));o.default='IR';o.rmempty=false;
  o = s.option(form.Value, 'discovery_candidates', tr('Extra discovery candidates', 'ریزالورهای اضافی برای کشف')); o.placeholder='1.1.1.1:53,9.9.9.9:53';
  plaintextOption(o);
  o.description = zui.optionDescription('discovery_candidates');
  function addCandidates(addresses) {
   var field = m.lookupOption('discovery_candidates','main')[0];
   var widget = field && field.getUIElement('main');
   if (!widget) return;
   var existing = String(widget.getValue()||'').split(',').map(function(value){return value.trim();}).filter(Boolean);
   var combined = existing.concat(addresses).filter(function(value,index,all){return all.indexOf(value)===index;});
   widget.setValue(combined.join(','));
   outcome.textContent=tr('Candidates added. Enable resolver discovery and save and apply. Only paths that pass the tunnel handshake are verified.','آدرس‌ها اضافه شدند. کشف ریزالورها را روشن و ذخیره و اعمال کنید. فقط مسیرهایی که اتصال واقعی تونل را عبور دهند تایید می‌شوند.');
   outcome.hidden=false;
  }
  // Iran shortlist: pinned Soar data/resolvers/ir.txt, KevinNet snapshot 2026-05-26.
  // Attribution and MIT license: licenses/Soar-resolver-data-LICENSE.
  var candidateSets = [
   ['iran',tr('Candidates inside Iran','نامزدهای داخل ایران'),['185.212.51.144:53','46.100.40.59:53','5.202.171.124:53','93.114.104.185:53','2.188.162.78:53','87.107.82.213:53'],tr('Historical candidates from the bundled Iran list, not verified on your connection. Availability depends on the operator and restrictions.','نامزدهای فهرست پیشین ایران هستند و روی اتصال شما هنوز آزمایش نشده‌اند. دسترسی به اپراتور و محدودیت‌ها بستگی دارد.')],
   ['public',tr('Public resolvers','ریزالورهای عمومی'),['1.1.1.1:53','1.0.0.1:53','8.8.8.8:53','8.8.4.4:53','9.9.9.9:53','149.112.112.112:53'],tr('Cloudflare, Google and Quad9. Public resolvers may be unreachable during severe restrictions. A normal DNS reply does not prove tunnel connectivity.','کلادفلر، گوگل و کواد۹. در محدودیت شدید ممکن است در دسترس نباشند. پاسخ عادی DNS به معنی کار کردن تونل نیست.')]
  ];
  candidateSets.forEach(function(set) {
   var button = s.option(form.Button,'_candidates_'+set[0],set[1]);
   plaintextOption(button);
   button.inputtitle=tr('Add suggested addresses','افزودن آدرس‌های پیشنهادی');
   button.inputstyle='apply';
   button.description=E('span',{},[set[3],E('br'),E('bdi',{dir:'ltr',style:'overflow-wrap:anywhere'},set[2].join(', '))]);
   button.onclick=function(){addCandidates(set[2]);};
  });
  o = s.option(form.Flag, 'discovery_cache', tr('Save healthy resolvers', 'نگهداری ریزالورهای سالم')); o.default='1'; o.rmempty=false;
  plaintextOption(o);
  o.description = zui.optionDescription('discovery_cache');
  o = s.option(form.ListValue, 'network_id', tr('Network ID', 'شناسه شبکه'), tr('Automatic detects the network without manual selection. Otherwise choose Mobile, Home or Office to identify saved resolvers for that connection. Change this selection when switching internet connections. This does not change Wi-Fi or internet settings.', 'در حالت خودکار، روزنه شبکه را تشخیص می‌دهد. در حالت دستی، موبایل، خانه یا دفتر را انتخاب کنید تا ریزالورهای ذخیره‌شده با همان اینترنت شناخته شوند. با عوض کردن اینترنت، این انتخاب را هم تغییر دهید. این گزینه تنظیمات وای‌فای یا اینترنت را تغییر نمی‌دهد.'));
  o.value('', tr('Automatic', 'خودکار'));
  plaintextOption(o);
  o.value('mobile', tr('Mobile', 'موبایل'));
  o.value('home', tr('Home', 'خانه'));
  o.value('office', tr('Office', 'دفتر'));
  o.default = ''; o.rmempty = true;
  var savedNetwork = uci.get('rozaneh', 'main', 'network_id');
  if (savedNetwork && ['mobile','home','office'].indexOf(savedNetwork) === -1) o.value(savedNetwork, savedNetwork);
  return m.render().then(function(formNode) {
   return zui.optionHelp(E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' }, [
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/theme.css')+'?v=1.0.0-r1' }),
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/app.css')+'?v=1.0.0-r1' }),
    zui.hero('DNS'), zui.menu('dns'),
    outcome, E('div', { 'class':'mk-card', style:'--accent:#06b6d4' }, formNode), zui.footer()
   ]));
  });
 }
});

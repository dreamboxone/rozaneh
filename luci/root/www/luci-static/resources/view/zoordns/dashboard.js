'use strict';
'require view';
'require form';
'require rpc';
'require uci';
'require poll';
'require ui';
'require zoordns.icons as icons';

var status = rpc.declare({ object: 'zoordns', method: 'status', expect: { '': {} } });
var logs = rpc.declare({ object: 'zoordns', method: 'logs', expect: { '': {} } });
var control = rpc.declare({ object: 'zoordns', method: 'control', params: ['action'], expect: { '': {} } });
var test = rpc.declare({ object: 'zoordns', method: 'test', expect: { '': {} } });
var install = rpc.declare({ object: 'zoordns', method: 'install_dependencies', expect: { '': {} } });
var sites = rpc.declare({ object: 'zoordns', method: 'sites', params: ['site'], expect: { '': {} } });
var traffic = rpc.declare({ object: 'zoordns', method: 'traffic', expect: { '': {} } });
var fa = localStorage.getItem('zoordns-language') !== 'en';
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
  'Service could not start. Check logs.': 'سرویس روشن نشد؛ گزارش‌ها را بررسی کنید.'
 };
 if (!fa) return text;
 if (text.indexOf('HTTPS tunnel verified. Exit IP: ') === 0) return text.replace('HTTPS tunnel verified. Exit IP: ', 'اتصال HTTPS از داخل تونل تایید شد. IP خروجی: ');
 return messages[text] || text;
}

return view.extend({
 addFooter: function() {
  var self = this;
  return E('div', { 'class': 'mk zd-footer', dir: fa ? 'rtl' : 'ltr' }, [
   E('button', { 'class': 'mk-btn success', click: ui.createHandlerFn(this, function(ev) { return self.handleSaveApply(ev, '0'); }), disabled: !!(self.map && self.map.readonly) }, tr('Save & apply', 'ذخیره و اعمال')),
   E('button', { 'class': 'mk-btn primary', click: ui.createHandlerFn(this, 'handleSave'), disabled: !!(self.map && self.map.readonly) }, tr('Save', 'ذخیره')),
   E('button', { 'class': 'mk-btn soft-red', click: ui.createHandlerFn(this, 'handleReset'), disabled: !!(self.map && self.map.readonly) }, tr('Reset', 'بازنشانی'))
  ]);
 },
 load: function() { return Promise.all([uci.load('zoordns'), status().catch(function() { return {}; })]); },
 render: function(data) {
  var self = this;
  var current = data[1];
  var root = E('div', { 'class': 'mk' + (localStorage.getItem('zoordns-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' });
  var serviceValue = E('div', { 'class': 'zd-value' });
  var depValue = E('div', { 'class': 'zd-value' });
  var outcome = E('div', { 'class': 'zd-result', role: 'status' }, tr('No connection test yet.', 'هنوز اتصال آزمایش نشده است.'));
  var output = E('pre', { 'class': 'zd-log' }, tr('Click refresh to read recent logs.', 'برای خواندن گزارش‌ها، بازخوانی را بزنید.'));
  var depInfo = E('p');
  var uptime = E('bdi', {}, uptimeText(current.uptime));
  var installButton;
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
     var ms=Math.round(Number(item.seconds)*1000);
     card.value.textContent=ms>0?ms+' ms':tr('N/A','ناموفق');
     card.value.className='pwp-check-value '+(ms>0?(ms<1000?'mk-on':ms<2000?'mk-wait':'mk-bad'):'mk-bad');
     card.button.title='HTTP '+item.http+(item.state==='restricted'?' · '+tr('Restricted response','پاسخ محدودشده'):'');
    });
   }).catch(function(){names.forEach(function(name){siteValues[name].value.textContent=tr('N/A','ناموفق');siteValues[name].value.className='pwp-check-value mk-bad';});}).finally(function(){names.forEach(function(name){siteValues[name].button.disabled=false;});});
  }
  var usageData={},usagePeriod=localStorage.getItem('zoordns-usage-period')||'today',usageValues={};
  function renderUsage(){var d=usageData[usagePeriod]||{};['up','down','total'].forEach(function(key){var bytes=key==='total'?Number(d.up||0)+Number(d.down||0):Number(d[key]||0);usageValues[key].textContent=(bytes/1000000000).toFixed(2);});}
  var usageButtons=[];
  var usageCard=E('section',{'class':'mk-card pwp-usage-card',style:'--accent:#10b981'},[
   E('div',{'class':'pwp-usage-title'},tr('Usage (GB)','مصرف (گیگابایت)')),
   E('div',{'class':'pwp-usage-body'},[
    E('div',{'class':'pwp-usage-values'},['up','down','total'].map(function(key,i){var value=E('strong',{dir:'ltr'},'0.00');usageValues[key]=value;return E('div',{'class':'pwp-usage-value pwp-usage-'+key},[E('span',{},[tr('Upload','آپلود'),tr('Download','دانلود'),tr('Total','مجموع')][i]),value]);})),
    E('div',{'class':'pwp-usage-periods',role:'tablist'},['today','week','month'].map(function(period,i){var b=E('button',{type:'button',role:'tab','aria-selected':String(period===usagePeriod),title:[tr('Today (UTC)','امروز (UTC)'),tr('Last 7 days','۷ روز اخیر'),tr('Last 30 days','۳۰ روز اخیر')][i],click:function(){usagePeriod=period;localStorage.setItem('zoordns-usage-period',period);usageButtons.forEach(function(item){item.b.setAttribute('aria-selected',String(item.p===period));});renderUsage();}},[tr('1d','۱ر'),tr('1w','۷ر'),tr('1m','۳۰ر')][i]);usageButtons.push({b:b,p:period});return b;}))
   ])
  ]);
  function message(result) {
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
  function refresh() {
   traffic().then(function(data){usageData=data;renderUsage();}).catch(function(){});
   return status().then(function(s) {
    uptime.textContent = uptimeText(s.uptime);
    serviceValue.replaceChildren(E('span', { 'class': 'zd-led' + (s.running ? ' on' : '') }), document.createTextNode(s.running ? tr('Running', 'در حال اجرا') : tr('Stopped', 'خاموش')));
    depValue.textContent = s.missing ? tr('Needs setup', 'نیاز به نصب') : tr('Ready', 'آماده');
    depInfo.textContent = s.missing ? tr('Missing packages: ', 'بسته‌های ناقص: ') + s.missing : tr('All tools needed for the HTTPS test are available.', 'ابزارهای موردنیاز برای آزمون HTTPS نصب هستند.');
    if (s.installing) depInfo.textContent = tr('Installing packages…', 'در حال نصب بسته‌ها…');
    if (s.install_log) depInfo.textContent += '\n' + translatedMessage(s.install_log);
    installButton.disabled = !s.missing || s.installing;
   }).catch(function() { serviceValue.textContent = tr('Unavailable', 'در دسترس نیست'); });
  }
  function metric(label, value, accent) { return E('div', { 'class': 'mk-metric', style: '--accent:' + accent }, [E('div', { 'class': 'zd-label' }, label), value]); }
  installButton = action(tr('Install missing dependencies', 'نصب وابستگی‌های ناقص'), 'primary', function() {
   outcome.textContent = tr('Installing packages. This may take a moment…', 'بسته‌ها در حال نصب هستند؛ کمی صبر کنید…');
   return install().then(message);
  });
  var hero = E('div', { 'class': 'mk-hero' }, [
   E('div', { 'class': 'mk-brand' }, [E('div', {}, [E('h2', { 'class': 'zd-heading' }, ['zoorDNS', E('span', { 'class': 'mk-version' }, '0.1.0')]), E('p', {}, tr('A powerful DNS client for your router', 'کلاینت DNS قدرتمند برای روتر شما')), E('small', { style: 'color:inherit;opacity:.85' }, [tr('Router uptime: ', 'مدت فعالیت روتر: '), uptime])])]),
   E('div', { 'class': 'mk-row' }, [action(fa ? 'English' : 'فارسی', 'glass', function() { localStorage.setItem('zoordns-language', fa ? 'en' : 'fa'); window.location.reload(); }), action(tr('Light / dark', 'روشن / تیره'), 'glass', function() { root.classList.toggle('mk-dark'); localStorage.setItem('zoordns-dark', root.classList.contains('mk-dark') ? '1' : '0'); })])
  ]);
  var m = new form.Map('zoordns', tr('Tunnel settings', 'تنظیمات تونل'));
  this.map = m;
  var s = m.section(form.NamedSection, 'main', 'client');
  s.anonymous = true;
  s.tab('server', tr('Server', 'سرور'));
  s.tab('resolver', tr('DNS paths', 'مسیرهای DNS'));
  s.tab('local', tr('Local proxy', 'پراکسی محلی'));
  var o = s.taboption('server', form.Flag, 'enabled', tr('Start on boot', 'اجرای خودکار'));
  o.rmempty = false;
  o = s.taboption('server', form.Value, 'zone', tr('Tunnel domain', 'دامنه تونل'), tr('A DNS zone delegated to your Soar server.', 'زیردامنه‌ای که رکورد NS آن به سرور Soar شما واگذار شده است.'));
  o.placeholder = 'tunnel.example.com'; o.datatype = 'hostname'; o.rmempty = false;
  o = s.taboption('server', form.Value, 'pubkey', tr('Server public key', 'کلید عمومی سرور'));
  o.rmempty = false;
  o.validate = function(section, value) { return /^[A-Za-z0-9+/]{43}=$/.test(value || '') || tr('Enter a base64 32-byte public key.', 'کلید عمومی ۳۲ بایتی با قالب Base64 وارد کنید.'); };
  o = s.taboption('resolver', form.Value, 'resolvers', tr('Resolvers', 'ریزالورها'), tr('Comma-separated IP:port addresses, for example 1.1.1.1:53,8.8.8.8:53.', 'آدرس‌ها را با ویرگول جدا کنید؛ نمونه: 1.1.1.1:53,8.8.8.8:53'));
  o.placeholder = '1.1.1.1:53,8.8.8.8:53';
  o = s.taboption('resolver', form.ListValue, 'tcp', tr('DNS transport', 'روش انتقال DNS'));
  o.value('auto', tr('Automatic', 'خودکار')); o.value('on', tr('UDP and TCP', 'UDP و TCP')); o.value('only', tr('TCP only', 'فقط TCP')); o.value('off', tr('UDP only', 'فقط UDP')); o.default = 'auto';
  o = s.taboption('resolver', form.Flag, 'discover', tr('Discover resolvers', 'کشف ریزالورهای بیشتر')); o.rmempty = false;
  o = s.taboption('resolver', form.ListValue, 'country', tr('Discovery region', 'منطقه کشف'));
  o.value('IR', tr('Iran', 'ایران')); o.value('', tr('Global', 'جهانی')); o.default = 'IR';
  o = s.taboption('local', form.Value, 'listen', tr('SOCKS5 listener', 'آدرس پراکسی SOCKS5'), tr('TCP proxy without authentication. Keep loopback for router-only access.', 'این پراکسی TCP احراز هویت ندارد. آدرس پیش‌فرض فقط روی روتر در دسترس است.'));
  o.placeholder = '127.0.0.1:1088'; o.rmempty = false;
  return m.render().then(function(formNode) {
   root.appendChild(E('link', { rel: 'stylesheet', href: L.resource('zoordns/theme.css') + '?v=0.1.0-r3' }));
   root.appendChild(E('link', { rel: 'stylesheet', href: L.resource('zoordns/app.css') + '?v=0.1.0-r3' }));
   root.appendChild(hero);
   root.appendChild(E('div', { 'class': 'mk-row' }, [E('a', { 'class': 'mk-btn soft-violet', href: L.url('admin/services/zoordns/dashboard') }, tr('Home', 'صفحه اصلی')), E('a', { 'class': 'mk-btn soft-blue', href: L.url('admin/services/zoordns/update') }, tr('Updates', 'به‌روزرسانی'))]));
   root.appendChild(E('div', { 'class': 'mk-grid mk-metrics pwp-status-row zd-top-row' }, [E('div',{'class':'mk-card',style:'--accent:#6366f1'},[E('h3',{},tr('Soar engine','موتور Soar')),serviceValue]),E('section',{'class':'mk-card pwp-social-card',style:'--accent:#db2777'},siteCards),usageCard]));
   root.appendChild(E('div', { 'class': 'mk-card', style: '--accent:#06b6d4' }, [E('h3', {}, tr('Router and operating system', 'روتر و سیستم‌عامل')), E('p', {}, current.model || '—'), E('small', { 'class': 'zd-code' }, current.os || '—'), E('small', { 'class': 'zd-code' }, current.architecture || '—')]));
   root.appendChild(E('div', { 'class': 'mk-card', style: '--accent:#6366f1' }, [E('h3', {}, tr('Connection', 'اتصال')), E('p', {}, tr('A running service is not proof of internet access. Run the HTTPS test to verify the tunnel.', 'روشن بودن سرویس به‌تنهایی اتصال اینترنت را تایید نمی‌کند. برای بررسی تونل، آزمون HTTPS را اجرا کنید.')), E('div', { 'class': 'mk-row' }, [action(tr('Start', 'روشن'), 'success', function() { return control('start').then(message); }), action(tr('Stop', 'خاموش'), 'danger', function() { return control('stop').then(message); }), action(tr('Restart', 'راه‌اندازی مجدد'), 'soft-violet', function() { return control('restart').then(message); }), action(tr('Test HTTPS', 'آزمون HTTPS'), 'primary', function() { outcome.textContent = tr('Testing through the tunnel…', 'در حال آزمایش از داخل تونل…'); return test().then(message); })]), outcome]));
   root.appendChild(E('div', { 'class': 'mk-row' }, [action(tr('URL test: all sites', 'آزمون URL همه سایت‌ها'), 'soft-violet', function() { return runSites('all'); }), E('small', {}, tr('Green < 1000 ms · amber < 2000 ms · red ≥ 2000 ms. Usage is tunnel payload, excluding DNS overhead.', 'سبز کمتر از ۱۰۰۰، نارنجی کمتر از ۲۰۰۰، قرمز ۲۰۰۰ میلی‌ثانیه و بیشتر. مصرف مربوط به داده تونل، بدون سربار DNS است.'))]));
   root.appendChild(E('div', { 'class': 'mk-card', style: '--accent:#8b5cf6' }, formNode));
   root.appendChild(E('div', { 'class': 'mk-card', style: '--accent:#10b981' }, [E('h3', {}, tr('Dependencies', 'وابستگی‌ها')),depValue, depInfo, installButton]));
   root.appendChild(E('div', { 'class': 'mk-card', style: '--accent:#f59e0b' }, [E('div', { 'class': 'mk-row' }, [E('h3', {}, tr('Recent logs', 'گزارش‌های اخیر')), action(tr('Refresh', 'بازخوانی'), 'soft-amber', function() { return logs().then(function(r) { output.textContent = r.text || tr('No recent logs.', 'گزارشی وجود ندارد.'); }); })]), output]));
   root.appendChild(E('small', {}, tr('TCP SOCKS5 mode. Whole-network routing is not enabled by this version.', 'حالت پراکسی TCP SOCKS5؛ این نسخه مسیریابی کل شبکه را فعال نمی‌کند.')));
   root.querySelectorAll('input[type=text]').forEach(function(input) { input.dir = 'ltr'; });
   refresh(); poll.add(refresh, 5);
   return root;
  });
 }
});

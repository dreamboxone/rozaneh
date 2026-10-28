/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
'use strict';
'require baseclass';
'require rozaneh.icons-v2 as icons';

['language','dark','usage-period'].forEach(function(key) {
 var previous=localStorage.getItem('zoordns-'+key);
 if(localStorage.getItem('rozaneh-'+key)==null&&previous!=null)localStorage.setItem('rozaneh-'+key,previous);
});

return baseclass.extend({
 optionHelp: function(root) {
  if (root._rozanehHelp) return root;
  root._rozanehHelp = true;
  var active = null, connected = false;
  function close() {
   if (!active) return;
   active.desc.hidden = true;
   active.button.setAttribute('aria-expanded','false');
   active = null;
  }
  function enhance() {
   root.querySelectorAll('.cbi-value-description').forEach(function(desc) {
    if (desc.dataset.rozanehHelp || !desc.textContent.trim()) return;
    var row = desc.closest('.cbi-value,.zd-kill-control'), title = row && row.querySelector('.cbi-value-title');
    if (!title) return;
    desc.dataset.rozanehHelp = '1';
    desc.hidden = true;
    desc.setAttribute('role','note');
    var fa = localStorage.getItem('rozaneh-language') !== 'en';
    var button = E('button', { type:'button', 'class':'zd-option-help', 'aria-label':(fa ? 'راهنمای ' : 'Help for ') + title.textContent.trim(), 'aria-expanded':'false' }, fa ? '؟' : '?');
    button.addEventListener('click', function(ev) {
     ev.preventDefault(); ev.stopPropagation();
     var opening = desc.hidden;
     close();
     if (!opening) return;
     desc.hidden = false;
     button.setAttribute('aria-expanded','true');
     active = {button:button,desc:desc};
     var anchor = button.getBoundingClientRect(), box = desc.getBoundingClientRect();
     var left = Math.max(16,Math.min(fa ? anchor.right-box.width : anchor.left,window.innerWidth-box.width-16));
     var top = anchor.bottom+8+box.height<=window.innerHeight-16 ? anchor.bottom+8 : anchor.top-box.height-8;
     desc.style.left = left+'px';
     desc.style.top = Math.max(16,Math.min(top,window.innerHeight-box.height-16))+'px';
    });
    button.addEventListener('keydown', function(ev) {
     if (ev.key === 'Escape') { close(); button.focus(); }
    });
    var label = E('span', { 'class':'zd-help-label' });
    while (title.firstChild) label.appendChild(title.firstChild);
    label.appendChild(document.createTextNode('\u00a0'));
    label.appendChild(button);
    title.appendChild(label);
    title.classList.add('zd-has-help');
   });
  }
  enhance();
  if (typeof MutationObserver !== 'undefined') {
   function outside(ev) { if (active && !active.button.contains(ev.target) && !active.desc.contains(ev.target)) close(); }
   function scroll(ev) { if (active && !active.desc.contains(ev.target)) close(); }
   function escape(ev) { if (ev.key==='Escape' && active) { var button=active.button;close();button.focus(); } }
   var observer = new MutationObserver(function() {
    if (root.isConnected) connected = true;
    else if (connected) {
     close();observer.disconnect();
     document.removeEventListener('click',outside);
     document.removeEventListener('keydown',escape);
     document.removeEventListener('scroll',scroll,true);
     window.removeEventListener('resize',close);
     return;
    }
    enhance();
   });
   observer.observe(document.body,{childList:true,subtree:true});
   document.addEventListener('click',outside);
   document.addEventListener('keydown',escape);
   document.addEventListener('scroll',scroll,true);
   window.addEventListener('resize',close);
  }
  return root;
 },
 optionDescription: function(name) {
  var descriptions = {
   pubkey:['Paste the public key provided by your server. Keep the private key on the server.','کلید عمومی سرور را وارد کنید. کلید خصوصی باید فقط روی سرور بماند.'],
   tcp:['Select automatic, UDP/TCP, TCP only or UDP only for plain DNS endpoints. tls:// and https:// endpoints use their own protocol.','برای آدرس‌های DNS معمولی حالت خودکار، UDP و TCP، فقط TCP یا فقط UDP را انتخاب کنید. مسیرهای tls:// و https:// از پروتکل خود استفاده می‌کنند.'],
   discover:['Find and check additional resolvers automatically. Disable to use only your configured resolvers.','ریزالورهای بیشتر به‌صورت خودکار پیدا و بررسی می‌شوند. با خاموش کردن، فقط ریزالورهای تنظیم‌شده استفاده می‌شوند.'],
   country:['Choose the country used to select resolver discovery candidates.','کشور مورد استفاده برای انتخاب نامزدهای کشف ریزالور را مشخص می‌کند.'],
   lan_devices:['Select local network interfaces such as br-lan whose traffic should enter the tunnel. WAN interfaces are not accepted.','رابط‌های شبکه داخلی مثل br-lan را که باید از تونل استفاده کنند انتخاب کنید. رابط WAN پذیرفته نمی‌شود.'],
   discovery_candidates:['Optional extra resolver addresses to check during discovery, separated by commas.','آدرس ریزالورهای اضافی برای بررسی هنگام کشف؛ با ویرگول جدا کنید.'],
   discovery_cache:['Save verified resolvers to make the next connection on the same network faster.','ریزالورهای تاییدشده را ذخیره می‌کند تا اتصال بعدی در همین شبکه سریع‌تر انجام شود.'],
   _automatic_paths:['Fill the DNS options with resilient defaults. Save and apply to activate them.','گزینه‌های DNS را با مقادیر مناسب محدودیت شدید پر می‌کند. برای فعال شدن، ذخیره و اعمال کنید.'],
   discovery_want:['Target number of healthy resolver paths to discover. More paths can take longer to find.','تعداد مسیر سالم هدف برای کشف؛ یافتن مسیرهای بیشتر ممکن است زمان بیشتری ببرد.'],
   discovery_concurrency:['Maximum simultaneous discovery checks. Higher values increase network and router load.','بیشترین بررسی همزمان هنگام کشف؛ مقدار بالاتر بار شبکه و روتر را بیشتر می‌کند.'],
   discovery_samples:['Number of optional country address samples to probe. Zero disables this sampling.','تعداد نمونه آدرس کشور برای بررسی؛ صفر این نمونه‌برداری را خاموش می‌کند.'],
   safe_qname:['Initial DNS query name length. Shorter names may work better on restrictive networks.','طول اولیه نام درخواست DNS؛ نام کوتاه‌تر ممکن است در شبکه محدود بهتر کار کند.'],
   max_qname:['Upper limit for DNS query name length. Leave the default unless the network requires shorter requests.','سقف طول نام درخواست DNS؛ مگر در شبکه نیازمند درخواست کوتاه‌تر، مقدار پیش‌فرض را نگه دارید.'],
   max_inflight:['Maximum DNS queries awaiting replies at once. Larger values use more resources.','بیشترین درخواست DNS منتظر پاسخ به‌طور همزمان؛ مقدار بالاتر منابع بیشتری مصرف می‌کند.'],
   path_window:['Maximum outstanding queries on each resolver path.','بیشترین درخواست منتظر پاسخ در هر مسیر ریزالور.'],
   max_answer:['Maximum DNS response size in bytes. Smaller replies may pass restrictive networks more reliably.','سقف اندازه پاسخ DNS بر حسب بایت؛ پاسخ کوچک‌تر ممکن است در شبکه محدود بهتر عبور کند.'],
   max_streams:['Maximum concurrent streams in one tunnel session.','بیشترین تعداد اتصال همزمان در یک نشست تونل.'],
   send_buffer:['Send buffer size per stream in bytes. Larger buffers increase memory use.','حجم حافظه ارسال برای هر اتصال بر حسب بایت؛ مقدار بالاتر حافظه بیشتری مصرف می‌کند.'],
   stream_window:['Receive window per stream in bytes. Larger windows increase memory use.','اندازه پنجره دریافت هر اتصال بر حسب بایت؛ مقدار بالاتر حافظه بیشتری مصرف می‌کند.'],
   max_clients:['Maximum number of simultaneous local SOCKS clients.','بیشترین تعداد کلاینت همزمان پراکسی SOCKS محلی.'],
   hold:['Maximum time the server holds a poll while waiting for data, such as 500ms.','بیشترین زمان انتظار سرور برای داده هنگام درخواست دریافت؛ مانند 500 م‌ثانیه.'],
   session_idle:['How long an unused tunnel session is retained, such as 90s.','مدت نگهداری نشست تونل بدون استفاده؛ مانند 90s.'],
   discovery_pace:['Delay between resolver discovery probes, such as 20ms. A shorter delay increases probe traffic.','فاصله بررسی‌های کشف ریزالور؛ مانند 20 م‌ثانیه. فاصله کمتر ترافیک بررسی را افزایش می‌دهد.'],
   idle_timeout:['How long an inactive SOCKS connection can remain open, such as 5m.','مدتی که اتصال SOCKS بدون فعالیت باز می‌ماند؛ مانند 5m.'],
   debug:['Enable detailed engine logs for troubleshooting. This increases log volume.','گزارش کامل موتور را برای عیب‌یابی فعال می‌کند و حجم گزارش‌ها را بیشتر می‌کند.']
  };
  var value = descriptions[name];
  if (!value) return '';
  return localStorage.getItem('rozaneh-language') !== 'en' ? value[1].replace(/[A-Za-z][A-Za-z0-9:/=.?_-]*/g,'\u2066$&\u2069') : value[0];
 },
 name: function() {
  return localStorage.getItem('rozaneh-language') !== 'en' ? 'روزنه' : 'Rozaneh';
 },
 connectionState: function(s, now) {
  if (!s.enabled) return 'stopped';
  if (s.conflict) return 'blocked';
  if (s.missing) return 'blocked';
  if (!s.running || !s.lan_running || !s.lan_ready) return 'starting';
  now = now == null ? Math.floor(Date.now() / 1000) : now;
  var age = now - Number(s.health_updated || 0);
  return s.connected && s.health_running && age >= -5 && age <= 60 ? 'connected' : 'connecting';
 },
 versionText: function(value) {
  var version = String(value || '0.1.0').replace(/^v/i, '');
  return localStorage.getItem('rozaneh-language') !== 'en' ? version.replace(/-r(\d+)$/, '-$1') : version;
 },
 engineVersion: function(value) {
  var match = /-(\d{4})(\d{2})(\d{2})\d{6}-([a-f0-9]+)$/.exec(value || '');
  return match ? match[1] + '.' + match[2] + '.' + match[3] + ' / ' + match[4] : value || '—';
 },
 hero: function(subtitle, version, uptime) {
  var fa = localStorage.getItem('rozaneh-language') !== 'en';
  var themes = E('div', { 'class': 'mk-seg', role: 'group', 'aria-label': fa ? 'حالت نمایش' : 'Display mode' }, [false, true].map(function(dark) {
   var selected = (localStorage.getItem('rozaneh-dark') === '1') === dark;
   return E('button', { type: 'button', 'class': 'mk-btn glass'+(selected?' mk-seg-on':''), 'aria-pressed':String(selected), click: function(ev) {
    localStorage.setItem('rozaneh-dark', dark ? '1' : '0');
    ev.currentTarget.closest('.mk').classList.toggle('mk-dark', dark);
    themes.querySelectorAll('button').forEach(function(button){var active=button===ev.currentTarget;button.classList.toggle('mk-seg-on',active);button.setAttribute('aria-pressed',String(active));});
   } }, [icons.icon(dark ? 'moon' : 'sun'), E('span', {}, dark ? (fa ? 'تیره' : 'Dark') : (fa ? 'روشن' : 'Light'))]);
  }));
  var language = E('select', { 'class': 'mk-btn glass mk-language', 'aria-label': fa ? 'زبان' : 'Language', change: function(ev) {
   localStorage.setItem('rozaneh-language', ev.currentTarget.value); window.location.reload();
  } }, [E('option', { value: 'fa', selected: fa ? '' : null }, 'فارسی'), E('option', { value: 'en', selected: fa ? null : '' }, 'English')]);
  var controls = E('div', { 'class': 'mk-language-column' }, [language]);
  if (uptime) controls.appendChild(E('div', { 'class': 'mk-uptime' }, [E('span', {}, fa ? 'مدت فعالیت روتر' : 'Router uptime'), uptime]));
  return E('div', { 'class': 'mk-hero' }, [E('div', { 'class': 'mk-brand' }, [E('div', {}, [this.wordmark(version), E('p', {}, subtitle)])]), E('div', { 'class': 'mk-row mk-hero-controls' }, [themes, controls])]);
 },
 card: function(label, icon, accent, body) {
  return E('section', { 'class': 'mk-card', style: '--accent:' + accent }, [this.title(label, icon, accent), body]);
 },
 siteResult: function(item) {
  var ms = Math.round(Number(item.seconds) * 1000);
  var ok = item.state === 'ok' && /^2\d\d$/.test(String(item.http)) && Number(item.exit_code) === 0 && Number.isFinite(ms) && ms > 0;
  return { ms: Number.isFinite(ms) && ms > 0 ? ms : 0, tone: ok ? (ms < 1000 ? 'mk-on' : ms < 2000 ? 'mk-wait' : 'mk-bad') : 'mk-bad' };
 },
 menu: function(active) {
  var fa = localStorage.getItem('rozaneh-language') !== 'en';
  return E('nav', { 'class': 'mk-tabs zd-menu', 'aria-label': fa ? 'منوی برنامه' : 'Application menu' }, [
   ['dashboard', 'home', 'Home', 'خانه', '#3b82f6'],
   ['server', 'server', 'Server', 'سرور', '#9333ea'],
   ['dns', 'dns', 'DNS', 'DNS', '#06b6d4'],
   ['network', 'router', 'Network', 'شبکه', '#16a34a'],
   ['settings', 'settings', 'Other settings', 'تنظیمات دیگر', '#8b5cf6'],
   ['update', 'update', 'Updates', 'به روزرسانی', '#0284c7'],
   ['log', 'log', 'Logs', 'گزارش', '#b45309']
  ].map(function(item) {
   return E('a', { 'class': 'mk-tab' + (active === item[0] ? ' mk-tab-on' : ''), style: '--tab-icon-colour:' + item[4], href: L.url('admin/services/rozaneh/' + item[0]), 'aria-current': active === item[0] ? 'page' : null }, [icons.icon(item[1]), E('span', {}, fa ? item[3] : item[2])]);
  }));
 },
 title: function(label, icon, accent) {
  return E('h3', { 'class': 'mk-card-title', style: '--accent:' + accent }, [E('span', { 'class': 'mk-badge-ico' }, icons.icon(icon)), E('span', {}, label)]);
 },
 logo: function() {
  return E('img', { 'class': 'zd-logo', src: L.resource('rozaneh/logo.png') + '?v=rozaneh-cinema-2', alt: this.name(), width: 60, height: 60 });
 },
 wordmark: function(value) {
  var version = E('bdi', { 'class': 'mk-version', dir: 'ltr' }, this.versionText(value));
  var fa = localStorage.getItem('rozaneh-language') !== 'en';
  var name = E('span', { dir: 'auto' }, this.name());
  var brand = E('span', { 'class': 'zd-brand-name', dir: 'ltr' }, fa ? [name, this.logo()] : [this.logo(), name]);
  return E('h2', { 'class': 'zd-heading zd-wordmark', dir: 'ltr', 'aria-label': this.name() }, fa ? [version, brand] : [brand, version]);
 },
 footer: function() {
  var fa = localStorage.getItem('rozaneh-language') !== 'en';
  return E('footer', { 'class': 'mk-footer' }, [
   E('img', { 'class': 'zd-footer-logo', src: L.resource('rozaneh/logo.png') + '?v=rozaneh-cinema-2', height: 26, width: 26, alt: this.name() }),
   E('div', {}, this.name() + ' · Copyright © 2026 dreamboxone · Apache-2.0 / AGPL-3.0'),
   E('a', { 'class': 'mk-link', href: 'https://t.me/routekernel1', target: '_blank', rel: 'noopener noreferrer' }, [icons.icon('send'), 'Telegram · t.me/routekernel1']),
   E('a', { 'class': 'mk-link', href: 'https://github.com/dreamboxone/rozaneh', target: '_blank', rel: 'noopener noreferrer' }, [icons.icon('code'), fa ? 'سورس برنامه' : 'Source code'])
  ]);
 }
});

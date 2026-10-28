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
function tr(en, persian) { return fa ? persian : en; }
function displayDuration(value) { return fa ? String(value||'').replace(/ms$/, ' م‌ثانیه') : value; }
function parseDuration(value) {
 return String(value||'').trim().replace(/[۰-۹]/g,function(n){return '۰۱۲۳۴۵۶۷۸۹'.indexOf(n);}).replace(/[٠-٩]/g,function(n){return '٠١٢٣٤٥٦٧٨٩'.indexOf(n);}).replace(/\s*م‌?ثانیه$/, 'ms');
}
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
  var m = new form.Map('rozaneh', tr('Other settings', 'تنظیمات دیگر'));
  this.map = m;
  var s = m.section(form.NamedSection, 'main', 'client');
  s.anonymous = true;
  var startup=s.option(form.ListValue,'startup_mode',tr('Startup mode','روش راه اندازی'));
  startup.value('logs',tr('Verified cached resolvers, with scan fallback','مسیرهای نگهداری شده با بررسی مجدد'));startup.value('resolvers',tr('Full resolver scan','بررسی کامل ریزالورها'));startup.default='resolvers';startup.rmempty=false;
  var fast=s.option(form.Flag,'fast_connect',tr('Start with the first validated paths','شروع با نخستین مسیرهای تاییدشده'));fast.default='1';fast.rmempty=false;
  fast.description=tr('Start with a freshly validated resolver pool and continue scanning in the background.','پس از تایید مسیرهای اولیه، اتصال شروع می شود و بررسی مسیرهای دیگر در پس زمینه ادامه می یابد.');
  var o=s.option(form.ListValue,'config_preset',tr('Network profile','پروفایل شبکه'));
  o.value('speed',tr('Balanced speed','سرعت متعادل'));o.value('survival',tr('Lossy network','شبکه با افت زیاد'));o.value('tcp-survival',tr('TCP fallback','مسیر TCP'));o.default='speed';o.rmempty=false;
  o=s.option(form.ListValue,'resolver_ip_mode',tr('IP family','نسخه IP'));
  [['auto','خودکار'],['ipv4','فقط IPv4'],['ipv6','فقط IPv6'],['dual','هر دو']].forEach(function(item){o.value(item[0],fa?item[1]:item[0]);});o.default='auto';o.rmempty=false;
  o=s.option(form.ListValue,'tunnel_mode',tr('Core mode','حالت هسته'));
  o.value('legacy','Normal');o.value('cottenmax',tr('CottenMax (experimental)','CottenMax (آزمایشی)'));o.default='legacy';o.rmempty=false;
  o=s.option(form.Value,'listen',tr('SOCKS5 listener','آدرس پراکسی SOCKS5'));o.default='127.0.0.1:1088';o.rmempty=false;
  o=s.option(form.TextValue,'advanced_config',tr('Advanced CottenDNS configuration','تنظیمات پیشرفته CottenDNS'));
  o.rows=18;o.wrap='off';
  o.description=tr('Native CottenDNS TOML options, including ARQ, MTU, duplication, FEC and recovery. Main form fields take precedence. Normal is the tested default; Max requires a compatible server.','گزینه های اصلی CottenDNS با قالب TOML، شامل ARQ، اندازه بسته، تکرار بسته، FEC و بازیابی. گزینه های فرم اصلی اولویت دارند. حالت Normal پیش فرض آزمایش شده است؛ Max به سرور سازگار نیاز دارد.');
  return m.render().then(function(formNode) {
   return zui.optionHelp(E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' }, [
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/theme.css')+'?v=1.0.0-r1' }),
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/app.css')+'?v=1.0.0-r1' }),
    zui.hero(tr('Other settings', 'تنظیمات دیگر')), zui.menu('settings'),
    E('div', { 'class':'mk-card', style:'--accent:#8b5cf6' }, formNode), zui.footer()
   ]));
  });
 }
});

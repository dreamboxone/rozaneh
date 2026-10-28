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
function renderBlurredWidget() {
 return Promise.resolve(form.Value.prototype.renderWidget.apply(this, arguments)).then(function(node) {
  if (node.matches && node.matches('input')) node.classList.add('zd-private-field');
  node.querySelectorAll('input').forEach(function(input) { input.classList.add('zd-private-field'); });
  return node;
 });
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
  var m = new form.Map('rozaneh', tr('Server', 'سرور'));
  this.map = m;
  var s = m.section(form.NamedSection, 'main', 'client');
  s.anonymous = true;
  var o;
  o = s.option(form.Value, 'zone', tr('Tunnel domain', 'دامنه تونل'), tr('A DNS zone delegated to your CottenDNS server.', 'زیردامنه‌ای که رکورد NS آن به سرور CottenDNS شما واگذار شده است.'));
  o.placeholder = 'tunnel.example.com'; o.datatype = 'hostname'; o.rmempty = false;
  o.renderWidget = renderBlurredWidget;
  o = s.option(form.Value, 'encryption_key', tr('Server shared key', 'کلید مشترک سرور'));
  o.password = true; o.description = tr('The secret from encrypt_key.txt on your CottenDNS server. Your old Soar public key cannot be used.', 'کلید محرمانه فایل encrypt_key.txt سرور CottenDNS را وارد کنید. کلید عمومی قبلی Soar قابل استفاده نیست.');
  o.rmempty = false;
  o.renderWidget = renderBlurredWidget;
  o.validate = function(section, value) { return (!!value && !/[\r\n]/.test(value)) || tr('Enter the server shared key.', 'کلید مشترک سرور را وارد کنید.'); };
  return m.render().then(function(formNode) {
   return zui.optionHelp(E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' }, [
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/theme.css')+'?v=1.0.0-r1' }),
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/app.css')+'?v=1.0.0-r1' }),
    zui.hero(tr('Server', 'سرور')), zui.menu('server'),
    E('div', { 'class':'mk-card', style:'--accent:#8b5cf6' }, formNode), zui.footer()
   ]));
  });
 }
});

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
  var m = new form.Map('rozaneh', tr('Network', 'شبکه'));
  this.map = m;
  var s = m.section(form.NamedSection, 'main', 'client');
  s.anonymous = true;
  var o;
  o = s.option(form.DynamicList, 'lan_devices', tr('Router interface', 'اینترفیس روتر'));
  o.description = zui.optionDescription('lan_devices');
  o.default = ['br-lan']; o.rmempty = false;
  o.validate = function(section, value) { return value === '' || /^[a-zA-Z0-9_.:-]{1,15}$/.test(value || '') || tr('Enter a Linux network device name, such as br-lan.', 'نام رابط شبکه مانند br-lan را وارد کنید.'); };
  return m.render().then(function(formNode) {
   return zui.optionHelp(E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' }, [
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/theme.css')+'?v=1.0.0-r1' }),
    E('link', { rel:'stylesheet', href:L.resource('rozaneh/app.css')+'?v=1.0.0-r1' }),
    zui.hero(tr('Network', 'شبکه')), zui.menu('network'),
    E('div', { 'class':'mk-card', style:'--accent:#8b5cf6' }, formNode), zui.footer()
   ]));
  });
 }
});

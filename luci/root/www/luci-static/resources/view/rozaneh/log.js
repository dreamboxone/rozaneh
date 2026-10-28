/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
'use strict';
'require view';
'require rpc';
'require poll';
'require form';
'require uci';
'require ui';
'require rozaneh.ui as zui';
'require rozaneh.logs as logs';
'require rozaneh.icons-v2 as icons';

var readLogs = rpc.declare({ object: 'rozaneh', method: 'logs', expect: { '': {} } });
var clearLogs = rpc.declare({ object: 'rozaneh', method: 'clear_logs', expect: { '': {} } });
var diagnosticLogs = rpc.declare({ object: 'rozaneh', method: 'diagnostic_logs', expect: { '': {} } });
var readStatus = rpc.declare({ object: 'rozaneh', method: 'status', expect: { '': {} } });
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
 load: function() { return Promise.all([readLogs(),readStatus().catch(function(){return {};}),uci.load('rozaneh')]); },
 render: function(initial) {
  var m = new form.Map('rozaneh');
  this.map = m;
  var section = m.section(form.NamedSection, 'main', 'client');
  section.anonymous = true;
  var debug = section.option(form.Flag, 'debug', tr('Detailed Rozaneh logs','گزارش کامل روزنه'));
  debug.default='0';debug.rmempty=false;debug.description=zui.optionDescription('debug');
  var history = [];
  var selectedLevel = localStorage.getItem('rozaneh-log-level') || 'info-error';
  if (['error','info-error','debug','info'].indexOf(selectedLevel)<0) selectedLevel='info-error';
  var generation = 0, clearing = false;
  var active = !!initial[1].running;
  var output = E('pre', { 'class': 'zd-log zd-runtime-log', dir:fa?'rtl':'ltr', style: 'max-height:65vh', role: 'log' });
  var failure = E('p', { 'class':'zd-label', role:'status', hidden:true });
  function visibleText() { return logs.format(logs.filter(history.join('\n'),selectedLevel),fa); }
  function draw(result) {
   if (typeof result.active === 'boolean') active=result.active;
   history = logs.merge(history,result.text,active?0:30);
   output.textContent = visibleText() || tr('No logs at this level.', 'گزارشی در این سطح وجود ندارد.');
   failure.hidden = true;
  }
  draw(initial[0]);
  function refresh() {
   if (clearing) return Promise.resolve();
   var requestGeneration = generation;
   return readLogs().then(function(result) { if (requestGeneration === generation) draw(result); }).catch(function() {
    failure.textContent = tr('Could not fetch new logs. Previous messages are retained.', 'دریافت گزارش تازه ممکن نشد؛ پیام‌های قبلی حفظ شده‌اند.');
    failure.hidden = false;
   });
  }
  function copyText(text) {
   if (navigator.clipboard && window.isSecureContext) return navigator.clipboard.writeText(text);
   var input = E('textarea', { style:'position:fixed;opacity:0', readonly:true });
   input.value = text; document.body.appendChild(input); input.select();
   try { if (!document.execCommand('copy')) throw new Error('copy'); }
   finally { input.remove(); }
   return Promise.resolve();
  }
  var copyButton = E('button', { type:'button', 'class':'mk-btn primary', click:ui.createHandlerFn(this,function() {
   return Promise.resolve().then(function(){return copyText(visibleText());}).then(function(){
    ui.addNotification(null,E('p',{},tr('Copied.','کپی شد.')),'info');
   }).catch(function(){ui.addNotification(null,E('p',{},tr('Could not copy logs.','کپی گزارش ها انجام نشد.')),'error');});
  }) }, [icons.icon('copy'),tr('Copy','کپی')]);
  var deleteButton = E('button', { type:'button', 'class':'mk-btn soft-red', disabled:m.readonly||null, click:ui.createHandlerFn(this,function() {
   clearing=true; generation++;
   return clearLogs().then(function(result){
    if (!result.ok) throw new Error('clear');
    history=[]; draw({text:''});
   }).catch(function(){ui.addNotification(null,E('p',{},tr('Could not delete logs.','حذف گزارش ها انجام نشد.')),'error');}).finally(function(){clearing=false;});
  }) }, [icons.icon('trash'),tr('Delete','حذف')]);
  var diagnosticButton=E('button',{type:'button','class':'mk-btn soft-blue',click:ui.createHandlerFn(this,function(){
   return diagnosticLogs().then(function(result){return copyText(String(result.text||''));}).then(function(){
    ui.addNotification(null,E('p',{},tr('Diagnostic report copied.','گزارش عیب یابی کپی شد.')),'info');
   }).catch(function(){ui.addNotification(null,E('p',{},tr('Could not copy diagnostic report.','کپی گزارش عیب یابی انجام نشد.')),'error');});
  })},[icons.icon('copy'),tr('Copy diagnostic report','کپی گزارش عیب یابی')]);
  var levelSelect=E('select',{id:'rozaneh-log-level','class':'mk-select',change:function(ev){
   selectedLevel=ev.target.value;localStorage.setItem('rozaneh-log-level',selectedLevel);draw({text:''});
  }},[['error',tr('Errors','خطا')],['info-error',tr('Information and errors','اطلاعات و خطا')],['debug',tr('Debug (all levels)','دیباگ')],['info',tr('Information','اطلاعات')]].map(function(item){
   return E('option',{value:item[0],selected:item[0]===selectedLevel?true:null},item[1]);
  }));
  poll.add(refresh, 5);
  return m.render().then(function(formNode) {
  return zui.optionHelp(E('div', { 'class': 'mk' + (localStorage.getItem('rozaneh-dark') === '1' ? ' mk-dark' : ''), dir: fa ? 'rtl' : 'ltr' }, [
   E('link', { rel: 'stylesheet', href: L.resource('rozaneh/theme.css') + '?v=1.0.0-r1' }),
   E('link', { rel: 'stylesheet', href: L.resource('rozaneh/app.css') + '?v=1.0.0-r1' }),
   zui.hero(tr('Logs','گزارش'),initial[1].version),
   zui.menu('log'),
   E('div', { 'class':'mk-card', style:'--accent:#8b5cf6' }, formNode),
   E('div', { 'class': 'mk-card', style: '--accent:#f59e0b' }, [
    E('h3', {}, tr('Recent logs', 'گزارش‌های اخیر')),
    E('div', { 'class':'mk-row', style:'display:flex;gap:10px;flex-wrap:wrap;align-items:center;margin-bottom:12px' }, [E('label',{'for':'rozaneh-log-level'},tr('Log level','سطح گزارش')),levelSelect,copyButton,deleteButton,diagnosticButton]),
    output, failure
   ]),zui.footer()
  ]));
  });
 }
});

/* SPDX-License-Identifier: AGPL-3.0-or-later
 * Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
 */
'use strict';
'require baseclass';

var messages = [
 ['inbound/tun[lan-tun]', 'ورودی تونل شبکه داخلی'],
 ['outbound/socks[tunnel]', 'خروجی پراکسی تونل'],
 ['inbound connection from ', 'اتصال ورودی از '],
 ['inbound connection to ', 'مقصد اتصال ورودی: '],
 ['outbound connection to ', 'اتصال خروجی به '],
 ['connection: open connection to ', 'اتصال: برقراری ارتباط با '],
 [' using ', ' از مسیر '],
 ['socks5: request rejected, code=', 'درخواست پراکسی SOCKS5 رد شد، کد: '],
 ['soar: stream reset by peer', 'Soar: جریان ارتباط از سمت مقابل قطع شد'],
 ['dns: exchanged ', 'DNS: پاسخ دریافت شد، نوع رکورد '],
 ['Zirgozar is on; Rozaneh will not start','زیرگذر روشن است؛ روزنه اجرا نمی‌شود'],
 ['Another transparent proxy is active; stop it before starting Rozaneh','پراکسی دیگری فعال است؛ پیش از روشن‌کردن روزنه آن را متوقف کنید'],
 ['Another transparent proxy is active; LAN mode was not started','پراکسی دیگری فعال است؛ مسیر شبکه داخلی اجرا نشد'],
 ['Missing zone or public key','دامنه تونل یا کلید عمومی وارد نشده است'],
 ['Missing packages: ','بسته‌های موردنیاز: '],
 ['LAN mode needs: ','نیازمندی‌های شبکه داخلی: '],
 ['LAN device not found: ','رابط شبکه داخلی پیدا نشد: '],
 ['Routing table 28028 is already in use','جدول مسیریابی 28028 از قبل در حال استفاده است'],
 ['Routing rule priority 12801 is already in use','قانون مسیریابی با اولویت 12801 از قبل در حال استفاده است'],
 ['LAN DNS table already exists','جدول DNS شبکه داخلی از قبل وجود دارد'],
 ['LAN tunnel did not start','تونل شبکه داخلی اجرا نشد'],
 ['LAN internet sharing started using ','اشتراک اینترنت شبکه داخلی آغاز شد با '],
 ['(TCP through Soar, DNS through HTTPS)','(TCP از طریق Soar و DNS از طریق HTTPS)'],
 ['SOCKS5 listening on ','پراکسی SOCKS5 آماده دریافت اتصال است در '],
 ['resolver diagnostics: ','بررسی مسیرهای ریزالور: '],
 ['zone and a valid base64 32-byte public key are required','دامنه و کلید عمومی معتبر ۳۲ بایتی با قالب Base64 لازم است'],
 ['configure resolvers or enable discovery','ریزالورها را تنظیم یا کشف خودکار را فعال کنید'],
 ['invalid tcp mode','حالت پروتکل DNS نامعتبر است'],
 ['invalid SOCKS greeting','درخواست آغاز SOCKS نامعتبر است'],
 ['unsupported SOCKS authentication','روش احراز هویت SOCKS پشتیبانی نمی‌شود'],
 ['invalid SOCKS request','درخواست SOCKS نامعتبر است'],
 ['only TCP CONNECT is supported','فقط اتصال TCP پشتیبانی می‌شود'],
 ['empty destination','مقصد خالی است'],
 ['unsupported address type','نوع آدرس پشتیبانی نمی‌شود'],
 ['tunnel connect: ','اتصال تونل: '],
 ['tunnel transfer: ', 'خطای انتقال تونل: '],
 ['tunnel idle timeout: ', 'پایان مهلت بی فعالیتی تونل: '],
 ['direction=upload', 'جهت=ارسال'],
 ['direction=download', 'جهت=دریافت'],
 ['bytes=', 'بایت='],
 ['error=', 'خطا='],
 ['idle=', 'مهلت بی فعالیتی='],
 ['destination=', 'مقصد='],
 ['elapsed=', 'زمان سپری شده='],
 ['context deadline exceeded','مهلت عملیات تمام شد'],
 ['context canceled','عملیات لغو شد'],
 ['connection refused','اتصال رد شد'],
 ['connection reset by peer','اتصال از سمت مقابل قطع شد'],
 ['network is unreachable','شبکه در دسترس نیست'],
 ['no route to host','مسیری به مقصد وجود ندارد'],
 ['i/o timeout','مهلت ارسال یا دریافت تمام شد'],
 ['EOF','پایان ارتباط']
];
function translate(message) {
 message=message.replace(/elapsed=(\d+(?:\.\d+)?)(ms|s)(?=:|\s|$)/g,function(_,value,unit){return 'elapsed='+value+' '+(unit==='ms'?'م‌ثانیه':'ثانیه');});
 messages.forEach(function(pair) { message = message.split(pair[0]).join(pair[1]); });
 return message;
}
var months = { Jan:0,Feb:1,Mar:2,Apr:3,May:4,Jun:5,Jul:6,Aug:7,Sep:8,Oct:9,Nov:10,Dec:11 };
function severity(raw) {
 var text=raw.replace(/\x1b\[[0-9;]*m/g,'');
 var runtime=/\b(INFO|DEBUG|WARN|ERROR|FATAL|TRACE)\[\d+\]/.exec(text);
 if(runtime)return /ERROR|FATAL|WARN/.test(runtime[1])?'error':/DEBUG|TRACE/.test(runtime[1])?'debug':'info';
 var structured=/"level"\s*:\s*"([a-z]+)"/i.exec(text);
 var syslog=/\b\w+\.(err|error|warn|warning|crit|alert|emerg|info|notice|debug)\s/.exec(text);
 var level=structured?structured[1].toLowerCase():syslog?syslog[1]:'info';
 return /^(err|error|warn|warning|crit|alert|emerg|fatal)$/.test(level)?'error':/^(debug|trace)$/.test(level)?'debug':'info';
}
function line(raw, fa) {
 if (!fa) return raw;
 var text = raw.replace(/\x1b\[[0-9;]*m/g, '');
 var runtimeLevel;
 var match = /^\w{3} (\w{3})\s+(\d+) (\d{2}):(\d{2}):(\d{2}) (\d{4})\s+([\w.-]+)\s+([^:]+):\s?(.*)$/.exec(text);
 var prefix = '';
 if (match && months[match[1]] !== undefined) {
  // logread supplies router-local wall time without an offset. Preserve it.
  var date = new Date(Date.UTC(+match[6],months[match[1]],+match[2],+match[3],+match[4],+match[5]));
  prefix = new Intl.DateTimeFormat('fa-IR-u-ca-persian', {year:'numeric',month:'2-digit',day:'2-digit',timeZone:'UTC'}).format(date) + ' ' +
   [match[3],match[4],match[5]].join(':').replace(/\d/g,function(n){return '۰۱۲۳۴۵۶۷۸۹'[n];}) + ' — ';
  var level = match[7].split('.').pop();
  var levels = {err:'خطا',error:'خطا',warn:'هشدار',warning:'هشدار',info:'اطلاعات',notice:'رویداد',debug:'اشکال‌زدایی',crit:'بحرانی',alert:'هشدار فوری',emerg:'اضطراری'};
  if (levels[level]) prefix += '['+levels[level]+'] ';
  text = match[9];
  text = text.replace(/^\d{4}\/\d{2}\/\d{2} \d{2}:\d{2}:\d{2}\s+/, '');
  var runtime = /^(INFO|DEBUG|WARN|ERROR|FATAL|TRACE)\[(\d+)\]\s*/.exec(text);
  if (runtime) {
   runtimeLevel = {INFO:'اطلاعات',DEBUG:'اشکال زدایی',WARN:'هشدار',ERROR:'خطا',FATAL:'بحرانی',TRACE:'ردیابی'}[runtime[1]];
   prefix = prefix.replace(/\[[^\]]+\]\s*$/, '') + '['+runtimeLevel+'] ';
   text = text.slice(runtime[0].length);
   text = text.replace(/^\[(\d+)\s+([\d.]+)(ms|s)\]\s*/, function(_,id,duration,unit){
    return '[شناسه '+id+'، '+duration+' '+(unit==='ms'?'م‌ثانیه':'ثانیه')+'] ';
   });
  }
 }
 if (text.charAt(0) === '{') {
  try {
   var entry = JSON.parse(text);
   if (typeof entry.msg === 'string') {
    text = entry.msg;
    Object.keys(entry).filter(function(k){return k!=='msg'&&k!=='time'&&k!=='level';}).forEach(function(k){
     var labels = {error:'خطا',err:'خطا',resolver:'ریزالور',transport:'روش انتقال',rtt:'زمان پاسخ',loss:'افت',address:'آدرس',path:'مسیر'};
     text += ' · '+(labels[k]||k)+': '+(typeof entry[k]==='object'?JSON.stringify(entry[k]):entry[k]);
    });
   }
  } catch (_) { /* Preserve malformed diagnostic output. */ }
 }
 return prefix + translate(text);
}

return baseclass.extend({
 filter: function(text,level) { return String(text||'').split('\n').filter(function(raw){var value=severity(raw);return level==='debug'||level==='all'||(level==='info-error'&&value!=='debug')||value===level;}).join('\n'); },
 format: function(text, fa) { return String(text||'').split('\n').map(function(value){return line(value,fa);}).join('\n'); },
 merge: function(previous, snapshot, limit) {
  if (limit === undefined) limit=30;
  function trim(lines) { return limit>0?lines.slice(-limit):lines; }
  var incoming = String(snapshot||'').split('\n').filter(function(value){return value.length;});
  if (!incoming.length) return trim(previous);
  var overlap = Math.min(previous.length,incoming.length);
  for (;overlap>0;overlap--) {
   if (previous.slice(-overlap).join('\n')===incoming.slice(0,overlap).join('\n')) break;
  }
  return trim(previous.concat(incoming.slice(overlap)));
 }
});

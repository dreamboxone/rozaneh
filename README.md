<div dir="rtl" align="right">

<p align="center">
  <img src="luci/root/www/luci-static/resources/rozaneh/logo.png" alt="لوگوی روزنه" width="200">
</p>

<h1 align="center">روزنه | Rozaneh</h1>
<h3 align="center">تونل DNS برای روتر OpenWrt، با هسته CottenDNS</h3>

<p align="center" dir="ltr">
  <a href="https://github.com/dreamboxone/rozaneh/releases/latest"><img alt="Latest release" src="https://img.shields.io/github/v/release/dreamboxone/rozaneh?style=for-the-badge&amp;color=2563eb&amp;label=release"></a>
  <img alt="OpenWrt" src="https://img.shields.io/badge/OpenWrt-00B5E2?style=for-the-badge&amp;logo=openwrt&amp;logoColor=white">
  <a href="https://github.com/WhiteDNS/CottenDNS"><img alt="CottenDNS core" src="https://img.shields.io/badge/engine-CottenDNS-7c3aed?style=for-the-badge"></a>
  <a href="LICENSE"><img alt="License" src="https://img.shields.io/badge/license-Apache--2.0%20%2F%20AGPL--3.0-16a34a?style=for-the-badge"></a>
  <a href="https://t.me/routekernel1"><img alt="Telegram" src="https://img.shields.io/badge/Telegram-routekernel1-26A5E4?style=for-the-badge&amp;logo=telegram&amp;logoColor=white"></a>
</p>

<p align="center">
  <a href="https://github.com/dreamboxone/rozaneh/releases/latest">⬇️ <b>دانلود آخرین نسخه</b></a> ·
  <a href="README.md">🇮🇷 <b>فارسی</b></a> ·
  <a href="README.en.md">🇬🇧 <b>English</b></a> ·
  <a href="https://t.me/routekernel1">💬 <b>پشتیبانی</b></a>
</p>

<p align="right"><img src="assets/server-required-fa.svg" alt="برای استفاده از این برنامه نیاز به سرور مجازی لینوکسی خارج از ایران دارید." width="1000"></p>

---

روزنه برنامه تونل DNS برای روترهای OpenWrt بر پایه هسته CottenDNS است. برای استفاده، علاوه بر روتر سازگار به سرور مجازی لینوکسی خارج از ایران، دامنه با واگذاری NS و کلید مشترک سرور نیاز دارید. راهنمای ساخت سرور در ادامه آمده است.


<p align="center">
  <a href="#server">🌐 ۱. سرور و کلید</a> · <a href="#install">⬇️ ۲. نصب روتر</a> · <a href="#manual-install">📂 نصب دستی</a> · <a href="#remove">🗑️ حذف</a> · <a href="#features">✨ امکانات</a> · <a href="#support">💚 حمایت</a>
</p>

<a id="server"></a>

## 🌐 ۱. ابتدا سرور را راه اندازی کنید

**پیش از نصب روزنه روی روتر، یک سرور مجازی خارج از ایران با Ubuntu 24.04 LTS یا جدیدتر، IP عمومی و دسترسی root یا sudo آماده کنید.** دستورهای این مرحله برای ترمینال سرور هستند، نه روتر OpenWrt.

### ۱.۱. دامنه را به سرور وصل کنید

در پنل DNS دامنه خود این دو رکورد را بسازید. این نمونه برای `example.com` است؛ دامنه و IP را با اطلاعات واقعی خود جایگزین کنید:

| نوع | نام در پنل دامنه | مقدار | حالت |
|---|---|---|---|
| A | `ns-tunnel` | IP عمومی سرور | فقط DNS؛ ابر نارنجی خاموش |
| NS | `tunnel` | `ns-tunnel.example.com` | واگذاری زیردامنه به سرور |

دامنه تونل در این نمونه `tunnel.example.com` است. IP نمونه یا آدرس عادی سایت را به جای آن وارد نکنید. پورت **53 برای هر دو UDP و TCP** باید در فایروال شرکت میزبان و فایروال خود سرور باز باشد. سرویس دیگری نباید این پورت را اشغال کرده باشد.

در [MobaXterm](https://mobaxterm.mobatek.net/download-home-edition.html) به IP سرور وصل شوید و روی **سرور Ubuntu** اجرا کنید:

<div dir="ltr" align="left">

```bash
sudo apt update
sudo apt install -y curl ca-certificates dnsutils
sudo ss -lntup | grep ':53 '
dig +short A ns-tunnel.example.com
dig +short NS tunnel.example.com
```

</div>

خروجی A باید IP سرور و خروجی NS باید نام `ns-tunnel.example.com` باشد. اگر پورت 53 اشغال است، پیش از نصب تداخل آن سرویس را برطرف کنید. اگر UFW فعال است، روی سرور اجرا کنید:

<div dir="ltr" align="left">

```bash
sudo ufw allow 53/udp
sudo ufw allow 53/tcp
```

</div>

### ۱.۲. نصب CottenDNS روی Ubuntu

این دستورها را روی **سرور** اجرا کنید. نصب در `/opt/cottendns` انجام می شود و نسخه سازگار با هسته فعلی روزنه انتخاب می شود:

<div dir="ltr" align="left">

```bash
sudo install -d -m 755 /opt/cottendns
cd /opt/cottendns
sudo curl -fL https://raw.githubusercontent.com/WhiteDNS/CottenDNS/main/server_linux_install.sh -o server_linux_install.sh
sudo bash server_linux_install.sh --version v2026.10.24.183015-7db7b73
```

</div>

تگ دستور بالا با نسخه هسته همان انتشار روزنه هماهنگ است. هنگام انتشار روزنه با هسته جدید، این تگ در راهنما نیز خودکار تغییر می کند. برای ارتقا، راهنمای همان نسخه روزنه را بخوانید.

وقتی نصب کننده دامنه را درخواست کرد، **دامنه تونل واگذارشده** مانند `tunnel.example.com` را وارد کنید. نصب کننده نسخه مناسب معماری سرور را دریافت و سرویس `cottendns` را راه اندازی می کند. این روش برای سرور مستقل است؛ اگر سرور را با CottenRouter مدیریت می کنید، از روش بروزرسانی خود CottenRouter استفاده کنید.

### ۱.۳. سلامت سرور و دریافت کلید

پس از نصب روی **سرور** اجرا کنید. بررسی سلامت تا ۱۰ نوبت تکرار می شود تا سرویس فرصت آماده شدن داشته باشد:

<div dir="ltr" align="left">

```bash
sudo systemctl is-active cottendns
for i in $(seq 1 10); do
  curl -fsS --max-time 2 http://127.0.0.1:9090/healthz && break
  sleep 1
done
sudo ss -lntup | grep -E ':(53|9090)[[:space:]]'
sudo cat /opt/cottendns/encrypt_key.txt
```

</div>

وضعیت سرویس باید `active` و پاسخ سلامت `ok` باشد. در صورت خطا، پیش از ادامه نصب روتر، لاگ را بررسی کنید:

<div dir="ltr" align="left">

```bash
sudo journalctl -u cottendns -n 60 --no-pager
```

</div>

**کلید تولیدشده در `/opt/cottendns/encrypt_key.txt` روی سرور قرار دارد.** کل متن کلید را بدون فاصله و خط اضافی کپی و محرمانه نگهداری کنید. برای CottenMax باید در لاگ سرور پیام پشتیبانی از Normal و CottenMax دیده شود؛ هر دو از همان دامنه و کلید استفاده می کنند.

### 🔑 این کلید را کجای روتر ذخیره کنیم؟

پس از نصب روزنه در مرحله بعد، در پنل روتر به **Services → Rozaneh → سرور** بروید:

| گزینه در روزنه | چه چیزی وارد شود؟ |
|---|---|
| **دامنه تونل** | همان دامنه مرحله اول؛ مانند `tunnel.example.com` |
| **کلید مشترک سرور** | متن فایل `/opt/cottendns/encrypt_key.txt` سرور |

روی **ذخیره و اعمال** بزنید. برنامه کلید را در گزینه `encryption_key` بخش `main` فایل `/etc/config/rozaneh` روی روتر ذخیره می کند؛ لازم نیست فایل کلید سرور را در پوشه `/tmp` روتر کپی کنید. این کلید، رمز ورود SSH یا کلید عمومی Soar نیست.

برای اتصال Max، در **تنظیمات دیگر → حالت هسته** گزینه **CottenMax** و در بخش DNS روش اتصال **خودکار** را انتخاب کنید. در هر دو حالت Normal و Max، کلید و دامنه یکسان هستند. [راهنمای اصلی نصب سرور CottenDNS](https://github.com/WhiteDNS/CottenDNS#step-2--prepare-the-server)

---

<a id="install"></a>

## ⬇️ ۲. نصب روزنه روی روتر با تشخیص خودکار معماری

پس از آماده شدن سرور و نگهداری کلید، در [MobaXterm](https://mobaxterm.mobatek.net/download-home-edition.html) یک اتصال SSH به IP روتر باز کنید و با کاربر `root` وارد شوید. دستور زیر را در ترمینال **روتر** اجرا کنید:

<div dir="ltr" align="left">

```sh
wget -O /tmp/rozaneh-install.sh https://raw.githubusercontent.com/dreamboxone/rozaneh/main/scripts/install.sh && sh /tmp/rozaneh-install.sh
```

</div>

اسکریپت معماری بسته OpenWrt را از `DISTRIB_ARCH` می خواند، مدیر بسته APK یا OPKG را تشخیص می دهد و بسته کامل مناسب را از آخرین انتشار پایدار دانلود می کند. SHA-256 فایل با فهرست انتشار تطبیق داده می شود و سپس نصب انجام می شود. هسته و رابط LuCI در یک بسته هستند. روتر باید به گیتهاب و مخزن بسته های OpenWrt دسترسی داشته باشد.

**این دستور پس از انتشار اسکریپت روی شاخه اصلی و انتشار بسته ها همراه `manifest.json` در بخش Releases قابل استفاده است. فایل های محلی لپ تاپ با این دستور دانلود نمی شوند.**

<a id="manual-install"></a>

## 📂 روش جایگزین مرحله ۲: نصب با [MobaXterm](https://mobaxterm.mobatek.net/download-home-edition.html)

۱. از [بخش Releases](https://github.com/dreamboxone/rozaneh/releases) بسته روزنه مناسب معماری روتر را دانلود کنید. برای دیدن معماری، این دستور را در SSH روتر اجرا کنید:

<div dir="ltr" align="left">

```sh
. /etc/openwrt_release
echo "$DISTRIB_ARCH"
```

</div>

۲. اگر روتر مدیر بسته `apk` دارد فایل `.apk` و اگر `opkg` دارد فایل `.ipk` بگیرید. نام معماری باید با خروجی دستور بالا یکسان باشد.

۳. در پنل انتقال فایل [MobaXterm](https://mobaxterm.mobatek.net/download-home-edition.html) مسیر `/tmp` روتر را باز کنید و فایل دانلود شده را در آن بگذارید. اگر پنل SFTP باز نشد، در تنظیمات نشست، روش SSH-browser را روی SCP بگذارید.

۴. در ترمینال روتر، فقط دستور مناسب مدیر بسته خود را اجرا کنید. مثال زیر برای نسخه ۱.۰.۰ ویرایش ۱ و معماری `arm_cortex-a7_neon-vfpv4` است؛ نام فایل را با فایل خود عوض کنید:

<div dir="ltr" align="left">

```sh
# APK
apk add --allow-untrusted /tmp/rozaneh-1.0.0-r1-arm_cortex-a7_neon-vfpv4.apk

# OPKG
opkg update
opkg install /tmp/rozaneh-1.0.0-r1-arm_cortex-a7_neon-vfpv4.ipk
```

</div>

وابستگی های پایه با مدیر بسته نصب می شوند. سپس به **Services → Rozaneh** بروید. پیش از محدودیت اینترنت، نیازمندی های باقی مانده را از بخش به روزرسانی نصب کنید. دامنه تونل، کلید مشترک و ریزالورها را تنظیم و ذخیره کنید. ابتدا زیرگذر را خاموش و سپس کلید اصلی روزنه را روشن کنید.

<a id="remove"></a>

## 🗑️ ۳. حذف برنامه از روتر

در SSH روتر فقط دستور مناسب مدیر بسته خود را اجرا کنید:

<div dir="ltr" align="left">

```sh
# APK
apk del rozaneh

# OPKG
opkg remove rozaneh
```

</div>

حذف بسته، سرویس روزنه را متوقف می کند. این دستور برای حذف بسته است؛ برای پاک کردن تنظیمات باقی مانده، ابتدا از تنظیمات مورد نیاز خود پشتیبان بگیرید.

<a id="features"></a>

## ✨ امکانات و محدودیت ها

هسته اصلی CottenDNS با رابط فارسی و انگلیسی، حالت روشن و تیره، گزارش ها با کپی و پاک کردن، پراکسی SOCKS5، مسیریابی شبکه و کنترل سرویس در یک بسته نصب می شود. تنظیمات پیشرفته TOML هسته نیز در دسترس است. بعضی گزینه های رابط قدیمی هنوز در حال انتقال هستند. مسیریابی فعلی شبکه برای TCP است.

<a id="updates"></a>

## 🔄 نسخه هسته و بروزرسانی

نسخه هسته در رابط با تاریخ میلادی مانند `2026.10.24` نمایش داده می شود. تگ کامل انتشار و شناسه سورس برای تشخیص دقیق نسخه نگهداری می شوند. هسته همراه بسته کامل روزنه و رابط LuCI بروزرسانی می شود. گردش کار روزانه گیتهاب نسخه منتشرشده CottenDNS را بررسی و پس از بیلد و تست، بسته جدید روزنه را منتشر می کند.

<a id="build"></a>

## 🛠️ ساخت از سورس

از `scripts/build-release.py` برای ساخت معماری های پشتیبانی شده و `scripts/package.py` برای بسته بندی استفاده کنید. نسخه منبع در `cotten-upstream.json` قفل شده است. بیلد به Python، Git و Go سازگار با هسته نیاز دارد؛ فایل های مجوز هسته و وابستگی ها همراه بسته نگهداری می شوند.

<a id="thanks"></a>

## 🤝 سپاسگزاری

از گروه WhiteDNS، سازندگان و مشارکت کنندگان [CottenDNS](https://github.com/WhiteDNS/CottenDNS)، به ویژه [TaJirax](https://github.com/TaJirax)، برای توسعه هسته و انتشار سورس آن سپاسگزاریم. روزنه از هسته اصلی این پروژه استفاده می کند. مجوزها و حقوق سازندگان اصلی محفوظ هستند.

<a id="support"></a>

## 💚 حمایت مالی

اگر روزنه به کارتان آمده، می توانید با واریز تتر از توسعه این پروژه حمایت کنید.

**USDT (تتر) — فقط شبکه BEP20 (BSC)**

<div dir="ltr" align="left">

```
0x56daaa6b76d88ee0c8dba8042121f4b77de0a813
```

</div>

این آدرس برای تتر در شبکه BEP20 (BSC) است؛ ارز دیگر یا شبکه دیگری استفاده نکنید.

</div>

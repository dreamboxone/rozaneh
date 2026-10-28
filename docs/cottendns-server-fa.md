<div dir="rtl" align="right">

# آزمایش خود CottenDNS روی سرور فعلی

این راهنما برای Ubuntu یا Debian با systemd است. هدف، آزمایش CottenDNS اصلی در حالت Normal است؛ این حالت با سرور فعلی Soar و کلید عمومی آن کار نمی کند. نصب روی سرور را خودتان انجام می دهید؛ زیرگذر روی روتر روشن می ماند.

## ۱. بررسی و پشتیبان

در SSH سرور اجرا کنید:

```sh
uname -m
sudo systemctl status soar-server --no-pager
sudo ss -lntup | grep ':53 '
sudo install -d -m 700 /root/soar-before-cotten
sudo cp -a /etc/soar /root/soar-before-cotten/
sudo cp -a /etc/systemd/system/soar-server.service /root/soar-before-cotten/
```

دستورهای پشتیبان فرض می کنند سرور قبلی طبق راهنمای روزنه نصب شده است. اگر مسیر یا نام سرویس متفاوت بود، ابتدا مسیر واقعی را پیدا کنید. کلید قبلی را حذف یا جایگزین نکنید.

## ۲. دامنه و پورت

می توانید همان دامنه تونل و رکورد NS فعلی را استفاده کنید، به شرطی که به همین سرور اشاره کنند. دو برنامه نمی توانند همزمان روی یک IP و پورت UDP/53 گوش کنند. برای این آزمایش، سرویس Soar را متوقف کنید؛ حذف یا غیرفعال کردن دائمی آن لازم نیست:

```sh
sudo systemctl stop soar-server
sudo ss -lntup | grep ':53 '
```

اگر سرویس دیگری پورت را گرفته، آن را بدون بررسی متوقف نکنید. در فایروال پنل سرور، ورودی UDP/53 و TCP/53 باز باشد. پورت بررسی سلامت 9090 فقط محلی می ماند.

## ۳. نصب CottenDNS اصلی

```sh
sudo install -d -m 0755 /opt/cottendns
cd /opt/cottendns
curl -fL https://raw.githubusercontent.com/WhiteDNS/CottenDNS/7db7b73658c791cce62300d29647e4d369d6e1e4/server_linux_install.sh -o /tmp/cottendns-server-install.sh
sudo bash /tmp/cottendns-server-install.sh
```

هنگام پرسش دامنه، دامنه تونل خود را وارد کنید؛ برای مثال `tunnel.example.com`، نه نام سرور NS. اسکریپت از نسخه مشخصی از مخزن گرفته می شود، ولی بسته سرور را از انتشار موجود دریافت می کند. نصاب می تواند تنظیمات فایروال و DNS سیستم را برای آزاد کردن پورت ۵۳ تغییر دهد؛ آن را روی سروری اجرا کنید که تغییر این تنظیمات در آن مجاز است.

## ۴. بررسی نصب

```sh
sudo systemctl is-active cottendns
curl -fS --max-time 5 http://127.0.0.1:9090/healthz
sudo ss -lnup | grep ':53 '
sudo ss -lntp | grep ':53 '
sudo journalctl -u cottendns -n 50 --no-pager
```

فعال بودن سرویس و پاسخ healthz فقط سلامت محلی سرور را نشان می دهند. دسترسی واقعی از ایران باید با کلاینت CottenDNS و انتقال داده بررسی شود.

کلید CottenDNS در این مسیر است:

```sh
sudo cat /opt/cottendns/encrypt_key.txt
```

این یک کلید مشترک محرمانه است و با کلید عمومی Soar تفاوت دارد. آن را در پیام، تصویر یا گزارش عمومی نفرستید؛ بعدا در تنظیمات محلی کلاینت وارد می شود. تنظیمات سرور در `/opt/cottendns/server_config.toml` قرار دارد.

## ۵. بازگشت به Soar

اگر نصب یا آزمایش موفق نبود:

```sh
sudo systemctl disable --now cottendns
sudo systemctl start soar-server
sudo systemctl status soar-server --no-pager
sudo ss -lnup | grep ':53 '
```

این مرحله فایل ها و کلید CottenDNS را حذف نمی کند. اگر نصب کننده تنظیمات DNS یا فایروال سیستم را تغییر داده باشد، بازگشت آن تغییرات جداگانه بررسی می شود.

راهنمای مرجع: [مخزن رسمی CottenDNS](https://github.com/WhiteDNS/CottenDNS#step-2--prepare-the-server).

</div>

# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
include $(TOPDIR)/rules.mk

PKG_NAME:=rozaneh
PKG_VERSION:=1.0.0
PKG_RELEASE:=7
PKG_LICENSE:=Apache-2.0 AGPL-3.0-or-later MIT BSD-3-Clause OFL-1.1
PKG_LICENSE_FILES:=LICENSE SOAR-LICENSE luci/LICENSE licenses/CottenDNS-LICENSE licenses/quic-go-LICENSE licenses/qpack-LICENSE licenses/Go-LICENSE luci/root/www/luci-static/resources/rozaneh/fonts/OFL.txt
PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk

COTTEN_VERSION:=$(shell sed -n 's/.*"revision": *"\([^"]*\)".*/\1/p' $(CURDIR)/cotten-upstream.json)

ROZANEH_BINARY?=$(CURDIR)/prebuilt/$(ARCH_PACKAGES)/rozaneh

define Package/rozaneh
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Web Servers/Proxies
  TITLE:=Rozaneh DNS tunnel client
  URL:=https://github.com/dreamboxone/rozaneh
  USERID:=rozaneh:rozaneh
  DEPENDS:=+jshn +curl +ca-bundle +luci-base +rpcd +jsonfilter
  PROVIDES:=luci-app-rozaneh
  CONFLICTS:=luci-app-rozaneh
endef

define Package/rozaneh/description
 TCP SOCKS5 proxy using the CottenDNS engine, managed by procd and UCI.
 Requires a compatible server and delegated DNS zone. Disabled by default.
endef

define Build/Prepare
	$(INSTALL_DIR) $(PKG_BUILD_DIR)
	test -f "$(ROZANEH_BINARY)"
	$(CP) "$(ROZANEH_BINARY)" $(PKG_BUILD_DIR)/rozaneh
endef

define Build/Compile
endef

define Package/rozaneh/conffiles
/etc/config/rozaneh
/etc/rozaneh/
endef

define Package/rozaneh/install
	$(INSTALL_DIR) $(1)/usr/bin $(1)/etc/init.d $(1)/etc/config $(1)/usr/share/licenses/rozaneh
	$(INSTALL_DIR) $(1)/usr/share/rozaneh
	echo '$(PKG_VERSION)-r$(PKG_RELEASE)' > $(1)/usr/share/rozaneh/core.version
	echo '$(PKG_VERSION)-r$(PKG_RELEASE)' > $(1)/usr/share/rozaneh/luci.version
	echo '$(COTTEN_VERSION)' > $(1)/usr/share/rozaneh/engine.version
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/rozaneh $(1)/usr/bin/rozaneh
	$(INSTALL_DIR) $(1)/usr/libexec/rozaneh
	$(INSTALL_BIN) $(CURDIR)/files/dependencies.sh $(CURDIR)/files/lan-config.sh $(CURDIR)/files/lan-run.sh $(CURDIR)/files/lan-cleanup.sh $(CURDIR)/files/lan-firewall.sh $(CURDIR)/files/health-check.sh $(CURDIR)/files/health-run.sh $(1)/usr/libexec/rozaneh/
	$(INSTALL_BIN) $(CURDIR)/files/migrate-paths.sh $(1)/usr/libexec/rozaneh/
	$(INSTALL_BIN) $(CURDIR)/files/cotten-config.sh $(1)/usr/libexec/rozaneh/
	$(INSTALL_BIN) $(CURDIR)/files/rozaneh.init $(1)/etc/init.d/rozaneh
	$(INSTALL_BIN) $(CURDIR)/files/rozaneh.init $(1)/usr/share/rozaneh/service.init
	$(INSTALL_CONF) $(CURDIR)/files/rozaneh.config $(1)/etc/config/rozaneh
	chmod 600 $(1)/etc/config/rozaneh
	$(INSTALL_DATA) $(CURDIR)/LICENSE $(CURDIR)/SOAR-LICENSE $(CURDIR)/NOTICE $(CURDIR)/licenses/* $(1)/usr/share/licenses/rozaneh/
	$(INSTALL_DIR) $(1)/usr/share/licenses/rozaneh/engine
	$(CP) $(dir $(ROZANEH_BINARY))engine-licenses/* $(1)/usr/share/licenses/rozaneh/engine/
	$(CP) $(CURDIR)/luci/root/* $(1)/
	chmod 755 $(1)/usr/libexec/rpcd/rozaneh $(1)/usr/libexec/rozaneh/*
	$(INSTALL_DIR) $(1)/usr/share/licenses/rozaneh-ui
	$(INSTALL_DATA) $(CURDIR)/luci/LICENSE $(CURDIR)/luci/NOTICE $(1)/usr/share/licenses/rozaneh-ui/
endef

define Package/rozaneh/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
cp /usr/share/rozaneh/service.init /etc/init.d/rozaneh || exit 1
chmod 755 /etc/init.d/rozaneh || exit 1
/usr/libexec/rozaneh/migrate-paths.sh || exit 1
/etc/init.d/rozaneh enable
/etc/init.d/rpcd reload >/dev/null 2>&1 || true
rm -f /tmp/luci-indexcache.*
[ "$$(uci -q get rozaneh.main.enabled)" = 1 ] && /etc/init.d/rozaneh restart
exit 0
endef

define Package/rozaneh/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || /etc/init.d/rozaneh stop
exit 0
endef

define Package/rozaneh/postrm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
if [ ! -x /usr/bin/rozaneh ]; then
 nft delete table inet rozaneh_guard 2>/dev/null || true
 if [ "$$(uci -q get firewall.rozaneh_guard.path)" = /usr/libexec/rozaneh/lan-firewall.sh ]; then
  uci -q delete firewall.rozaneh_guard
  uci commit firewall
 fi
fi
exit 0
endef

$(eval $(call BuildPackage,rozaneh))

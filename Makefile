include $(TOPDIR)/rules.mk

PKG_NAME:=zoordns
PKG_VERSION:=0.1.0
PKG_RELEASE:=2
PKG_LICENSE:=Apache-2.0
PKG_LICENSE_FILES:=LICENSE SOAR-LICENSE
PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)

include $(INCLUDE_DIR)/package.mk

ZOORDNS_BINARY?=$(CURDIR)/prebuilt/$(ARCH_PACKAGES)/zoordns

define Package/zoordns
  SECTION:=net
  CATEGORY:=Network
  SUBMENU:=Web Servers/Proxies
  TITLE:=zoorDNS DNS tunnel client
  URL:=https://github.com/dreamboxone/zoordns
  USERID:=zoordns:zoordns
endef

define Package/zoordns/description
 TCP SOCKS5 proxy using the Soar DNS tunnel engine, managed by procd and UCI.
 Requires a compatible server and delegated DNS zone. Disabled by default.
endef

define Build/Prepare
	$(INSTALL_DIR) $(PKG_BUILD_DIR)
	test -f "$(ZOORDNS_BINARY)"
	$(CP) "$(ZOORDNS_BINARY)" $(PKG_BUILD_DIR)/zoordns
endef

define Build/Compile
endef

define Package/zoordns/conffiles
/etc/config/zoordns
/etc/zoordns/
endef

define Package/zoordns/install
	$(INSTALL_DIR) $(1)/usr/bin $(1)/etc/init.d $(1)/etc/config $(1)/usr/share/licenses/zoordns
	$(INSTALL_DIR) $(1)/usr/share/zoordns
	echo '$(PKG_VERSION)-r$(PKG_RELEASE)' > $(1)/usr/share/zoordns/core.version
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/zoordns $(1)/usr/bin/zoordns
	$(INSTALL_BIN) $(CURDIR)/files/zoordns.init $(1)/etc/init.d/zoordns
	$(INSTALL_CONF) $(CURDIR)/files/zoordns.config $(1)/etc/config/zoordns
	chmod 600 $(1)/etc/config/zoordns
	$(INSTALL_DATA) $(CURDIR)/LICENSE $(CURDIR)/SOAR-LICENSE $(CURDIR)/NOTICE $(CURDIR)/licenses/* $(1)/usr/share/licenses/zoordns/
endef

define Package/zoordns/prerm
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] || /etc/init.d/zoordns stop
exit 0
endef

$(eval $(call BuildPackage,zoordns))

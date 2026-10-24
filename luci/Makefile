include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-zoordns
PKG_VERSION:=0.1.0
PKG_RELEASE:=3
PKG_LICENSE:=AGPL-3.0-or-later
PKG_LICENSE_FILES:=LICENSE
PKG_ARCH:=all

include $(INCLUDE_DIR)/package.mk

define Package/luci-app-zoordns
  SECTION:=luci
  CATEGORY:=LuCI
  SUBMENU:=3. Applications
  TITLE:=LuCI interface for zoorDNS
  URL:=https://github.com/dreamboxone/zoordns
  DEPENDS:=+zoordns +luci-base +rpcd +jsonfilter
  PKGARCH:=all
endef

define Build/Compile
endef

define Package/luci-app-zoordns/install
	$(CP) ./root/* $(1)/
	chmod 755 $(1)/usr/libexec/rpcd/zoordns
	chmod 755 $(1)/usr/libexec/zoordns/install-deps
	chmod 755 $(1)/usr/libexec/zoordns/probe-sites $(1)/usr/libexec/zoordns/update
	$(INSTALL_DIR) $(1)/usr/share/zoordns
	echo '$(PKG_VERSION)-r$(PKG_RELEASE)' > $(1)/usr/share/zoordns/luci.version
	$(INSTALL_DIR) $(1)/usr/share/licenses/luci-app-zoordns
	$(INSTALL_DATA) ./LICENSE ./NOTICE $(1)/usr/share/licenses/luci-app-zoordns/
endef

define Package/luci-app-zoordns/postinst
#!/bin/sh
[ -n "$${IPKG_INSTROOT}" ] && exit 0
/etc/init.d/rpcd restart >/dev/null 2>&1 || true
rm -f /tmp/luci-indexcache.*
exit 0
endef

$(eval $(call BuildPackage,luci-app-zoordns))

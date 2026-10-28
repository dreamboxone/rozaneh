# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
import importlib.util
import io
from pathlib import Path
import tarfile
import tempfile
import unittest


spec = importlib.util.spec_from_file_location("package", Path(__file__).resolve().parents[1] / "scripts/package.py")
package = importlib.util.module_from_spec(spec)
spec.loader.exec_module(package)


class PackageArchiveTests(unittest.TestCase):
    def test_unified_package_contains_engine_interface_and_licenses(self):
        with tempfile.TemporaryDirectory() as temporary:
            stage = Path(temporary)
            package.stage_component(stage, "rozaneh", "arm_cortex-a7_neon-vfpv4")
            for name in ("usr/bin/rozaneh", "usr/libexec/rpcd/rozaneh", "usr/libexec/rozaneh/update", "www/luci-static/resources/view/rozaneh/dashboard.js", "www/luci-static/resources/view/rozaneh/update.js", "usr/share/licenses/rozaneh/LICENSE", "usr/share/licenses/rozaneh-ui/LICENSE"):
                self.assertTrue((stage / name).is_file(), name)
            self.assertEqual((stage / "usr/share/rozaneh/core.version").read_text(), (stage / "usr/share/rozaneh/luci.version").read_text())
            self.assertEqual(list(package.VERSIONS), ["rozaneh"])
            for license_name in ("CottenDNS-LICENSE","quic-go-LICENSE","qpack-LICENSE","golang-x-net-LICENSE","golang-x-text-LICENSE"):
                self.assertTrue((stage / "usr/share/licenses/rozaneh" / license_name).is_file(),license_name)
            self.assertIn("tajirax",(stage / "usr/share/licenses/rozaneh/CottenDNS-LICENSE").read_text())
            self.assertIn("encrypted_dns_only",(stage / "etc/config/rozaneh").read_text())
            self.assertIn("-config /var/run/rozaneh/client.toml",(stage / "etc/init.d/rozaneh").read_text())
            self.assertEqual((stage / "usr/share/rozaneh/service.init").read_bytes(),(stage / "etc/init.d/rozaneh").read_bytes())
            self.assertIn("cp /usr/share/rozaneh/service.init /etc/init.d/rozaneh",package.postinst("rozaneh"))

    def test_portable_paths_and_install_permissions(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for name, mode in {"usr/libexec/rozaneh/probe-sites": 0o755, "usr/bin/rozaneh": 0o755, "etc/config/rozaneh": 0o600, "usr/share/licenses/rozaneh/LICENSE": 0o644}.items():
                file = root / name
                file.parent.mkdir(parents=True, exist_ok=True)
                file.write_text("fixture\n")
                file.chmod(mode)
            with tarfile.open(fileobj=io.BytesIO(package.archive(root)), mode="r:gz") as archive:
                members = {member.name: member for member in archive.getmembers()}
                self.assertTrue(all("\\" not in name for name in members))
                self.assertEqual(members["./usr/libexec/rozaneh/probe-sites"].mode, 0o755)
                self.assertEqual(members["./usr/bin/rozaneh"].mode, 0o755)
                self.assertEqual(members["./etc/config/rozaneh"].mode, 0o600)
                self.assertEqual(members["./usr/share/licenses/rozaneh/LICENSE"].mode, 0o644)


if __name__ == "__main__":
    unittest.main()

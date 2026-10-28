# SPDX-License-Identifier: Apache-2.0
# Copyright (C) 2026 dreamboxone <https://t.me/routekernel1>
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parents[1]


def load(name, filename):
    spec = importlib.util.spec_from_file_location(name, ROOT / "scripts" / filename)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


release = load("soar_release", "soar-release.py")
builder = load("build_release", "build-release.py")


class ReleaseTests(unittest.TestCase):
    def test_pseudo_version_of_same_release_is_not_an_update(self):
        self.assertFalse(release.needs_update("v0.0.0-20261022160106-fea80aae6c60", "v0.2.4", "fea80aae6c60abcdef", "2026-10-22T16:01:06Z"))

    def test_downgrades_and_prereleases_are_rejected(self):
        self.assertFalse(release.needs_update("v0.2.5", "v0.2.4", "abcd", "2026-10-22T16:01:06Z"))
        self.assertFalse(release.needs_update("v0.0.0-20261022160106-fea80aae6c60", "v0.2.3", "abcd", "2026-10-21T16:01:06Z"))
        with self.assertRaises(ValueError):
            release.stable_version("v0.2.5-rc1")
        with self.assertRaises(ValueError):
            release.stable_version("v0.2.5;command")

    def test_new_stable_version_and_revision(self):
        self.assertTrue(release.needs_update("v0.2.4", "v0.2.5", "abcd", "2026-10-23T16:01:06Z"))
        self.assertEqual(release.next_package_version('VERSIONS = {"rozaneh": "0.1.0-r11"}'), "0.1.0-r12")

    def test_all_published_targets_have_build_settings(self):
        for target in json.loads((ROOT / "targets.json").read_text()):
            self.assertIn("GOARCH", builder.target_environment(target))
        self.assertEqual(builder.target_environment("arm_arm1176jzf-s_vfp")["GOARM"], "6")
        self.assertEqual(builder.target_environment("arm_xscale")["GOARM"], "5")
        self.assertNotEqual(builder.target_environment("mips_24kc"), builder.target_environment("mipsel_24kc"))
        with self.assertRaises(ValueError):
            builder.target_environment("unknown")

    def test_verifier_rejects_incomplete_packages(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            state = root / "state.json"
            state.write_text(json.dumps({"engine":"v0.2.5", "package_version":"0.1.0-r12"}))
            (root / "manifest.json").write_text(json.dumps({"components":{"rozaneh":{"engine_version":"v0.2.5", "version":"0.1.0-r12", "bundled_luci":True, "assets":{"ipk":{}, "apk":{}}}}}))
            with self.assertRaisesRegex(ValueError, "Incomplete architecture"):
                release.verify(root, state)

    def test_prepare_rejects_changed_upstream_tag(self):
        with tempfile.TemporaryDirectory() as temporary:
            state = Path(temporary) / "state.json"
            state.write_text(json.dumps({"engine":"v0.2.5", "engine_sha":"expected"}))
            with patch.object(release, "run", side_effect=["", "", json.dumps({"Version":"v0.2.5", "Origin":{"Hash":"different"}})]):
                with self.assertRaisesRegex(ValueError, "release commit"):
                    release.prepare(state)


if __name__ == "__main__":
    unittest.main()

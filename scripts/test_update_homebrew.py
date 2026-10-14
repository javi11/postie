import copy
import unittest
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

import update_homebrew


class HomebrewReleaseTests(unittest.TestCase):
    def setUp(self):
        self.release = {
            "tag_name": "v1.2.3", "draft": False, "prerelease": False,
            "assets": [
                {"name": f"postie-cli_v1.2.3_{platform}_{arch}.tar.gz",
                 "browser_download_url": f"https://github.com/kipsilabs/postie/releases/download/v1.2.3/postie-cli_v1.2.3_{platform}_{arch}.tar.gz",
                 "digest": "sha256:" + "a" * 64}
                for platform in ("darwin", "linux") for arch in ("arm64", "amd64")
            ],
        }

    def test_formula_pins_version_and_covers_all_four_platforms(self):
        formula = update_homebrew.render_formula(self.release)
        self.assertIn('version "1.2.3"', formula)
        self.assertEqual(formula.count('sha256 "' + "a" * 64 + '"'), 4)
        for asset in self.release["assets"]:
            self.assertIn(asset["browser_download_url"], formula)
        self.assertIn('bin.install "postie-cli-#{os}-#{arch}" => "postie"', formula)

    def test_missing_platform_asset_is_rejected(self):
        self.release["assets"].pop()
        with self.assertRaisesRegex(ValueError, "Missing release asset"):
            update_homebrew.render_formula(self.release)

    def test_invalid_or_missing_checksum_is_rejected(self):
        for digest in (None, "sha256:abc", "sha512:" + "a" * 64, "sha256:" + "z" * 64):
            with self.subTest(digest=digest):
                self.release["assets"][0]["digest"] = digest
                with self.assertRaisesRegex(ValueError, "SHA256"):
                    update_homebrew.render_formula(self.release)

    def test_unexpected_download_origin_is_rejected(self):
        self.release["assets"][0]["browser_download_url"] = "https://example.com/archive.tar.gz"
        with self.assertRaisesRegex(ValueError, "URL"):
            update_homebrew.render_formula(self.release)

    def test_unstable_or_invalid_tag_is_rejected(self):
        for tag in ("v1.2.3-rc1", "latest", 'v1.2.3"'):
            with self.subTest(tag=tag):
                release = copy.deepcopy(self.release)
                release["tag_name"] = tag
                with self.assertRaisesRegex(ValueError, "stable"):
                    update_homebrew.render_formula(release)
        for field in ("draft", "prerelease"):
            release = copy.deepcopy(self.release)
            release[field] = True
            with self.assertRaisesRegex(ValueError, "stable"):
                update_homebrew.render_formula(release)

    def test_invalid_input_leaves_existing_formula_unchanged(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root / "scripts").mkdir()
            (root / "Formula").mkdir()
            script = root / "scripts" / "update_homebrew.py"
            shutil.copyfile(update_homebrew.__file__, script)
            formula = root / "Formula" / "postie.rb"
            formula.write_text("existing formula")
            for payload in ("not JSON", "[]", json.dumps({"message": "Not Found"})):
                with self.subTest(payload=payload):
                    result = subprocess.run([sys.executable, str(script)], input=payload,
                                            text=True, capture_output=True, check=False)
                    self.assertEqual(result.returncode, 1, result.stderr)
                    self.assertIn("Cannot update Homebrew formula", result.stderr)
                    self.assertEqual(formula.read_text(), "existing formula")

    def test_api_error_response_is_rejected(self):
        with self.assertRaisesRegex(ValueError, "stable"):
            update_homebrew.render_formula({"message": "Not Found"})


if __name__ == "__main__":
    unittest.main()

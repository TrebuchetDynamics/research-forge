"""Offline contract for the Hermes skill-only registration seam."""

import importlib.util
from pathlib import Path
import sys
import unittest
from unittest.mock import patch


class HermesSkillTest(unittest.TestCase):
    def test_registers_only_public_skill(self):
        root = Path(__file__).resolve().parents[1]
        for directory in (root, root / "hermes-plugin"):
            with self.subTest(directory=directory):
                self.assertTrue((directory / "__init__.py").is_file())
                spec = importlib.util.spec_from_file_location("research_forge_plugin", directory / "__init__.py")
                plugin = importlib.util.module_from_spec(spec)
                # Loading the contract must not add files to the scanned package.
                with patch.object(sys, "dont_write_bytecode", True):
                    spec.loader.exec_module(plugin)
                registered = []

                class Context:
                    # Any hook, command, tool, settings or background API call fails.
                    def register_skill(self, name, path, **metadata):
                        registered.append((name, path))

                plugin.register(Context())
                self.assertEqual(registered, [("research-forge", directory / "skills/research-forge/SKILL.md")])
                self.assertTrue(registered[0][1].is_file())

    def test_isolated_package_contains_only_synchronized_skill(self):
        root = Path(__file__).resolve().parents[1]
        canonical = root / "skills/research-forge"
        packaged = root / "hermes-plugin"
        skill_files = {path.relative_to(canonical): path.read_bytes()
                       for path in canonical.rglob("*") if path.is_file()}
        files = {path.relative_to(packaged)
                 for path in packaged.rglob("*") if path.is_file()}
        self.assertEqual(files, {Path("plugin.yaml"), Path("__init__.py")} |
                         {Path("skills/research-forge") / path for path in skill_files})
        for path, content in skill_files.items():
            with self.subTest(path=path):
                self.assertEqual((packaged / "skills/research-forge" / path).read_bytes(), content)


if __name__ == "__main__":
    unittest.main()

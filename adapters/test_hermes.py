"""Offline contract for the Hermes skill-only registration seam."""

import importlib.util
from pathlib import Path
import unittest


class HermesSkillTest(unittest.TestCase):
    def test_registers_only_public_skill(self):
        root = Path(__file__).resolve().parents[1]
        spec = importlib.util.spec_from_file_location("research_forge_plugin", root / "__init__.py")
        plugin = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(plugin)
        registered = []

        class Context:
            # Any hook, command, tool, settings or background API call fails.
            def register_skill(self, name, path, **metadata):
                registered.append((name, path))

        plugin.register(Context())
        self.assertEqual(registered, [("research-forge", root / "skills/research-forge/SKILL.md")])
        self.assertTrue(registered[0][1].is_file())


if __name__ == "__main__":
    unittest.main()

"""Passive Hermes entry point: expose only the public methodology skill."""

from pathlib import Path


def register(ctx):
    """Register read-only skill content; never provision defaults or run work."""
    ctx.register_skill(
        "research-forge",
        Path(__file__).resolve().parent / "skills/research-forge/SKILL.md",
        description="Research with rforge provenance, source coverage, and human review gates.",
    )

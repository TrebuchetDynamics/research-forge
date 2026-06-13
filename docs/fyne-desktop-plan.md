# Superseded Fyne desktop implementation plan

This file is retained only as a historical pointer. ResearchForge no longer plans a native Fyne desktop GUI as the primary interface.

Current direction:

- [ADR 0006](adr/0006-rescope-fyne-desktop-to-local-web-gui.md) re-scopes the UI to a local browser-based web GUI.
- [web-gui-plan.md](web-gui-plan.md) contains the active implementation plan.
- `make web-gui-smoke` runs the current Go + HTMX handler smoke tests in `internal/webui`.

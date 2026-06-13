# Implementation tracker: web_gui_stack_scope

## Decision ID

web_gui_stack_scope

## Status

complete

## Selected stack

Go + HTMX local research cockpit, per ADR 0006.

## Tracking issue

https://github.com/TrebuchetDynamics/research-forge/issues/2

## Completed TODO items

- Add Go + HTMX web GUI workspace/dependencies
- Add web GUI search screen
- Add web GUI library screen
- Create/open a research project from the web GUI
- View OSS repository studies in the web GUI
- View CLI-generated papers, meta-analysis outputs, PRISMA/citation diagrams, and report artifacts in the web GUI
- Guide small local review actions without replacing CLI automation

## Evidence

- `internal/webui` shell, search, library, project, OSS, and artifact handlers
- `internal/webui/*_test.go` Go + HTMX handler tests
- `web/assets/researchforge.css`
- `make check`
- `rforge decisions --completion-audit TODO.md docs/todo-completion-audit.md`

## Outcome

The implementation tracker no longer blocks TODO completion. The final purpose is a local research cockpit: project review, artifact navigation, and guided local actions over CLI-generated state. Remaining unchecked TODO coverage is limited to `project_license` pending owner decision.

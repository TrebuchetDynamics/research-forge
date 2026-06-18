#!/usr/bin/env bash
set -euo pipefail

# Colors — disabled when stdout is not a TTY
if [ -t 1 ]; then
  BOLD='\033[1m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  RED='\033[0;31m'
  CYAN='\033[0;36m'
  DIM='\033[2m'
  RESET='\033[0m'
else
  BOLD='' GREEN='' YELLOW='' RED='' CYAN='' DIM='' RESET=''
fi

step() { printf "  ${CYAN}→${RESET}  %s\n" "$*"; }
ok()   { printf "  ${GREEN}✓${RESET}  %s\n" "$*"; }
warn() { printf "  ${YELLOW}!${RESET}  %s\n" "$*"; }
die()  { printf "\n  ${RED}✗${RESET}  %s\n\n" "$*" >&2; exit 1; }

printf "\n${BOLD}ResearchForge${RESET}${DIM} — rforge installer${RESET}\n\n"

# ── Check Go ───────────────────────────────────────────────────────────────

step "Checking Go..."
command -v go >/dev/null 2>&1 \
  || die "Go not found. Install Go 1.22+ from https://go.dev/dl/ then re-run."

GO_VERSION=$(go version | sed 's/.*go\([0-9][0-9]*\.[0-9][0-9]*\).*/\1/')
GO_MAJOR=$(printf '%s' "$GO_VERSION" | cut -d. -f1)
GO_MINOR=$(printf '%s' "$GO_VERSION" | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || { [ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 22 ]; }; then
  die "Go 1.22+ required (found go${GO_VERSION}). Upgrade at https://go.dev/dl/"
fi

ok "Go ${GO_VERSION}"

# ── Install ────────────────────────────────────────────────────────────────

step "Running go install..."
go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@latest
ok "Binary → $(go env GOPATH)/bin/rforge"

# ── PATH check ─────────────────────────────────────────────────────────────

if ! command -v rforge >/dev/null 2>&1; then
  GOBIN="$(go env GOPATH)/bin"
  printf "\n"
  warn "rforge is not on your PATH."
  printf "  Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):\n\n"
  printf "    ${DIM}export PATH=\"\$PATH:%s\"${RESET}\n\n" "$GOBIN"
  printf "  Then reload your shell or run:\n\n"
  printf "    ${DIM}source ~/.bashrc${RESET}\n\n"
else
  VERSION=$(rforge --version 2>/dev/null || true)
  ok "${VERSION:-rforge} → $(command -v rforge)"
fi

# ── Next steps ─────────────────────────────────────────────────────────────

printf "\n${BOLD}Next steps${RESET}\n\n"
printf "  ${DIM}# Create a project${RESET}\n"
printf "  rforge project create ./my-review --title \"My systematic review\"\n\n"
printf "  ${DIM}# Search and import papers${RESET}\n"
printf "  rforge search import --source openalex --query \"...\" --project ./my-review\n\n"
printf "  ${DIM}# Open the local UI${RESET}\n"
printf "  rforge --project ./my-review ui\n\n"
printf "  ${DIM}# Guided review package workflow${RESET}\n"
printf "  rforge forge init --project ./my-review --question \"...\"\n\n"

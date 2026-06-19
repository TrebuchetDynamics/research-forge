#!/usr/bin/env bash
set -euo pipefail

# Colors — disabled when stdout is not a TTY.
if [ -t 1 ]; then
  BOLD='\033[1m'
  GREEN='\033[0;32m'
  YELLOW='\033[1;33m'
  RED='\033[0;31m'
  CYAN='\033[0;36m'
  DIM='\033[2m'
  RESET='\033[0m'
else
  BOLD=''
  GREEN=''
  YELLOW=''
  RED=''
  CYAN=''
  DIM=''
  RESET=''
fi

header() { printf "\n${BOLD}%s${RESET}${DIM} — %s${RESET}\n\n" "$1" "$2"; }
section() { printf "\n${BOLD}%s${RESET}\n\n" "$1"; }
cmd() { printf "  ${DIM}%s${RESET}\n" "$*"; }
step() { printf "  ${CYAN}→${RESET}  %s\n" "$*"; }
ok() { printf "  ${GREEN}✓${RESET}  %s\n" "$*"; }
warn() { printf "  ${YELLOW}!${RESET}  %s\n" "$*"; }
die() {
  printf "\n  ${RED}✗${RESET}  %s\n\n" "$*" >&2
  exit 1
}

usage() {
  cat <<EOF
Usage: install.sh [--version vX.Y.Z] [--dir DIR] [--from-source] [--dry-run]

Defaults to the latest GitHub release binary. Use --from-source only when you
want to build with Go locally.
EOF
}

command_exists() { command -v "$1" >/dev/null 2>&1; }

choose_install_dir() {
  if [ -w /usr/local/bin ]; then
    printf /usr/local/bin
  else
    printf '%s/.local/bin' "$HOME"
  fi
}

VERSION="${RFORGE_VERSION:-latest}"
INSTALL_DIR="${RFORGE_INSTALL_DIR:-}"
FROM_SOURCE="${RFORGE_INSTALL_FROM_SOURCE:-0}"
DRY_RUN=0

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    --dir)
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --from-source)
      FROM_SOURCE=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h | --help)
      usage
      exit 0
      ;;
    *)
      die "unknown option: $1"
      ;;
  esac
done

[ -n "$VERSION" ] || die "--version needs a value"
[ -n "$INSTALL_DIR" ] || INSTALL_DIR=$(choose_install_dir)

header "ResearchForge" "rforge installer"

platform() {
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)

  case "$os" in
    linux) os=linux ;;
    darwin) os=darwin ;;
    msys* | mingw* | cygwin*) os=windows ;;
    *) die "unsupported OS: $os" ;;
  esac

  case "$arch" in
    x86_64 | amd64) arch=amd64 ;;
    arm64 | aarch64) arch=arm64 ;;
    *) die "unsupported architecture: $arch" ;;
  esac

  printf '%s-%s' "$os" "$arch"
}

asset_name() {
  p=$(platform)
  if [ "${p%%-*}" = windows ]; then
    printf 'rforge-%s.exe' "$p"
  else
    printf 'rforge-%s' "$p"
  fi
}

install_name() {
  case "$(platform)" in
    windows-*) printf 'rforge.exe' ;;
    *) printf 'rforge' ;;
  esac
}

release_url() {
  asset=$(asset_name)
  if [ "$VERSION" = latest ]; then
    printf 'https://github.com/TrebuchetDynamics/research-forge/releases/latest/download/%s' "$asset"
  else
    printf 'https://github.com/TrebuchetDynamics/research-forge/releases/download/%s/%s' "$VERSION" "$asset"
  fi
}

source_install() {
  step "Checking Go..."
  command_exists go || die "Go not found. Re-run without --from-source to use release binaries, or install Go from https://go.dev/dl/."

  step "Running go install..."
  if [ "$DRY_RUN" -eq 1 ]; then
    ok "dry-run: go install github.com/TrebuchetDynamics/research-forge/cmd/rforge@${VERSION}"
    return
  fi

  go install "github.com/TrebuchetDynamics/research-forge/cmd/rforge@${VERSION}"
  ok "Binary → $(go env GOPATH)/bin/rforge"
}

release_install() {
  url=$(release_url)
  name=$(install_name)

  step "Installing release binary: $url"
  if [ "$DRY_RUN" -eq 1 ]; then
    ok "dry-run: install $(asset_name) to $INSTALL_DIR/$name"
    return
  fi

  command_exists curl || command_exists wget || die "Need curl or wget to download release binaries. Or run with --from-source if Go is installed."

  mkdir -p "$INSTALL_DIR"
  tmp=$(mktemp -d)
  trap 'rm -rf "$tmp"' EXIT

  bin="$tmp/$name"
  if command_exists curl; then
    curl -fsSL "$url" -o "$bin"
  else
    wget -q -O "$bin" "$url"
  fi

  chmod +x "$bin"
  mv "$bin" "$INSTALL_DIR/$name"
  ok "Binary → $INSTALL_DIR/$name"
}

if [ "$FROM_SOURCE" = 1 ]; then
  source_install
else
  release_install
fi

# PATH check.
if ! command_exists rforge && [ "$INSTALL_DIR" != /usr/local/bin ]; then
  printf "\n"
  warn "rforge may not be on your PATH."
  printf "  Add the following to your shell profile (~/.bashrc, ~/.zshrc, etc.):\n\n"
  printf "    ${DIM}export PATH=\"\$PATH:%s\"${RESET}\n\n" "$INSTALL_DIR"
else
  VERSION_TEXT=$(rforge version 2>/dev/null || rforge --version 2>/dev/null || true)
  ok "${VERSION_TEXT:-rforge} → $(command -v rforge 2>/dev/null || printf '%s/%s' "$INSTALL_DIR" "$(install_name)")"
fi

section "Next steps"
printf "  ${CYAN}Create a project${RESET}\n"
cmd "rforge project create ./my-review --title \"My systematic review\""
printf "\n  ${CYAN}Search and import papers${RESET}\n"
cmd "rforge search import --source openalex --query \"...\" --project ./my-review"
printf "\n  ${CYAN}Open the local UI${RESET}\n"
cmd "rforge --project ./my-review ui"
printf "\n  ${CYAN}Guided review package workflow${RESET}\n"
cmd "rforge forge init --project ./my-review --question \"...\""
printf "\n"

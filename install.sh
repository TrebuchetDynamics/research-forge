#!/usr/bin/env bash
# ResearchForge unified installer.
#
# One entry point for the rforge engine, the public agent skills and the native
# Claude Code, Codex, Pi and Hermes integrations. Components install at user
# scope (never system-wide, never with sudo), failures stay isolated per
# component, and --dry-run changes nothing.
#
#   curl -fsSL .../install.sh | bash          the full suite for available hosts
#   ./install.sh --menu                       interactive component selection
#   ./install.sh --only binary,skills,hermes  exactly these components
#   ./install.sh --list                       components and availability
#   ./install.sh update                       upgrade what is already installed
#   ./install.sh --dry-run ...                print commands, change nothing
#
# Written for bash 3.2 (macOS default): no associative arrays, no ${x,,}.
set -euo pipefail

REPO_SLUG="TrebuchetDynamics/research-forge"
PI_SOURCE="git:github.com/${REPO_SLUG}"
PLUGIN_URL="https://github.com/${REPO_SLUG}"
RAW_BASE="${RFORGE_RAW_BASE:-https://raw.githubusercontent.com/${REPO_SLUG}}"
RELEASE_BASE="${RFORGE_RELEASE_BASE:-https://github.com/${REPO_SLUG}/releases}"
MIN_RFORGE_VERSION="0.1.22"
REQUIRED_CAPABILITIES="search=1 paper=1 brief=1 compare=1 provenance=1 mcp=1"
# Public, user-facing skills. Developer *-tdd skills are never installed.
PUBLIC_SKILLS="research-forge lit paper brief compare replicate"
COMPONENT_KEYS="binary skills claude codex pi hermes"

# --- defaults -------------------------------------------------------------------

COMMAND="install"
VERSION="${RFORGE_VERSION:-latest}"
INSTALL_DIR="${RFORGE_INSTALL_DIR:-}"
FROM_SOURCE="${RFORGE_INSTALL_FROM_SOURCE:-0}"
DRY_RUN=0
MENU=0
SHOW_LIST=0
ONLY=""
ONLY_GIVEN=0
PROFILE=""
SKIP=""
VERBOSE=1
QUIET=0
COLOR="auto"
HERMES_PROFILES=""
HERMES_PROFILES_GIVEN=0

SELF_PATH="${BASH_SOURCE[0]:-}"
if [ -n "$SELF_PATH" ] && [ -f "$SELF_PATH" ]; then
  SELF_DIR=$(cd -- "$(dirname -- "$SELF_PATH")" && pwd -P)
else
  SELF_DIR=""
fi

USE_COLOR=0
BOLD=""; DIM=""; GREEN=""; YELLOW=""; RED=""; CYAN=""; RESET=""
HERMES_ARGV=()

setup_colors() {
  case "$COLOR" in
    always) USE_COLOR=1 ;;
    never) USE_COLOR=0 ;;
    auto)
      if [ -t 1 ] && [ -z "${NO_COLOR:-}" ] && [ "${TERM:-}" != "dumb" ]; then USE_COLOR=1; fi
      ;;
  esac
  if [ "$USE_COLOR" = 1 ]; then
    BOLD=$'\033[1m'; DIM=$'\033[2m'; GREEN=$'\033[0;32m'
    YELLOW=$'\033[0;33m'; RED=$'\033[0;31m'; CYAN=$'\033[0;36m'; RESET=$'\033[0m'
  fi
}

# --- output ---------------------------------------------------------------------

heading() { printf '\n%s%s%s\n\n' "$BOLD" "$*" "$RESET"; }
step() { printf '  %s->%s  %s\n' "$CYAN" "$RESET" "$*"; }
ok() { printf '  %s✓%s  %s\n' "$GREEN" "$RESET" "$*"; }
warn() { printf '  %s!%s  %s\n' "$YELLOW" "$RESET" "$*"; }
err() { printf '  %s✗%s  %s\n' "$RED" "$RESET" "$*" >&2; }
hint() { printf '  %s%s%s\n' "$DIM" "$*" "$RESET"; }
say() { printf '%s\n' "$*"; }

die() {
  err "$*"
  exit 2
}

usage() {
  cat <<'EOF'
Usage: install.sh [COMMAND] [OPTIONS]

Installs ResearchForge for the current user: the rforge engine, the public agent
skills, and the native Claude Code, Codex, Pi and Hermes integrations. Nothing is
installed system-wide and no command needs sudo.

Commands:
  install             install the selected components (default)
  update              upgrade the components that are already installed

Profiles (what you want to use):
  --for cli           just the rforge engine
  --for skills        the public skills, delivered through each installed host
  --for mcp           the engine plus typed MCP tool registration in each host
  --for all           everything below (default)

Components:
  binary              rforge engine in ~/.local/bin
  skills              public agent skills in ~/.agents/skills
  claude              Claude Code plugin (native, user scope)
  codex               Codex plugin (native, user scope)
  pi                  Pi package (native, user scope)
  hermes              Hermes native plugin (your current profile)

Selection:
  --all               every component available on this machine (default)
  --only LIST         comma-separated components, e.g. binary,skills,hermes
  --skip LIST         comma-separated components to leave out
  --menu              interactive checkbox selection (needs a terminal)
  --list              show the components and whether each is available

Engine options:
  --version TAG       release tag to install (default: latest, or RFORGE_VERSION)
  --dir DIR           binary directory (default: ~/.local/bin, or RFORGE_INSTALL_DIR)
  --from-source       build with Go instead of downloading a release (inside
                      a checkout, builds that checkout; piped, uses go install)

Hermes options:
  --hermes-profile NAME
                      install into this profile instead of the current one;
                      repeat the flag or use commas for several profiles

Output:
  --dry-run           print every command, change nothing
  -v, --verbose       show exact commands as well as live output (default)
  -q, --quiet         show the summary and failures only
  --color auto|always|never
  -h, --help          this help

Environment: RFORGE_VERSION, RFORGE_INSTALL_DIR, RFORGE_INSTALL_FROM_SOURCE,
HERMES_PROFILE, NO_COLOR. Mirrors: RFORGE_RELEASE_BASE, RFORGE_RAW_BASE.
RFORGE_INSTALL_UI=text drives --menu from stdin (for scripts and tests).
EOF
}

# --- argument parsing -----------------------------------------------------------

require_value() {
  [ "$#" -ge 2 ] && [ -n "${2:-}" ] || die "$1 needs a value"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    install|update)
      COMMAND="$1"; shift ;;
    --all|--full)
      ONLY=""; shift ;;
    --only)
      require_value "$@"; ONLY="$2"; ONLY_GIVEN=1; shift 2 ;;
    --for)
      require_value "$@"; PROFILE="$2"; shift 2 ;;
    --skip)
      require_value "$@"; SKIP="$2"; shift 2 ;;
    --menu)
      MENU=1; shift ;;
    --list)
      SHOW_LIST=1; shift ;;
    --version)
      require_value "$@"; VERSION="$2"; shift 2 ;;
    --dir)
      require_value "$@"; INSTALL_DIR="$2"; shift 2 ;;
    --from-source)
      FROM_SOURCE=1; shift ;;
    --hermes-profile|--hermes-profiles)
      require_value "$@"
      if [ "$HERMES_PROFILES_GIVEN" = 1 ]; then HERMES_PROFILES="$HERMES_PROFILES,$2"; else HERMES_PROFILES="$2"; fi
      HERMES_PROFILES_GIVEN=1
      shift 2 ;;
    --dry-run)
      DRY_RUN=1; shift ;;
    -v|--verbose)
      VERBOSE=1; QUIET=0; shift ;;
    -q|--quiet)
      QUIET=1; VERBOSE=0; shift ;;
    --color)
      require_value "$@"; COLOR="$2"; shift 2 ;;
    -h|--help)
      usage; exit 0 ;;
    *)
      die "unknown option: $1 (see --help)" ;;
  esac
done

case "$COLOR" in
  auto|always|never) ;;
  *) die "--color must be auto, always or never" ;;
esac
setup_colors

[ -n "$VERSION" ] || die "--version needs a value"
if [ -z "$INSTALL_DIR" ]; then INSTALL_DIR="$HOME/.local/bin"; fi

validate_component_list() {
  local list="$1" flag="$2" key rest available="" candidate
  [ -n "$list" ] || return 0
  for candidate in $COMPONENT_KEYS; do
    if [ -n "$available" ]; then available="$available, $candidate"; else available="$candidate"; fi
  done
  rest=",$list,"
  while [ -n "$rest" ]; do
    rest="${rest#,}"
    [ -n "$rest" ] || break
    key="${rest%%,*}"
    rest="${rest#*,}"
    case " $COMPONENT_KEYS " in
      *" $key "*) ;;
      *) die "unknown component: $key (choose from: $available) for $flag" ;;
    esac
  done
}

if [ -n "$PROFILE" ]; then
  if [ "$ONLY_GIVEN" = 1 ]; then
    die "--for and --only select components in different ways; use one of them (--for still accepts --skip)"
  fi
  case "$PROFILE" in
    cli) ONLY="binary" ;;
    skills) ONLY="skills,claude,codex,pi,hermes" ;;
    mcp) ONLY="binary,claude,codex,pi,hermes" ;;
    all) ONLY="" ;;
    *) die "unknown profile: $PROFILE (choose from: cli, skills, mcp, all)" ;;
  esac
fi

validate_component_list "$ONLY" "--only"
validate_component_list "$SKIP" "--skip"

if [ "$MENU" = 1 ] && [ "${RFORGE_INSTALL_UI:-}" != "text" ] && { [ ! -t 0 ] || [ ! -t 1 ]; }; then
  die "no terminal for --menu; choose with --all, --only or --skip (see --list)"
fi

# --- command execution ----------------------------------------------------------

LOG_FILE=""
TMP_DIR=""
cleanup() {
  if [ -n "$LOG_FILE" ]; then rm -f "$LOG_FILE"; fi
  if [ -n "$TMP_DIR" ]; then rm -rf "$TMP_DIR"; fi
  return 0
}
trap cleanup EXIT

quote_args() {
  local out="" arg
  for arg in "$@"; do out="$out $(printf '%q' "$arg")"; done
  printf '%s' "${out# }"
}

# run executes a mutating command, or prints it under --dry-run. Failures return
# the command's status so the calling component can report and stop.
run() {
  if [ "$DRY_RUN" = 1 ] || [ "$VERBOSE" = 1 ]; then
    printf '  %s$ %s%s\n' "$DIM" "$(quote_args "$@")" "$RESET"
  fi
  [ "$DRY_RUN" = 1 ] && return 0
  if [ "$QUIET" = 1 ]; then
    if "$@" >>"$LOG_FILE" 2>&1; then
      return 0
    else
      local status=$?
    fi
    err "command failed: $(quote_args "$@")"
    tail -n 5 "$LOG_FILE" >&2 || true
    return "$status"
  fi
  "$@" || return $?
}

# run_in runs a mutating command inside dir, with the same dry-run and output
# handling as run.
run_in() {
  local dir="$1"; shift
  if [ "$DRY_RUN" = 1 ] || [ "$VERBOSE" = 1 ]; then
    printf '  %s$ (cd %s && %s)%s\n' "$DIM" "$dir" "$(quote_args "$@")" "$RESET"
  fi
  if [ "$DRY_RUN" = 1 ]; then return 0; fi
  if [ "$QUIET" = 1 ]; then
    if ( cd "$dir" && "$@" ) >>"$LOG_FILE" 2>&1; then
      return 0
    else
      local status=$?
    fi
    err "command failed: $(quote_args "$@")"
    tail -n 5 "$LOG_FILE" >&2 || true
    return "$status"
  fi
  ( cd "$dir" && "$@" ) || return $?
}

# ensure_dir creates an install directory, reporting a clear cause instead of
# leaking mkdir output when the path is (say) an existing file.
ensure_dir() {
  local dir="$1"
  [ "$DRY_RUN" = 1 ] && return 0
  if ! mkdir -p "$dir" 2>/dev/null; then
    detail "cannot create directory: $dir"
    return 1
  fi
  return 0
}

# capture runs a read-only probe (safe under --dry-run) and never fails the run.
capture() { "$@" 2>/dev/null || true; }

have() { command -v "$1" >/dev/null 2>&1; }

# --- component metadata ---------------------------------------------------------

component_label() {
  case "$1" in
    binary) say "rforge engine binary" ;;
    skills) say "public agent skills (generic Agent Skills)" ;;
    claude) say "Claude Code plugin" ;;
    codex) say "Codex plugin" ;;
    pi) say "Pi package" ;;
    hermes) say "Hermes native plugin" ;;
  esac
}

component_need() {
  case "$1" in
    claude|codex|pi|hermes) say "$1" ;;
    *) say "" ;;
  esac
}

component_availability() {
  local key="$1" need
  need=$(component_need "$key")
  if [ -n "$need" ] && ! have "$need"; then
    say "$need not found"
    return 0
  fi
  if [ "$COMMAND" = "update" ]; then
    if component_installed "$key"; then say "available"; else say "not installed"; fi
    return 0
  fi
  say "available"
}

component_installed() {
  case "$1" in
    binary) [ -x "$INSTALL_DIR/rforge" ] || have rforge ;;
    skills) [ -f "$HOME/.agents/skills/research-forge/SKILL.md" ] ;;
    claude) claude_installed ;;
    codex) codex_installed ;;
    pi) pi_installed ;;
    hermes) hermes_any_profile_installed ;;
    *) return 1 ;;
  esac
}

# --- detection ------------------------------------------------------------------

claude_installed() {
  have claude || return 1
  capture claude plugin list --json | grep -q 'research-forge'
}

codex_installed() {
  have codex || return 1
  capture codex plugin list --json | grep -q 'research-forge'
}

pi_installed() {
  have pi || return 1
  capture pi list | grep -q "$PI_SOURCE"
}

hermes_profile_names() {
  have hermes || return 1
  capture hermes profile list | awk '
    NR > 1 {
      name = $1
      gsub(/^◆/, "", name)
      if (name != "Profile" && name != "Model" && name ~ /^[A-Za-z0-9._-]+$/) print name
    }'
}

# hermes_argv fills HERMES_ARGV for the current profile (empty name) or a named
# one. Callers pass the resolved argv to run/capture so the printed command is
# the command that actually runs.
hermes_argv() {
  local profile="$1"; shift
  HERMES_ARGV=(hermes)
  if [ -n "$profile" ]; then
    HERMES_ARGV=(hermes --profile "$profile")
  fi
  HERMES_ARGV+=("$@")
}

# Unlike capture, native Hermes probes must retain their exit status and errors.
# Plain rows are status, source (which may contain spaces), version, exact name.
hermes_plugin_status() {
  local profile="$1" listing
  hermes_argv "$profile" plugins list --plain --no-bundled
  listing=$("${HERMES_ARGV[@]}") || return 1
  printf '%s\n' "$listing" | awk '
    NF >= 4 && $NF == "research-forge" {
      print ($1 == "enabled" ? "enabled" : "disabled")
      found = 1
      exit
    }
    END { if (!found) print "absent" }'
}

hermes_installed() {
  local status
  have hermes || return 1
  status=$(hermes_plugin_status "$1") || return 1
  [ "$status" != "absent" ]
}

hermes_any_profile_installed() {
  hermes_installed "" && return 0
  local profile
  for profile in $(hermes_profile_names); do
    hermes_installed "$profile" && return 0
  done
  return 1
}

selected_hermes_profiles() {
  if [ "$HERMES_PROFILES_GIVEN" = 1 ]; then
    printf '%s\n' "$HERMES_PROFILES" | tr ',' '\n'
  else
    printf '\n'
  fi
}

# --- platform and release helpers -----------------------------------------------

platform() {
  local os arch
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  arch=$(uname -m)
  case "$os" in
    linux) os=linux ;;
    darwin) os=darwin ;;
    msys*|mingw*|cygwin*) os=windows ;;
    *) die "unsupported OS: $os" ;;
  esac
  case "$arch" in
    x86_64|amd64) arch=amd64 ;;
    arm64|aarch64) arch=arm64 ;;
    *) die "unsupported architecture: $arch" ;;
  esac
  printf '%s-%s' "$os" "$arch"
}

asset_name() {
  local p
  p=$(platform)
  if [ "${p%%-*}" = windows ]; then printf 'rforge-%s.exe' "$p"; else printf 'rforge-%s' "$p"; fi
}

install_name() {
  case "$(platform)" in windows-*) printf 'rforge.exe' ;; *) printf 'rforge' ;; esac
}

release_url() {
  local asset
  asset=$(asset_name)
  if [ "$VERSION" = latest ]; then
    printf '%s/latest/download/%s' "$RELEASE_BASE" "$asset"
  else
    printf '%s/download/%s/%s' "$RELEASE_BASE" "$VERSION" "$asset"
  fi
}

checksums_url() {
  if [ "$VERSION" = latest ]; then
    printf '%s/latest/download/checksums.txt' "$RELEASE_BASE"
  else
    printf '%s/download/%s/checksums.txt' "$RELEASE_BASE" "$VERSION"
  fi
}

download() {
  local url="$1" dest="$2"
  if have curl; then
    curl -fsSL "$url" -o "$dest"
  elif have wget; then
    wget -q -O "$dest" "$url"
  else
    return 1
  fi
}

sha256_of() {
  local file="$1"
  if have sha256sum; then sha256sum "$file" | awk '{print $1}'
  elif have shasum; then shasum -a 256 "$file" | awk '{print $1}'
  elif have openssl; then openssl dgst -sha256 "$file" | awk '{print $NF}'
  else return 1
  fi
}

# --- results --------------------------------------------------------------------

RESULTS=""
DETAIL=""
COMPONENT_STATUS="installed"

detail() {
  if [ -n "$DETAIL" ]; then DETAIL="$DETAIL; $*"; else DETAIL="$*"; fi
}

record() { RESULTS="${RESULTS}$1"$'\t'"$2"$'\t'"$3"$'\n'; }

selected_explicitly() {
  [ "$ONLY_GIVEN" = 1 ] || return 1
  case ",$ONLY," in *",$1,"*) return 0 ;; esac
  return 1
}

selected_keys() {
  local key out=""
  for key in $COMPONENT_KEYS; do
    if [ -n "$ONLY" ]; then
      case ",$ONLY," in *",$key,"*) ;; *) continue ;; esac
    fi
    if [ -n "$SKIP" ]; then
      case ",$SKIP," in *",$key,"*) continue ;; esac
    fi
    out="$out $key"
  done
  printf '%s' "${out# }"
}

# set_aside moves a path to <name>.bak, or .bak.N when that is taken, so an
# earlier backup is never overwritten.
set_aside() {
  local path="$1" backup n
  backup="${path}.bak"; n=1
  while [ -e "$backup" ] || [ -L "$backup" ]; do
    backup="${path}.bak.${n}"; n=$((n + 1))
  done
  mv "$path" "$backup"
  printf '%s' "$backup"
}

# --- binary ---------------------------------------------------------------------

# build_checkout compiles this checkout into <dir>/<name>, stamping the same
# metadata the Makefile does so provenance records a real engine identity.
build_checkout() {
  local dir="$1" name="$2" source_version source_commit source_date source_ldflags exact_tag
  ensure_dir "$dir" || return 1
  source_version="dev"; source_commit="unknown"
  if have git; then
    # Only an exact tag is that release. A describe version such as
    # v0.1.21-162-gabc would fail the capability gate's version compare even
    # when the build has every capability, so anything else stays "dev".
    exact_tag=$(cd "$SELF_DIR" && git describe --tags --exact-match 2>/dev/null || true)
    if [ -n "$exact_tag" ]; then source_version="$exact_tag"; fi
    source_commit=$(cd "$SELF_DIR" && git rev-parse --short HEAD 2>/dev/null || printf 'unknown')
  fi
  source_date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
  source_ldflags="-X main.version=$source_version -X main.commit=$source_commit -X main.date=$source_date"
  run_in "$SELF_DIR" go build -ldflags "$source_ldflags" -o "$dir/$name" ./cmd/rforge
}

verify_checksum() {
  local file="$1" asset="$2" expected actual
  if ! download "$(checksums_url)" "$TMP_DIR/checksums.txt"; then
    warn "release checksums unavailable for $VERSION; skipping verification"
    return 0
  fi
  # Released checksums.list build paths ("dist/rforge-linux-amd64"), and some
  # sha256sum formats mark binary files with a leading "*". Match the basename.
  expected=$(awk -v asset="$asset" '
    {
      name = $2
      sub(/^\*/, "", name)
      sub(/^.*\//, "", name)
      if (name == asset) { print $1; exit }
    }' "$TMP_DIR/checksums.txt")
  if [ -z "$expected" ]; then
    warn "checksums.txt has no entry for $asset; skipping verification"
    return 0
  fi
  if ! actual=$(sha256_of "$file"); then
    warn "no sha256 tool available; skipping verification"
    return 0
  fi
  if [ "$actual" != "$expected" ]; then
    detail "checksum mismatch for $asset (want $expected, got $actual)"
    return 1
  fi
  ok "checksum verified"
}

verify_binary() {
  local bin="$1" caps="" cap
  [ "$DRY_RUN" = 1 ] && return 0
  if ! "$bin" version >/dev/null 2>&1; then
    detail "installed binary does not run: $bin version"
    return 1
  fi
  for cap in $REQUIRED_CAPABILITIES; do caps="$caps --require $cap"; done
  # shellcheck disable=SC2086
  if ! "$bin" capabilities --json --min-rforge-version "$MIN_RFORGE_VERSION" $caps >/dev/null 2>&1; then
    detail "installed engine $("$bin" version 2>/dev/null | head -n1) does not meet the required rforge $MIN_RFORGE_VERSION with these capabilities: $(printf '%s' "$REQUIRED_CAPABILITIES" | sed 's/=[0-9]*//g' | tr ' ' ',')"
    return 1
  fi
  ok "$("$bin" version 2>/dev/null | head -n1)"
}

path_check() {
  local dir="$1" found
  case ":$PATH:" in
    *":$dir:"*) ;;
    *)
      warn "$dir is not on your PATH. Add this to your shell profile:"
      hint "export PATH=\"\$PATH:$dir\""
      ;;
  esac
  found=$(command -v rforge 2>/dev/null || true)
  if [ -n "$found" ] && [ "$found" != "$dir/rforge" ]; then
    warn "another rforge appears earlier on PATH: $found"
  fi
}

install_binary() {
  local dir="$INSTALL_DIR" asset name url
  COMPONENT_STATUS="installed"
  asset=$(asset_name)
  name=$(install_name)

  if [ "$COMMAND" = "update" ] && ! component_installed binary; then
    COMPONENT_STATUS="skipped"; detail "not installed"; return 0
  fi

  if [ "$FROM_SOURCE" = 1 ]; then
    if ! have go; then
      detail "Go not found; install Go or drop --from-source to use release binaries"
      return 1
    fi
    if [ -n "$SELF_DIR" ] && [ -d "$SELF_DIR/cmd/rforge" ] && [ "$VERSION" = latest ]; then
      # A checkout builds the engine under development. That is also how a
      # capability gate ahead of the next release is satisfied.
      step "Building this checkout with Go into $dir"
      build_checkout "$dir" "$name" || return 1
    else
      step "Building rforge@${VERSION} with Go into $dir"
      ensure_dir "$dir" || return 1
      run env GOBIN="$dir" go install "github.com/${REPO_SLUG}/cmd/rforge@${VERSION}" || return 1
    fi
  else
    url=$(release_url)
    step "Installing release binary: $url"
    if [ "$DRY_RUN" = 1 ]; then
      run download "$url" "$dir/$name"
      run chmod +x "$dir/$name"
    else
      have curl || have wget || { detail "need curl or wget to download release binaries (or use --from-source)"; return 1; }
      TMP_DIR=$(mktemp -d "${TMPDIR:-/tmp}/rforge-install.XXXXXX")
      if ! download "$url" "$TMP_DIR/$asset"; then
        detail "release binary unavailable: $url"
        return 1
      fi
      verify_checksum "$TMP_DIR/$asset" "$asset" || return 1
      ensure_dir "$dir" || return 1
      if ! chmod +x "$TMP_DIR/$asset"; then
        detail "cannot mark $asset executable"
        return 1
      fi
      if ! mv "$TMP_DIR/$asset" "$dir/$name" 2>/dev/null; then
        detail "cannot install into $dir (not a writable directory)"
        return 1
      fi
      ok "Binary → $dir/$name"
    fi
  fi

  if ! verify_binary "$dir/$name"; then
    if [ -n "$SELF_DIR" ] && [ -d "$SELF_DIR/cmd/rforge" ] && have go; then
      # The published release can lag the capability gate the plugins require.
      # A buildable checkout is a real fix, not a workaround: say so and build.
      warn "release binary does not satisfy the capability gate; building this checkout instead"
      # The release failure no longer describes what will be installed.
      DETAIL=""
      build_checkout "$dir" "$name" || return 1
      if ! verify_binary "$dir/$name"; then
        return 1
      fi
    else
      warn "this checkout builds an engine that satisfies the gate: run 'make install'"
      return 1
    fi
  fi
  if [ "$DRY_RUN" = 0 ]; then
    detail "$("$dir/$name" version 2>/dev/null | head -n1) at $dir/$name"
  fi
  path_check "$dir"
  return 0
}

# --- skills ---------------------------------------------------------------------

fetch_ref() {
  if [ "$VERSION" = latest ]; then printf 'main'; else printf '%s' "$VERSION"; fi
}

skill_source_dir() {
  local name="$1"
  if [ -n "$SELF_DIR" ] && [ -f "$SELF_DIR/adapters/skills/$name/SKILL.md" ]; then
    printf '%s' "$SELF_DIR/adapters/skills/$name"
  fi
}

# install_one_skill <name> <target-root>; prints current/installed and returns 0.
install_one_skill() {
  local name="$1" root="$2" src dst
  src=$(skill_source_dir "$name")
  dst="$root/$name"

  if [ "$DRY_RUN" = 1 ]; then
    if [ -n "$src" ]; then
      run cp -R "$src/." "$dst/"
    else
      run download "$RAW_BASE/$(fetch_ref)/adapters/skills/$name/SKILL.md" "$dst/SKILL.md"
    fi
    return 0
  fi

  if [ -n "$src" ]; then
    if [ -d "$dst" ] && diff -r -q "$src" "$dst" >/dev/null 2>&1; then
      detail "current: $name"
      return 0
    fi
    if [ -e "$dst" ] || [ -L "$dst" ]; then
      local backup; backup=$(set_aside "$dst")
      detail "replaced $name (old copy kept at $(basename "$backup"))"
    fi
    mkdir -p "$dst"
    cp -R "$src/." "$dst/"
    return 0
  fi

  mkdir -p "$dst"
  local fetched="$dst/.SKILL.md.fetch"
  if ! download "$RAW_BASE/$(fetch_ref)/adapters/skills/$name/SKILL.md" "$fetched"; then
    rm -f "$fetched"
    detail "could not download the $name skill"
    return 1
  fi
  if [ -f "$dst/SKILL.md" ] && cmp -s "$fetched" "$dst/SKILL.md"; then
    rm -f "$fetched"
    detail "current: $name"
    return 0
  fi
  mv "$fetched" "$dst/SKILL.md"
  return 0
}

quarantine_skill() {
  local path="$1" backup
  if [ ! -e "$path" ] && [ ! -L "$path" ]; then return 0; fi
  if [ "$DRY_RUN" = 1 ]; then
    run mv "$path" "$path.bak"
    return 0
  fi
  backup=$(set_aside "$path")
  detail "moved legacy skill $(basename "$(dirname "$path")")/$(basename "$path") to $(basename "$backup")"
}

# quarantine_legacy_skills removes standalone copies that would shadow a native
# plugin in the same host. Only the public skills are touched.
quarantine_legacy_skills() {
  local root="$1" skill
  [ -d "$root" ] || return 0
  for skill in $PUBLIC_SKILLS; do
    quarantine_skill "$root/$skill"
  done
}

install_skills() {
  local root="$HOME/.agents/skills" skill failed=0
  COMPONENT_STATUS="installed"

  # Pi discovers ~/.agents/skills as well as its own package, so a generic copy
  # next to the Pi package registers every skill twice.
  if pi_installed; then
    COMPONENT_STATUS="skipped"
    detail "the Pi package already provides the public skills at user scope"
    return 0
  fi

  if [ "$COMMAND" = "update" ] && ! component_installed skills; then
    COMPONENT_STATUS="skipped"; detail "not installed"; return 0
  fi

  step "Installing public agent skills into $root"
  if [ "$DRY_RUN" = 0 ]; then mkdir -p "$root"; fi
  for skill in $PUBLIC_SKILLS; do
    install_one_skill "$skill" "$root" || failed=1
  done
  if [ "$failed" = 1 ]; then
    detail "one or more public skills could not be installed"
    return 1
  fi
  if [ "$DRY_RUN" = 0 ]; then
    detail "$(printf '%s' "$PUBLIC_SKILLS" | wc -w | tr -d ' ') public skills at $root"
  fi
  return 0
}

# --- Claude Code ----------------------------------------------------------------

install_claude() {
  COMPONENT_STATUS="installed"
  if [ "$COMMAND" = "update" ] && ! claude_installed; then
    COMPONENT_STATUS="skipped"; detail "not installed"; return 0
  fi
  step "Claude Code plugin"
  if capture claude plugin marketplace list | grep -q 'research-forge'; then
    run claude plugin marketplace update research-forge || return 1
  else
    run claude plugin marketplace add "$REPO_SLUG" || return 1
  fi
  if [ "$COMMAND" = "update" ]; then
    run claude plugin update research-forge@research-forge || return 1
  else
    run claude plugin install research-forge@research-forge || return 1
  fi
  if [ "$DRY_RUN" = 0 ] && ! claude_installed; then
    detail "claude does not list research-forge after install"
    return 1
  fi
  quarantine_legacy_skills "$HOME/.claude/skills"
  detail "research-forge@research-forge"
  return 0
}

# --- Codex ----------------------------------------------------------------------

install_codex() {
  COMPONENT_STATUS="installed"
  if [ "$COMMAND" = "update" ] && ! codex_installed; then
    COMPONENT_STATUS="skipped"; detail "not installed"; return 0
  fi
  step "Codex plugin"
  if capture codex plugin marketplace list | grep -q 'research-forge'; then
    run codex plugin marketplace upgrade research-forge || return 1
  else
    run codex plugin marketplace add "$REPO_SLUG" --ref main || return 1
  fi
  if [ "$COMMAND" = "update" ]; then
    # Codex caches a plugin by version, so drop the cached copy before re-adding.
    run codex plugin remove research-forge@research-forge || true
  fi
  run codex plugin add research-forge@research-forge || return 1
  if [ "$DRY_RUN" = 0 ] && ! codex_installed; then
    detail "codex does not list research-forge after install"
    return 1
  fi
  quarantine_legacy_skills "$HOME/.codex/skills"
  detail "research-forge@research-forge"
  return 0
}

# --- Pi -------------------------------------------------------------------------

install_pi() {
  COMPONENT_STATUS="installed"
  if [ "$COMMAND" = "update" ] && ! pi_installed; then
    COMPONENT_STATUS="skipped"; detail "not installed"; return 0
  fi
  step "Pi package"
  if [ "$COMMAND" = "update" ]; then
    run pi update "$PI_SOURCE" || return 1
  else
    run pi install "$PI_SOURCE" || return 1
  fi
  if [ "$DRY_RUN" = 0 ] && ! pi_installed; then
    detail "pi does not list $PI_SOURCE after install"
    return 1
  fi
  quarantine_legacy_skills "$HOME/.pi/agent/skills"
  detail "$PI_SOURCE"
  return 0
}

# --- Hermes ---------------------------------------------------------------------

# Preserve native failure status in quiet mode as well, without changing the
# other hosts' command handling.
hermes_run() {
  if [ "$QUIET" != 1 ] || [ "$DRY_RUN" = 1 ]; then
    run "${HERMES_ARGV[@]}"
    return $?
  fi
  if [ "$VERBOSE" = 1 ]; then
    printf '  %s$ %s%s\n' "$DIM" "$(quote_args "${HERMES_ARGV[@]}")" "$RESET"
  fi
  local status=0
  "${HERMES_ARGV[@]}" >>"$LOG_FILE" 2>&1 || status=$?
  [ "$status" = 0 ] && return 0
  err "command failed: $(quote_args "${HERMES_ARGV[@]}")"
  tail -n 5 "$LOG_FILE" >&2 || true
  return "$status"
}

validate_hermes_profiles() {
  local profile
  while IFS= read -r profile; do
    [ "$HERMES_PROFILES_GIVEN" = 0 ] && continue
    if ! hermes profile show "$profile" >/dev/null; then
      detail "could not validate Hermes profile: $profile (detected: $(hermes_profile_names | tr '\n' ' ' | sed 's/ $//'))"
      return 1
    fi
  done < <(selected_hermes_profiles)
  return 0
}

install_hermes() {
  local profile status any=0 i=0
  local states=()
  COMPONENT_STATUS="installed"
  step "Hermes native plugin"
  if ! validate_hermes_profiles; then return 1; fi

  # Probe every target before changing any of them. A failed list is not absence.
  while IFS= read -r profile; do
    if ! status=$(hermes_plugin_status "$profile"); then
      detail "could not list Hermes plugins in profile $(hermes_profile_label "$profile")"
      return 1
    fi
    states+=("$status")
  done < <(selected_hermes_profiles)

  while IFS= read -r profile; do
    status="${states[$i]}"
    i=$((i + 1))
    if [ "$COMMAND" = "update" ] && [ "$status" = "absent" ]; then
      detail "not installed in profile $(hermes_profile_label "$profile")"
      continue
    fi
    any=1
    if [ "$COMMAND" = "update" ]; then
      # Native update owns pin policy; never unpin or reinstall on refusal.
      hermes_argv "$profile" plugins update research-forge
      hermes_run || return 1
    elif [ "$status" = "enabled" ]; then
      detail "already enabled in profile $(hermes_profile_label "$profile"); keeping existing source/revision"
      continue
    else
      if [ "$status" = "absent" ]; then
        # Scan only the passive plugin subdirectory, without dependency installs.
        hermes_argv "$profile" plugins install "$PLUGIN_URL/tree/main/hermes-plugin" --no-deps
        hermes_run || return 1
      fi
      hermes_argv "$profile" plugins enable research-forge
      hermes_run || return 1
    fi
    if [ "$DRY_RUN" = 0 ]; then
      if ! status=$(hermes_plugin_status "$profile"); then
        detail "could not verify Hermes plugins in profile $(hermes_profile_label "$profile")"
        return 1
      fi
      if [ "$status" != "enabled" ]; then
        detail "research-forge is not enabled in profile $(hermes_profile_label "$profile")"
        return 1
      fi
    fi
  done < <(selected_hermes_profiles)

  if [ "$any" = 0 ]; then
    COMPONENT_STATUS="skipped"
    return 0
  fi
  detail "research-forge in profile(s): $(selected_hermes_profiles | while read -r p; do hermes_profile_label "$p"; done | tr '\n' ' ' | sed 's/ $//')"
  return 0
}

hermes_profile_label() {
  if [ -z "$1" ]; then printf 'current'; else printf '%s' "$1"; fi
}

# --- component runner -----------------------------------------------------------

run_component() {
  local key="$1" need
  DETAIL=""
  COMPONENT_STATUS="installed"
  need=$(component_need "$key")

  if [ -n "$need" ] && ! have "$need"; then
    if selected_explicitly "$key"; then
      record "$key" failed "$need not found"
      err "$key: $need not found"
      return 1
    fi
    record "$key" skipped "$need not found"
    return 0
  fi

  if "install_$key"; then
    if [ "$DRY_RUN" = 1 ]; then
      record "$key" planned "${DETAIL:-planned}"
    else
      record "$key" "$COMPONENT_STATUS" "$DETAIL"
    fi
    return 0
  fi
  record "$key" failed "${DETAIL:-failed}"
  err "$key: ${DETAIL:-failed}"
  return 1
}

# --- menu -----------------------------------------------------------------------

MENU_CHOSEN=""

menu_key_for_number() {
  local want="$1" i=0 key
  for key in $COMPONENT_KEYS; do
    i=$((i + 1))
    if [ "$i" = "$want" ]; then printf '%s' "$key"; return 0; fi
  done
  return 1
}

menu_choose() {
  local chosen="$1" answer part key i mark
  while :; do
    say ""
    say "ResearchForge installer — choose components:"
    i=0
    for key in $COMPONENT_KEYS; do
      i=$((i + 1))
      case " $chosen " in
        *" $key "*) mark="x" ;;
        *) mark=" " ;;
      esac
      printf '  [%s] %d %-8s %-42s %s\n' "$mark" "$i" "$key" "$(component_label "$key")" "$(component_availability "$key")"
    done
    printf 'Toggle numbers (e.g. 1 3), a = all, n = none, Enter = install, q = quit: '
    if [ "${RFORGE_INSTALL_UI:-}" = "text" ]; then
      if ! read -r answer; then return 1; fi
    else
      if ! read -r answer < /dev/tty; then return 1; fi
    fi
    answer=$(printf '%s' "$answer" | tr -d '[:space:]')
    case "$answer" in
      "") MENU_CHOSEN="$chosen"; return 0 ;;
      q|Q) return 1 ;;
      a|A) chosen="$COMPONENT_KEYS"; continue ;;
      n|N) chosen=""; continue ;;
    esac
    for part in $(printf '%s' "$answer" | tr ',' ' '); do
      case "$part" in
        ''|*[!0-9]*) continue ;;
      esac
      key=$(menu_key_for_number "$part") || continue
      # An unavailable component stays inert instead of failing the run later.
      if [ "$(component_availability "$key")" != "available" ]; then continue; fi
      case " $chosen " in
        *" $key "*) chosen=$(printf '%s' "$chosen" | sed "s/\(^\| \)$key\( \|\$\)/\1\2/g" | sed 's/  */ /g; s/^ //; s/ $//') ;;
        *) chosen="$chosen $key" ;;
      esac
    done
  done
}

# --- list -----------------------------------------------------------------------

print_list() {
  local key
  heading "ResearchForge components"
  for key in $COMPONENT_KEYS; do
    printf '  %-8s %-14s %s\n' "$key" "$(component_availability "$key")" "$(component_label "$key")"
  done
  if have hermes; then
    printf '\n  Hermes profiles: %s\n' "$(hermes_profile_names | tr '\n' ' ' | sed 's/ $//')"
  fi
  printf '\n  Profiles: cli=binary | skills=skills,claude,codex,pi,hermes | mcp=binary,claude,codex,pi,hermes | all=everything\n'
  printf '  Binary directory: %s\n' "$INSTALL_DIR"
  printf '  Public skills:    %s\n' "$PUBLIC_SKILLS"
}

# --- summary --------------------------------------------------------------------

print_summary() {
  local key status text installed=0 planned=0 skipped=0 failed=0
  heading "Summary"
  while IFS=$'\t' read -r key status text; do
    [ -n "$key" ] || continue
    printf '  %-8s %-9s %s\n' "$key" "$status" "$text"
    case "$status" in
      installed) installed=$((installed + 1)) ;;
      planned) planned=$((planned + 1)) ;;
      skipped) skipped=$((skipped + 1)) ;;
      failed) failed=$((failed + 1)) ;;
    esac
  done <<EOF
$RESULTS
EOF
  printf '\n'
  if [ "$DRY_RUN" = 1 ]; then
    say "$planned planned, $skipped skipped, $failed failed — nothing was changed"
  else
    say "$installed installed, $skipped skipped, $failed failed"
  fi
  [ "$failed" = 0 ] || return 1
  return 0
}

# --- main -----------------------------------------------------------------------

if [ "$SHOW_LIST" = 1 ]; then
  print_list
  exit 0
fi

heading "ResearchForge"
if [ "$DRY_RUN" = 1 ]; then
  say "Unified installer — engine, public skills and native agent integrations (dry-run: nothing is changed)."
else
  say "Unified installer — engine, public skills and native agent integrations."
fi
say "User scope only: nothing is installed system-wide and sudo is never needed."

KEYS=$(selected_keys)

if [ "$MENU" = 1 ]; then
  default_chosen=""
  for key in $KEYS; do
    if [ "$(component_availability "$key")" = "available" ]; then default_chosen="$default_chosen $key"; fi
  done
  default_chosen="${default_chosen# }"
  if ! menu_choose "$default_chosen"; then
    say "Nothing installed."
    exit 0
  fi
  KEYS="$MENU_CHOSEN"
fi

if [ -z "$KEYS" ]; then
  die "no components selected (see --list)"
fi

if [ "$QUIET" = 1 ] && [ "$DRY_RUN" = 0 ]; then
  LOG_FILE=$(mktemp "${TMPDIR:-/tmp}/rforge-install.XXXXXX")
fi

TOTAL=0
for key in $KEYS; do TOTAL=$((TOTAL + 1)); done
say ""
say "Selected: $(printf '%s' "$KEYS" | tr ' ' ',')"

INDEX=0
FAILED=0
for key in $KEYS; do
  INDEX=$((INDEX + 1))
  heading "[$INDEX/$TOTAL] $(component_label "$key")"
  if ! run_component "$key"; then FAILED=1; fi
done

if ! print_summary; then exit 1; fi
if [ "$FAILED" = 1 ]; then exit 1; fi

if [ "$DRY_RUN" = 0 ]; then
  case " $KEYS " in
    *" claude "*|*" codex "*|*" pi "*|*" hermes "*)
      say ""
      say "Open a new host session to discover the installed skills." ;;
  esac
fi

if [ "$DRY_RUN" = 0 ] && [ "$COMMAND" = "install" ]; then
  heading "Next steps"
  say "  Create a project"
  hint "rforge project create ./my-review --title \"My systematic review\""
  say "  Search and import papers"
  hint "rforge search batch --out ./research/my-topic --query \"...\" --sources openalex,arxiv"
  say "  Guided review package workflow"
  hint "rforge forge init --project ./my-review --question \"...\""
fi

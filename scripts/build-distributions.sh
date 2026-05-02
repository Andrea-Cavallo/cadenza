#!/usr/bin/env bash
set -euo pipefail

VERSION=""
WAILS_BIN="${WAILS:-}"
OUTPUT_DIR=""
DESKTOP_PLATFORMS="linux/amd64"
SKIP_DESKTOP=0
SKIP_CHECKS=0
CLEAN=0

while [[ $# -gt 0 ]]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    --wails) WAILS_BIN="$2"; shift 2 ;;
    --output-dir) OUTPUT_DIR="$2"; shift 2 ;;
    --desktop-platforms) DESKTOP_PLATFORMS="$2"; shift 2 ;;
    --skip-desktop) SKIP_DESKTOP=1; shift ;;
    --skip-checks) SKIP_CHECKS=1; shift ;;
    --clean) CLEAN=1; shift ;;
    -h|--help)
      cat <<'EOF'
Usage: scripts/build-distributions.sh [options]

Options:
  --version <version>              Release version label. Defaults to git describe.
  --wails <path>                   Path to Wails CLI. Defaults to $WAILS or PATH.
  --output-dir <dir>               Distribution directory. Defaults to ./dist.
  --desktop-platforms <csv>        Wails platforms to include. Default: linux/amd64.
  --skip-desktop                   Package CLI only.
  --skip-checks                    Skip go test ./...
  --clean                          Remove dist before packaging.
EOF
      exit 0
      ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "$script_dir/.." && pwd)"
backend_dir="$repo_root/backend"
desktop_dir="$backend_dir/cmd/desktop"
dist_dir="${OUTPUT_DIR:-$repo_root/dist}"
stage_root="$dist_dir/_stage"

step() {
  printf '\n==> %s\n' "$1"
}

has_platform() {
  local needle="$1"
  IFS=',' read -ra platforms <<< "$DESKTOP_PLATFORMS"
  for platform in "${platforms[@]}"; do
    [[ "${platform// /}" == "$needle" ]] && return 0
  done
  return 1
}

resolve_wails() {
  if [[ -n "$WAILS_BIN" ]]; then
    printf '%s' "$WAILS_BIN"
    return
  fi
  if command -v wails >/dev/null 2>&1; then
    command -v wails
    return
  fi
  local gopath
  gopath="$(go env GOPATH 2>/dev/null || true)"
  if [[ -n "$gopath" && -x "$gopath/bin/wails" ]]; then
    printf '%s' "$gopath/bin/wails"
  fi
}

safe_rm_dir() {
  local target="$1"
  local parent="$2"
  [[ -e "$target" ]] || return 0
  local resolved_target resolved_parent
  resolved_target="$(cd "$(dirname "$target")" && pwd)/$(basename "$target")"
  resolved_parent="$(cd "$parent" && pwd)"
  case "$resolved_target" in
    "$resolved_parent"/*) rm -rf "$resolved_target" ;;
    *) echo "Refusing to delete outside expected parent: $resolved_target" >&2; exit 1 ;;
  esac
}

build_cli() {
  local goos="$1" goarch="$2" outfile="$3"
  (
    cd "$backend_dir"
    GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
      go build -ldflags="-s -w" -o "$outfile" ./cmd/cadenza/
  )
}

try_build_desktop() {
  local platform="$1" stage_dir="$2" wails_path="$3"
  local bin_dir="$desktop_dir/build/bin"
  local desktop_stage="$stage_dir/desktop"

  if [[ -z "$wails_path" ]]; then
    echo "Desktop app not included: Wails CLI was not found."
    return 1
  fi

  safe_rm_dir "$bin_dir" "$desktop_dir"
  (
    cd "$desktop_dir"
    "$wails_path" build -platform "$platform"
  )

  if [[ ! -d "$bin_dir" ]]; then
    echo "Desktop app not included for $platform: Wails did not create build/bin."
    return 1
  fi

  mkdir -p "$desktop_stage"
  cp -R "$bin_dir"/. "$desktop_stage"/
  echo "Desktop app included from Wails platform $platform."
}

write_distribution_note() {
  local stage_dir="$1" name="$2" desktop_note="$3"
  cat > "$stage_dir/DISTRIBUTION.txt" <<EOF
Cadenza $name distribution

CLI:
- Run the cadenza binary from this folder.
- Offline example: cadenza --bpm 122 --key Am --no-llm

Desktop:
$desktop_note

Output:
- CLI default output is ./output unless another output path is selected.
- Desktop default output is ~/cadenza-output.
EOF
}

assert_package_contents() {
  local stage_dir="$1" package_name="$2" desktop_expected="$3"
  case "$package_name" in
    windows-*) [[ -f "$stage_dir/cadenza.exe" ]] || { echo "Missing cadenza.exe" >&2; exit 1; } ;;
    macos-*)
      [[ -f "$stage_dir/cadenza-darwin-amd64" ]] || { echo "Missing cadenza-darwin-amd64" >&2; exit 1; }
      [[ -f "$stage_dir/cadenza-darwin-arm64" ]] || { echo "Missing cadenza-darwin-arm64" >&2; exit 1; }
      ;;
    linux-*) [[ -f "$stage_dir/cadenza" ]] || { echo "Missing cadenza" >&2; exit 1; } ;;
  esac

  if [[ "$desktop_expected" == "1" ]]; then
    [[ -d "$stage_dir/desktop" ]] || { echo "Missing desktop folder for $package_name" >&2; exit 1; }
    if [[ "$package_name" == windows-* ]]; then
      [[ -f "$stage_dir/desktop/cadenza-desktop.exe" ]] || {
        echo "Windows package must contain desktop/cadenza-desktop.exe" >&2
        exit 1
      }
    fi
  fi
}

if [[ -z "$VERSION" ]]; then
  VERSION="$(git -C "$repo_root" describe --tags --always --dirty 2>/dev/null || true)"
  VERSION="${VERSION:-dev}"
fi

if [[ "$CLEAN" == "1" ]]; then
  safe_rm_dir "$dist_dir" "$repo_root"
fi
mkdir -p "$dist_dir"
safe_rm_dir "$stage_root" "$dist_dir"
mkdir -p "$stage_root"

step "Building Cadenza distributions ($VERSION)"
echo "Repo: $repo_root"
echo "Dist: $dist_dir"

if [[ "$SKIP_CHECKS" != "1" ]]; then
  step "Running Go tests"
  (cd "$backend_dir" && go test ./...)
fi

wails_path=""
if [[ "$SKIP_DESKTOP" != "1" ]]; then
  wails_path="$(resolve_wails || true)"
  [[ -n "$wails_path" ]] && echo "Wails: $wails_path" || echo "Wails CLI not found. Desktop binaries will be skipped."
fi

packages=(
  "windows-amd64|windows|cadenza-$VERSION-windows-amd64.zip|windows/amd64|windows:amd64:cadenza.exe"
  "macos-universal|macos|cadenza-$VERSION-macos-universal.zip|darwin/universal|darwin:amd64:cadenza-darwin-amd64,darwin:arm64:cadenza-darwin-arm64"
  "linux-amd64|linux|cadenza-$VERSION-linux-amd64.zip|linux/amd64|linux:amd64:cadenza"
)

for pkg in "${packages[@]}"; do
  IFS='|' read -r name stage zip_name desktop_platform cli_specs <<< "$pkg"
  step "Packaging $name"
  stage_dir="$stage_root/$stage"
  mkdir -p "$stage_dir"

  IFS=',' read -ra specs <<< "$cli_specs"
  for spec in "${specs[@]}"; do
    IFS=':' read -r goos goarch outfile <<< "$spec"
    echo "CLI $goos/$goarch -> $stage_dir/$outfile"
    build_cli "$goos" "$goarch" "$stage_dir/$outfile"
  done

  desktop_note="Desktop app not requested."
  desktop_expected=0
  if [[ "$SKIP_DESKTOP" != "1" ]]; then
    if has_platform "$desktop_platform"; then
      desktop_expected=1
      echo "Desktop $desktop_platform"
      desktop_note="$(try_build_desktop "$desktop_platform" "$stage_dir" "$wails_path")"
      echo "$desktop_note"
    else
      desktop_note="Desktop app skipped by --desktop-platforms."
    fi
  fi

  cp "$repo_root/README.md" "$stage_dir/"
  cp "$repo_root/LICENSE" "$stage_dir/"
  write_distribution_note "$stage_dir" "$name" "$desktop_note"
  assert_package_contents "$stage_dir" "$name" "$desktop_expected"

  zip_path="$dist_dir/$zip_name"
  rm -f "$zip_path"
  (
    cd "$stage_dir"
    zip -qr "$zip_path" .
  )
  echo "Created $zip_path"
done

step "Done"
find "$dist_dir" -maxdepth 1 -name '*.zip' -printf '%f %s bytes\n'

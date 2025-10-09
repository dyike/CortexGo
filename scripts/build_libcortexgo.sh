#!/usr/bin/env bash
set -euo pipefail

# Build the libcortexgo C shared library for macOS.
# Produces build/libcortexgo.dylib and build/libcortexgo.h.

ARCHS_DEFAULT=("arm64" "amd64")
DEPLOYMENT_TARGET_DEFAULT="11.0"

print_usage() {
  cat <<'USAGE'
Usage: scripts/build_libcortexgo.sh [--arch ARCH ...] [--deployment-target VERSION]

Options:
  --arch                Target GOARCH to build (may be repeated). Default: arm64 amd64
  --deployment-target   Minimum macOS version for the produced binary. Default: 11.0
  -h, --help            Show this message.

Environment variables:
  GO         Override the Go command (default: go)
  OUT_DIR    Override output directory (default: <repo>/build)
  PKG_PATH   Override the Go package to build (default: ./cmd/libcortexgo)
  GOMODCACHE Override the module cache (default: <repo>/.cache/gomod)
  GOCACHE    Override Go build cache (default: <repo>/.cache/gocache)
USAGE
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command '$1' is not available" >&2
    exit 1
  fi
}

# Parse arguments
ARCHS=()
DEPLOYMENT_TARGET="${DEPLOYMENT_TARGET:-${DEPLOYMENT_TARGET_DEFAULT}}"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --arch)
      shift || { echo "error: --arch expects a value" >&2; exit 1; }
      ARCHS+=("$1")
      ;;
    --deployment-target)
      shift || { echo "error: --deployment-target expects a value" >&2; exit 1; }
      DEPLOYMENT_TARGET="$1"
      ;;
    -h|--help)
      print_usage
      exit 0
      ;;
    *)
      echo "error: unknown argument '$1'" >&2
      print_usage >&2
      exit 1
      ;;
  esac
  shift
done

if [[ ${#ARCHS[@]} -eq 0 ]]; then
  ARCHS=("${ARCHS_DEFAULT[@]}")
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "error: this script must be run on macOS" >&2
  exit 1
fi

require_command go
require_command xcrun

GO_CMD="${GO:-go}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-${REPO_ROOT}/build}"
PKG_PATH="${PKG_PATH:-./cmd/libcortexgo}"
GOMODCACHE="${GOMODCACHE:-${REPO_ROOT}/.cache/gomod}"
GOCACHE="${GOCACHE:-${REPO_ROOT}/.cache/gocache}"

mkdir -p "${OUT_DIR}" "${GOMODCACHE}" "${GOCACHE}"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

CLANG_BIN="$(xcrun --sdk macosx --find clang)"
CLANGXX_BIN="$(xcrun --sdk macosx --find clang++)"
LIPO_BIN="$(xcrun --sdk macosx --find lipo)"
SDK_PATH="$(xcrun --sdk macosx --show-sdk-path)"

map_arch_to_clang() {
  case "$1" in
    amd64) echo "x86_64" ;;
    arm64) echo "arm64" ;;
    *)
      echo "error: unsupported architecture '$1'" >&2
      exit 1
      ;;
  esac
}

check_go_version() {
  local goversion golabel major minor patch
  goversion="$(${GO_CMD} env GOVERSION 2>/dev/null || true)"
  if [[ -z "${goversion}" ]]; then
    return
  fi
  golabel="${goversion#go}"
  major="${golabel%%.*}"
  local rest="${golabel#${major}.}";
  minor="${rest%%.*}"
  rest="${rest#${minor}.}"
  patch="${rest%%.*}"
  if [[ -z "${minor}" ]]; then
    minor=0
  fi
  if [[ "${major}" =~ ^[0-9]+$ && "${minor}" =~ ^[0-9]+$ ]]; then
    if (( major > 1 || (major == 1 && minor >= 25) )); then
      if [[ "${ALLOW_GO125:-}" == "1" ]]; then
        cat <<'WARN' >&2
warning: Detected Go 1.25+. Proceeding because ALLOW_GO125=1, but build is expected to fail
         until dependencies such as github.com/bytedance/sonic publish Go 1.25 support.
WARN
      else
        cat <<'ERR' >&2
error: Detected Go 1.25+. github.com/bytedance/sonic (an indirect dependency) has not
       published Go 1.25-compatible runtime bindings yet, so the CGO build cannot succeed.

       Install Go 1.24 and rerun the script, for example:
         brew install go@1.24
         GO=$(brew --prefix go@1.24)/bin/go ./scripts/build_libcortexgo.sh

       To force the script to continue anyway (not recommended), set ALLOW_GO125=1.
ERR
        exit 1
      fi
    fi
  fi
}

check_go_version

BUILD_OUTPUTS=()
HEADER_PATH=""

for arch in "${ARCHS[@]}"; do
  clang_arch="$(map_arch_to_clang "$arch")"
  arch_out="${WORK_DIR}/libcortexgo_${arch}.dylib"

  echo "[libcortexgo] Building for GOARCH=${arch} (clang -arch ${clang_arch})"
  env \
    CGO_ENABLED=1 \
    GOOS=darwin \
    GOARCH="${arch}" \
    CC="${CLANG_BIN}" \
    CXX="${CLANGXX_BIN}" \
    CGO_CFLAGS="-arch ${clang_arch} -mmacosx-version-min=${DEPLOYMENT_TARGET} -isysroot ${SDK_PATH}" \
    CGO_LDFLAGS="-arch ${clang_arch} -mmacosx-version-min=${DEPLOYMENT_TARGET} -isysroot ${SDK_PATH}" \
    SDKROOT="${SDK_PATH}" \
    MACOSX_DEPLOYMENT_TARGET="${DEPLOYMENT_TARGET}" \
    "${GO_CMD}" build \
      -trimpath \
      -buildmode=c-shared \
      -o "${arch_out}" \
      "${PKG_PATH}"

  arch_header="${arch_out%.dylib}.h"
  if [[ ! -f "${arch_header}" ]]; then
    echo "error: expected header file '${arch_header}' was not generated" >&2
    exit 1
  fi

  BUILD_OUTPUTS+=("${arch_out}")
  HEADER_PATH="${arch_header}"

  echo "[libcortexgo] ✔ Built ${arch_out##${WORK_DIR}/}"
done

FINAL_DYLIB="${OUT_DIR}/libcortexgo.dylib"
FINAL_HEADER="${OUT_DIR}/libcortexgo.h"

if [[ ${#BUILD_OUTPUTS[@]} -gt 1 ]]; then
  echo "[libcortexgo] Creating universal binary at ${FINAL_DYLIB}"
  "${LIPO_BIN}" -create -output "${FINAL_DYLIB}" "${BUILD_OUTPUTS[@]}"
else
  cp "${BUILD_OUTPUTS[0]}" "${FINAL_DYLIB}"
fi

cp "${HEADER_PATH}" "${FINAL_HEADER}"

cat <<EOF_SUMMARY

Build complete.
  Library : ${FINAL_DYLIB}
  Header  : ${FINAL_HEADER}
  Arch(s) : ${ARCHS[*]}
EOF_SUMMARY

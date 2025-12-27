#!/bin/bash
set -e

# Configuration
LIB_NAME="libcortex"
MAIN_PKG="./cmd/libcortex/..."
OUTPUT_DIR="build"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

print_help() {
    echo "Usage: $0 [target]"
    echo ""
    echo "Targets:"
    echo "  darwin-amd64      Build macOS Intel dylib"
    echo "  darwin-arm64      Build macOS Apple Silicon dylib"
    echo "  darwin            Build macOS Universal dylib (default on macOS)"
    echo "  linux-amd64       Build Linux x86_64 .so"
    echo "  linux-arm64       Build Linux arm64 .so"
    echo "  linux             Build Linux amd64 + arm64 .so"
    echo "  all               Build all platforms"
    echo "  clean             Remove build directory"
    echo ""
    echo "Examples:"
    echo "  $0                # Build for current platform"
    echo "  $0 linux-amd64    # Build Linux amd64 only"
    echo "  $0 all            # Build all platforms"
}

clean() {
    rm -rf "$OUTPUT_DIR"
    echo -e "${GREEN}✅ Cleaned${NC}"
}

build_darwin_amd64() {
    echo -e "${YELLOW}🛠  Building macOS amd64...${NC}"
    mkdir -p "$OUTPUT_DIR/darwin-amd64"
    CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
        go build -buildmode=c-shared -trimpath -ldflags="-s -w" \
        -o "$OUTPUT_DIR/darwin-amd64/$LIB_NAME.dylib" $MAIN_PKG
    echo -e "${GREEN}✅ Built: $OUTPUT_DIR/darwin-amd64/$LIB_NAME.dylib${NC}"
}

build_darwin_arm64() {
    echo -e "${YELLOW}🛠  Building macOS arm64...${NC}"
    mkdir -p "$OUTPUT_DIR/darwin-arm64"
    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
        go build -buildmode=c-shared -trimpath -ldflags="-s -w" \
        -o "$OUTPUT_DIR/darwin-arm64/$LIB_NAME.dylib" $MAIN_PKG
    echo -e "${GREEN}✅ Built: $OUTPUT_DIR/darwin-arm64/$LIB_NAME.dylib${NC}"
}

build_darwin_universal() {
    build_darwin_amd64
    build_darwin_arm64

    echo -e "${YELLOW}🔗 Creating Universal Binary...${NC}"
    mkdir -p "$OUTPUT_DIR/darwin-universal"
    lipo -create \
        "$OUTPUT_DIR/darwin-amd64/$LIB_NAME.dylib" \
        "$OUTPUT_DIR/darwin-arm64/$LIB_NAME.dylib" \
        -output "$OUTPUT_DIR/darwin-universal/$LIB_NAME.dylib"
    cp "$OUTPUT_DIR/darwin-arm64/$LIB_NAME.h" "$OUTPUT_DIR/darwin-universal/$LIB_NAME.h"
    echo -e "${GREEN}✅ Built: $OUTPUT_DIR/darwin-universal/$LIB_NAME.dylib${NC}"
    file "$OUTPUT_DIR/darwin-universal/$LIB_NAME.dylib"
}

# Find cross-compiler for Linux builds from macOS
find_linux_cc() {
    local arch=$1
    if command -v zig &> /dev/null; then
        if [ "$arch" = "amd64" ]; then
            echo "zig cc -target x86_64-linux-gnu"
        else
            echo "zig cc -target aarch64-linux-gnu"
        fi
    elif [ "$arch" = "amd64" ] && command -v x86_64-linux-musl-gcc &> /dev/null; then
        echo "x86_64-linux-musl-gcc"
    elif [ "$arch" = "arm64" ] && command -v aarch64-linux-musl-gcc &> /dev/null; then
        echo "aarch64-linux-musl-gcc"
    else
        echo ""
    fi
}

build_linux_amd64() {
    echo -e "${YELLOW}🛠  Building Linux amd64...${NC}"
    mkdir -p "$OUTPUT_DIR/linux-amd64"

    local CC=""
    local current_os=$(uname -s)

    if [ "$current_os" = "Linux" ]; then
        # Native Linux build
        CC="gcc"
    else
        # Cross-compile from macOS
        CC=$(find_linux_cc "amd64")
        if [ -z "$CC" ]; then
            echo -e "${RED}❌ Error: No cross-compiler found for Linux amd64${NC}"
            echo "Install zig: brew install zig"
            echo "Or musl-cross: brew install filosottile/musl-cross/musl-cross"
            exit 1
        fi
    fi

    CGO_ENABLED=1 GOOS=linux GOARCH=amd64 CC="$CC" \
        go build -buildmode=c-shared -trimpath -ldflags="-s -w" \
        -o "$OUTPUT_DIR/linux-amd64/$LIB_NAME.so" $MAIN_PKG
    echo -e "${GREEN}✅ Built: $OUTPUT_DIR/linux-amd64/$LIB_NAME.so${NC}"
}

build_linux_arm64() {
    echo -e "${YELLOW}🛠  Building Linux arm64...${NC}"
    mkdir -p "$OUTPUT_DIR/linux-arm64"

    local CC=""
    local current_os=$(uname -s)
    local current_arch=$(uname -m)

    if [ "$current_os" = "Linux" ] && [ "$current_arch" = "aarch64" ]; then
        # Native Linux arm64 build
        CC="gcc"
    else
        # Cross-compile
        CC=$(find_linux_cc "arm64")
        if [ -z "$CC" ]; then
            echo -e "${RED}❌ Error: No cross-compiler found for Linux arm64${NC}"
            echo "Install zig: brew install zig"
            exit 1
        fi
    fi

    CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC="$CC" \
        go build -buildmode=c-shared -trimpath -ldflags="-s -w" \
        -o "$OUTPUT_DIR/linux-arm64/$LIB_NAME.so" $MAIN_PKG
    echo -e "${GREEN}✅ Built: $OUTPUT_DIR/linux-arm64/$LIB_NAME.so${NC}"
}

build_linux() {
    build_linux_amd64
    build_linux_arm64
}

build_current_platform() {
    local os=$(uname -s)
    local arch=$(uname -m)

    case "$os" in
        Darwin)
            build_darwin_universal
            ;;
        Linux)
            if [ "$arch" = "x86_64" ]; then
                build_linux_amd64
            elif [ "$arch" = "aarch64" ]; then
                build_linux_arm64
            else
                echo -e "${RED}❌ Unsupported architecture: $arch${NC}"
                exit 1
            fi
            ;;
        *)
            echo -e "${RED}❌ Unsupported OS: $os${NC}"
            exit 1
            ;;
    esac
}

build_all() {
    build_darwin_universal
    build_linux
    echo ""
    echo -e "${GREEN}✅ All builds complete!${NC}"
    echo "📂 Output files:"
    find "$OUTPUT_DIR" -name "*.dylib" -o -name "*.so" | while read f; do
        echo "   $f"
    done
}

# Main
cd "$(dirname "$0")/.."

case "${1:-}" in
    darwin-amd64)
        build_darwin_amd64
        ;;
    darwin-arm64)
        build_darwin_arm64
        ;;
    darwin)
        build_darwin_universal
        ;;
    linux-amd64)
        build_linux_amd64
        ;;
    linux-arm64)
        build_linux_arm64
        ;;
    linux)
        build_linux
        ;;
    all)
        build_all
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        print_help
        ;;
    "")
        build_current_platform
        ;;
    *)
        echo -e "${RED}Unknown target: $1${NC}"
        print_help
        exit 1
        ;;
esac

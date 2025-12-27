# CortexGo Build Makefile

LIB_NAME := libcortex
MAIN_PKG := ./cmd/libcortex/...
OUTPUT_DIR := build
DEMO_PKG := ./cmd/demo

# Go build flags
GO_BUILD_FLAGS := -trimpath -ldflags="-s -w"

.PHONY: all clean demo lib-darwin lib-linux lib-linux-amd64 lib-linux-arm64 lib-darwin-amd64 lib-darwin-arm64 lib-darwin-universal help

all: demo lib-darwin-universal

help:
	@echo "CortexGo Build Targets:"
	@echo ""
	@echo "  make demo                  - Build demo CLI for current platform"
	@echo "  make lib-darwin            - Build macOS universal dylib (amd64 + arm64)"
	@echo "  make lib-darwin-amd64      - Build macOS amd64 dylib"
	@echo "  make lib-darwin-arm64      - Build macOS arm64 dylib"
	@echo "  make lib-linux-amd64       - Build Linux amd64 .so (requires cross-compiler)"
	@echo "  make lib-linux-arm64       - Build Linux arm64 .so (requires cross-compiler)"
	@echo "  make lib-linux             - Build Linux amd64 + arm64 .so"
	@echo "  make clean                 - Remove build artifacts"
	@echo ""
	@echo "Cross-compilation for Linux requires:"
	@echo "  - zig (recommended): brew install zig"
	@echo "  - or musl-cross: brew install filosottile/musl-cross/musl-cross"

# Demo CLI
demo:
	@echo "Building demo CLI..."
	go build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/cortexgo $(DEMO_PKG)
	@echo "✅ Demo built: $(OUTPUT_DIR)/cortexgo"

# Clean
clean:
	rm -rf $(OUTPUT_DIR)
	@echo "✅ Cleaned build directory"

# macOS builds
lib-darwin-amd64:
	@mkdir -p $(OUTPUT_DIR)/darwin-amd64
	@echo "Building macOS amd64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/darwin-amd64/$(LIB_NAME).dylib $(MAIN_PKG)
	@echo "✅ Built: $(OUTPUT_DIR)/darwin-amd64/$(LIB_NAME).dylib"

lib-darwin-arm64:
	@mkdir -p $(OUTPUT_DIR)/darwin-arm64
	@echo "Building macOS arm64..."
	CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/darwin-arm64/$(LIB_NAME).dylib $(MAIN_PKG)
	@echo "✅ Built: $(OUTPUT_DIR)/darwin-arm64/$(LIB_NAME).dylib"

lib-darwin-universal: lib-darwin-amd64 lib-darwin-arm64
	@echo "Creating macOS Universal Binary..."
	@mkdir -p $(OUTPUT_DIR)/darwin-universal
	lipo -create \
		$(OUTPUT_DIR)/darwin-amd64/$(LIB_NAME).dylib \
		$(OUTPUT_DIR)/darwin-arm64/$(LIB_NAME).dylib \
		-output $(OUTPUT_DIR)/darwin-universal/$(LIB_NAME).dylib
	cp $(OUTPUT_DIR)/darwin-arm64/$(LIB_NAME).h $(OUTPUT_DIR)/darwin-universal/$(LIB_NAME).h
	@echo "✅ Built: $(OUTPUT_DIR)/darwin-universal/$(LIB_NAME).dylib"

lib-darwin: lib-darwin-universal

# Linux builds (using zig as cross-compiler)
lib-linux-amd64:
	@mkdir -p $(OUTPUT_DIR)/linux-amd64
	@echo "Building Linux amd64..."
	@if command -v zig >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC="zig cc -target x86_64-linux-gnu" \
		CXX="zig c++ -target x86_64-linux-gnu" \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/linux-amd64/$(LIB_NAME).so $(MAIN_PKG); \
	elif command -v x86_64-linux-musl-gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=linux GOARCH=amd64 \
		CC=x86_64-linux-musl-gcc \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/linux-amd64/$(LIB_NAME).so $(MAIN_PKG); \
	else \
		echo "❌ Error: No cross-compiler found. Install zig or musl-cross"; \
		echo "   brew install zig"; \
		exit 1; \
	fi
	@echo "✅ Built: $(OUTPUT_DIR)/linux-amd64/$(LIB_NAME).so"

lib-linux-arm64:
	@mkdir -p $(OUTPUT_DIR)/linux-arm64
	@echo "Building Linux arm64..."
	@if command -v zig >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC="zig cc -target aarch64-linux-gnu" \
		CXX="zig c++ -target aarch64-linux-gnu" \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/linux-arm64/$(LIB_NAME).so $(MAIN_PKG); \
	elif command -v aarch64-linux-musl-gcc >/dev/null 2>&1; then \
		CGO_ENABLED=1 GOOS=linux GOARCH=arm64 \
		CC=aarch64-linux-musl-gcc \
		go build -buildmode=c-shared $(GO_BUILD_FLAGS) \
		-o $(OUTPUT_DIR)/linux-arm64/$(LIB_NAME).so $(MAIN_PKG); \
	else \
		echo "❌ Error: No cross-compiler found. Install zig or musl-cross"; \
		echo "   brew install zig"; \
		exit 1; \
	fi
	@echo "✅ Built: $(OUTPUT_DIR)/linux-arm64/$(LIB_NAME).so"

lib-linux: lib-linux-amd64 lib-linux-arm64
	@echo "✅ Linux builds complete"

# Build all platforms
lib-all: lib-darwin-universal lib-linux
	@echo "✅ All platform builds complete"

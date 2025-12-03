#!/bin/bash

# --- 配置部分 ---
LIB_NAME="libcortex"       # 你的库名称
MAIN_FILE="./cmd/libcortex/..."      # 入口文件路径
OUTPUT_DIR="build"   # 输出目录
# ----------------

# 出错立即停止
set -e

echo "🚀 开始编译 macOS Universal Binary..."

# 1. 清理旧文件
rm -rf $OUTPUT_DIR
mkdir -p $OUTPUT_DIR/amd64
mkdir -p $OUTPUT_DIR/arm64

# 2. 编译 AMD64 (Intel) 架构
echo "🛠  编译 AMD64 (Intel)..."
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -buildmode=c-shared -o $OUTPUT_DIR/amd64/$LIB_NAME.dylib $MAIN_FILE

# 3. 编译 ARM64 (Apple Silicon) 架构
echo "🛠  编译 ARM64 (Apple Silicon)..."
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -buildmode=c-shared -o $OUTPUT_DIR/arm64/$LIB_NAME.dylib $MAIN_FILE

# 4. 使用 lipo 合并成通用二进制 (Universal Binary)
echo "🔗 合并架构 (Creating Universal Binary)..."
lipo -create \
    $OUTPUT_DIR/amd64/$LIB_NAME.dylib \
    $OUTPUT_DIR/arm64/$LIB_NAME.dylib \
    -output $OUTPUT_DIR/$LIB_NAME.dylib

# 5. 复制头文件 (两个架构的头文件是一样的，取其中一个即可)
cp $OUTPUT_DIR/arm64/$LIB_NAME.h $OUTPUT_DIR/$LIB_NAME.h

# 6. 清理临时文件夹 (可选)
# rm -rf $OUTPUT_DIR/amd64 $OUTPUT_DIR/arm64

echo "✅ 编译完成！"
echo "📂 输出文件位置:"
echo "   库文件: $OUTPUT_DIR/$LIB_NAME.dylib"
echo "   头文件: $OUTPUT_DIR/$LIB_NAME.h"

# 7. 检查架构信息
echo "ℹ️  文件架构信息:"
file $OUTPUT_DIR/$LIB_NAME.dylib

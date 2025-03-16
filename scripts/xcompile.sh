#!/bin/bash

# 设置编译参数
VERSION=${VERSION:-$(git describe --tags --always --dirty)}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_SHA=$(git rev-parse --short HEAD)

# 编译参数
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitSHA=${COMMIT_SHA}"
#GOX_OS="linux darwin windows"
GOX_OS="linux "
GOX_ARCH="amd64 arm64"

# 确保输出目录存在
mkdir -p build

# 使用 gox 进行跨平台编译
gox \
    -os="${GOX_OS}" \
    -arch="${GOX_ARCH}" \
    -ldflags="${LDFLAGS}" \
    -output="build/k8s-ai-agent_{{.OS}}_{{.Arch}}" \
    ./cmd/kube-copilot

# 重命名 Windows 可执行文件
for file in build/k8s-ai-agent_windows_*; do
    if [ -f "$file" ]; then
        mv "$file" "${file}.exe"
    fi
done

# 打印编译信息
echo "Build completed:"
ls -lh build/ 
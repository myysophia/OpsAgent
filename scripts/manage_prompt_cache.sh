#!/bin/bash

# 设置错误时退出
set -e

# 导入环境变量
source .env

# 检查缓存目录是否存在
CACHE_DIR="./cache/prompts"
if [ ! -d "$CACHE_DIR" ]; then
    mkdir -p "$CACHE_DIR"
fi

# 更新缓存
echo "正在更新本地 prompt 缓存..."
for prompt_file in prompts/*; do
    if [ -f "$prompt_file" ]; then
        filename=$(basename "$prompt_file")
        cp "$prompt_file" "$CACHE_DIR/$filename"
        echo "已缓存: $filename"
    fi
done

echo "Prompt 缓存更新完成！" 
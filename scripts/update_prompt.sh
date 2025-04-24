#!/bin/bash

# 设置错误时退出
set -e

# 导入环境变量
source .env

# 上传 prompt 到 OSS
echo "正在上传 prompt 到 OSS..."
./scripts/upload_prompt.sh

# 更新本地缓存
echo "正在更新本地缓存..."
./scripts/manage_prompt_cache.sh

echo "Prompt 更新完成！" 
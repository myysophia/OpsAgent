#!/bin/bash

# 设置错误时退出
set -e

# 导入环境变量
source .env

# 检查必要的环境变量
if [ -z "$OSS_ENDPOINT" ] || [ -z "$OSS_ACCESS_KEY_ID" ] || [ -z "$OSS_ACCESS_KEY_SECRET" ] || [ -z "$OSS_BUCKET_NAME" ]; then
    echo "错误：缺少必要的环境变量"
    echo "请确保在 .env 文件中设置了以下变量："
    echo "OSS_ENDPOINT"
    echo "OSS_ACCESS_KEY_ID"
    echo "OSS_ACCESS_KEY_SECRET"
    echo "OSS_BUCKET_NAME"
    exit 1
fi

# 创建 prompts 目录（如果不存在）
if [ ! -d "prompts" ]; then
    echo "创建 prompts 目录..."
    mkdir -p prompts
fi

# 从 OSS 下载 prompts
echo "正在从 OSS 下载 prompts..."
ossutil cp -r oss://${OSS_BUCKET_NAME}/prompts/ prompts/

echo "Prompt 下载完成！" 
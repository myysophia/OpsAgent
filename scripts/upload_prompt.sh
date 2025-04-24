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

# 检查 prompts 目录是否存在
if [ ! -d "prompts" ]; then
    echo "错误：prompts 目录不存在"
    exit 1
fi

# 上传 prompts 到 OSS
echo "正在上传 prompts 到 OSS..."
ossutil cp -r prompts/ oss://${OSS_BUCKET_NAME}/prompts/

echo "Prompt 上传完成！" 
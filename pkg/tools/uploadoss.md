# IoTDB Tools OSS上传功能使用说明

## 功能描述
iotdbtools工具提供了将Kubernetes Pod中的文件直接上传到阿里云OSS的功能。这个功能特别适用于需要备份或迁移Pod中数据的场景。

## 使用方式

### 基本命令格式
```bash
iotdbtools backup [options]
```

### 参数说明
- `--pods`: 指定要操作的Pod名称（必填）
- `--datadir`: 指定Pod中的数据目录路径（必填）
- `--containers`: 指定要操作的容器名称（必填）
- `--uploadoss`: 是否上传到OSS（可选，默认为false）
- `--bucketname`: OSS存储桶名称（当uploadoss为true时必填）
- `--keep-local`: 是否保留本地文件（可选，默认为true）
- `--verbose`: 日志详细程度（可选，默认为1）

### 使用示例

1. 备份Pod中的日志文件并上传到OSS：
```bash
iotdbtools backup \
  --pods vnnox-middle-oauth-f88d9c78c-247tf \
  --datadir /app/logs \
  --containers vnnox-middle-oauth \
  --uploadoss true \
  --bucketname iotdb-backup \
  --keep-local true \
  --verbose 2
```

2. 仅备份Pod中的文件到本地：
```bash
iotdbtools backup \
  --pods vnnox-middle-oauth-f88d9c78c-247tf \
  --datadir /app/logs \
  --containers vnnox-middle-oauth \
  --uploadoss false \
  --keep-local true
```

## 注意事项
1. 确保Pod中已经安装了iotdbtools工具
2. 确保有足够的权限访问OSS存储桶
3. 建议在非高峰期执行备份操作
4. 对于大文件，建议使用`--keep-local true`选项，以便在传输失败时可以重试

## 错误处理
如果遇到错误，工具会返回详细的错误信息，包括：
- 命令执行失败的原因
- 文件传输状态
- OSS上传结果

## 最佳实践
1. 在执行大规模备份前，建议先用小数据量测试
2. 定期检查OSS存储桶的容量和访问权限
3. 对于重要数据，建议启用`--keep-local true`选项
4. 使用`--verbose 2`可以获取更详细的执行日志 
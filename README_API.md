# OpsAgent API Documentation

OpsAgent 是一个基于 AI 的 Kubernetes 运维助手，该文档详细描述了 OpsAgent 所提供的 API 接口。

## Getting Started

### Authentication
- 使用 `/login` 接口获取 JWT token，随后在请求头中加入 `Authorization: Bearer <token>`。

### API Key
- 执行命令相关的接口需要在请求头中添加 `X-API-Key`。

## API Endpoints

### Authorization
- **`POST /login`**: 用户登录获取 JWT Token

### Core Functionality
- **`POST /api/execute`**: 使用 AI 智能执行 Kubernetes 相关命令
- **`POST /api/diagnose`**: 诊断 Kubernetes 资源，识别潜在问题
- **`POST /api/analyze`**: 分析 Kubernetes 资源，提供优化建议

### Performance
- **`GET /api/perf/stats`**: 获取性能统计信息
- **`POST /api/perf/reset`**: 重置性能统计信息

### Context Management
- **`POST /api/v1/switch-context`**: 切换 Kubernetes 上下文

### System Information
- **`GET /api/version`**: 获取 API 版本信息

## Usage
- 所有 API 调用均需要提供正确的身份验证信息和 API Key

- 示例请求：
  ```bash
  curl -X POST \
    'https://api.opsagent.com/api/execute' \
    -H 'Authorization: Bearer YOUR_JWT_TOKEN' \
    -H 'X-API-Key: YOUR_API_KEY' \
    -H 'Content-Type: application/json' \
    -d '{"instructions": "查看中国节点的 pod 状态","args": ""}'
  ```

## License
- 项目采用 MIT 开源许可证

详细的 OpenAPI 文档请参见 [openapi.yaml](./openapi.yaml) 和 [openapi.json](./openapi.json)。


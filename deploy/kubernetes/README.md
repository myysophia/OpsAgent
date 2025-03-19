# OpsAgent Kubernetes 部署指南

本文档提供了如何将 OpsAgent 部署到 Kubernetes 集群的说明。

## 部署文件

我们提供了三个不同环境的部署配置文件：

- `deployment.yaml` - 基础部署配置
- `deployment-dev.yaml` - 开发环境配置（资源需求较低，单副本）
- `deployment-prod.yaml` - 生产环境配置（高可用性，多副本，网络策略等）

## 前置条件

- Kubernetes 集群 (v1.19+)
- kubectl 命令行工具
- 安装了 Ingress Controller（推荐 nginx-ingress）
- 对于生产环境，建议安装 cert-manager 来自动管理 TLS 证书

## 快速部署

### 开发环境

1. 部署应用
```bash
kubectl apply -f deployment-dev.yaml
```

2. 验证部署
```bash
kubectl -n ops-agent-dev get pods
kubectl -n ops-agent-dev get svc
kubectl -n ops-agent-dev get ingress
```

3. 配置本地访问（可选）
```bash
# 添加hosts记录
echo "127.0.0.1 ops-agent-dev.example.com" | sudo tee -a /etc/hosts

# 使用端口转发快速访问服务
kubectl -n ops-agent-dev port-forward svc/ops-agent 8080:80
```

### 生产环境

1. 修改配置

部署前，请修改以下内容：
- 在 Secret 中设置安全的 JWT 密钥
- 在 `kubeconfig-secret` 中添加有效的 kubeconfig 配置
```bash
# 生成 kubeconfig Secret
cat ~/.kube/config | base64 -w 0
# 复制输出内容并替换 deployment-prod.yaml 中的 kubeconfig-secret 数据
```
- 在 Ingress 配置中设置正确的域名
- 在 Ingress 的 `nginx.ingress.kubernetes.io/whitelist-source-range` 注解中设置允许访问的 IP 地址
```bash
# 格式为 CIDR 表示法，多个地址用逗号分隔
# 例如: 10.0.0.1/32,192.168.1.0/24
```
- 根据实际需求调整资源限制

2. 部署应用
```bash
kubectl apply -f deployment-prod.yaml
```

3. 验证部署
```bash
kubectl -n ops-agent get pods
kubectl -n ops-agent get svc
kubectl -n ops-agent get ingress
kubectl -n ops-agent get secrets
kubectl -n ops-agent get configmaps
```

## 常见问题排查

### Pod无法启动

检查Pod状态和日志：
```bash
kubectl -n ops-agent describe pod <pod-name>
kubectl -n ops-agent logs <pod-name>
```

### 日志目录权限问题

如果遇到类似 `"error": "创建日志目录失败: mkdir logs: permission denied"` 的错误，请检查：

1. 确保部署配置中有 initContainer 来设置日志目录权限：
```bash
kubectl -n ops-agent get pod <pod-name> -o yaml | grep -A 10 initContainers
```

2. 查看日志目录权限：
```bash
kubectl -n ops-agent exec <pod-name> -- ls -la /app/logs
```

3. 手动修复权限（紧急情况下使用）：
```bash
kubectl -n ops-agent exec <pod-name> -- mkdir -p /app/logs
kubectl -n ops-agent exec <pod-name> -- chmod 755 /app/logs
```

### 无法访问服务

检查Ingress配置和服务状态：
```bash
kubectl -n ops-agent get ingress
kubectl -n ops-agent describe ingress ops-agent
kubectl -n ops-agent get svc
kubectl -n ops-agent get endpoints ops-agent
```

### IP访问限制问题

如果您无法访问应用，请检查您的IP是否在白名单中：
```bash
kubectl -n ops-agent get ingress ops-agent -o yaml | grep whitelist-source-range
```

### kubectl命令问题

如果容器内的kubectl命令无法正常工作，请检查kubeconfig的挂载和权限：
```bash
kubectl -n ops-agent exec -it <pod-name> -- ls -la /root/.kube
kubectl -n ops-agent exec -it <pod-name> -- cat /root/.kube/config
kubectl -n ops-agent exec -it <pod-name> -- kubectl version
```

### 自动扩缩容问题

检查HPA状态和指标：
```bash
kubectl -n ops-agent get hpa
kubectl -n ops-agent describe hpa ops-agent
```

## 配置说明

### 环境变量

部署中使用了以下重要环境变量：

- `JWT_KEY` - JWT 认证密钥，从 Secret 中获取
- `TZ` - 时区设置，默认为 Asia/Shanghai
- `PYTHONPATH` - Python 包路径，确保Python工具正常运行
- `KUBECONFIG` - kubeconfig 文件路径，用于容器内执行 kubectl 命令
- `LOG_PATH` - 日志存储路径，默认设置为 `/app/logs`
- `ENV` - 环境标识，用于区分不同环境

### 日志目录配置

为了解决在非root用户下运行时的权限问题，我们：

1. 使用 initContainer 预先创建日志目录并设置正确权限
2. 使用 emptyDir 卷来存储应用日志
3. 通过环境变量 `LOG_PATH` 告知应用使用指定的日志路径

如果您需要持久化日志，可以将 `emptyDir` 替换为 `persistentVolumeClaim`。

### 资源配置

请根据实际需求调整资源请求和限制：

- 开发环境：
  - 请求：CPU 100m，内存 128Mi
  - 限制：CPU 500m，内存 512Mi

- 生产环境：
  - 请求：CPU 500m，内存 512Mi 
  - 限制：CPU 2000m，内存 2Gi

### 容器内使用 kubectl

在容器内使用 kubectl 的配置说明：

1. 通过 `kubeconfig-secret` 存储 kubeconfig 文件
2. 将 Secret 挂载到容器的 `/root/.kube` 目录
3. 设置 `KUBECONFIG` 环境变量指向 `/root/.kube/config`
4. 容器内可直接使用 kubectl 命令操作集群

## 安全注意事项

1. 生产环境中，请替换默认的 JWT 密钥为强密码
2. 考虑使用 NetworkPolicy 限制 Pod 的网络访问
3. 启用 TLS，使用有效的证书保护通信
4. 配置适当的 RBAC 权限，只授予必要的权限
5. 使用 IP 白名单限制 Ingress 访问，提高安全性
6. 注意 kubeconfig 中的权限级别，建议使用最小权限原则 
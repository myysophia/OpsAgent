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

如果遇到日志相关的问题，可以检查以下几点：

1. 查看日志目录是否存在并有正确权限：
```bash
kubectl -n ops-agent exec <pod-name> -- ls -la /app/logs
```

2. 检查环境变量是否正确设置：
```bash
kubectl -n ops-agent exec <pod-name> -- env | grep LOG_PATH
```

3. 如果使用的是自定义镜像，确保镜像中已经包含了日志目录：
```bash
# 确认镜像设置了正确的日志目录和权限
docker inspect ninesun0318/opsagent:main | grep -A 10 "Volumes"
docker inspect ninesun0318/opsagent:main | grep -A 10 "User"
```

> 注意：我们已经在 Dockerfile 中正确设置了日志目录权限，镜像构建时会自动创建 `/app/logs` 目录并设置适当权限。

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
- `ENV` - 环境标识，用于区分不同环境

### 日志目录配置

为了解决在非root用户下运行时的权限问题，我们在 Dockerfile 中:

1. 创建所需目录结构：`mkdir -p /app/logs`
2. 设置正确的所有权：`chown -R 1000:1000 /app/logs`
3. 设置适当的权限：`chmod 755 /app/logs`
4. 设置 `LOG_PATH` 环境变量指向 `/app/logs`
5. 使用 `USER 1000` 指令确保容器以非root用户运行

在 Kubernetes 部署配置中:
1. 使用 emptyDir 卷挂载到 `/app/logs` 路径
2. 这样应用程序可以在 pod 生命周期内使用此卷进行日志写入

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
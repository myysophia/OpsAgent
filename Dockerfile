# 使用多阶段构建减小最终镜像大小
FROM golang:1.24-alpine AS builder

# 添加代理设置和必要的构建工具
RUN apk add --no-cache git make
# 设置多个 GOPROXY 源以提高可靠性
ENV GOPROXY=https://goproxy.io,https://proxy.golang.org,https://goproxy.cn,direct
ENV GO111MODULE=on
ENV GOSUMDB=off

WORKDIR /app
COPY go.mod go.sum ./
# 增加重试机制和超时设置
RUN for i in 1 2 3 4 5; do go mod download && break || sleep 10; done

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o OpsAgent ./cmd/kube-copilot

# 使用轻量级基础镜像
FROM alpine:3.18

# 安装运行时依赖和 Python 依赖
RUN apk update --no-cache
RUN apk add --no-cache ca-certificates tzdata curl bash python3 py3-pip jq
RUN curl --retry 3 -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN chmod +x kubectl && mv kubectl /usr/local/bin/

# 创建应用所需目录结构并设置权限
RUN mkdir -p /app/logs && \
    mkdir -p /app/configs && \
    mkdir -p /root/.kube && \
    chmod 755 /app/logs && \
    chown -R 1000:1000 /app/logs /app/configs /root/.kube

WORKDIR /app
COPY --from=builder /app/OpsAgent .

# 下载并安装iotdbtools到工作目录
RUN curl -s "https://api.github.com/repos/myysophia/iotdbtool/releases/latest" | \
    grep "browser_download_url" | \
    grep "iotdbtools_linux_amd64" | \
    cut -d '"' -f 4 | \
    xargs curl -L -o iotdbtools && \
    chmod +x iotdbtools

# 安装 Python 依赖并设置 Python 环境
RUN pip3 install --no-cache-dir --upgrade pip
RUN pip3 install --no-cache-dir kubernetes==29.0.0 pyyaml==6.0.1 pandas==2.2.1
RUN mkdir -p /app/k8s/python-cli

# 创建并配置 Python 虚拟环境
RUN python3 -m venv /app/k8s/python-cli/k8s-env && \
    . /app/k8s/python-cli/k8s-env/bin/activate && \
    pip install --no-cache-dir --upgrade pip && \
    pip install --no-cache-dir kubernetes==29.0.0 pyyaml==6.0.1 pandas==2.2.1 && \
    deactivate

# 清理缓存
RUN rm -rf /var/cache/apk/*

# 创建软链接，确保环境路径一致
RUN ln -s /app/k8s /root/k8s

# 设置环境变量
ENV GIN_MODE=release
ENV PYTHONPATH=/app/k8s/python-cli/k8s-env/lib/python3.*/site-packages
ENV LOG_PATH=/app/logs

# 设置用户
USER 1000

EXPOSE 8080
ENTRYPOINT ["./OpsAgent"]
CMD ["server", "--port", "8080", "--jwt-key", "123456"]
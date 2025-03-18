# 使用多阶段构建减小最终镜像大小
FROM golang:1.24-alpine AS builder

# 添加代理设置和必要的构建工具
RUN apk add --no-cache git make
# 设置 GOPROXY 环境变量以解决下载问题
ENV GOPROXY=https://goproxy.cn,direct

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o OpsAgent ./cmd/kube-copilot

# 使用轻量级基础镜像
FROM alpine:3.18

# 安装运行时依赖和 Python 依赖
RUN apk update --no-cache
RUN apk add --no-cache ca-certificates tzdata curl bash python3 py3-pip jq
RUN curl --retry 3 -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
RUN chmod +x kubectl && mv kubectl /usr/local/bin/

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

WORKDIR /app
COPY --from=builder /app/OpsAgent .

ENV GIN_MODE=release
ENV PYTHONPATH=/app/k8s/python-cli/k8s-env/lib/python3.*/site-packages

EXPOSE 8080
ENTRYPOINT ["./OpsAgent"]
CMD ["server", "--port", "8080"]
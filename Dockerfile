# 使用多阶段构建减小最终镜像大小
FROM golang:1.24-alpine AS builder

# 安装必要的构建工具
RUN apk add --no-cache git make

# 设置工作目录
WORKDIR /app

# 复制go.mod和go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -o kube-copilot ./cmd/kube-copilot

# 使用轻量级基础镜像
FROM alpine:3.19

# 安装必要的运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    kubectl \
    curl \
    jq \
    python3 \
    py3-pip \
    gcc \
    python3-dev \
    musl-dev \
    linux-headers \
    && pip3 install --no-cache-dir kubernetes \
    && apk del gcc python3-dev musl-dev linux-headers

# 设置工作目录
WORKDIR /app

# 从builder阶段复制二进制文件
COPY --from=builder /app/kube-copilot .

# 设置环境变量
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080

# 启动命令
ENTRYPOINT ["./kube-copilot"]
CMD ["server", "--port", "8080"]

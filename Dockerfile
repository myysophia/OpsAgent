# 使用多阶段构建减小最终镜像大小
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git make
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o OpsAgent ./cmd/OpsAgent

# 使用轻量级基础镜像
FROM alpine:3.19

# 安装运行时依赖和 Python 依赖
RUN apk update && apk add --no-cache ca-certificates tzdata curl bash python3 py3-pip jq && \
    # 手动安装 kubectl
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    # 安装 Python 依赖
    pip3 install --no-cache-dir --upgrade pip && \
    pip3 install --no-cache-dir kubernetes==29.0.0 pyyaml==6.0.1 pandas==2.2.1 && \
    mkdir -p /app/k8s/python-cli && \
    rm -rf /var/cache/apk/*

WORKDIR /app
COPY --from=builder /app/OpsAgent .

ENV GIN_MODE=release
EXPOSE 8080
ENTRYPOINT ["./OpsAgent"]
CMD ["server", "--port", "8080"]
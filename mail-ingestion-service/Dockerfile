# 多阶段构建：构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git

# 先复制 go.mod 文件（所有服务）
# 注意：只复制 go.mod，go.sum 会在 go mod download 时自动生成（如果不存在）
COPY go.mod ./
COPY api-gateway/go.mod ./api-gateway/
COPY mail-ingestion-service/go.mod ./mail-ingestion-service/
COPY worker-service/go.mod ./worker-service/

# 下载依赖（如果 go.sum 不存在会自动生成）
RUN go mod download

# 复制源代码
COPY . .

# 构建 mail-ingestion-service
WORKDIR /build/mail-ingestion-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o mail-ingestion-service ./cmd/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/mail-ingestion-service/mail-ingestion-service .
COPY --from=builder /build/mail-ingestion-service/config.yaml .

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8081

CMD ["./mail-ingestion-service"]


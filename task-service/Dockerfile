# 多阶段构建：构建阶段
FROM golang:1.25-alpine AS builder

WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git

# 先复制 go.mod 文件（所有服务）
COPY go.mod ./
COPY task-service/go.mod ./task-service/

# 下载依赖
RUN go mod download

# 复制源代码
COPY . .

# 构建 task-service
WORKDIR /build/task-service
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o task-service ./cmd/main.go

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata wget

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/task-service/task-service .
COPY --from=builder /build/task-service/config.yaml .

# 设置时区
ENV TZ=Asia/Shanghai

EXPOSE 8082

CMD ["./task-service"]


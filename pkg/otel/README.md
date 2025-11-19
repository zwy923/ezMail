# OpenTelemetry 全链路追踪

本包提供了 OpenTelemetry 全链路追踪的集成，支持 HTTP、MQ 和数据库操作的自动追踪。

## 功能特性

- ✅ HTTP 请求追踪（Gin 中间件）
- ✅ MQ 发布/消费追踪（自动传播 trace context）
- ✅ 数据库查询追踪（PostgreSQL）
- ✅ 跨服务 trace context 传播（W3C Trace Context 标准）

## 使用方法

### 1. 初始化 OpenTelemetry

在每个服务的 `main.go` 中初始化：

```go
import (
    "os"
    "mygoproject/pkg/otel"
    "go.uber.org/zap"
)

func main() {
    logger := logger.NewLogger()
    
    // 初始化 OpenTelemetry
    otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
    if otelEndpoint == "" {
        otelEndpoint = "otel-collector:4317"
    }
    shutdown, err := otel.Init(otel.Config{
        ServiceName:    "your-service-name",
        ServiceVersion: "1.0.0",
        Endpoint:       otelEndpoint,
        Enabled:        true,
    }, logger)
    if err != nil {
        logger.Fatal("Failed to init OpenTelemetry", zap.Error(err))
    }
    defer shutdown()
    
    // ... 其他初始化代码
}
```

### 2. HTTP 追踪

在 Gin 路由中添加中间件：

```go
import "mygoproject/pkg/otel"

func NewRouter() *gin.Engine {
    r := gin.Default()
    
    // OpenTelemetry 追踪中间件（必须在最前面）
    r.Use(otel.GinMiddleware())
    
    // ... 其他中间件和路由
}
```

### 3. MQ 追踪

MQ 发布和消费已自动集成追踪，无需额外配置。

**发布消息：**
```go
// 自动创建 Producer span 并传播 trace context
err := publisher.PublishWithContext(ctx, "task.created", payload)
```

**消费消息：**
```go
// 自动从消息头提取 trace context 并创建 Consumer span
consumer.SetHandler(func(ctx context.Context, data json.RawMessage) error {
    // ctx 已包含 trace context
    // ... 处理消息
})
```

### 4. 数据库追踪

使用辅助函数包装数据库查询：

```go
import "mygoproject/pkg/otel"

// QueryRow 示例
var id int
err := otel.QueryRow(ctx, "select", "SELECT id FROM users WHERE email = $1", func(ctx context.Context) error {
    return db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", email).Scan(&id)
})

// Exec 示例
err := otel.Exec(ctx, "insert", "INSERT INTO tasks (...) VALUES (...)", func(ctx context.Context) error {
    _, err := db.Exec(ctx, "INSERT INTO tasks (...) VALUES (...)", ...)
    return err
})
```

或者手动创建 span：

```go
import "mygoproject/pkg/otel"

ctx, span := otel.DBSpan(ctx, "select", "SELECT * FROM users WHERE id = $1")
defer span.End()

var user User
err := db.QueryRow(ctx, "SELECT * FROM users WHERE id = $1", id).Scan(&user)
otel.WrapDBError(span, err)
```

## 查看追踪数据

1. 启动服务：
```bash
docker-compose up -d
```

2. 访问 Jaeger UI：
   - 地址：http://localhost:16686
   - 选择服务名称
   - 点击 "Find Traces" 查看完整的调用链路

## 配置

### 环境变量

- `OTEL_EXPORTER_OTLP_ENDPOINT`: OpenTelemetry Collector 地址（默认：`otel-collector:4317`）

### Docker Compose

确保 `docker-compose.yml` 中包含：
- `otel-collector`: OpenTelemetry Collector 服务
- `jaeger`: Jaeger 追踪后端

## 示例：完整的调用链路

一次任务创建的完整链路：

```
POST /email/simulate (api-gateway)
  └─> POST /email/simulate (mail-ingestion-service)
      └─> mq.publish (email.received.agent)
          └─> mq.consume (email-processor-service)
              └─> db.select (查询邮件)
              └─> HTTP POST /decide (agent-service)
              └─> db.insert (保存元数据)
              └─> mq.publish (task.created)
                  └─> mq.consume (task-service)
                      └─> db.insert (创建任务)
```

所有步骤都会在 Jaeger UI 中显示为一个完整的 trace，包括每个步骤的延迟和错误信息。


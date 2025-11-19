# 项目结构与事件交互文档

## 📁 项目结构

### 微服务架构

```
MyGoProject/
├── api-gateway/              # API 网关服务（Go）
│   ├── cmd/main.go          # 入口文件
│   ├── internal/
│   │   ├── handler/         # 请求处理器
│   │   │   ├── auth_handler.go
│   │   │   ├── email_query_handler.go
│   │   │   ├── mail_proxy_handler.go
│   │   │   └── task_controller.go      # 统一的任务控制器
│   │   ├── httpserver/      # HTTP 服务器
│   │   ├── repository/     # 数据访问层
│   │   └── service/         # 业务逻辑层
│   └── config.yaml
│
├── mail-ingestion-service/   # 邮件接收服务（Go）
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/         # 邮件接收处理器
│   │   ├── repository/      # 邮件和失败事件存储
│   │   └── service/         # 邮件处理服务
│   └── config.yaml
│
├── email-processor-service/  # 邮件处理服务（Go）
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── mqhandler/       # MQ 消息处理器
│   │   │   ├── agent_handler.go        # AI 决策处理（发布 task.created 和 notification.created）
│   │   │   ├── notification_handler.go # 发布 notification.created 事件
│   │   │   └── notification_log_handler.go
│   │   ├── repository/      # 数据访问层（email, metadata, notification_log）
│   │   └── service/         # Agent 客户端
│   └── config.yaml
│
├── task-service/             # 任务管理服务（Go）
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── handler/         # HTTP 请求处理器
│   │   ├── mqhandler/       # MQ 消息处理器
│   │   │   ├── task_created_handler.go
│   │   │   ├── task_bulk_created_handler.go
│   │   │   ├── habit_created_handler.go
│   │   │   ├── project_created_handler.go
│   │   │   ├── task_overdue_handler.go
│   │   │   ├── task_unlocked_handler.go
│   │   │   └── habit_task_generated_handler.go
│   │   ├── repository/      # 数据访问层
│   │   │   ├── task_repo.go
│   │   │   ├── habit_repo.go
│   │   │   ├── project_repo.go
│   │   │   └── milestone_repo.go
│   │   └── model/           # 数据模型
│   └── config.yaml
│
│   **注意：** 任务编排逻辑（定时扫描、逾期检查、依赖解锁、习惯生成）已迁移到 `task-runner-service`
│
├── task-runner-service/      # 任务编排引擎（Go）
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── repository/      # 数据访问层
│   │   │   ├── task_repo.go
│   │   │   └── habit_repo.go
│   │   ├── service/         # 编排服务
│   │   │   └── orchestrator.go      # 任务编排器
│   │   ├── httpserver/      # HTTP 服务器（健康检查）
│   │   └── config/          # 配置管理
│   └── config.yaml
│
├── notification-service/     # 通知服务（Go）
│   ├── cmd/main.go
│   ├── internal/
│   │   ├── mqhandler/       # MQ 消息处理器
│   │   │   └── notification_created_handler.go
│   │   ├── repository/      # 数据访问层
│   │   │   └── notification_repo.go
│   │   ├── service/         # 业务服务
│   │   │   └── notification_sender.go      # 通知发送器
│   │   ├── httpserver/      # HTTP 服务器（健康检查）
│   │   └── config/          # 配置管理
│   └── config.yaml
│
├── agent-service/            # AI 代理服务（Python/FastAPI）
│   ├── app/
│   │   ├── main.py          # FastAPI 应用入口
│   │   ├── agent/
│   │   │   ├── chain.py              # 邮件决策链
│   │   │   ├── text_to_tasks_chain.py # 文本转任务链
│   │   │   └── project_planner_chain.py # 项目规划链
│   │   ├── schema.py        # Pydantic 模型
│   │   └── config.py        # 配置管理
│   └── Dockerfile
│
├── contracts/                 # 共享契约
│   ├── mq/                   # MQ 事件契约
│   │   ├── email_received.go
│   │   └── task.go
│   └── db/                   # 数据库契约
│
├── pkg/                      # 共享包
│   ├── db/                   # 数据库连接
│   │   ├── db.go             # 数据库连接池
│   │   └── slow_query_hook.go # 慢查询监控 Tracer
│   ├── mq/                   # MQ 连接（Publisher/Consumer）
│   ├── logger/               # 日志工具
│   ├── redis/                # Redis 客户端
│   ├── util/                 # 工具函数（JWT, 密码, 去重, 重试计数）
│   ├── outbox/               # Outbox 模式（可靠事件发布）
│   │   ├── outbox.go         # Repository 层
│   │   ├── dispatcher.go     # 后台 Dispatcher
│   │   ├── replay.go         # Replay 服务
│   │   └── helper.go         # 辅助函数
│   ├── circuitbreaker/       # 熔断器（Circuit Breaker）
│   ├── rbac/                 # 基于角色的访问控制
│   ├── trace/                # 分布式追踪（Trace ID，向后兼容）
│   ├── otel/                 # OpenTelemetry 全链路追踪
│   │   ├── otel.go           # OpenTelemetry 初始化
│   │   ├── http.go           # HTTP 追踪中间件
│   │   ├── mq.go             # MQ 追踪（Publisher/Consumer）
│   │   └── db.go             # 数据库追踪
│   ├── metrics/              # Prometheus 指标
│   └── config/               # 统一配置中心
│
└── migrations/               # 数据库迁移
    ├── 001_init_schema.sql
    └── 002_add_outbox.sql    # Outbox 表结构
```

---

## 🗄️ 数据库表结构

### 1. users（用户表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 用户ID |
| email | VARCHAR(255) UNIQUE | 邮箱（唯一） |
| password_hash | VARCHAR(255) | 密码哈希 |
| created_at | TIMESTAMP | 创建时间 |

### 2. emails_raw（原始邮件表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 邮件ID |
| user_id | INT | 用户ID（外键 → users.id） |
| subject | TEXT | 邮件主题 |
| body | TEXT | 邮件正文 |
| raw_json | JSONB | 原始JSON数据 |
| status | email_status ENUM | 状态：'received' / 'classified' |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- `idx_emails_raw_user` (user_id)
- `idx_emails_raw_status` (status)

### 3. emails_metadata（邮件元数据表）
| 字段 | 类型 | 说明 |
|------|------|------|
| email_id | INT PRIMARY KEY | 邮件ID（外键 → emails_raw.id） |
| categories | TEXT[] | 分类数组：["WORK","ACTION_REQUIRED"] |
| priority | TEXT | 优先级：LOW / MEDIUM / HIGH |
| summary | TEXT | 摘要（1-3句话） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

### 4. habits（习惯表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 习惯ID |
| user_id | INT | 用户ID（外键 → users.id） |
| title | VARCHAR(255) | 习惯标题 |
| recurrence_pattern | VARCHAR(100) | 重复模式："weekly Wednesday", "daily", "monthly 1" |
| is_active | BOOLEAN | 是否激活（默认 TRUE） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_habits_user` (user_id)
- `idx_habits_active` (is_active) WHERE is_active = TRUE

### 5. projects（项目表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 项目ID |
| user_id | INT | 用户ID（外键 → users.id） |
| title | VARCHAR(255) | 项目标题 |
| description | TEXT | 项目描述 |
| target_date | DATE | 项目截止日期 |
| status | VARCHAR(50) | 状态：'active' / 'completed' / 'cancelled'（默认 'active'） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_projects_user` (user_id)
- `idx_projects_status` (status)

### 6. milestones（里程碑/阶段表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 里程碑ID |
| project_id | INT | 项目ID（外键 → projects.id） |
| title | VARCHAR(255) | 阶段标题 |
| description | TEXT | 阶段描述 |
| phase_order | INT | 阶段顺序（1, 2, 3, ...） |
| target_date | DATE | 阶段截止日期 |
| status | VARCHAR(50) | 状态：'pending' / 'in_progress' / 'completed'（默认 'pending'） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_milestones_project` (project_id)
- `idx_milestones_status` (status)
- `idx_milestones_order` (project_id, phase_order)

### 7. tasks（任务表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 任务ID |
| user_id | INT | 用户ID（外键 → users.id） |
| email_id | INT | 邮件ID（外键 → emails_raw.id，可为 NULL） |
| habit_id | INT | 习惯ID（外键 → habits.id，可为 NULL） |
| project_id | INT | 项目ID（外键 → projects.id，可为 NULL） |
| milestone_id | INT | 里程碑ID（外键 → milestones.id，可为 NULL） |
| title | VARCHAR(255) | 任务标题 |
| due_date | DATE | 截止日期 |
| priority | VARCHAR(20) | 优先级：LOW / MEDIUM / HIGH（默认 'MEDIUM'） |
| status | VARCHAR(50) | 状态：'pending' / 'done' / 'overdue'（默认 'pending'） |
| completed_at | TIMESTAMP | 完成时间（可为 NULL） |
| created_at | TIMESTAMP | 创建时间 |

**任务来源说明：**
- **来自邮件：** `email_id > 0`（插入实际值），`habit_id` 和 `project_id` 为 NULL
  - 通过 `task.created` 事件创建，`TaskCreatedHandler` 验证 `email_id > 0`
  - `Insert` 方法：当 `email_id > 0` 时插入实际值，否则插入 NULL
- **来自文本：** `email_id = 0`（插入 NULL），`habit_id` 和 `project_id` 为 NULL（一次性任务）
  - 通过 `task.bulk_created` 事件创建，`TaskBulkCreatedHandler` 设置 `EmailID: 0`
  - `BulkInsert` 方法：当 `email_id <= 0` 时插入 NULL，避免外键冲突
- **来自习惯：** `habit_id` 不为 NULL，`email_id` 为 NULL（不设置），`project_id` 为 NULL
  - 通过 `habit.task.generated` 事件创建，`InsertFromHabit` 方法不包含 `email_id` 字段
- **来自项目：** `project_id` 和 `milestone_id` 不为 NULL，`email_id` 为 NULL（不设置），`habit_id` 为 NULL
  - 通过 `project.created` 事件创建，`InsertFromProject` 方法不包含 `email_id` 字段

**重要：** 所有插入方法都正确处理 `email_id` 为 NULL 的情况，避免外键冲突。`ListByUser` 方法使用 `sql.NullInt32` 正确读取 NULL 值。

**索引：**
- `idx_tasks_user` (user_id)
- `idx_tasks_status` (status)
- `idx_tasks_habit` (habit_id)
- `idx_tasks_project` (project_id)
- `idx_tasks_milestone` (milestone_id)
- `idx_tasks_due_date` (due_date)
- `idx_tasks_priority` (priority)

**唯一约束：**
- `idx_tasks_unique_pending_email_user`：同一 email_id + user_id 只能有一个 pending 任务
- `idx_tasks_unique_pending_habit_date`：同一 habit_id + due_date 只能有一个 pending 任务（幂等性）

### 8. task_dependencies（任务依赖表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 依赖关系ID |
| task_id | INT | 任务ID（外键 → tasks.id） |
| depends_on_task_id | INT | 依赖的任务ID（外键 → tasks.id） |
| created_at | TIMESTAMP | 创建时间 |

**约束：**
- `task_dependencies_no_self_reference`：task_id != depends_on_task_id（不能依赖自己）

**索引：**
- `idx_task_dependencies_task` (task_id)
- `idx_task_dependencies_depends_on` (depends_on_task_id)

### 9. notifications（通知表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 通知ID |
| user_id | INT | 用户ID（外键 → users.id） |
| email_id | INT | 邮件ID（外键 → emails_raw.id） |
| channel | TEXT | 通知渠道：EMAIL / PUSH / SMS |
| message | TEXT | 通知消息 |
| is_read | BOOLEAN | 是否已读（默认 FALSE） |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- `idx_notifications_user` (user_id)
- `idx_notifications_email` (email_id)
- `idx_notifications_is_read` (is_read)

### 10. notifications_log（通知日志表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 日志ID |
| user_id | INT | 用户ID（外键 → users.id） |
| email_id | INT | 邮件ID（外键 → emails_raw.id） |
| message | TEXT | 日志消息 |
| created_at | TIMESTAMP | 创建时间 |

**索引：**
- `idx_notifications_log_user` (user_id)
- `idx_notifications_log_email` (email_id)

### 11. failed_events（失败事件表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | SERIAL PRIMARY KEY | 失败事件ID |
| email_id | INT | 邮件ID（外键 → emails_raw.id） |
| user_id | INT | 用户ID（外键 → users.id） |
| event_type | VARCHAR(50) | 事件类型 |
| routing_key | VARCHAR(100) | 路由键 |
| payload | JSONB | 事件负载（JSON） |
| error_message | TEXT | 错误消息 |
| retry_count | INT | 重试次数（默认 0） |
| status | VARCHAR(20) | 状态：'pending' / 'retried' / 'failed'（默认 'pending'） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_failed_events_status` (status)
- `idx_failed_events_email` (email_id)
- `idx_failed_events_pending_retry` (status, retry_count) WHERE status = 'pending'

### 12. outbox_events（Outbox 事件表）
| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PRIMARY KEY | 事件ID |
| aggregate_type | VARCHAR(50) | 聚合类型：email/task/habit/project/notification |
| aggregate_id | BIGINT | 关联对象ID（可选） |
| routing_key | VARCHAR(100) | MQ 路由键 |
| payload | JSONB | 事件负载（JSON） |
| status | VARCHAR(20) | 状态：'pending' / 'sent' / 'failed'（默认 'pending'） |
| retry_count | INT | 重试次数（默认 0） |
| next_retry_at | TIMESTAMP | 下次重试时间（失败后） |
| created_at | TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | 更新时间 |

**索引：**
- `idx_outbox_pending` (status, next_retry_at) WHERE status = 'pending'
- `idx_outbox_aggregate` (aggregate_type, aggregate_id)
- `idx_outbox_failed` (status) WHERE status = 'failed'

**说明：**
- 每个服务都有自己的 `outbox_events` 表（服务自治）
- 使用 Outbox 模式确保事件发布的可靠性和事务一致性
- Dispatcher 后台自动发送待处理事件

---

## 🔄 MQ 事件交互逻辑

### Outbox 模式（可靠事件发布）

**使用 Outbox 的服务：**
- ✅ **mail-ingestion-service** - `email.received.*` 事件（3个路由键：agent, log, notify）
- ✅ **email-processor-service** - `task.created`、`notification.created` 事件（最重要，在事务中同时写入 metadata 和 outbox）
- ⚠️ **api-gateway** - `project.created` 事件（已使用 Outbox），但 `habit.created` 和 `task.bulk_created` **仍使用直接发布**
- ✅ **task-runner-service** - `task.overdue`、`task.unlocked`、`habit.task.generated` 事件（只写入 outbox，不更新业务数据）
- ✅ **notification-service** - `notification.sent`、`notification.failed` 事件（在发送后写入 outbox）

**Outbox 工作流程：**
1. **事务写入：** 业务数据和事件在同一事务中写入 `outbox_events` 表
2. **后台发送：** Outbox Dispatcher 每秒扫描待处理事件并自动发送到 MQ
3. **自动重试：** 发送失败时自动重试（最多 5 次，指数退避）
4. **手动重放：** 通过 Replay API 手动重放失败事件

**优势：**
- ✅ 事务一致性：业务数据和事件保证一致
- ✅ 可靠性：MQ 发布失败不影响业务数据
- ✅ 可追溯：所有事件都有 trace_id
- ✅ 可恢复：失败事件可以手动重放

### MQ 路由键和队列总览

| 路由键 | 队列名 | 发布者 | 消费者 | Outbox | 说明 |
|--------|--------|--------|--------|--------|------|
| `email.received.agent` | `email.received.agent.q` | mail-ingestion-service | email-processor-service | ✅ | AI 决策处理 |
| `email.received.log` | `email.received.log.q` | mail-ingestion-service | email-processor-service | ✅ | 通知日志记录 |
| `email.received.notify` | `email.received.notify.q` | mail-ingestion-service | email-processor-service | ✅ | 通知创建 |
| `task.created` | `task.created.q` | email-processor-service | task-service | ✅ | 单个任务创建（来自邮件） |
| `task.bulk_created` | `task.bulk_created.q` | api-gateway | task-service | ⚠️ | 批量任务创建（**直接发布，未使用 Outbox**） |
| `habit.created` | `habit.created.q` | api-gateway | task-service | ⚠️ | 习惯创建（**直接发布，未使用 Outbox**） |
| `project.created` | `project.created.q` | api-gateway | task-service | ✅ | 项目创建（已使用 Outbox） |
| `task.overdue` | `task.overdue.q` | task-runner-service | task-service | ✅ | 任务逾期 |
| `task.unlocked` | `task.unlocked.q` | task-runner-service | task-service | ✅ | 任务解锁（依赖完成） |
| `habit.task.generated` | `habit.task.generated.q` | task-runner-service | task-service | ✅ | 习惯任务生成 |
| `notification.created` | `notification.created.q` | email-processor-service | notification-service | ✅ | 通知创建 |
| `notification.sent` | `notification.sent.q` | notification-service | - | ✅ | 通知发送成功 |
| `notification.failed` | `notification.failed.q` | notification-service | - | ✅ | 通知发送失败 |

**死信队列（DLQ）：**
- 每个路由键都有对应的 DLQ：`{routing_key}.dlq`
- 例如：`task.created.dlq`, `email.received.agent.dlq`

### 事件流程图

```
┌─────────────────┐
│  API Gateway    │
│  (用户请求)      │
└────────┬────────┘
         │
         ├─ POST /email/simulate ──┐
         │                          │
         ├─ POST /tasks/from-text ─┤
         │                          │
         └─ POST /tasks/plan-project
                                    │
                                    ▼
                    ┌───────────────────────────┐
                    │  Mail Ingestion Service   │
                    │  1. 保存邮件到 emails_raw │
                    │  2. 发布 email.received  │
                    └──────────────┬───────────┘
                                    │
                                    │ 发布 3 个路由键
                                    ├─ email.received.agent
                                    ├─ email.received.log
                                    └─ email.received.notify
                                    │
        ┌───────────────────────────┼───────────────────────────┐
        │                           │                           │
        ▼                           ▼                           ▼
┌───────────────┐         ┌───────────────┐         ┌───────────────┐
│Email Processor│         │Email Processor│         │Email Processor│
│ (Agent Handler)│        │ (Log Handler) │        │ (Notify Handler)│
│               │         │               │         │               │
│ 1. 调用 Agent │         │ 记录日志      │         │ 发布通知事件  │
│ 2. 保存元数据 │         │               │         │               │
│ 3. 发布任务   │         │               │         │               │
└───────┬───────┘         └───────────────┘         └───────────────┘
        │
        │ 发布 task.created
        ▼
┌───────────────┐
│ Task Service  │
│ 创建任务      │
└───────────────┘
```

### 事件列表

#### 1. email.received（邮件接收事件）

**发布者：** `mail-ingestion-service`（使用 Outbox 模式）  
**路由键：**
- `email.received.agent` - AI 决策处理
- `email.received.log` - 日志记录
- `email.received.notify` - 通知处理

**发布方式：**
- 在事务中同时写入 `emails_raw` 和 `outbox_events` 表
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `EmailReceivedPayload`
```go
{
    email_id: int
    user_id: int
    subject: string
    body: string
    received_at: time.Time
}
```

**消费者：**
- `email-processor-service` (email.received.agent.q) → `AgentDecisionHandler`
- `email-processor-service` (email.received.log.q) → `NotificationLogHandler`
- `email-processor-service` (email.received.notify.q) → `NotificationHandler`

**处理流程：**
1. **Agent Handler（email-processor-service/internal/mqhandler/agent_handler.go）：**
   - **Step 1:** 解码 payload，提取 trace_id 并注入 context
   - **Step 2:** 加载邮件，检查幂等性（如果已 classified 则跳过）
   - **Step 3:** Redis 去重（避免并发重复消费）
   - **Step 4:** 调用 `agent-service /decide`（带熔断器和 fallback）
     - 熔断器配置：失败阈值 3，超时 30 秒
     - Fallback：返回默认决策（不创建任务、不发送通知），确保 ingestion-service 继续运行
     - 记录 `agent_call_latency_ms` 指标
   - **Step 5-8:** 在**单个事务**中执行：
     - 写入 `emails_metadata`（InsertDecisionTx）
     - 如果 `should_create_task`，写入 `outbox_events` (task.created)
     - 如果 `should_notify`，写入 `outbox_events` (notification.created)
     - 更新 `emails_raw.status = 'classified'`（UpdateStatusTx）
   - **Step 9:** 记录 metrics（IncrementEmailProcessed, IncrementTaskGeneration）
   - **错误处理：**
     - 可重试错误：返回错误（nack，触发重试）
     - 不可重试错误：写入 unknown + classified，返回 nil（ack）
     - 超过最大重试次数（5次）：写入 unknown + classified，返回 nil（ack）

2. **Log Handler：**
   - 记录通知日志到 `notifications_log`

3. **Notify Handler：**
   - 发布 `notification.created` 事件（由 notification-service 处理）

---

#### 2. task.created（单个任务创建事件）

**发布者：** `email-processor-service` (AgentDecisionHandler，使用 Outbox 模式)  
**路由键：** `task.created`  
**队列：** `task.created.q`

**发布方式：**
- 在事务中同时写入 `emails_metadata`、`outbox_events` 和更新 `emails_raw.status`
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `TaskCreatedPayload`
```go
{
    email_id: int
    user_id: int
    title: string
    due_in_days: int
}
```

**消费者：** `task-service` → `TaskCreatedHandler`

**处理流程（task-service/internal/mqhandler/task_created_handler.go）：**
- 验证 `email_id > 0`（task.created 事件必须来自邮件）
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 插入任务到 `tasks` 表
- 关联 `email_id` 和 `user_id`
- 计算 `due_date = now + due_in_days`
- 幂等性保证：唯一索引 `idx_tasks_unique_pending_email_user` 确保同一 email_id + user_id 只能有一个 pending 任务

---

#### 3. task.bulk_created（批量任务创建事件）

**发布者：** `api-gateway` (TaskController.CreateTasksFromText，使用 Outbox 模式)  
**路由键：** `task.bulk_created`  
**队列：** `task.bulk_created.q`

**发布方式：**
- 在事务中写入 `outbox_events` 表（不写业务数据）
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `TaskBulkCreatedPayload`
```go
{
    user_id: int
    tasks: [
        {
            title: string
            due_in_days: int
        }
    ]
}
```

**消费者：** `task-service` → `TaskBulkCreatedHandler`

**处理流程（task-service/internal/mqhandler/task_bulk_created_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 使用事务批量插入任务
- `email_id` 为 0 时插入 NULL（文本转任务没有关联邮件，避免外键冲突）
- `Insert` 和 `BulkInsert` 方法自动处理：当 `email_id <= 0` 时插入 NULL

---

#### 4. habit.created（习惯创建事件）

**发布者：** `api-gateway` (TaskController.CreateTasksFromText，使用 Outbox 模式)  
**路由键：** `habit.created`  
**队列：** `habit.created.q`

**发布方式：**
- 在事务中写入 `outbox_events` 表（不写业务数据）
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `HabitCreatedPayload`
```go
{
    user_id: int
    title: string
    recurrence_pattern: string  // "weekly Wednesday", "daily", "monthly 1"
}
```

**消费者：** `task-service` → `HabitCreatedHandler`

**处理流程（task-service/internal/mqhandler/habit_created_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 插入习惯到 `habits` 表
- `is_active = TRUE`
- 幂等性：如果习惯已存在（相同 user_id + title），跳过或更新

**后续处理：**
- `task-runner-service` 的 `Orchestrator` 每天凌晨 00:00 自动生成当天的习惯任务
- 发布 `habit.task.generated` 事件，由 `task-service` 处理
- 使用唯一索引保证幂等性（同一习惯同一天只生成一次）
- 定时任务在 `task-runner-service/cmd/main.go` 中实现，使用 `time.Ticker` 每 24 小时运行一次

---

#### 5. project.created（项目创建事件）

**发布者：** `api-gateway` (TaskController.PlanProject，使用 Outbox 模式)  
**路由键：** `project.created`  
**队列：** `project.created.q`

**发布方式：**
- 在事务中写入 `outbox_events` 表（不写业务数据）
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `ProjectCreatedPayload`
```go
{
    user_id: int
    title: string
    description: string
    target_days: int
    milestones: [
        {
            title: string
            order: int
            due_in_days: int
            tasks: [
                {
                    title: string
                    due_in_days: int
                    priority: string  // LOW / MEDIUM / HIGH
                    depends_on: []string  // 依赖的任务标题列表
                }
            ]
        }
    ]
}
```

**消费者：** `task-service` → `ProjectCreatedHandler`

**处理流程（task-service/internal/mqhandler/project_created_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 使用事务执行：
  1. 创建项目到 `projects` 表
  2. 为每个 milestone 创建阶段到 `milestones` 表
  3. 为每个任务创建任务到 `tasks` 表（关联 `project_id` 和 `milestone_id`）
     - `email_id` 为 NULL（项目任务不关联邮件，`InsertFromProject` 方法不包含 `email_id` 字段）
  4. 解析任务依赖关系，创建 `task_dependencies` 记录
     - 依赖关系基于任务标题（`depends_on` 字段）
     - 在同一 milestone 内查找依赖任务

---

#### 6. task.overdue（任务逾期事件）

**发布者：** `task-runner-service` (Orchestrator.CheckAndMarkOverdue，使用 Outbox 模式)  
**路由键：** `task.overdue`  
**队列：** `task.overdue.q`

**发布方式（task-runner-service/internal/service/orchestrator.go）：**
- **已使用 Outbox 模式**
- 在事务中同时执行：
  - 更新 `tasks.status = 'overdue'`（MarkExpiredTx）
  - 为每个过期的任务写入 `outbox_events` (task.overdue)
- Outbox Dispatcher 自动发送到 MQ
- 定时任务：每 1 分钟运行一次（CheckAndMarkOverdue）

**Payload：** `TaskOverduePayload`
```go
{
    task_id: int
}
```

**消费者：** `task-service` → `TaskOverdueHandler`

**处理流程（task-service/internal/mqhandler/task_overdue_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- **注意：** 任务已在数据库中标记为 overdue（由 task-runner-service 的 CheckAndMarkOverdue 完成）
- Handler 可用于额外处理（如通知、分析等）

---

#### 7. task.unlocked（任务解锁事件）

**发布者：** `task-runner-service` (Orchestrator.CheckAndUnlockTasks，使用 Outbox 模式)  
**路由键：** `task.unlocked`  
**队列：** `task.unlocked.q`

**发布方式：**
- 在事务中写入 `outbox_events` 表
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `TaskUnlockedPayload`
```go
{
    task_id: int
    user_id: int
    title: string
}
```

**消费者：** `task-service` → `TaskUnlockedHandler`

**处理流程（task-service/internal/mqhandler/task_unlocked_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- **注意：** 任务的所有依赖已完成（由 task-runner-service 的 CheckAndUnlockTasks 检查）
- Handler 可用于额外处理（如通知用户、分析等）

---

#### 8. habit.task.generated（习惯任务生成事件）

**发布者：** `task-runner-service` (Orchestrator.GenerateHabitTasks，使用 Outbox 模式)  
**路由键：** `habit.task.generated`  
**队列：** `habit.task.generated.q`

**发布方式：**
- 在事务中写入 `outbox_events` 表
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `HabitTaskGeneratedPayload`
```go
{
    habit_id: int
    user_id: int
    title: string
    due_date: string  // YYYY-MM-DD format
}
```

**消费者：** `task-service` → `HabitTaskGeneratedHandler`

**处理流程（task-service/internal/mqhandler/habit_task_generated_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 插入任务到 `tasks` 表（关联 `habit_id`）
- `email_id` 为 NULL（习惯任务不关联邮件）
- 使用唯一索引保证幂等性（同一习惯同一天只生成一次）
  - 唯一索引：`idx_tasks_unique_pending_habit_date` (habit_id, due_date)
  - 使用 `ON CONFLICT DO NOTHING` 避免重复插入

---

#### 9. notification.created（通知创建事件）

**发布者：** `email-processor-service` (AgentDecisionHandler, EmailReceivedNotificationHandler，使用 Outbox 模式)  
**路由键：** `notification.created`  
**队列：** `notification.created.q`

**发布方式（email-processor-service/internal/mqhandler/agent_handler.go）：**
- **已使用 Outbox 模式**
- 在事务中同时执行：
  - 写入 `emails_metadata`（InsertDecisionTx）
  - 写入 `outbox_events` (notification.created)
  - 更新 `emails_raw.status = 'classified'`（UpdateStatusTx）
- Outbox Dispatcher 自动发送到 MQ
- **注意：** 通知失败不影响主流程（只记录日志，继续执行）

**Payload：** `NotificationCreatedPayload`
```go
{
    user_id: int
    email_id: int (optional)
    task_id: int (optional)
    channel: string  // EMAIL / PUSH / SMS / WEBHOOK
    message: string
    created_at: time.Time
}
```

**消费者：** `notification-service` → `NotificationCreatedHandler`

**处理流程（notification-service/internal/mqhandler/notification_created_handler.go）：**
- 提取 trace_id 并注入 context
- Redis 去重（避免重复消费）
- 1. 插入通知到 `notifications` 表
- 2. 调用 `NotificationSender.SendNotification` 发送通知
  - 支持多种渠道：EMAIL、PUSH、SMS、WEBHOOK
  - 当前为模拟实现（sleep 100ms）
- 3. 根据发送结果在事务中写入 `outbox_events`：
  - 成功：写入 `notification.sent`
  - 失败：写入 `notification.failed`（包含错误信息）
- Outbox Dispatcher 自动发送到 MQ

---

#### 10. notification.sent（通知发送成功事件）

**发布者：** `notification-service` (NotificationSender，使用 Outbox 模式)  
**路由键：** `notification.sent`  
**队列：** `notification.sent.q`（可选，用于监控和分析）

**发布方式：**
- 在事务中写入 `outbox_events` 表
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `NotificationSentPayload`
```go
{
    notification_id: int
    user_id: int
    channel: string
    sent_at: time.Time
}
```

**消费者：** 无（可用于监控、分析、审计）

---

#### 11. notification.failed（通知发送失败事件）

**发布者：** `notification-service` (NotificationSender，使用 Outbox 模式)  
**路由键：** `notification.failed`  
**队列：** `notification.failed.q`（可选，用于重试和监控）

**发布方式：**
- 在事务中写入 `outbox_events` 表
- Outbox Dispatcher 自动发送到 MQ

**Payload：** `NotificationFailedPayload`
```go
{
    notification_id: int
    user_id: int
    channel: string
    error: string
    retry_count: int
}
```

**消费者：** 无（可用于重试机制、监控、告警）

---

## 🔌 API 端点

### API Gateway 端点

#### 公开端点
- `POST /register` - 用户注册
- `POST /login` - 用户登录

#### 需要认证的端点（JWT Token）
- `POST /email/simulate` - 模拟接收邮件
- `GET /emails` - 查询用户邮件列表
- `GET /tasks` - 获取用户任务列表（代理到 task-service）
- `POST /tasks/:id/complete` - 完成任务（代理到 task-service）
- `POST /tasks/from-text` - 文本转任务（调用 agent-service + Outbox 发布 MQ）
- `POST /tasks/plan-project` - 项目规划（调用 agent-service + Outbox 发布 MQ）

#### Admin 端点（需要认证）
- `POST /admin/outbox/replay?id=xxx` - 重放指定的 Outbox 事件
- `POST /admin/outbox/replay-failed?limit=100` - 重放所有失败的事件

#### 健康检查
- `GET /healthz` - Liveness 检查
- `GET /readyz` - Readiness 检查（检查 DB）

### Task Service 端点
- `GET /tasks?user_id=xxx` - 获取用户任务列表
- `POST /tasks/:id/complete` - 完成任务
- `GET /healthz` - Liveness 检查
- `GET /readyz` - Readiness 检查（检查 DB 和 MQ）

### Agent Service 端点
- `POST /decide` - 邮件决策（返回分类、优先级、是否创建任务等）
- `POST /text-to-tasks` - 文本转任务（返回任务列表和习惯列表）
- `POST /plan-project` - 项目规划（返回项目结构：阶段和任务）
- `GET /health` - 健康检查

---

## 🔄 完整事件流示例

### 示例 1：邮件处理流程

```
1. 用户请求：POST /email/simulate
   └─> API Gateway → Mail Ingestion Service

2. Mail Ingestion Service：
   ├─> 事务开始
   ├─> 保存邮件到 emails_raw
   ├─> 写入 outbox_events（3个路由键：agent, log, notify）
   └─> 事务提交
   └─> Outbox Dispatcher 自动发送事件到 MQ

3. Email Processor Service 处理：
   ├─> email.received.agent → AgentDecisionHandler（email-processor-service/internal/mqhandler/agent_handler.go）
   │   ├─> 幂等性检查：如果已 classified，跳过
   │   ├─> Redis 去重（避免并发重复消费）
   │   ├─> 调用 agent-service /decide（带熔断器和 fallback）
   │   │   ├─> 熔断器：失败阈值 3，超时 30 秒
   │   │   ├─> Fallback：返回默认决策（不创建任务、不发送通知）
   │   │   └─> 记录 agent_call_latency_ms 指标
   │   ├─> 事务开始
   │   ├─> 保存元数据到 emails_metadata（InsertDecisionTx）
   │   ├─> 如果 should_create_task → 写入 outbox_events (task.created)
   │   │   └─> aggregate_type="task", aggregate_id=emailID
   │   ├─> 如果 should_notify → 写入 outbox_events (notification.created)
   │   │   └─> aggregate_type="email", aggregate_id=emailID
   │   ├─> 更新邮件状态为 'classified'（UpdateStatusTx）
   │   └─> 事务提交
   │   └─> 记录 metrics（IncrementEmailProcessed, IncrementTaskGeneration）
   │   └─> Outbox Dispatcher 自动发送事件到 MQ
   │
   ├─> email.received.log → NotificationLogHandler
   │   └─> 记录日志到 notifications_log
   │
   └─> email.received.notify → NotificationHandler
       └─> 发布 notification.created 事件

4. Task Service 处理：
   └─> task.created → TaskCreatedHandler
       └─> 创建任务到 tasks 表

5. Notification Service 处理：
   └─> notification.created → NotificationCreatedHandler
       ├─> 插入通知到 notifications 表
       ├─> 发送通知（EMAIL/PUSH/SMS/WEBHOOK）
       └─> 发布 notification.sent 或 notification.failed 事件
```

### 示例 2：文本转任务流程

```
1. 用户请求：POST /tasks/from-text
   Body: { "text": "我每周三跑步，每天读书" }
   └─> API Gateway → TaskController.CreateTasksFromText

2. API Gateway：
   ├─> 调用 agent-service /text-to-tasks（带熔断器）
   │   └─> 返回：{ tasks: [...], habits: [...] }
   │
   ├─> 事务开始
   ├─> 写入 outbox_events (habit.created) - 每个习惯
   ├─> 写入 outbox_events (task.bulk_created) - 如果有任务
   └─> 事务提交
   └─> Outbox Dispatcher 自动发送事件到 MQ
   │
   └─> Task Service 处理：
       ├─> habit.created → HabitCreatedHandler → 保存习惯
       └─> task.bulk_created → TaskBulkCreatedHandler → 批量创建任务

3. Task Service：
   └─> habit.created → HabitCreatedHandler → 保存习惯

4. Task Runner Service 定时任务（每天00:00）：
   └─> Orchestrator.GenerateHabitTasks()
       ├─> 扫描所有活动习惯
       ├─> 检查今天是否应该生成任务
       └─> 发布 habit.task.generated 事件

5. Task Service 处理：
   └─> habit.task.generated → HabitTaskGeneratedHandler
       └─> 插入任务到 tasks 表（幂等性保证）
```

### 示例 3：项目规划流程

```
1. 用户请求：POST /tasks/plan-project
   Body: { "text": "I want to launch a personal blog in 2 weeks." }
   └─> API Gateway → TaskController.PlanProject

2. API Gateway（api-gateway/internal/handler/task_controller.go）：
   ├─> 调用 agent-service /plan-project（带熔断器）
   │   ├─> 熔断器：失败阈值 3，超时 30 秒
   │   └─> 返回项目结构：
   │       {
   │         project: {
   │           title: "Launch Personal Blog",
   │           milestones: [
   │             {
   │               title: "Phase 1: Setup",
   │               order: 1,
   │               tasks: [...]
   │             }
   │           ]
   │         }
   │       }
   │
   ├─> **已使用 Outbox 模式**
   ├─> RBAC 验证：确保 user_id 匹配 token
   ├─> 事务开始
   ├─> 写入 outbox_events (project.created)
   │   └─> aggregate_type="project", aggregate_id=nil
   └─> 事务提交
   └─> Outbox Dispatcher 自动发送事件到 MQ

3. Task Service：
   └─> ProjectCreatedHandler
       ├─> 创建项目到 projects 表
       ├─> 创建阶段到 milestones 表
       ├─> 创建任务到 tasks 表
       └─> 创建依赖关系到 task_dependencies 表

4. Task Runner Service 定时检查（每1分钟）：
   └─> Orchestrator.CheckAndUnlockTasks()
       ├─> 扫描有依赖的任务
       ├─> 检查依赖是否完成
       └─> 如果完成 → 发布 task.unlocked 事件
```

### 示例 4：任务编排流程

```
1. Task Runner Service 定时任务（每1分钟，task-runner-service/internal/service/orchestrator.go）：
   ├─> Orchestrator.CheckAndMarkOverdue()
   │   ├─> 扫描过期的 pending 任务（ListExpiredPendingTasks）
   │   ├─> 事务开始
   │   ├─> 标记为 overdue（MarkExpiredTx）
   │   ├─> 写入 outbox_events (task.overdue) - 每个任务
   │   │   └─> aggregate_type="task", aggregate_id=taskID
   │   └─> 事务提交
   │   └─> Outbox Dispatcher 自动发送事件到 MQ
   │
   └─> Orchestrator.CheckAndUnlockTasks()
       ├─> 扫描有依赖的任务（ListTasksWithDependencies）
       ├─> 检查依赖是否完成（CompletedDepCount == DepCount）
       ├─> 事务开始
       ├─> 写入 outbox_events (task.unlocked) - 已解锁的任务
       │   └─> aggregate_type="task", aggregate_id=taskID
       │   └─> **注意：只写入 outbox，不更新任务状态（由 task-service handler 处理）**
       └─> 事务提交
       └─> Outbox Dispatcher 自动发送事件到 MQ

2. Task Runner Service 定时任务（每天00:00，task-runner-service/internal/service/orchestrator.go）：
   └─> Orchestrator.GenerateHabitTasks()
       ├─> 扫描所有活动习惯（ListAllActive）
       ├─> 检查今天是否应该生成任务（shouldGenerateToday）
       │   ├─> daily：每天生成
       │   ├─> weekly Monday/Tuesday/...：每周指定日期生成
       │   └─> monthly 1/2/...：每月指定日期生成
       ├─> 事务开始
       ├─> 写入 outbox_events (habit.task.generated) - 每个习惯
       │   └─> aggregate_type="habit", aggregate_id=habitID
       │   └─> **注意：只写入 outbox，不创建任务（由 task-service handler 处理）**
       └─> 事务提交
       └─> Outbox Dispatcher 自动发送事件到 MQ

3. Task Service 处理：
   ├─> task.overdue → TaskOverdueHandler（可用于通知、分析）
   ├─> task.unlocked → TaskUnlockedHandler（可用于通知用户）
   └─> habit.task.generated → HabitTaskGeneratedHandler
       └─> 插入任务到 tasks 表（幂等性保证）
```

### 示例 5：通知发送流程

```
1. Email Processor Service 发布：
   └─> notification.created 事件
       Payload: { user_id, email_id, channel: "EMAIL", message }

2. Notification Service 处理（notification-service/internal/mqhandler/notification_created_handler.go）：
   └─> NotificationCreatedHandler
       ├─> 提取 trace_id 并注入 context
       ├─> Redis 去重（避免重复消费）
       ├─> 插入通知到 notifications 表
       ├─> NotificationSender.SendNotification()（notification-service/internal/service/notification_sender.go）
       │   ├─> 根据 channel 发送（EMAIL/PUSH/SMS/WEBHOOK）
       │   │   └─> 当前为模拟实现（sleep 100ms）
       │   ├─> 事务开始
       │   ├─> 如果成功 → 写入 outbox_events (notification.sent)
       │   │   └─> aggregate_type="notification", aggregate_id=notificationID
       │   ├─> 如果失败 → 写入 outbox_events (notification.failed)
       │   │   └─> aggregate_type="notification", aggregate_id=notificationID
       │   │   └─> 包含错误信息（Error 字段）
       │   └─> 事务提交
       │   └─> Outbox Dispatcher 自动发送事件到 MQ
       └─> 支持重试机制（可配置）
```

---

## 🛠️ 技术栈

### Go 服务
- **Web 框架：** Gin
- **数据库：** PostgreSQL (pgxpool)
- **消息队列：** RabbitMQ (amqp091-go)
- **日志：** zap
- **JWT：** 自定义实现
- **可观测性：** Prometheus 指标、OpenTelemetry 全链路追踪（Jaeger）
- **可靠性：** Outbox 模式、熔断器（Circuit Breaker）
- **安全：** RBAC（基于角色的访问控制）

### Python 服务
- **Web 框架：** FastAPI
- **AI：** OpenAI API (gpt-4o-mini)
- **数据验证：** Pydantic

### 基础设施
- **数据库：** PostgreSQL
- **消息队列：** RabbitMQ
- **缓存：** Redis（用于去重和重试计数）
- **容器化：** Docker + Docker Compose
- **可观测性：** OpenTelemetry Collector + Jaeger（分布式追踪）

---

## 📊 数据关系图

```
users
  ├─> emails_raw (1:N)
  │     ├─> emails_metadata (1:1)
  │     ├─> notifications (1:N)
  │     ├─> notifications_log (1:N)
  │     └─> failed_events (1:N)
  │
  ├─> habits (1:N)
  │     └─> tasks (1:N, via habit_id)
  │
  ├─> projects (1:N)
  │     └─> milestones (1:N)
  │           └─> tasks (1:N, via milestone_id)
  │
  └─> tasks (1:N)
        ├─> task_dependencies (N:M, 自关联)
        └─> task_dependencies (N:M, 自关联)
```

---

## ⏰ 定时任务

### Task Runner Service 定时任务（任务编排引擎）

#### 1. 任务过期检查器
- **频率：** 每 1 分钟运行一次
- **功能：** 扫描过期的 pending 任务，标记为 overdue，使用 Outbox 发布 `task.overdue` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(1 * time.Minute)`
- **方法：** `Orchestrator.CheckAndMarkOverdue()`（使用事务 + Outbox）

#### 2. 任务依赖解锁检查器
- **频率：** 每 1 分钟运行一次（与过期检查一起运行）
- **功能：** 检查有依赖的任务，如果所有依赖已完成，使用 Outbox 发布 `task.unlocked` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(1 * time.Minute)`
- **方法：** `Orchestrator.CheckAndUnlockTasks()`（使用事务 + Outbox）

#### 3. 习惯任务生成器
- **频率：** 每天凌晨 00:00 运行一次
- **功能：** 为所有活动习惯生成当天的任务，使用 Outbox 发布 `habit.task.generated` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(24 * time.Hour)`
- **方法：** `Orchestrator.GenerateHabitTasks()`（使用事务 + Outbox）
- **幂等性：** task-service 的 handler 使用唯一索引保证同一天只生成一次

**注意：** 任务编排逻辑已从 `task-service` 迁移到 `task-runner-service`，实现关注点分离。所有事件发布都使用 Outbox 模式确保可靠性。

---

## 🔐 安全与幂等性

### 幂等性保证
1. **任务创建：**
   - 同一 email_id + user_id 只能有一个 pending 任务（唯一索引）
   - 同一 habit_id + due_date 只能有一个 pending 任务（唯一索引）

2. **习惯任务生成：**
   - 使用唯一索引 + `ON CONFLICT DO NOTHING` 避免重复生成

3. **MQ 消息处理：**
   - Redis 去重机制（Deduper）
   - 重试计数（RetryCounter）

### 数据完整性保证
1. **email_id 外键约束处理：**
   - 当 `email_id > 0` 时，必须存在对应的 `emails_raw.id`（外键约束）
   - 当 `email_id = 0` 或未设置时，插入 NULL，避免外键冲突
   - `Insert` 方法：当 `email_id <= 0` 时自动插入 NULL
   - `BulkInsert` 方法：当 `email_id <= 0` 时自动插入 NULL
   - `InsertFromHabit` 方法：不包含 `email_id` 字段，自动插入 NULL
   - `InsertFromProject` 方法：不包含 `email_id` 字段，自动插入 NULL
   - `ListByUser` 方法：使用 `sql.NullInt32` 正确读取 NULL 值

2. **任务来源验证：**
   - `task.created` 事件必须包含有效的 `email_id > 0`（`TaskCreatedHandler` 验证）
   - 文本转任务、习惯任务、项目任务的 `email_id` 为 NULL，符合业务逻辑

### 认证授权
- JWT Token 认证
- 所有任务相关操作都需要 user_id（从 JWT 中提取）
- RBAC：敏感操作需要权限验证（project.created, habit.created, task.bulk_created）

### 事件发布可靠性（Outbox 模式）

**问题：** 直接发布 MQ 事件存在双写不一致问题（业务数据写入成功，但 MQ 发布失败）

**解决方案：** Outbox 模式
1. **事务写入：** 业务数据和事件在同一事务中写入 `outbox_events` 表
2. **后台发送：** Outbox Dispatcher 每秒扫描待处理事件并自动发送
3. **自动重试：** 发送失败时自动重试（最多 5 次，指数退避）
4. **手动重放：** 通过 Replay API 手动重放失败事件

**优势：**
- ✅ 事务一致性：业务数据和事件保证一致
- ✅ 可靠性：MQ 发布失败不影响业务数据
- ✅ 可追溯：所有事件都有 trace_id
- ✅ 可恢复：失败事件可以手动重放

**使用 Outbox 的服务：**
- mail-ingestion-service：`email.received.*` 事件
- email-processor-service：`task.created`、`notification.created` 事件
- api-gateway：`habit.created`、`task.bulk_created`、`project.created` 事件
- task-runner-service：`task.overdue`、`task.unlocked`、`habit.task.generated` 事件
- notification-service：`notification.sent`、`notification.failed` 事件

---

## 🌐 服务端口配置

| 服务 | 端口 | 说明 |
|------|------|------|
| api-gateway | 8080 | API 网关 |
| mail-ingestion-service | 8081 | 邮件接收服务 |
| task-service | 8082 | 任务管理服务 |
| task-runner-service | 8084 | 任务编排引擎 |
| notification-service | 8085 | 通知服务 |
| email-processor-service | 8083 | 邮件处理服务 |
| agent-service | 8000 | AI 代理服务（Python） |
| postgres | 5432 | PostgreSQL 数据库 |
| rabbitmq | 5672 | RabbitMQ 消息队列 |
| redis | 6379 | Redis 缓存 |
| otel-collector | 4317 | OpenTelemetry Collector (gRPC) |
| otel-collector | 4318 | OpenTelemetry Collector (HTTP) |
| jaeger | 16686 | Jaeger UI |

---

## 📝 总结

这是一个基于微服务架构的智能任务管理系统，支持：
- ✅ 邮件智能处理和任务创建
- ✅ 文本转任务（一次性任务）
- ✅ 习惯追踪（重复任务）
- ✅ 项目规划（多级任务结构）
- ✅ 任务依赖管理
- ✅ 优先级管理
- ✅ 通知系统
- ✅ **Outbox 模式**：可靠的事件发布，保证事务一致性
- ✅ **熔断器**：防止级联故障，agent-service 失败不影响其他服务
- ✅ **RBAC**：基于角色的访问控制，防止越权操作
- ✅ **OpenTelemetry 全链路追踪**：跨服务 Trace，支持 Jaeger 可视化
- ✅ **Prometheus 指标**：完整的可观测性
- ✅ **慢查询监控**：自动检测 >100ms 的 SQL 查询，记录警告日志和 Prometheus 指标

所有服务通过 RabbitMQ 异步通信，使用 PostgreSQL 持久化数据，Redis 提供去重和重试计数功能。

### 核心架构特性

1. **可靠事件发布（Outbox 模式）：**
   - 所有事件发布都使用 Outbox 模式，确保事务一致性
   - 后台 Dispatcher 自动发送待处理事件
   - 支持自动重试和手动重放

2. **容错机制：**
   - 熔断器：快速失败，防止级联故障
   - Fallback：agent-service 失败时返回默认决策，确保服务继续运行

3. **安全与权限：**
   - RBAC：敏感操作需要权限验证
   - user_id 匹配验证：防止越权操作

4. **可观测性：**
   - OpenTelemetry 全链路追踪：API → Mail Ingestion → MQ → Email Processor → Task Service → DB
   - Jaeger 可视化：支持跨服务 Trace，延迟可在 Jaeger UI 中查看
   - Prometheus 指标：HTTP、MQ、Agent、DB 延迟统计、慢查询计数
   - 慢查询监控：使用 pgx Tracer 自动检测 >100ms 的 SQL 查询，记录警告日志和 `db_slow_query_total` 指标
   - 结构化日志：自动注入 trace_id（向后兼容）

### 核心功能模块

1. **邮件处理流程：** 邮件接收（Outbox）→ AI 分类（带熔断器+fallback）→ 自动创建任务/通知（Outbox）
2. **文本转任务：** 自然语言输入 → LLM 解析（带熔断器）→ 批量创建任务（Outbox）
3. **习惯追踪：** 习惯定义（Outbox）→ 定时生成重复任务（Outbox）
4. **项目规划：** 项目目标 → 多阶段拆分（Outbox）→ 任务依赖管理
5. **任务编排引擎：** 定时扫描 → 逾期标记（Outbox）→ 依赖解锁（Outbox）→ 习惯生成（Outbox）
6. **通知服务：** 多通道通知 → 发送结果（Outbox）→ 重试机制 → Webhook 支持

### 服务职责分离

- **task-service：** 任务 CRUD 操作，事件消费（不包含定时任务逻辑）
- **task-runner-service：** 任务编排引擎（定时扫描、逾期检查、依赖解锁、习惯生成），使用 Outbox 发布事件
- **notification-service：** 通知发送（EMAIL/PUSH/SMS/WEBHOOK），使用 Outbox 发布事件
- **email-processor-service：** AI 决策处理，使用 Outbox 发布事件（task.created 和 notification.created）
- **mail-ingestion-service：** 邮件接收，使用 Outbox 发布 email.received 事件
- **api-gateway：** API 网关，使用 Outbox 发布事件（habit.created、task.bulk_created、project.created）

### Outbox 模式实现

**每个服务的 Outbox 表：**
- 每个服务在各自的数据库中创建 `outbox_events` 表（服务自治）
- 运行 migration `002_add_outbox.sql` 创建表结构

**Outbox Dispatcher：**
- 每个服务启动时自动启动 Dispatcher
- 每秒扫描一次待处理事件
- 自动重试失败事件（最多 5 次，指数退避）

**Replay API：**
- `POST /admin/outbox/replay?id=xxx` - 手动重放指定事件
- `POST /admin/outbox/replay-failed?limit=100` - 重放所有失败事件

### 已移除的组件

- **task-service/internal/service/habit_generator.go：** 已迁移到 task-runner-service
- **email-processor-service/internal/repository/task_repo.go：** 已移除（任务创建改为事件驱动）
- **email-processor-service/internal/repository/notification_repo.go：** 已移除（通知创建改为事件驱动）
- **各服务的 config.yaml：** 已迁移到统一配置中心（config/base.yaml, local.yaml, production.yaml, docker.yaml）
- **直接 MQ 发布：** 所有事件发布已改为 Outbox 模式，确保事务一致性

### 新增的组件

- **pkg/outbox/：** Outbox 模式实现（Repository、Dispatcher、Replay）
- **pkg/circuitbreaker/：** 熔断器实现
- **pkg/rbac/：** RBAC 权限控制
- **pkg/trace/：** 分布式追踪（Trace ID，向后兼容）
- **pkg/otel/：** OpenTelemetry 全链路追踪（HTTP、MQ、DB）
- **pkg/metrics/：** Prometheus 指标
- **pkg/config/：** 统一配置中心
- **migrations/002_add_outbox.sql：** Outbox 表结构迁移
- **api-gateway/internal/handler/admin_handler.go：** Replay API 处理器
- **config/otel-collector-config.yaml：** OpenTelemetry Collector 配置
- **docker-compose.yml：** 添加 otel-collector 和 Jaeger 服务

### OpenTelemetry 全链路追踪

**架构：**
- 所有 Go 服务通过 OpenTelemetry SDK 发送 trace 数据到 otel-collector
- otel-collector 接收 trace 数据并转发到 Jaeger
- Jaeger 提供 UI 界面（http://localhost:16686）查看完整的调用链路

**自动追踪：**
- ✅ HTTP 请求：Gin 中间件自动创建 span
- ✅ MQ 发布：Publisher 自动创建 Producer span
- ✅ MQ 消费：Consumer 自动创建 Consumer span，支持跨服务传播
- ✅ PostgreSQL 查询：支持数据库操作追踪（可选）

**效果：**
- 支持跨服务 Trace，将一次任务创建链路从 API → Mail Ingestion → MQ → Email Processor → Task Service → DB 全流程串起来
- 延迟可在 Jaeger 中可视化，方便性能分析和问题排查
- 所有服务使用统一的 trace context，支持 W3C Trace Context 标准

### 慢查询监控

**实现方式：**
- 使用 pgx v5 的 `QueryTracer` 接口实现慢查询监控
- 自动拦截所有数据库查询，检测执行时间超过 100ms 的查询

**功能特性：**
- ✅ 自动检测：所有通过 `pgxpool.Pool` 执行的查询都会被监控
- ✅ 警告日志：超过阈值的查询会记录 `WARN slow-query` 日志，包含 SQL 和耗时
- ✅ Prometheus 指标：记录 `db_slow_query_total` 计数器，标签包含 SQL 语句（截断到 200 字符）
- ✅ 可配置阈值：默认 100ms，可通过 `NewSlowQueryTracer` 自定义

**日志格式：**
```
WARN slow-query: SELECT * FROM tasks WHERE user_id = $1 took=189ms
```

**Prometheus 指标：**
- `db_slow_query_total{sql="SELECT * FROM tasks..."}` - 慢查询总数

**使用方式：**
- 已在 `pkg/db/db.go` 中自动集成，所有使用 `db.NewConnection` 创建的连接池都会自动启用慢查询监控
- 无需额外配置，开箱即用


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
│   ├── mq/                   # MQ 连接（Publisher/Consumer）
│   ├── logger/               # 日志工具
│   ├── redis/                # Redis 客户端
│   └── util/                 # 工具函数（JWT, 密码, 去重, 重试计数）
│
└── migrations/               # 数据库迁移
    └── 001_init_schema.sql
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
- **来自邮件：** `email_id` 不为 NULL，`habit_id` 和 `project_id` 为 NULL
- **来自习惯：** `habit_id` 不为 NULL，`email_id` 和 `project_id` 为 NULL
- **来自项目：** `project_id` 和 `milestone_id` 不为 NULL，`email_id` 和 `habit_id` 为 NULL
- **来自文本：** `email_id`、`habit_id`、`project_id` 都为 NULL（一次性任务）

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

---

## 🔄 MQ 事件交互逻辑

### MQ 路由键和队列总览

| 路由键 | 队列名 | 发布者 | 消费者 | 说明 |
|--------|--------|--------|--------|------|
| `email.received.agent` | `email.received.agent.q` | mail-ingestion-service | email-processor-service | AI 决策处理 |
| `email.received.log` | `email.received.log.q` | mail-ingestion-service | email-processor-service | 通知日志记录 |
| `email.received.notify` | `email.received.notify.q` | mail-ingestion-service | email-processor-service | 通知创建 |
| `task.created` | `task.created.q` | email-processor-service | task-service | 单个任务创建（来自邮件） |
| `task.bulk_created` | `task.bulk_created.q` | api-gateway | task-service | 批量任务创建（来自文本） |
| `habit.created` | `habit.created.q` | api-gateway | task-service | 习惯创建 |
| `project.created` | `project.created.q` | api-gateway | task-service | 项目创建 |
| `task.overdue` | `task.overdue.q` | task-runner-service | task-service | 任务逾期 |
| `task.unlocked` | `task.unlocked.q` | task-runner-service | task-service | 任务解锁（依赖完成） |
| `habit.task.generated` | `habit.task.generated.q` | task-runner-service | task-service | 习惯任务生成 |
| `notification.created` | `notification.created.q` | email-processor-service | notification-service | 通知创建 |
| `notification.sent` | `notification.sent.q` | notification-service | - | 通知发送成功 |
| `notification.failed` | `notification.failed.q` | notification-service | - | 通知发送失败 |

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

**发布者：** `mail-ingestion-service`  
**路由键：**
- `email.received.agent` - AI 决策处理
- `email.received.log` - 日志记录
- `email.received.notify` - 通知处理

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
1. **Agent Handler：**
   - 调用 `agent-service` 进行 AI 决策
   - 保存邮件元数据到 `emails_metadata`
   - 如果 `should_create_task`，发布 `task.created` 事件
   - 如果 `should_notify`，发布 `notification.created` 事件
   - 更新邮件状态为 'classified'

2. **Log Handler：**
   - 记录通知日志到 `notifications_log`

3. **Notify Handler：**
   - 发布 `notification.created` 事件（由 notification-service 处理）

---

#### 2. task.created（单个任务创建事件）

**发布者：** `email-processor-service` (AgentDecisionHandler)  
**路由键：** `task.created`  
**队列：** `task.created.q`

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

**处理流程：**
- 插入任务到 `tasks` 表
- 关联 `email_id` 和 `user_id`
- 计算 `due_date = now + due_in_days`

---

#### 3. task.bulk_created（批量任务创建事件）

**发布者：** `api-gateway` (TaskController.CreateTasksFromText)  
**路由键：** `task.bulk_created`  
**队列：** `task.bulk_created.q`

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

**处理流程：**
- 使用事务批量插入任务
- `email_id` 为 NULL（文本转任务没有关联邮件）

---

#### 4. habit.created（习惯创建事件）

**发布者：** `api-gateway` (TaskController.CreateTasksFromText)  
**路由键：** `habit.created`  
**队列：** `habit.created.q`

**Payload：** `HabitCreatedPayload`
```go
{
    user_id: int
    title: string
    recurrence_pattern: string  // "weekly Wednesday", "daily", "monthly 1"
}
```

**消费者：** `task-service` → `HabitCreatedHandler`

**处理流程：**
- 插入习惯到 `habits` 表
- `is_active = TRUE`

**后续处理：**
- `task-runner-service` 的 `Orchestrator` 每天凌晨 00:00 自动生成当天的习惯任务
- 发布 `habit.task.generated` 事件，由 `task-service` 处理
- 使用唯一索引保证幂等性（同一习惯同一天只生成一次）
- 定时任务在 `task-runner-service/cmd/main.go` 中实现，使用 `time.Ticker` 每 24 小时运行一次

---

#### 5. project.created（项目创建事件）

**发布者：** `api-gateway` (TaskController.PlanProject)  
**路由键：** `project.created`  
**队列：** `project.created.q`

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

**处理流程：**
1. 创建项目到 `projects` 表
2. 为每个 milestone 创建阶段到 `milestones` 表
3. 为每个任务创建任务到 `tasks` 表（关联 `project_id` 和 `milestone_id`）
4. 解析任务依赖关系，创建 `task_dependencies` 记录

---

#### 6. task.overdue（任务逾期事件）

**发布者：** `task-runner-service` (Orchestrator.CheckAndMarkOverdue)  
**路由键：** `task.overdue`  
**队列：** `task.overdue.q`

**Payload：** `TaskOverduePayload`
```go
{
    task_id: int
}
```

**消费者：** `task-service` → `TaskOverdueHandler`

**处理流程：**
- 任务已在数据库中标记为 overdue
- Handler 可用于额外处理（如通知、分析等）

---

#### 7. task.unlocked（任务解锁事件）

**发布者：** `task-runner-service` (Orchestrator.CheckAndUnlockTasks)  
**路由键：** `task.unlocked`  
**队列：** `task.unlocked.q`

**Payload：** `TaskUnlockedPayload`
```go
{
    task_id: int
    user_id: int
    title: string
}
```

**消费者：** `task-service` → `TaskUnlockedHandler`

**处理流程：**
- 任务的所有依赖已完成，任务已解锁
- Handler 可用于额外处理（如通知用户、分析等）

---

#### 8. habit.task.generated（习惯任务生成事件）

**发布者：** `task-runner-service` (Orchestrator.GenerateHabitTasks)  
**路由键：** `habit.task.generated`  
**队列：** `habit.task.generated.q`

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

**处理流程：**
- 插入任务到 `tasks` 表（关联 `habit_id`）
- 使用唯一索引保证幂等性（同一习惯同一天只生成一次）

---

#### 9. notification.created（通知创建事件）

**发布者：** `email-processor-service` (AgentDecisionHandler, EmailReceivedNotificationHandler)  
**路由键：** `notification.created`  
**队列：** `notification.created.q`

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

**处理流程：**
1. 插入通知到 `notifications` 表
2. 调用 `NotificationSender` 发送通知
3. 根据发送结果发布 `notification.sent` 或 `notification.failed` 事件

---

#### 10. notification.sent（通知发送成功事件）

**发布者：** `notification-service` (NotificationSender)  
**路由键：** `notification.sent`  
**队列：** `notification.sent.q`（可选，用于监控和分析）

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

**发布者：** `notification-service` (NotificationSender)  
**路由键：** `notification.failed`  
**队列：** `notification.failed.q`（可选，用于重试和监控）

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
- `POST /tasks/from-text` - 文本转任务（调用 agent-service + 发布 MQ）
- `POST /tasks/plan-project` - 项目规划（调用 agent-service + 发布 MQ）

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
   ├─> 保存邮件到 emails_raw
   └─> 发布 email.received 事件（3个路由键）

3. Email Processor Service 处理：
   ├─> email.received.agent → AgentDecisionHandler
   │   ├─> 调用 agent-service /decide
   │   ├─> 保存元数据到 emails_metadata
   │   ├─> 如果 should_create_task → 发布 task.created
   │   └─> 如果 should_notify → 发布 notification.created
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
   ├─> 调用 agent-service /text-to-tasks
   │   └─> 返回：{ tasks: [...], habits: [...] }
   │
   ├─> 发布 habit.created 事件（每个习惯）
   │   └─> Task Service → HabitCreatedHandler → 保存习惯
   │
   └─> 发布 task.bulk_created 事件（如果有任务）
       └─> Task Service → TaskBulkCreatedHandler → 批量创建任务

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

2. API Gateway：
   ├─> 调用 agent-service /plan-project
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
   └─> 发布 project.created 事件

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
1. Task Runner Service 定时任务（每1分钟）：
   ├─> Orchestrator.CheckAndMarkOverdue()
   │   ├─> 扫描过期的 pending 任务
   │   ├─> 标记为 overdue
   │   └─> 发布 task.overdue 事件
   │
   └─> Orchestrator.CheckAndUnlockTasks()
       ├─> 扫描有依赖的任务
       ├─> 检查依赖是否完成
       └─> 如果完成 → 发布 task.unlocked 事件

2. Task Runner Service 定时任务（每天00:00）：
   └─> Orchestrator.GenerateHabitTasks()
       ├─> 扫描所有活动习惯
       ├─> 检查今天是否应该生成任务
       └─> 发布 habit.task.generated 事件

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

2. Notification Service 处理：
   └─> NotificationCreatedHandler
       ├─> 插入通知到 notifications 表
       ├─> NotificationSender.SendNotification()
       │   ├─> 根据 channel 发送（EMAIL/PUSH/SMS/WEBHOOK）
       │   ├─> 如果成功 → 发布 notification.sent
       │   └─> 如果失败 → 发布 notification.failed
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

### Python 服务
- **Web 框架：** FastAPI
- **AI：** OpenAI API (gpt-4o-mini)
- **数据验证：** Pydantic

### 基础设施
- **数据库：** PostgreSQL
- **消息队列：** RabbitMQ
- **缓存：** Redis（用于去重和重试计数）
- **容器化：** Docker + Docker Compose

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
- **功能：** 扫描过期的 pending 任务，标记为 overdue，发布 `task.overdue` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(1 * time.Minute)`
- **方法：** `Orchestrator.CheckAndMarkOverdue()`

#### 2. 任务依赖解锁检查器
- **频率：** 每 1 分钟运行一次（与过期检查一起运行）
- **功能：** 检查有依赖的任务，如果所有依赖已完成，发布 `task.unlocked` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(1 * time.Minute)`
- **方法：** `Orchestrator.CheckAndUnlockTasks()`

#### 3. 习惯任务生成器
- **频率：** 每天凌晨 00:00 运行一次
- **功能：** 为所有活动习惯生成当天的任务，发布 `habit.task.generated` 事件
- **实现：** `task-runner-service/cmd/main.go` 中的 `time.Ticker(24 * time.Hour)`
- **方法：** `Orchestrator.GenerateHabitTasks()`
- **幂等性：** task-service 的 handler 使用唯一索引保证同一天只生成一次

**注意：** 任务编排逻辑已从 `task-service` 迁移到 `task-runner-service`，实现关注点分离。

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

### 认证授权
- JWT Token 认证
- 所有任务相关操作都需要 user_id（从 JWT 中提取）

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

---

## 📝 总结

这是一个基于微服务架构的智能任务管理系统，支持：
- ✅ 邮件智能分类和任务创建
- ✅ 文本转任务（一次性任务）
- ✅ 习惯追踪（重复任务）
- ✅ 项目规划（多级任务结构）
- ✅ 任务依赖管理
- ✅ 优先级管理
- ✅ 通知系统

所有服务通过 RabbitMQ 异步通信，使用 PostgreSQL 持久化数据，Redis 提供去重和重试计数功能。

### 核心功能模块

1. **邮件处理流程：** 邮件接收 → AI 分类 → 自动创建任务/通知
2. **文本转任务：** 自然语言输入 → LLM 解析 → 批量创建任务
3. **习惯追踪：** 习惯定义 → 定时生成重复任务
4. **项目规划：** 项目目标 → 多阶段拆分 → 任务依赖管理
5. **任务编排引擎：** 定时扫描 → 逾期标记 → 依赖解锁 → 习惯生成
6. **通知服务：** 多通道通知 → 重试机制 → Webhook 支持

### 服务职责分离

- **task-service：** 任务 CRUD 操作，事件消费（不包含定时任务逻辑）
- **task-runner-service：** 任务编排引擎（定时扫描、逾期检查、依赖解锁、习惯生成）
- **notification-service：** 通知发送（EMAIL/PUSH/SMS/WEBHOOK），重试机制
- **email-processor-service：** AI 决策处理，事件发布（发布 task.created 和 notification.created，不再直接操作数据库）

### 已移除的组件

- **task-service/internal/service/habit_generator.go：** 已迁移到 task-runner-service
- **email-processor-service/internal/repository/task_repo.go：** 已移除（任务创建改为事件驱动）
- **email-processor-service/internal/repository/notification_repo.go：** 已移除（通知创建改为事件驱动）


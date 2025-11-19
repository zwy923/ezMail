# 统一配置中心

## 目录结构

```
config/
├── base.yaml              # 基础配置（所有环境共享）
├── local.yaml             # 本地开发环境配置（直接运行，不使用 Docker）
├── docker.yaml            # Docker 环境配置（Docker Compose 中使用）
├── production.yaml        # 生产环境配置
├── secrets.env.example    # 密钥配置示例
└── secrets.env            # 密钥配置（不提交到 Git）
```

## 配置加载顺序

配置按以下优先级加载（后面的覆盖前面的）：

1. **base.yaml** - 基础配置
2. **{env}.yaml** - 环境特定配置（如 local.yaml, production.yaml）
3. **secrets.env** - 密钥配置（覆盖 YAML 中的占位符）
4. **系统环境变量** - 最高优先级

## 使用方法

### 1. 设置环境

通过环境变量 `CONFIG_ENV` 指定环境：

```bash
# 本地开发（直接运行，不使用 Docker）
export CONFIG_ENV=local

# Docker 环境（在 Docker Compose 中运行）
export CONFIG_ENV=docker

# 生产环境
export CONFIG_ENV=production
```

**重要：** 
- **本地开发（直接运行）**：使用 `CONFIG_ENV=local`，服务地址为 `localhost`
- **Docker 环境**：使用 `CONFIG_ENV=docker`，服务地址为 Docker 服务名称（如 `postgres`, `rabbitmq`, `agent-service`）

### 2. 配置密钥

复制示例文件并填入实际值：

```bash
cp config/secrets.env.example config/secrets.env
# 编辑 config/secrets.env，填入实际密钥
```

**重要：** `secrets.env` 已添加到 `.gitignore`，不会被提交到 Git。

### 3. 自定义配置目录

通过环境变量 `CONFIG_DIR` 指定配置目录：

```bash
export CONFIG_DIR=/path/to/config
```

## 环境变量覆盖

所有配置都可以通过环境变量覆盖，优先级最高：

- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`
- `MQ_URL`
- `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB`
- `JWT_SECRET`
- `SERVER_PORT`
- `AGENT_SERVICE_URL`, `TASK_SERVICE_URL`, 等

## 配置示例

### base.yaml
包含所有环境的默认配置，使用空值或占位符。

### local.yaml
本地开发环境配置（直接运行，不使用 Docker），包含开发用的默认值，使用 `localhost` 作为服务地址。

### docker.yaml
Docker 环境配置（在 Docker Compose 中运行），使用 Docker 服务名称作为地址：
- `postgres` 而不是 `localhost`
- `rabbitmq` 而不是 `localhost`
- `agent-service` 而不是 `localhost`
- `mail-ingestion-service` 而不是 `localhost`
- 等等

### production.yaml
生产环境配置，使用环境变量占位符 `${VAR_NAME}`，实际值从 `secrets.env` 或系统环境变量获取。

### secrets.env
包含所有敏感信息：
- 数据库密码
- JWT 密钥
- OpenAI API Key
- 其他密钥

## 迁移说明

所有服务已更新为使用统一配置中心。旧的 `{service}/config.yaml` 文件已被删除。

新的配置加载逻辑会自动：
1. 从 `config/` 目录加载配置
2. 根据 `CONFIG_ENV` 选择环境
3. 应用环境变量覆盖

## Docker 环境使用

在 Docker Compose 中运行时，`docker-compose.yml` 会自动设置 `CONFIG_ENV=docker`，服务会使用 `docker.yaml` 配置，其中：

- **数据库地址**：`postgres`（而不是 `localhost`）
- **消息队列地址**：`rabbitmq:5672`（而不是 `localhost:5672`）
- **Redis 地址**：`redis:6379`（而不是 `localhost:6379`）
- **服务间调用**：使用 Docker 服务名称（如 `http://agent-service:8000`）

**示例：**
```bash
# 启动所有服务（自动使用 docker.yaml）
docker-compose up -d

# 查看服务日志
docker-compose logs -f api-gateway
```

## 注意事项

1. **不要提交 secrets.env** - 已添加到 `.gitignore`
2. **生产环境** - 建议使用系统环境变量或密钥管理服务，而不是文件
3. **配置变更** - 修改配置后需要重启服务才能生效
4. **配置位置** - 所有配置统一在 `config/` 目录管理
5. **Docker vs 本地** - Docker 环境使用服务名称，本地直接运行使用 `localhost`


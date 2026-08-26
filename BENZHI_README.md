# LogAlert - 实时日志聚合与告警规则引擎

## 项目简介

LogAlert 是一个基于 Go 语言的实时日志聚合与告警规则引擎。它接收多个服务发送的日志条目，支持按级别、来源、关键词存储和查询，并能根据自定义规则自动触发告警。

### 核心功能

- **日志接收**：通过 HTTP POST 接收日志条目，支持 INFO/WARN/ERROR/DEBUG/FATAL 五种级别
- **日志查询**：按时间范围、级别、关键词、来源过滤查询日志
- **告警规则管理**：CRUD 操作管理告警规则，支持按级别、数量、关键词等条件触发
- **定时扫描**：定时扫描日志并根据规则触发告警，记录告警事件
- **统计分析**：日志量统计、错误率趋势、按小时分桶聚合

### 技术栈

- 纯 Go 标准库（net/http, sync, encoding/json 等）
- 无第三方依赖
- 端口：8080

---

## 目录结构

```
.
├── cmd/
│   └── server/
│       └── main.go              # 应用入口
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构定义
│   │   ├── loader.go            # 配置加载（文件+环境变量）
│   │   └── validator.go        # 配置校验
│   ├── handler/
│   │   ├── log_handler.go      # 日志相关 HTTP 处理器
│   │   ├── rule_handler.go      # 规则相关 HTTP 处理器
│   │   ├── alert_handler.go    # 告警相关 HTTP 处理器
│   │   ├── stats_handler.go    # 统计相关 HTTP 处理器
│   │   ├── health_handler.go    # 健康检查处理器
│   │   ├── middleware.go        # 中间件（CORS、日志、限流等）
│   │   ├── scheduler_handler.go # 调度器处理器
│   │   ├── config_handler.go    # 配置处理器
│   │   └── metrics_handler.go  # 指标处理器
│   ├── model/
│   │   ├── log_entry.go        # 日志条目模型
│   │   ├── alert_rule.go       # 告警规则模型
│   │   ├── alert_event.go      # 告警事件模型
│   │   ├── enums.go            # 枚举和常量
│   │   ├── requests.go         # 请求模型
│   │   ├── query_params.go     # 查询参数模型
│   │   └── log_query.go        # 日志查询模型
│   ├── service/
│   │   ├── log_service.go      # 日志业务逻辑
│   │   ├── rule_service.go     # 规则业务逻辑
│   │   ├── alert_service.go    # 告警业务逻辑
│   │   ├── stats_service.go    # 统计业务逻辑
│   │   ├── scheduler.go        # 定时扫描调度器
│   │   ├── cleanup_service.go  # 数据清理服务
│   │   └── service_manager.go  # 服务管理器
│   └── store/
│       ├── store.go            # 存储接口定义
│       ├── memory_log_store.go # 内存日志存储
│       ├── memory_rule_store.go# 内存规则存储
│       ├── memory_alert_store.go# 内存告警存储
│       ├── filter_utils.go    # 过滤工具
│       ├── file_persistence.go # 文件持久化
│       ├── backup_log_store.go # 备份日志存储
│       └── retry_log_store.go # 重试日志存储
├── pkg/
│   ├── logger/
│   │   ├── logger.go           # 日志接口和实现
│   │   ├── log_entry.go        # 日志条目结构
│   │   ├── log_writer.go       # 日志写入器
│   │   └── logger_builder.go    # 日志构建器
│   ├── response/
│   │   ├── response.go         # 统一响应格式
│   │   └── error_codes.go      # 错误码定义
│   ├── timeutil/
│   │   ├── clock.go            # 时钟抽象
│   │   └── timewindow.go       # 时间窗口工具
│   ├── jsonutil/
│   │   └── json_io.go          # JSON 读写工具
│   ├── validator/
│   │   └── validator.go        # 参数校验
│   ├── errors/
│   │   ├── app_error.go        # 应用错误类型
│   │   └── wrap.go             # 错误包装
│   ├── cache/
│   │   └── cache.go            # 内存缓存
│   ├── retry/
│   │   └── retry.go            # 重试逻辑
│   └── metrics/
│       └── metrics.go          # 指标收集
├── static/
│   └── index.html              # 前端页面
├── go.mod
├── benzhi.Dockerfile
├── build_benzhi_docker.sh
└── BENZHI_README.md
```

---

## API 文档

### 健康检查

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/health` | 健康检查，返回内存和 goroutine 状态 |
| GET | `/ready` | 就绪检查 |
| GET | `/info` | 服务信息 |

### 日志管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/logs` | 创建一条日志 |
| POST | `/api/logs/batch` | 批量创建日志 |
| GET | `/api/logs` | 查询日志列表 |
| GET | `/api/logs/{id}` | 获取单条日志 |
| DELETE | `/api/logs/{id}` | 删除日志 |
| GET | `/api/logs/sources` | 获取所有来源列表 |
| GET | `/api/logs/services` | 获取所有服务列表 |

**POST /api/logs 请求体：**
```json
{
    "level": "ERROR",
    "source": "order-service",
    "message": "Failed to process payment",
    "service": "orders",
    "timestamp": "2024-01-15T10:30:00Z",
    "tags": {
        "user_id": "123"
    }
}
```

**GET /api/logs 查询参数：**
- `levels`: 日志级别过滤（逗号分隔，如 `INFO,WARN,ERROR`）
- `sources`: 来源过滤（逗号分隔）
- `service`: 服务过滤
- `keywords`: 关键词搜索（逗号分隔）
- `start_time`: 开始时间（RFC3339 格式）
- `end_time`: 结束时间
- `limit`: 返回条数（默认 100）
- `offset`: 偏移量

### 告警规则

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/rules` | 创建告警规则 |
| GET | `/api/rules` | 获取所有规则 |
| GET | `/api/rules/active` | 获取活跃规则 |
| GET | `/api/rules/{id}` | 获取单条规则 |
| PUT | `/api/rules/{id}` | 更新规则 |
| DELETE | `/api/rules/{id}` | 删除规则 |
| PUT | `/api/rules/{id}/status` | 切换规则状态 |

**POST /api/rules 请求体：**
```json
{
    "name": "订单服务错误监控",
    "condition": {
        "type": "count",
        "level": "ERROR",
        "source": "order-service"
    },
    "window": 300000000000,
    "threshold": 5,
    "severity": "high",
    "cooldown": 300000000000
}
```

### 告警事件

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/alerts` | 查询告警列表 |
| GET | `/api/alerts/recent` | 获取最近告警 |
| GET | `/api/alerts/{id}` | 获取单条告警 |
| POST | `/api/alerts/{id}/acknowledge` | 确认告警 |
| POST | `/api/alerts/{id}/resolve` | 解决告警 |
| DELETE | `/api/alerts/{id}` | 删除告警 |

### 统计分析

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/stats` | 获取日志统计 |
| GET | `/api/stats/hourly` | 按小时统计 |
| GET | `/api/stats/error-rate` | 错误率趋势 |
| GET | `/api/stats/sources` | 按来源统计 |
| GET | `/api/stats/levels` | 按级别统计 |

### 调度器

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/scheduler/status` | 调度器状态 |
| POST | `/api/scheduler/scan` | 立即执行规则扫描 |

### 配置

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/config` | 获取当前配置 |
| PUT | `/api/config` | 更新配置 |
| POST | `/api/config/reload` | 重新加载配置 |

### 指标

| 方法 | 路径 | 描述 |
|------|------|------|
| GET | `/api/metrics` | 获取服务指标 |
| POST | `/api/metrics` | 重置指标 |

---

## 本地运行

```bash
# 克隆项目后，在项目根目录执行
go run ./cmd/server

# 默认端口 8080，可通过环境变量配置
SERVER_PORT=9090 go run ./cmd/server

# 使用配置文件
CONFIG_PATH=config.json go run ./cmd/server
```

访问 http://localhost:8080 查看前端页面。

---

## Docker 构建与运行

### 使用评测 Dockerfile

```bash
# 构建镜像
docker build -f benzhi.Dockerfile -t logalert:latest .

# 运行容器
docker run -d --name logalert -p 8080:8080 logalert:latest

# 查看日志
docker logs -f logalert

# 停止并删除容器
docker stop logalert && docker rm logalert
```

### 使用构建脚本

```bash
# 默认构建
./build_benzhi_docker.sh

# 自定义参数
./build_benzhi_docker.sh my-logalert v1.0 linux/amd64

# 运行示例
docker run -d -p 8080:8080 my-logalert:v1.0
```

---

## 测试命令

```bash
# 编译检查
go build ./...

# 静态分析
go vet ./...

# 全量测试（含竞态检测）
go test -race -count=5 ./...

# 仅存储层测试
go test -race -count=10 ./internal/store/

# 仅服务层测试
go test -race -count=5 ./internal/service/

# 手动 API 测试
curl http://localhost:8080/health
curl -X POST http://localhost:8080/api/logs \
    -H 'Content-Type: application/json' \
    -d '{"level":"INFO","source":"test","message":"Hello World"}'
curl http://localhost:8080/api/logs?limit=10
curl -X POST http://localhost:8080/api/rules \
    -H 'Content-Type: application/json' \
    -d '{"name":"Test Rule","condition":{"type":"count","level":"ERROR"},"window":300000000000,"threshold":3,"severity":"medium"}'
curl -X POST http://localhost:8080/api/scheduler/scan
```

---

## 特性

- ✅ **优雅关闭**：监听 SIGINT/SIGTERM 信号，平滑关闭 HTTP 服务和定时任务
- ✅ **健康检查**：`/health`、`/ready` 端点
- ✅ **结构化日志**：支持多种日志级别和结构化字段
- ✅ **参数校验**：请求参数验证和错误处理
- ✅ **统一响应格式**：所有 API 返回统一的 JSON 格式
- ✅ **CORS 支持**：跨域请求支持
- ✅ **限流**：简单的请求限流中间件
- ✅ **前端页面**：内置 Web UI

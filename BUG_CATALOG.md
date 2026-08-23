# BUG_CATALOG.md - 实时日志聚合与告警规则引擎 缺陷候选清单

> 本文档列出了 logalert 项目中可注入的 30 个缺陷候选。
> 每个缺陷跨至少 2 个文件，验证命令包含 `-race` 和 `-count=N` 参数。

## 缺陷统计

| 类别 | 数量 |
|------|------|
| concurrency | 5 |
| nil | 4 |
| slice | 4 |
| error | 5 |
| context | 4 |
| defer | 4 |
| other | 4 |
| **合计** | **30** |

---

## 缺陷清单

| bug_id | bug_category | 缺陷描述 | 植入位置 | 预期表现 | 触发方式 | 缺陷难度 |
|--------|--------------|----------|----------|----------|----------|----------|
| logalert-001 | concurrency | 存储驱逐时数据竞争导致日志丢失 | internal/store/memory_log_store.go.evictOldest, internal/service/scheduler.go.evaluateRule | 高并发下 panic "concurrent map read and map write" 或日志条目丢失 | 向系统以 >500 QPS 发送日志同时执行规则扫描 `go test -race -count=5 ./internal/store/` | ⭐⭐⭐ |
| logalert-002 | concurrency | WaitGroup 计数不匹配导致程序无法正常退出 | cmd/server/main.go.main, internal/service/scheduler.go.Start | 程序关闭时永远挂起，无法完成优雅关闭 | 启动服务后发送 SIGTERM 信号，观察进程不退出 `go build -race -o server ./cmd/server/` | ⭐⭐ |
| logalert-003 | concurrency | channel 未关闭导致 goroutine 泄露 | internal/service/scheduler.go.runLoop, internal/service/cleanup_service.go.runLoop | goroutine 数量持续增长，内存泄漏 | 启动服务运行数小时，观察 goroutine 数量不断增加 `go test -race -count=3 ./internal/service/` | ⭐⭐⭐ |
| logalert-004 | concurrency | 统计计算时未加锁导致读到不一致状态 | internal/store/memory_log_store.go.Statistics, internal/store/filter_utils.go.FilterLogEntries | 统计数据偶发不一致，total_count 与 by_level 总和不符 | 高并发写入同时查询统计接口 `go test -race -count=10 ./internal/store/` | ⭐⭐⭐⭐ |
| logalert-005 | concurrency | 告警记录写入与查询并发导致数据竞争 | internal/store/memory_alert_store.go.Record, internal/service/alert_service.go.RecordAlert | 高并发下 panic 或告警列表偶发为空 | 同时 POST /api/alerts 和触发规则扫描 `go test -race -count=5 ./internal/service/` | ⭐⭐⭐ |
| logalert-006 | nil | 向未初始化的 map 写入导致 panic | internal/model/log_entry.go.ToMap, internal/model/alert_event.go.ToMap | 调用 ToMap 时 panic "assignment to entry in nil map" | 创建 LogEntry/AlertEvent 未初始化 Tags 字段直接调用 ToMap `go test -race -count=5 ./internal/model/` | ⭐⭐ |
| logalert-007 | nil | 规则评估时 nil 指针解引用 | internal/service/scheduler.go.evaluateRule, internal/service/alert_service.go.RecordAlert | 规则触发时 panic "nil pointer dereference" | 创建规则后发送满足条件的日志触发告警 `go test -race -count=3 ./internal/service/` | ⭐⭐⭐ |
| logalert-008 | nil | 接口装 nil 指针与 nil 比较恒为假导致漏记 | internal/handler/log_handler.go.CreateLog, internal/handler/alert_handler.go.GetAlert | nil 错误被忽略，请求返回成功但实际失败 | 向不存在的 source 发送日志，期望看到错误但实际返回成功 `go test -race -count=5 ./internal/handler/` | ⭐⭐⭐⭐ |
| logalert-009 | nil | nil 切片未检查导致索引越界 | internal/model/log_query.go.Validate, internal/model/alert_query.go.Validate | 空查询请求时 panic "slice bounds out of range" | 发送空 JSON 对象作为查询请求 `go test -race -count=5 ./internal/model/` | ⭐⭐⭐ |
| logalert-010 | slice | append 后共享底层数组导致数据污染 | internal/store/memory_log_store.go.Query, internal/handler/log_handler.go.QueryLogs | 分页查询结果异常，第二页数据与第一页重叠 | 发送 >100 条日志然后分页查询 `go test -race -count=5 ./internal/store/` | ⭐⭐⭐⭐ |
| logalert-011 | slice | 子切片写回污染原数组导致统计错误 | internal/service/stats_service.go.GetErrorRateTrend, internal/store/filter_utils.go.FilterAlertEvents | 错误率统计数据偶发异常 | 同时请求统计接口和触发规则扫描 `go test -race -count=5 ./internal/service/` | ⭐⭐⭐⭐ |
| logalert-012 | slice | 容量估算错误导致越界访问 | internal/handler/log_handler.go.QueryLogs, internal/handler/rule_handler.go.ListRules | 查询规则列表时 panic "index out of range" | 创建大量规则后查询列表 `go test -race -count=5 ./internal/handler/` | ⭐⭐⭐ |
| logalert-013 | slice | 未检查空切片导致越界访问 | internal/handler/health_handler.go.HandleHealth, internal/handler/scheduler_handler.go.GetStatus | 空健康检查数据时 panic | 在内存指标为 0 时访问 /health 端点 `go test -race -count=3 ./internal/handler/` | ⭐⭐⭐ |
| logalert-014 | error | 错误包装丢失 %w 导致 errors.Is/As 失效 | pkg/errors/wrap.go.NewDetailedError, pkg/errors/app_error.go.Wrap | errors.Is(err, ErrNotFound) 永远返回 false | 使用 errors.Is 检查包装后的错误类型 `go test -race -count=5 ./pkg/errors/` | ⭐⭐⭐⭐ |
| logalert-015 | error | err 被 := 遮蔽导致原始错误丢失 | cmd/server/main.go.main, internal/config/loader.go.Load | 配置加载失败时返回 nil error，服务静默使用默认配置 | 删除配置文件后启动服务 `go build -race -o server ./cmd/server/` | ⭐⭐⭐ |
| logalert-016 | error | 仅比较错误字符串而非类型判断 | internal/service/log_service.go.CreateLog, internal/service/alert_service.go.AcknowledgeAlert | 自定义错误被当作未知错误处理 | 使用自定义错误类型调用 CreateLog `go test -race -count=5 ./internal/service/` | ⭐⭐⭐ |
| logalert-017 | error | 错误类型在包装器中丢失导致类型断言失败 | pkg/retry/retry.go.Do, pkg/cache/cache.go.GetOrSet | 重试逻辑无法识别可重试错误，持续重试 | 模拟临时网络错误调用 retry.Do `go test -race -count=5 ./pkg/...` | ⭐⭐⭐⭐ |
| logalert-018 | error | 清理操作忽略错误导致数据不一致 | internal/service/cleanup_service.go.CleanupOnce, internal/store/file_persistence.go.SaveState | 清理失败后不重试，数据丢失 | 磁盘空间不足时执行清理操作 `go test -race -count=5 ./internal/service/` | ⭐⭐⭐ |
| logalert-019 | context | 取消信号未传播到下游导致操作超时 | internal/service/scheduler.go.runLoop, internal/handler/middleware.go.TimeoutMiddleware | 客户端断开后服务器仍处理请求，资源浪费 | 客户端断开后检查服务器 goroutine 数 `go test -race -count=3 ./internal/service/` | ⭐⭐⭐⭐ |
| logalert-020 | context | context 存入结构体后复用导致请求间串扰 | internal/service/service_manager.go.RegisterService, internal/service/scheduler.go.evaluateRule | 并发请求使用相同 context，trace_id 混乱 | 并发发送多个带 trace_id 的请求 `go test -race -count=10 ./internal/service/` | ⭐⭐⭐⭐ |
| logalert-021 | context | 忽略 ctx.Err() 导致取消后继续执行 | internal/service/log_service.go.QueryLogs, internal/service/rule_service.go.ListActiveRules | 请求超时后仍然执行数据库查询 | 使用超时 context 调用 QueryLogs `go test -race -count=5 ./internal/service/` | ⭐⭐⭐ |
| logalert-022 | context | 使用 Background context 而非请求 context | internal/handler/health_handler.go.HandleHealth, internal/handler/config_handler.go.GetConfig | 请求取消后健康检查仍执行，可能阻塞 | 客户端断开时检查 /health 处理时间 `go test -race -count=3 ./internal/handler/` | ⭐⭐⭐ |
| logalert-023 | defer | 循环内 defer 直到函数返回才执行导致资源堆积 | internal/service/scheduler.go.ScanOnce, internal/service/cleanup_service.go.CleanupOnce | 每次扫描堆积资源，长时间运行后 OOM | 启动服务运行数小时观察内存增长 `go test -race -count=5 ./internal/service/` | ⭐⭐⭐⭐ |
| logalert-024 | defer | defer 修改命名返回值导致返回值错误 | internal/service/log_service.go.CreateLog, internal/service/rule_service.go.CreateRule | 创建日志/规则返回 nil 但实际已创建 | 调用 CreateLog 检查返回值 `go test -race -count=5 ./internal/service/` | ⭐⭐⭐ |
| logalert-025 | defer | error 分支跳过了资源释放导致资源泄漏 | internal/store/memory_log_store.go.Store, internal/store/memory_alert_store.go.Record | 存储失败时锁未释放，后续所有请求阻塞 | 模拟存储失败后尝试新的写入请求 `go test -race -count=5 ./internal/store/` | ⭐⭐⭐⭐ |
| logalert-026 | defer | defer 释放锁在提前 return 时不执行导致死锁 | internal/store/file_persistence.go.SaveLogs, pkg/cache/cache.go.Set | 并发写文件时死锁，服务无法响应 | 并发调用 SaveLogs `go test -race -count=5 ./internal/store/` | ⭐⭐⭐⭐ |
| logalert-027 | other | 时区不一致导致时间窗口判断错误 | pkg/timeutil/clock.go.Now, pkg/timeutil/timewindow.go.Contains | UTC 与本地时区混用导致日志查询结果不对 | 跨时区查询日志时间范围 `go test -race -count=5 ./pkg/timeutil/` | ⭐⭐⭐ |
| logalert-028 | other | 状态字符串比较大小写不一致 | internal/handler/log_handler.go.CreateLog, internal/handler/alert_handler.go.QueryAlerts | lowercase 状态值被当作非法值 | 发送大小写混合的 level 值 `go test -race -count=5 ./internal/handler/` | ⭐⭐ |
| logalert-029 | other | 配置更新后未重新加载导致变更不生效 | internal/handler/config_handler.go.UpdateConfig, internal/config/config.go.LoadFromFile | PUT /api/config 后新配置不生效 | 通过 API 更新日志级别后发送日志 `go test -race -count=5 ./internal/handler/` | ⭐⭐⭐ |
| logalert-030 | other | 缓存无限增长导致内存泄漏 | pkg/cache/cache.go.Set, pkg/metrics/metrics.go.RecordRequest | 长时间运行后内存持续增长 | 高频请求数小时后观察内存占用 `go test -race -count=5 ./pkg/cache/` | ⭐⭐⭐ |

---

## 验证命令示例

```bash
# 全量测试（含竞态检测）
go test -race -count=5 ./...

# 仅存储层测试
go test -race -count=10 ./internal/store/

# 仅服务层测试
go test -race -count=5 ./internal/service/

# 仅处理器层测试
go test -race -count=5 ./internal/handler/

# 压力测试（需更大的 count 值）
go test -race -count=20 ./internal/store/
```

## 注意事项

1. 所有缺陷均为**运行时缺陷**，项目能通过 `go build ./...` 和 `go vet ./...`
2. 缺陷注入后项目仍能编译通过并正常运行
3. 每个缺陷的触发条件明确且可稳定复现
4. 缺陷之间相互独立，改动一个不影响另一个的触发
5. 单个文件的缺陷数量不超过 9 个（30% 规则）

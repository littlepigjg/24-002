# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
在短链服务中，当客户端请求携带上下文超时或主动取消时，服务端的存储加载操作未能及时响应上下文取消信号，继续执行直至完成。这会导致在高并发或网络不稳定的场景下，服务端积累大量无效的后台操作，占用宝贵的计算资源和协程资源。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块名：logalert
- 关键依赖：无外部依赖，纯标准库项目
- 运行参数：测试时使用 context.WithTimeout 设置超时，默认超时时间 50ms
- 硬件信息：与并发表现相关，建议在 4 核以上 CPU 环境下测试

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 初始化配置对象，调用 `config.DefaultConfig()` 获取默认配置
3. 使用配置创建 URLStore 实例：`store.NewURLStore(cfg)`
4. 创建一个带短超时的 context：`ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)`
5. 等待 context 超时：`time.Sleep(100 * time.Millisecond)`
6. 调用 `s.Load(ctx)` 并检查返回值
7. 观察：Load 方法成功返回 nil（无错误），但期望的行为是返回 context deadline exceeded 错误

## 4. 实际结果（Actual Behavior / Observed Output）
- Load 方法返回 nil（无错误），即使 context 已经超时
- 健康检查接口在客户端断开后仍继续执行存储加载操作
- 配置查询接口在客户端断开后仍继续执行存储加载操作
- 协程数量持续增长，在高并发场景下可能导致资源耗尽

## 5. 期望结果（Expected Behavior）
- 当 context 被取消或超时后，Load 方法应立即返回 context cancelled 错误
- 健康检查接口应在客户端断开后立即停止后续操作
- 配置查询接口应在客户端断开后立即停止后续操作
- 无不必要的后台协程持续运行
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%），只要传入已取消或已超时的 context，必然出现此问题

## 7. 影响范围（Impact / Scope）
- 服务端协程泄漏：每个被取消的请求都会在服务端留下一个继续运行的协程
- 资源浪费：无效的存储加载操作占用 CPU 和内存资源
- 性能下降：大量积压的后台操作拖慢正常请求的响应速度
- 可用性风险：极端情况下可能导致服务内存溢出或协程数超限
- 影响所有涉及 URLStore.Load 的功能，包括健康检查、配置查询等接口

## 8. 附加说明（Additional Notes / Workaround）
目前暂无安全的临时规避方法。如果必须临时处理，可以在调用 Load 之前手动检查 context 状态，但这只是治标不治本，因为问题的根因在于 Load 内部忽略了传入的 context 参数。

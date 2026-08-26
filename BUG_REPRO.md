# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
在日志告警服务中，当客户端请求的 context 超时或被取消后，服务器端的存储层操作（如 Count、Query 等）仍然继续执行并返回成功结果，而不是正确返回 context 取消错误。同时，调度器在 context 取消后仍继续扫描规则和触发告警，中间件在 context 已取消时仍继续将请求传递到下游。这导致客户端断开后服务器仍浪费资源进行无效处理。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go build ./... 确保编译通过
- 硬件信息：CPU（多核）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录：cd /home/admin/code/24/002/24-002-19
2. 确保项目编译通过：go build ./...
3. 使用 logger.NewDiscardWriter() 创建日志实例，构造默认配置初始化系统
4. 向存储层写入日志条目（至少5条）
5. 调用 SetProcessingDelay(100 * time.Millisecond) 设置存储层处理延迟
6. 创建一个带有 5ms 超时的 context
7. 调用 Count 或 Query 方法，传入该短超时 context
8. 观察方法返回结果

## 4. 实际结果（Actual Behavior / Observed Output）
- Count 方法在 context 超时（5ms）后仍成功返回条目数量，例如返回 count=5, err=nil
- Query 方法在 context 超时后仍成功返回完整的日志条目列表
- 存储层的 time.Sleep(100ms) 完整执行完毕，未响应 context 取消
- 调度器在 context 取消后仍继续遍历规则，evaluateRule 在 context 取消后仍触发告警
- 中间件在 context 已取消时仍继续处理请求，未拦截或终止
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）

## 5. 期望结果（Expected Behavior）
- Count 和 Query 在 context 超时（5ms）后应立即返回 context deadline exceeded 错误
- 存储层的处理延迟应使用 select + ctx.Done() 响应式等待，context 取消时立即返回错误
- 调度器 ScanOnce 在 context 取消时应立即中止遍历并返回错误
- 调度器 evaluateRule 在 context 取消时应返回错误且不触发告警
- 中间件 TimeoutMiddleware 在父 context 已取消时应正确拦截并终止处理
- RED/GREEN 判定结果应为 GREEN
- go build 与 go vet 全部通过
- SetProcessingDelay 钩子方法应保持原有功能不变

## 6. 触发频率（Frequency）
必现（100%）：每当存储层设置了处理延迟（通过 SetProcessingDelay）且 context 超时时间短于处理延迟时，必定触发该缺陷。单次调用即可稳定复现。

## 7. 影响范围（Impact / Scope）
- 客户端断开连接后，服务器端仍继续处理请求，造成 CPU 和内存资源浪费
- 存储层操作在 context 取消后仍可能写入数据，导致数据不一致
- 调度器在 context 取消后仍继续触发告警，产生误告警
- 高并发场景下大量无效处理可能导致服务器资源耗尽
- 中间件层未能正确拦截已取消的请求，使得无效请求穿透到下游

## 8. 附加说明（Additional Notes / Workaround）
目前无临时规避方法。该问题影响所有使用 context 超时控制的调用场景，包括但不限于 HTTP 请求超时、调度器扫描超时等。SetProcessingDelay 方法是故障演练/诊断 API，用于模拟存储层处理延迟，应在修复后保持其功能不变。
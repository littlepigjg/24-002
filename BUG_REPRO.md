# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
日志告警服务在长时间运行后，进程内存持续上涨，goroutine 数量不断增加。每次重启服务后内存立即恢复正常，但运行一段时间后问题复现。该问题表现为后台周期性任务的 goroutine 在服务停止后未正确释放，导致资源泄漏，最终可能引发内存溢出或服务不可用。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go version go1.21.x
- 项目模块：logalert
- 运行参数：go test . -count=1 -run '^TestRedGreen$'，go test -race -count=3 . -run '^TestRedGreen$'
- 硬件信息：CPU 4 核及以上

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test . -count=1 -run '^TestRedGreen$'` 运行验证测试
3. 观察测试输出中 "scheduler idle" 和 "cleanup idle" 的值
4. 若输出显示 idle 值为 false，则表明 goroutine 泄漏已触发
5. 可重复执行多次以确认问题稳定性：`go test -race -count=3 . -run '^TestRedGreen$'`

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：
```
=== RUN   TestRedGreen
RED (红灯，缺陷未修复)
scheduler idle: false, cleanup idle: false
FAIL    logalert        1.206s
```
- RED/GREEN 判定结果：RED（红灯）
- scheduler 和 cleanup service 的 AwaitIdle 均在 500ms 超时后返回 false，表明 notifier goroutine 未能在服务停止后正常退出
- go test -race 未报告 DATA RACE（该缺陷为 goroutine 泄漏，非数据竞争）
- 退出码：1

## 5. 期望结果（Expected Behavior）
- 测试输出：
```
=== RUN   TestRedGreen
GREEN (绿灯，缺陷已修复)
scheduler idle: true, cleanup idle: true
ok      logalert        0.2xxs
```
- 无 goroutine 泄漏，所有后台 goroutine 在服务停止后正常退出
- go test -race -count=3 无 DATA RACE 报告
- RED/GREEN 判定结果应为 GREEN
- go build ./... 与 go vet ./... 全部通过
- 退出码：0

## 6. 触发频率（Frequency）
- 100% 必现：每次启动服务并停止后，均能稳定触发 goroutine 泄漏
- 单次运行即可复现，无需多次执行
- 并发场景下同样稳定复现

## 7. 影响范围（Impact / Scope）
- 服务长时间运行后 goroutine 数量持续增长，内存泄漏
- 每次 Start/Stop 周期都会累积泄漏的 goroutine
- 高并发场景下泄漏速度加快
- 最终可能导致进程内存溢出（OOM），服务崩溃不可用
- 影响所有依赖周期性任务的功能：规则扫描、数据清理
- 线上可用性下降，需要频繁重启服务作为临时规避方案

## 8. 附加说明（Additional Notes / Workaround）
- 临时规避方案：定期重启服务以释放泄漏的 goroutine 和内存
- 建议设置监控告警，当 goroutine 数量超过阈值时触发告警
- 通过 pprof 可以观察到卡在 channel receive 操作上的 goroutine
- 无其他有效的临时 workaround
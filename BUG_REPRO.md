# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
日志告警服务在接收到终止信号（SIGTERM/SIGINT）后无法正常退出。服务端口已停止监听，但进程本身持续挂起不退出，必须使用 kill -9 强制终止才能结束进程。此问题稳定复现，影响服务的正常发布和重启流程。

## 2. 环境信息（Environment）
- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go build -race -o server ./cmd/server/，go test . -count=1 -run '^TestRedGreen$' -v -timeout 10s
- 硬件信息：CPU 4 核及以上

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 执行 go test . -count=1 -run '^TestRedGreen$' -v -timeout 10s
3. 观察测试输出及日志

（或通过以下方式复现服务层面的问题：）
4. 执行 go build -race -o server ./cmd/server/ 构建二进制
5. 执行 ./server 启动服务
6. 等待服务完全启动后，执行 kill -SIGTERM <PID> 发送终止信号
7. 观察进程是否在 3 秒内退出

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出：RED (红灯，缺陷未修复)
- 测试错误信息：WaitGroup did not complete after scheduler stop - goroutine leak detected
- 测试退出码：1（失败）
- 服务层面：发送 SIGTERM 后，日志显示"scheduler stopped"和"shutting down gracefully"，但进程不退出，必须 kill -9
- go test -race 未报告 DATA RACE（本缺陷不涉及数据竞争，为生命周期管理缺陷）

## 5. 期望结果（Expected Behavior）
- 执行 go test . -count=1 -run '^TestRedGreen$' 判定为 GREEN（绿灯，缺陷已修复）
- 发送 SIGTERM 后进程在 3 秒内正常退出，日志显示"server shutdown complete"
- go test -race -count=20 . -run '^TestRedGreen$' 全部通过且无数据竞争
- go build ./... 与 go vet ./... 全部通过
- 高并发场景下关闭服务仍能稳定退出

## 6. 触发频率（Frequency）
稳定复现（100%）。只要启动服务后发送终止信号，进程必然挂起。测试用例每次运行均超时。

## 7. 影响范围（Impact / Scope）
- 服务无法优雅关闭，发布/重启时必须 kill -9，可能导致数据丢失
- Kubernetes/Docker 等容器编排环境中，容器无法正常终止，影响滚动更新和服务伸缩
- 在云原生环境中可能触发健康检查失败，导致服务被标记为不健康
- 每次重启操作都需要手动干预，增加运维成本和故障风险

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：在发送 SIGTERM 等待约 5 秒后，发送 SIGKILL（kill -9）强制终止进程。此方法存在数据丢失风险，不建议在生产环境使用。
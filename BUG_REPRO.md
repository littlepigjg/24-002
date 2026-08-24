# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

告警服务在高并发场景下出现不稳定行为。当多个请求同时录制告警记录时，服务可能触发 panic 崩溃（错误信息为 "concurrent map iteration and map write"），或者告警查询接口返回的数据偶发为空、缺失部分记录。单条写入或低并发下表现正常，只有在高并发压力下才会暴露。

## 2. 环境信息（Environment）

- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：需启用 -race 检测（go test -race），并发数量 ≥ 30 goroutines
- 硬件信息：CPU 核数 ≥ 4（影响竞态触发概率，核心越多越容易复现）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -race -count=20 -run '^TestRedGreen$' .` 启动并发告警录制测试
3. 观察测试输出和进程行为
4. 若未复现，可增加并发数量或执行次数（如 -count=50）

## 4. 实际结果（Actual Behavior / Observed Output）

- 具体的 panic 错误信息：
  ```
  WARNING: DATA RACE
  Read at 0x00c0000c3050 by goroutine 22:
    logalert/internal/store.(*MemoryAlertStore).CountAlerts()
        /path/to/memory_alert_store.go:44
    logalert.TestRedGreen.func1()
        /path/to/red_green_test.go:22
    logalert/internal/store.(*MemoryAlertStore).Record()
        /path/to/memory_alert_store.go:63

  fatal error: concurrent map iteration and map write

  goroutine 45 [running]:
  internal/runtime/maps.fatal(...)
  internal/runtime/maps.(*Iter).Next(...)
  logalert/internal/store.(*MemoryAlertStore).SnapshotAlerts(...)
      /path/to/memory_alert_store.go:50
  logalert.TestRedGreen.func1(...)
      /path/to/red_green_test.go:23
  logalert/internal/store.(*MemoryAlertStore).Record(...)
      /path/to/memory_alert_store.go:63

  FAIL    logalert
  ```
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 其他异常现象：告警列表查询返回空数据或部分数据丢失
- go test -race 报告 DATA RACE，涉及 CountAlerts、SnapshotAlerts 与 Record 之间的并发读写

## 5. 期望结果（Expected Behavior）

- 无 panic、无数据竞争（go test -race 无警告）
- RED/GREEN 判定结果应为 GREEN
- 6000 条告警全部正确写入，查询接口返回完整数据
- 高并发下告警录制和查询均保持稳定，无崩溃、无数据丢失
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）

偶发（高并发下约 80% 概率复现），需 -race -count=20 才能稳定捕获。在 30 goroutines × 200 records 的压力下，不使用 -race 时也有较高概率触发 panic。使用 -race 时 100% 复现数据竞争。

## 7. 影响范围（Impact / Scope）

- 服务 panic 崩溃：高并发告警录制时可能导致整个服务进程崩溃
- 数据不一致：部分告警可能丢失，告警列表查询返回不完整数据
- 接口返回错误：告警录制接口间歇性返回错误，影响告警功能可用性
- 线上可用性下降：告警系统在高负载下不稳定，影响监控告警能力
- 规则扫描触发的告警录制也会受影响，因为调度器与 API 可能同时录制告警

## 8. 附加说明（Additional Notes / Workaround）

无临时 workaround。降低并发量可以减少触发概率，但无法根除问题。建议在业务低峰期进行修复，修复期间系统应避免高并发写入场景。

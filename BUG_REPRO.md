# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

在高并发场景下，日志统计接口返回的数据存在偶发不一致问题。具体表现为：返回的 `total_count`（总日志条数）与 `by_level`（按日志级别分组的计数）的总和不相等。当有并发写入同时发生时，统计数据会出现偏差，导致前端展示或数据分析时出现异常。

## 2. 环境信息（Environment）

- 操作系统：Linux (x86_64)
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go test -race -count=20 .
- 硬件信息：多核 CPU（并发缺陷在多核环境下更易复现）

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录，执行 `go build ./...` 确保编译通过
2. 执行 `go test -race -count=20 -v -run '^TestRedGreen$' .` 运行验证测试
3. 观察测试输出结果

## 4. 实际结果（Actual Behavior / Observed Output）

- 测试输出：`RED (红灯，缺陷未修复)`
- 测试退出码：1（FAIL）
- 统计数据不一致：`total_count` 与 `by_level` 总和不相等
- 每次运行均可稳定复现（20 次运行全部失败）
- go test -race 未报告 DATA RACE（该缺陷为逻辑竞态，非内存级竞态）

## 5. 期望结果（Expected Behavior）

- 测试输出：`GREEN (绿灯，缺陷已修复)`
- 测试退出码：0（PASS）
- 统计数据一致：`total_count` 始终等于 `by_level` 总和
- go test -race 无 DATA RACE 警告
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）

- 必现（100%），所有 20 次测试运行均稳定触发
- 缺陷在并发写入与统计查询同时发生时必然出现
- 即使没有并发写入，只要统计过程中数据发生变化也可能触发

## 7. 影响范围（Impact / Scope）

- 统计数据偶发不一致，total_count 与 by_level 总和不符
- 影响所有依赖日志统计接口的功能：仪表盘、报表、告警分析等
- 在高并发写入场景下问题尤为严重，可能导致数据分析结果偏差
- 不影响单条日志写入和读取功能

## 8. 附加说明（Additional Notes / Workaround）

无

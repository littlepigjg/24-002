# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
日志告警系统在处理带有附加元数据的日志条目或告警事件时，程序异常崩溃，抛出运行时 panic。具体表现为：当日志条目包含标签（如主机名、环境变量等键值对）或告警事件包含详情字段（如计数、时间窗口等信息）时，系统在数据序列化过程中直接崩溃，导致请求失败、功能不可用。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块名：logalert
- 运行参数：go test . -count=1 -run '^TestRedGreen$'
- 硬件信息：与缺陷无关（非并发竞态问题）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录 `/home/admin/code/24/002/24-002-6`
2. 执行 `go build ./...` 确保项目编译通过
3. 执行 `go test . -count=1 -run '^TestRedGreen$'` 运行验证测试
4. 观察测试输出，查看 RED/GREEN 判定结果及 panic 堆栈信息

## 4. 实际结果（Actual Behavior / Observed Output）
执行测试后，程序崩溃并输出以下关键信息：
- panic: assignment to entry in nil map
- 测试判定结果：RED（红灯，缺陷未修复）
- 退出码：1
- 受影响的测试用例：
  - LogEntry.ToMap() 带标签场景：FAIL，panic: assignment to entry in nil map
  - AlertEvent.ToMap() 带详情场景：FAIL，panic: assignment to entry in nil map
  - LogEntry.ToMapWithGuard()：PASS（带守卫的版本正常）
  - AlertEvent.ToMapWithGuard()：PASS（带守卫的版本正常）

## 5. 期望结果（Expected Behavior）
- 所有测试用例通过，无 panic、无异常崩溃
- 测试判定结果：GREEN（绿灯，缺陷已修复）
- 带标签的日志条目能正确序列化为 map，包含完整的标签键值对
- 带详情的告警事件能正确序列化为 map，包含完整的详情字段
- go build ./... 编译通过
- go vet ./... 无报错

## 6. 触发频率（Frequency）
必现（100%）。只要 LogEntry 的 Tags 字段非空或 AlertEvent 的 Details 字段非空，调用 ToMap() 或经过 service 层序列化流程时必定触发 panic。Tags/Details 为空时不会触发。

## 7. 影响范围（Impact / Scope）
- 所有涉及带标签日志条目的序列化操作全部失败
- 所有涉及带详情告警事件的序列化操作全部失败
- 日志查询详情页无法正常展示带标签的日志
- 告警历史详情无法正常展示带详情的告警
- 服务层的日志归档和告警记录功能瘫痪
- 影响线上日志告警系统的核心可用性

## 8. 附加说明（Additional Notes / Workaround）
目前无临时绕过方案。唯一的临时规避方式是确保所有 LogEntry 的 Tags 字段和 AlertEvent 的 Details 字段始终为空，但这会导致系统丢失关键元数据信息，严重影响可观测性能力。建议尽快修复。
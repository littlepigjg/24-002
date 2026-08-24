# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
日志告警服务在创建基于日志级别（Level）或错误率（ErrorRate）条件的告警规则后，发送满足条件的日志并触发规则扫描时，服务发生 panic 崩溃，错误信息为 "assignment to entry in nil map"。基于关键字（Keyword）或计数（Count）条件的规则不受影响，可正常触发告警。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go test -v -count=1 -run '^TestRedGreen$' .
- 硬件信息：CPU 多核

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录，执行 go build ./... 确保编译通过
2. 创建一个告警规则，条件类型为 ConditionLevel（或 ConditionErrorRate），指定日志级别为 ERROR，阈值为 2
3. 向系统中发送 2 条 ERROR 级别的日志
4. 触发一次规则扫描（调用 Scheduler.ScanOnce 或等待定时扫描）
5. 观察系统输出 / panic 信息

## 4. 实际结果（Actual Behavior / Observed Output）
- panic 错误信息："assignment to entry in nil map"
- 完整 panic 堆栈指向规则评估与告警记录相关代码路径
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 其他异常现象：基于关键字条件的规则可正常工作，不受影响
- go test -race 报告：无数据竞争，但存在 panic

## 5. 期望结果（Expected Behavior）
- 无 panic、无错误发生
- RED/GREEN 判定结果应为 GREEN
- 基于任意条件类型的规则均能正常触发告警，AlertEvent 的 Details 字段正确记录告警元数据
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）——只要规则条件类型为 ConditionLevel 或 ConditionErrorRate，且匹配日志数量达到阈值，每次触发规则评估都会稳定复现

## 7. 影响范围（Impact / Scope）
- 基于日志级别和错误率条件的告警规则完全无法使用
- 规则扫描过程 panic 会中断当次扫描，可能导致后续规则无法及时评估
- 告警记录丢失，运维人员无法收到关键告警通知
- 基于关键字和计数条件的规则不受影响

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：目前可使用 ConditionKeyword 或 ConditionCount 类型的规则替代 ConditionLevel 和 ConditionErrorRate 类型来实现告警需求。例如创建关键字条件规则，匹配 ERROR 关键字或特定错误信息模式，可达到类似效果。
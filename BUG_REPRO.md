# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
日志告警服务的错误率趋势统计功能存在数据异常。当告警存储中存在告警事件时，通过统计接口计算出的错误率会被异常放大，返回明显高于预期的错误率数值。在无告警数据的场景下，错误率计算完全正常。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go test -v -run '^TestRedGreen$' -count=1 .
- CPU 核数：与缺陷无关（本缺陷为非并发 slice 缺陷，不涉及竞态条件）

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录：cd /home/admin/code/24/002/24-002-11
2. 确保项目编译通过：go build ./...
3. 执行测试命令：go test -v -run '^TestRedGreen$' -count=1 .
4. 观察测试输出，检查错误率数值

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试输出显示：RED（红灯，缺陷未修复）
- 具体错误信息：point 0: expected error rate 0.725, got 1.5000
- TotalCount 和 ErrorCount 计数正确（20 和 10），但 ErrorRate 被异常放大
- go test -race 无 DATA RACE 报告（本缺陷为非并发问题）
- 异常仅在告警存储包含数据时出现

## 5. 期望结果（Expected Behavior）
- 测试输出显示：GREEN（绿灯，缺陷已修复）
- ErrorRate 应为 0.725（容差 0.01 内）
- TotalCount 为 20，ErrorCount 为 10
- 无 panic、无数据竞争
- go build 与 go vet 全部通过

## 6. 触发频率（Frequency）
必现（100%）：只要告警存储中存在匹配严重级别的告警事件，且日志存储中有对应的 ERROR/INFO 日志，错误率计算必然异常。单次测试即可稳定复现。

## 7. 影响范围（Impact / Scope）
- 服务统计接口返回错误的错误率数据，可能导致运维人员对系统状态做出错误判断
- 仅影响错误率趋势计算功能，不影响日志存储、告警触发等核心功能
- 当告警存储为空时不触发，影响范围与告警数据是否存在直接相关

## 8. 附加说明（Additional Notes / Workaround）
临时规避方法：在调用统计接口前清空告警存储中的告警数据，或在创建统计服务时传入 nil 作为告警存储参数，可绕过此问题。
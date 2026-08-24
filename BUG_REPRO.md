# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）
当向服务端发送空的查询请求或创建请求时（请求体中过滤条件为空、或缺少可选字段如 keywords/tags），服务端直接发生 panic 崩溃，错误信息为 "runtime error: index out of range [0] with length 0"，而不是返回正常的参数校验错误响应。该问题影响短链接创建、日志查询、告警查询及链接验证等多个接口。

## 2. 环境信息（Environment）
- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：go test -v -run '^TestRedGreen$' -count=1 .

## 3. 复现步骤（Steps to Reproduce）
1. 进入项目根目录 `/home/admin/code/24/002/24-002-9`
2. 执行 `go build ./...` 确保编译通过
3. 执行 `go test -v -run '^TestRedGreen$' -count=1 .`
4. 观察测试输出中的 panic 信息

## 4. 实际结果（Actual Behavior / Observed Output）
- 测试运行失败，退出码为 1
- RED/GREEN 判定结果：RED（红灯，缺陷未修复）
- 具体 panic 信息：
  - create_empty_request: runtime error: index out of range [0] with length 0
  - log_query_nil_filter: runtime error: index out of range [0] with length 0
  - alert_query_nil_filter: runtime error: index out of range [0] with length 0
  - short_url_validate: runtime error: index out of range [0] with length 0
- 正常的业务操作（创建带 keywords 的短链接、正常跳转、快照查询）不受影响

## 5. 期望结果（Expected Behavior）
- 无 panic，无运行时错误
- RED/GREEN 判定结果：GREEN（绿灯，缺陷已修复）
- 对于空的查询请求，应返回正常的参数校验错误提示（如 "keyword cannot be empty" 等），而非 panic 崩溃
- go build 与 go vet 全部通过
- 所有测试用例均通过

## 6. 触发频率（Frequency）
必现（100%）：只要请求中包含 nil 或空的切片字段（如空 Filter、无 Keywords、无 Tags），必定触发 panic。

## 7. 影响范围（Impact / Scope）
该缺陷导致服务端在接收到空查询请求时直接 panic 崩溃，影响以下业务场景：
- 用户创建短链接时不传 keywords 字段
- 用户查询日志时传空的过滤条件
- 用户查询告警时传空的过滤条件
- 系统内部验证 ShortURL 数据完整性时 tags 字段为空
严重情况下可能导致整个服务进程崩溃，所有请求不可用。

## 8. 附加说明（Additional Notes / Workaround）
当前无已知临时规避方法。建议在客户端确保所有可选数组字段均传入非空值，或在服务端修复后再上线。
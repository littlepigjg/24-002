# 缺陷复现报告（Bug Reproduction Report）

## 1. 问题概述（Summary）

日志告警服务的缓存加载路径中，当底层加载器返回永久性错误（如资源不存在、参数校验失败、冲突、上下文超时等）时，系统本应立即终止操作并将错误返回给调用方。但实际观察到的行为是：这些错误被重试了 3 次才最终返回，每次重试间隔约 100ms，导致额外 300ms 的响应延迟。这一问题仅在错误经过缓存层传递时出现，直接传递同类错误到重试逻辑则表现正常。受影响的错误类型包括 not_found、validation、conflict、context 等所有永久性错误类型。

## 2. 环境信息（Environment）

- 操作系统：Linux
- Go 版本：go 1.22
- 项目模块：logalert
- 运行参数：本次复现无需特殊运行参数，使用标准 go test 命令

## 3. 复现步骤（Steps to Reproduce）

1. 进入项目根目录
2. 确保编译通过：`go build ./...`
3. 执行复现测试：`go test -v -count=1 -run '^TestRedGreen$' .`
4. 观察输出中的 RED/GREEN 判定结果

## 4. 实际结果（Actual Behavior / Observed Output）

- not_found 类型的错误经过缓存层传递后，被重试 3 次（而不是立即返回）
- validation 类型的错误经过缓存层传递后，同样被重试 3 次
- conflict 类型的错误经过缓存层传递后，同样被重试 3 次
- context 类型的错误经过缓存层传递后，同样被重试 3 次
- retry.DoWithClassification 对缓存包装后的 not_found 错误分类错误：errType=""、isRetryable=true、category="unknown_retryable"
- 测试输出中，6 个子测试显示 **RED**（受影响的错误类型和代码路径）
- database 类型的错误经过缓存层传递后重试行为正常（该重试就重试）
- 直接传递 not_found / validation / conflict 等错误（不经缓存）重试行为正常（首次即返回）

关键输出片段：
```
=== RUN   TestRedGreen/not-found_error_via_GetOrSet_should_NOT_be_retried
    red_green_test.go:41: RED (红灯，缺陷未修复): not-found error was retried 3 times through cache GetOrSet, should have failed on first attempt
=== RUN   TestRedGreen/not-found_error_via_LoadOrCompute_should_NOT_be_retried
    red_green_test.go:74: RED (红灯，缺陷未修复): not-found error was retried 3 times through LoadOrCompute, should have failed on first attempt
=== RUN   TestRedGreen/validation_error_via_GetOrSet_should_NOT_be_retried
    red_green_test.go:103: RED (红灯，缺陷未修复): validation error was retried 3 times through cache, should have failed on first attempt
=== RUN   TestRedGreen/conflict_error_via_GetOrSet_should_NOT_be_retried
    red_green_test.go:132: RED (红灯，缺陷未修复): conflict error was retried 3 times through cache, should have failed on first attempt
=== RUN   TestRedGreen/not-found_error_misclassified_by_DoWithClassification_after_cache_wrapping
    red_green_test.go:153: RED (红灯，缺陷未修复): error type not preserved after cache wrapping, expected 'not_found' but got ''. category=unknown_retryable, retryable=true
=== RUN   TestRedGreen/context_error_via_GetOrSet_should_NOT_be_retried
    red_green_test.go:190: RED (红灯，缺陷未修复): context error was retried 3 times through cache, should have failed on first attempt
--- FAIL: TestRedGreen (0.12s)
    --- FAIL: ... (6 RED tests)
    --- PASS: ... (2 GREEN tests)
FAIL
```

## 5. 期望结果（Expected Behavior）

- not_found 类型的错误应在首次遇到时立即返回，不应触发重试
- 修复后所有子测试均显示 **GREEN**
- 测试命令退出码为 0（PASS）
- go build ./... 编译通过
- go vet ./... 无告警

## 6. 触发频率（Frequency）

必现（100%）。只要永久性错误（not_found、validation、conflict、context、serialization）经过缓存层传递给重试逻辑，就会触发。database、internal、concurrency 等临时性错误不受影响（本就应该重试）。

## 7. 影响范围（Impact / Scope）

- 所有经过缓存层加载且返回永久性错误的操作，包括：not_found、validation、conflict、context、serialization 等错误类型
- 所有经过 LoadOrCompute 路径加载且返回永久性错误的操作
- 所有依赖 DoWithClassification 对错误进行分类的操作
- 响应延迟增加约 300ms（3 次重试 × 100ms 间隔）
- 额外的数据库/存储访问压力（3 倍无效查询）
- 用户体验下降（响应时间增加）

## 8. 附加说明（Additional Notes / Workaround）

目前无临时规避方法。问题的核心现象是：同样的错误类型，直接传递给重试逻辑时行为正确，一旦经过缓存层包装后就会被错误地重试。建议排查缓存层错误传递链路与重试逻辑的错误分类判断之间的衔接。
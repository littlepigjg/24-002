# BUG_REPRO.md - Defer-in-Loop Resource Accumulation Defect

## 缺陷概述

| 项目 | 内容 |
|------|------|
| **缺陷类型** | Go 语言 defer-in-loop 反模式导致的资源堆积/数据丢失 |
| **严重程度** | 高 - 导致 URL 条目被意外消费，数据丢失 |
| **影响范围** | 跨服务：scheduler + cleanup 两个服务均受影响 |
| **触发条件** | 执行定时扫描（ScanOnce）或数据清理（CleanupOnce） |
| **修复难度** | 中 - 需理解 defer 执行机制，跨文件修复 |

## 缺陷位置

### 文件1: `internal/service/scheduler.go`
```go
// ScanOnce performs a single scan of all active rules.
func (s *scheduler) ScanOnce(ctx context.Context) error {
    // ... 规则扫描逻辑 ...

    if s.urlStore != nil {
        snapshot := s.urlStore.RawSnapshot()
        for code := range snapshot {
            // ❌ BUG: defer 在循环内注册，函数返回时才执行
            defer func(c string) {
                if s.urlStore != nil {
                    s.urlStore.ConsumeEntry(c)
                }
            }(code)
        }
    }
    // ...
}
```

### 文件2: `internal/service/cleanup_service.go`
```go
// CleanupOnce performs a single cleanup pass.
func (s *cleanupService) CleanupOnce(ctx context.Context) error {
    // ... 清理逻辑 ...

    if s.urlStore != nil {
        snapshot := s.urlStore.RawSnapshot()
        for code := range snapshot {
            // ❌ BUG: 同样的 defer-in-loop 反模式
            defer func(c string) {
                if s.urlStore != nil {
                    s.urlStore.ConsumeEntry(c)
                }
            }(code)
        }
    }

    return nil
}
```

## 缺陷原因

### Go defer 执行机制

在 Go 语言中，`defer` 语句将函数调用推入栈中，当外层函数返回时执行。
关键点：**defer 注册时不会执行，只有函数返回时才执行。**

### 问题分析

当 `for code := range snapshot` 循环时：
1. 每次迭代都注册一个新的 `defer` 调用
2. 循环结束后，所有 `defer` 都在栈上等待
3. 当 `ScanOnce`/`CleanupOnce` 返回时，所有 `defer` 以 LIFO 顺序一次性执行
4. 结果：所有 URL 条目被**一次性全部消费**，而非按需消费

### 缺陷影响

1. **数据丢失**：URL 条目被意外消费，无法正常访问
2. **状态污染**：consumedEntries 持续增长，影响统计
3. **跨服务叠加**：scheduler 和 cleanup 都执行消费，加剧问题
4. **累积效应**：多次扫描后，几乎所有条目都被消费

## 复现步骤

### 先决条件
- Go 1.22+
- 项目已编译通过

### 复现命令
```bash
cd /home/admin/code/24/002/24-002-23
go test -count=1 -v .
```

### 预期输出（缺陷存在时）
```
=== RUN   TestDeferInLoopResourceAccumulation
    red_green_test.go:68: Created 10 URLs successfully
    red_green_test.go:87: Final snapshot entries: 0
    red_green_test.go:88: Consumed entries: 10
    red_green_test.go:89: Pending tasks: 10
    red_green_test.go:95: RED: defer-in-loop defect detected - 10 entries consumed, 0 remaining
--- FAIL: TestDeferInLoopResourceAccumulation (0.00s)
```

### 缺陷验证逻辑

```
1. 创建 N 个短 URL → URLStore 中有 N 个条目
2. 设置 scheduler 和 cleanup 的 URLStore 引用
3. 执行 scheduler.ScanOnce()
   → 循环内注册 N 个 defer
   → 函数返回时，N 个 defer 全部执行
   → 所有 N 个条目被消费
4. 执行 cleanup.CleanupOnce()
   → snapshot 为空（已被 ScanOnce 消费）
   → 循环不执行，无新消费
5. 最终快照：0 条目可见，N 条目已消费
   → 测试失败（RED）
```

## 修复指南

### 修复原则
1. **禁止在循环内使用 defer**：将 defer 移至循环外或改为直接调用
2. **按需消费**：每次迭代立即执行消费操作，而非延迟
3. **跨文件一致**：两个服务的修复方式应保持一致

### 修复步骤

#### Step 1: 修复 scheduler.go ScanOnce
```go
// 修复后：直接调用，而非 defer
if s.urlStore != nil {
    snapshot := s.urlStore.RawSnapshot()
    for code := range snapshot {
        // ✅ FIX: 直接调用，立即消费
        s.urlStore.ConsumeEntry(code)
    }
}
```

#### Step 2: 修复 cleanup_service.go CleanupOnce
```go
// 修复后：同样改为直接调用
if s.urlStore != nil {
    snapshot := s.urlStore.RawSnapshot()
    for code := range snapshot {
        // ✅ FIX: 直接调用
        s.urlStore.ConsumeEntry(code)
    }
}
```

#### Step 3: 验证修复
```bash
go build ./...
go vet ./...
go test -count=1 -v .
```

### 修复后预期输出
```
=== RUN   TestDeferInLoopResourceAccumulation
    red_green_test.go:68: Created 10 URLs successfully
    red_green_test.go:87: Final snapshot entries: 10
    red_green_test.go:88: Consumed entries: 0
    red_green_test.go:95: GREEN: defer-in-loop defect fixed - all 10 entries preserved
--- PASS: TestDeferInLoopResourceAccumulation (0.00s)
```

## 相关 Go 特性

### defer-in-loop 反模式
- 这是 Go 社区公认的反模式
- 常见于资源清理、文件句柄关闭等场景
- 修复方式：使用匿名函数包裹循环体，或直接调用

### 参考示例
```go
// ❌ 错误：defer 在循环内
for _, f := range files {
    f, _ := os.Open(f)
    defer f.Close()  // 所有文件句柄在函数返回时才关闭！
}

// ✅ 正确：在匿名函数中使用 defer
for _, f := range files {
    func() {
        f, _ := os.Open(f)
        defer f.Close()  // 每次迭代结束时关闭
        // 使用 f...
    }()
}

// ✅ 正确：直接调用
for _, f := range files {
    f, _ := os.Open(f)
    // 使用 f...
    f.Close()  // 立即关闭
}
```

## 总结

本缺陷展示了 Go 语言中 `defer` 语句与循环结合使用时的经典陷阱。
修复方法简单明确，但需要跨文件（scheduler.go + cleanup_service.go）一致修改，
且需确保不破坏公开 API 契约。

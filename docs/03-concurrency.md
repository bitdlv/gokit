# 三、并发 / 调度

## goroutinepool — 统一协程池

合并原 `gochan` / `taskpool` / `utils/goroutinePool`。

**类型**：`Pool`、`Task`、`Handler`

```go
p := goroutinepool.New(
    goroutinepool.WithCap(50),
    goroutinepool.WithDefaultFunc(func() {}),
)
p.Put(goroutinepool.NewTask(func() error {
    return doWork()
}))
p.Wait()
p.Close()
```

| API | 说明 |
|---|---|
| `New(opts...)` | 构造 Pool |
| `WithCap(n)` | 容量 Option |
| `WithDefaultFunc(fn)` | 默认处理函数 |
| `Put(Task)` | 提交任务 |
| `NewTask(fn) / NewTaskWithHandler(fn, h)` | 构造 Task |
| `RunWorker()` | 手动启动 worker |
| `Wait()` | 等待全部完成 |
| `Close()` | 关闭池 |

### 子包 goroutinepool/dispatcher — 事件分发器

**类型**：`Dispatcher`、`DispatchItem`、`TaskDispatchFunc`

- `NewDispatcher(workers, fn)`
- `Dispatch(item)` — 按 key 哈希分发到 N 个 worker
- `Close()`

## scheduler — 定时任务

**类型**：`Scheduler`、`Job`、`JobOption`

```go
s := scheduler.NewScheduler()
s.MustRegisterAndRunImmediately(scheduler.NewJob("sync", fn,
    scheduler.WithInterval(5*time.Minute),
))
s.Run(ctx)
defer s.Close()
```

| API | 说明 |
|---|---|
| `NewScheduler()` | 构造调度器 |
| `NewJob(name, fn, opts...)` | 构造任务 |
| `WithInterval(d)` | 周期 |
| `WithOnce()` | 只跑一次 |
| `Register / RegisterAndRunImmediately / MustRegisterAndRunImmediately` | 注册 |
| `Run(ctx)` | 启动循环 |
| `Pop()` | 弹出到期任务 |
| `IsOnce() / IsTime()` | 状态查询 |
| `Close()` | 关闭 |

## locker — Redis 分布式锁

**类型**：`Lock`；子包 `redislocker`（go-zero 集成）

```go
lk := locker.NewLock(rds, "user:123")
lk.SetExpiration(10*time.Second)
if lk.TryLock() {
    defer lk.Unlock()
    // 临界区
}
```

| API | 说明 |
|---|---|
| `NewLock(rds, key)` | 构造锁 |
| `Lock()` | 阻塞加锁 |
| `TryLock() bool` | 非阻塞加锁 |
| `Unlock()` | 释放（Lua 校验 value） |
| `SetExpiration(d)` | 设置过期 |

**测试**：`miniredis` 起假 Redis，验证互斥 + 过期。

## 测试

```bash
go test -v ./goroutinepool/... ./scheduler/... ./locker/...
```

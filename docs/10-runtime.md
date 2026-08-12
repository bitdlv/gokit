# 十、运行时 / 日志 / 系统

## app — 全局启动 / 关闭钩子

```go
app.RegisterInitializer("db", func() error { return dbs.Init(cfg) })
app.Register("db", func() error { return dbs.Close() })

instance := app.GetInstance()
instance.Start()
defer instance.Stop()
```

| API | 说明 |
|---|---|
| `GetInstance()` | 单例 |
| `Register(name, closer)` | 关闭钩子 |
| `RegisterInitializer(name, initFn)` | 启动钩子 |

## amslogx — 日志适配器

**类型**：`AmsLogger`、`GormLogxLogger`

```go
l := amslogx.New(ctx)
l.Info("msg")
l.Errorf("boom: %v", err)

gormLog := amslogx.NewGormLogxLogger()  // 传给 gorm.Config.Logger
```

| API | 说明 |
|---|---|
| `New(ctx)` | 构造 |
| `NewGormLogxLogger()` | GORM 适配 |
| `LogMode(level)` | 切换级别 |
| `Info / Infof / Warn / Warnf / Error / Errorf / Trace` | 日志 |

## instrumentation — 性能采样

**类型**：`SystemStats`

| API | 说明 |
|---|---|
| `GetStackTrace()` | 当前 goroutine 堆栈 |
| `GetStackTraces()` | 全部 goroutine 堆栈 |
| `GetSystemStats()` | CPU / 内存快照 |

## environmentvariable — 语言环境开关

- `SetSystemLanguage(lang)`
- `SystemLanguageIsEnglish() bool`

## ssh — SSH 隧道（MySQL over bastion）

**类型**：`SSHConfig`、`ViaSSHDialer`

```go
cfg := &ssh.SSHConfig{Host:"bastion", User:"ops", KeyFile:"~/.ssh/id_rsa"}
dialer, err := ssh.GetSSHConnection(cfg)
mysql.RegisterDialContext("mysql+ssh", dialer.DialContext)
db, _ := sql.Open("mysql", "user:pw@mysql+ssh(target:3306)/db")
```

| API | 说明 |
|---|---|
| `GetSSHConnection(cfg)` | 建立隧道 |
| `AgentSSH(...)` | 使用 ssh-agent |
| `PublicKeyFile(path)` | 加载私钥 |
| `DialContext(ctx, net, addr)` | mysql-driver 回调 |

## 测试

```bash
go test -v ./app/... ./amslogx/... ./instrumentation/... ./environmentvariable/... ./ssh/...
```

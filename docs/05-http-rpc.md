# 五、HTTP / RPC / 中间件

## httpx — 统一 HTTP 客户端

**类型**：`IClient`、`OptFunc`

```go
c := httpx.NewClient(
    httpx.WithBaseUrl("https://api.x"),
    httpx.WithTimeout(5*time.Second),
    httpx.Debug(),
)
var r Resp
err := c.PostJson("/user", body).
    AddHeader("X-Token", "abc").
    ScanJsonBody(&r)
```

| API | 说明 |
|---|---|
| `NewClient(opts...)` | 构造 |
| `WithBaseUrl / WithTimeout / Debug` | Option |
| `Get / Delete` | 无 body |
| `PostJson / PostForm / PostXml` | 有 body |
| `PutJson / PutXml` | 更新 |
| `Request / Response` | 底层访问 |
| `AddHeader / SetHeader` | 头 |
| `AddCookie / SetCookies` | Cookie |
| `AddQuery / SetQuery` | Query |
| `SetTimeout` | 超时 |
| `ScanJsonBody(&v) / ScanXmlBody(&v)` | 反序列化响应 |
| `ParseParams(r) / XParse(r, &v)` | 解析请求 |

**测试**：`httptest.NewServer` 起 mock，断言方法/头/体。

## ws — WebSocket 长连接

**类型**：`Manager`、`GoroutineClient`、`GoroutineHandler`、`IClient`

```go
mgr := ws.NewManager()
mgr.Register("chat", handler)
http.HandleFunc("/ws", mgr.Serve)
```

| API | 说明 |
|---|---|
| `NewManager()` | 管理器 |
| `Register(name, h) / Unregister(name)` | 注册 handler |
| `Serve(w, r)` | 升级 HTTP → WS |
| `NewGoroutineClient(...)` | 客户端 |
| `StartProcess()` | 启动读写协程 |
| `Close()` | 关闭 |

## middleware

- `RequestIdMiddleware(next)` — 注入 `X-Request-Id` 到 context。

## validator — 参数校验

- `Validate(req) error` — 基于 go-playground/validator + zh 翻译，返回 gRPC error。

## result — 统一响应体

**类型**：`ResponseSuccessBean`、`ResponseErrorBean`、`NullJson`

```go
result.Success(w, data)
result.Error(w, err)
result.HttpResult(w, code, msg, data)
```

内部使用 `errx/legacy` 解析 uint64 错误码。

## 测试

```bash
go test -v ./httpx/... ./ws/... ./middleware/... ./validator/... ./result/...
```

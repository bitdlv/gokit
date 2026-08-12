# 七、微服务 / 网关

## kit — grpc-gateway 生成器 + gateway runtime

1322 LOC，最重的一个模块。

**类型**：`Gateway`、`Options`、`Option`、`RPC`、`Config`、`MiddleWare`、
`ResponseHandler`、`ErrorHandlerFunc`、`ResponseProcessor`、
`RouteEntry`、`ServiceGroup`、`ServiceRegistration`、`TemplateData`、`WrappedMarshaler`

**子包**：
- `kit/grpcgw` — 网关运行时
- `kit/gwgen`  — 代码生成
- `kit/clone`  — 深拷贝

### 生成 / 构建

| API | 说明 |
|---|---|
| `BuildServiceGroups(rpcs)` | 按服务分组 |
| `BuildRouteEntries(group)` | 生成路由表 |
| `Generate(cfg)` | 生成代码 |
| `ParseProto(path)` | 解析 proto |
| `ProtoPathToGoZero(p)` | 路径转换 |
| `ToRouteEntry(rpc)` | RPC → 路由项 |

### 运行时

```go
mux := grpcgw.NewGatewayMux(
    grpcgw.With(reg),
    grpcgw.WithMiddlewares(mw1, mw2),
    grpcgw.WithResponseProcessor(myResp),
    grpcgw.WithErrorHandler(myErrH),
)
http.ListenAndServe(":8080", mux)
```

| API | 说明 |
|---|---|
| `NewGatewayMux(opts...)` | 构造 |
| `NewResponseHandler / NewResponseHandler4Api` | 响应处理器 |
| `DefaultHandler / DefaultErrorHandler` | 默认处理 |
| `ResponseHandle / ResponseHandle2Api` | 直接调用 |

### Option

`With` / `WithMiddlewares` / `WithMarshaler` / `WithHeaderMatcher` /
`WithErrorHandler` / `WithResponseProcessor` / `WithIgnoreFields` /
`WithPrimaryKey` / `WithCreatedAt` / `WithDB`

### 中间件工具

- `MidWithWhiteList(paths...)` — 白名单跳过
- `NewHeaderMatcher(prefixes)` — 自定义 header 转发
- `MetadataFromReq(r)` — HTTP → gRPC metadata
- `HandleNonAsciiHeaders(r)` — 非 ASCII 编码处理
- `ApplyDynamicMask(...)` — 字段动态脱敏

### 上下文帮助

- `GetLang(ctx) string`
- `GetLoc(ctx) *time.Location`
- `GetUser(ctx) User`

### 其他
- `Copy(src, dst)` — 深拷贝
- `HTTPMethodConst(m)` — 方法常量

## nacos — 服务发现 / 配置

**类型**：`NsClient`

```go
cli := nacos.GetNsClientInstance()
cfg, err := nacos.LoadNsConfig("dataId", "group")
rpcConf, err := nacos.LoadRpcClientConf("svc")
insts, _ := nacos.SelectInstances("svc")
```

| API | 说明 |
|---|---|
| `GetNsClientInstance()` | 单例 |
| `LoadNsConfig(dataId, group)` | 加载配置 |
| `LoadRpcClientConf(name)` | 加载 RPC 配置 |
| `GetService(name)` / `GetAllService()` | 查询服务 |
| `SelectInstances(name)` | 获取实例列表 |

## pb — 动态字段承载

类似 `google.protobuf.Struct`，用于业务动态字段。

**类型**：
- `Value`（+ `Value_StrVal` / `Value_Int32Val` / `Value_Int64Val` / `Value_ListVal` / `Value_MapVal`）
- `List` — Value 列表
- `Map`  — Value 映射

| API | 说明 |
|---|---|
| `GetStrVal / GetInt32Val / GetInt64Val` | 取标量 |
| `GetListVal / GetMapVal` | 取容器 |
| `GetKind()` | 判类型 |
| `GetList() / GetFields()` | 遍历 |
| `Descriptor / ProtoMessage / ProtoReflect / Reset / String` | 标准 proto 接口 |

配合 `helper.Any2PbValue / Map2Pb / PbValue2Any / PbMap2MapStrAny` 使用。

## 测试

```bash
go test -v ./kit/... ./nacos/... ./pb/...
```

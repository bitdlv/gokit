# 六、错误码

## errx — 业务 / gRPC 错误码体系（主）

**类型**：`Error`、`ErrorDetails`

### 构造

```go
err := errx.New(codes.InvalidArgument, "参数错误")
err = errx.NewWithPos(codes.Internal, "boom", 1)
err = errx.NewErrCodeMsg(codes.NotFound, "用户不存在")
```

| API | 说明 |
|---|---|
| `New(code, msg)` | 新建 |
| `NewWithPos(code, msg, skip)` | 带位置 |
| `NewErrCode(code)` / `NewErrCodeMsg(code, msg)` / `NewErrMsg(msg)` | 简写 |
| `NewFromStatus(s)` | 从 gRPC status |

### 包装

```go
err = errx.Wrap(err, codes.Internal, "调用下游失败")
err = errx.WrapCode(err, codes.Internal)
err = errx.WrapMsg(err, "补充信息")
err = errx.WithDetails(err, "trace", "xxx")
```

### 预置业务错误

- 通用：`Conflict / NotFound / InvalidRequest / Timeout / ThirdPartyServiceFailure`
- DB：`QueryFailed / QueryTotalFailed / SaveFailed / DeleteFailed`
- JSON：`JsonEncodeFailed / JsonDecodeFailed`
- 文件：`FileImportFailed / GenFileFailed / FileCanNotGt`
- 分页：`LimitCanNotLt0 / LimitCanNotLtMinus1 / OffsetCanNotLt0`

### 查询 / 输出

| API | 说明 |
|---|---|
| `IsCodeErr(code)` | 是否已知码 |
| `MapErrMsg(code)` | 码 → 默认消息 |
| `ToHttpStatus(code)` | gRPC → HTTP |
| `PrintErrorDetail(err)` | 调试打印 |
| `Format` / `GRPCStatus` / `Unwrap` | 格式化/转 status/解包 |

## errx/legacy — uint64 错误码（原 xerr，兼容层）

**类型**：`CodeError`、`Error`

供旧代码和 `result` 包使用。新代码请优先用主包 `errx`。

```go
err := legacy.NewErrCodeMsg(legacy.REUQEST_PARAM_ERROR, "参数错")
err = legacy.WrapWithCode(err, legacy.DB_ERROR)
code := legacy.GetErrCode(err)
msg  := legacy.MapErrMsg(code)
```

| API | 说明 |
|---|---|
| `C(code)` / `E(err)` / `F(fmt, ...)` | 链式构造 |
| `Errorf(code, fmt, ...)` | fmt 构造 |
| `New / NewErrCode / NewErrCodeMsg / NewErrMsg` | 构造 |
| `WrapWithCode / WrapWithCodeMsg / WrapWithCodeOriginalMsg / WrapWithMsg` | 包装 |
| `MapErrMsg / IsCodeErr / ParseError` | 查询 |
| `GetErrCode / GetErrMsg` | 提取 |

## 测试

```bash
go test -v ./errx/...
```

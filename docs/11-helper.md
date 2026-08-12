# 十一、杂项胶水 helper

1237 LOC，反射 + 表达式 + ctx + pb + 批处理。原 `utils` 已合并入此。

**类型**：`IntBase`、`Number`、`Supported`（泛型约束）、`LogicResponse`

**子包**：`helper/testconfig` — 测试配置加载

## Ctx / 请求参数

| API | 说明 |
|---|---|
| `GetValueFromCtxByKey(ctx, key) (string, bool)` | 取 ctx 值 |
| `Parse(r, &v)` | 解析请求 |
| `ParseForm(r, &v)` | form 解析 |
| `GetFormValues(r)` | 取所有 form |
| `NewMetadataForRestToRpc(r)` | HTTP → gRPC metadata |
| `FormatResponseBody(data)` | 统一响应体 |

## 泛型集合操作

| API | 说明 |
|---|---|
| `Filter[T](src, f)` | 过滤 |
| `MapSlice[T,K](data, f)` | 转换 |
| `FindSlice[T](s, finder)` | 查找 |
| `Contains[T](finds, data)` | 包含 |
| `Diff[T](a, b)` | 差集 |
| `Ternary[T](e, t, f)` | 三元 |
| `Rotate / Set` | 旋转 / 集合 |
| `ParseSlice[T](input, sep)` | 分隔串转泛型切片 |
| `SliceOperator[T](A, B, eq)` | 并/交/差 |
| `ArrayColumn[T,R](items, getField)` | 抽取列 |

## 反射抽取字段

| API | 说明 |
|---|---|
| `Pluck(data, container)` | 反射抽字段 |
| `AdvPluck[T](data, field)` | 泛型版 |
| `GetField[T](s, field, def...)` | 取字段值 |
| `GetItems(...)` | 通用 items 取值 |
| `JoinField(slice, field, sep)` | 抽字段拼接 |
| `JoinAnySlice(slice, sep)` | 任意切片拼接 |

## 数学 / 统计

`Avg[T]` / `Max[T]` / `Length(...)` / `Sum_Flt32Arr` /
`Average_Flt32Arr` / `Average_Flt64Arr` / `Average_StrArr`

## pb ↔ any

配合 `pb` 包使用：

| API | 说明 |
|---|---|
| `Any2PbValue(v)` | any → *pb.Value |
| `Map2Pb(v)` | map → *pb.Map |
| `PbValue2Any(pv)` | *pb.Value → any |
| `PbList2SliceAny(pl)` | *pb.List → []any |
| `PbMap2MapStrAny(pm)` | *pb.Map → map[string]any |

## JSON / 配置

- `Json2Map(str)` — JSON 转 map
- `Map2Json(m)` — map 转 JSON
- `LoadConf(configStr, &v)` — 配置加载

## 时间 / 字符串

- `FormatTs(unix, format)` — Unix → 格式化字符串
- `Time2LocalString(t)` — Time → 本地字符串
- `TruncateString(s, length)` — 按宽度截断（含中文）
- `GetRandString(n)` — 随机串
- `GetColumnName(index)` — Excel 列名（A/B/.../AA）

## 表达式引擎（govaluate）

- `ExpressionToPlaceholderExpression(expr) (placeholderExpr, vars, noCalc)`
- `GeExpressionComputeVal(expr, params) (result, error)`

## 批处理 / 超时 / 反射调用

| API | 说明 |
|---|---|
| `BatchProcess[T](s, batchSize, f)` | 顺序分批 |
| `AsyncBatchProcess[T](s, batchSize, f, max)` | 并发分批 |
| `Timeout(ctxFn, dur)` | 超时执行 |
| `AnyCall[T](obj, method, args...)` | 反射调用 |
| `Check(v)` | 通用校验 |
| `NewRepeatChecker[G,T](cap)` | 重复检查 |
| `NewHBaseDataInfo(size)` | HBase 数据结构 |
| `InitForTest()` | 测试初始化 |

## helper/testconfig 子包

加载测试环境配置。

## 测试

```bash
go test -v ./helper/...
```

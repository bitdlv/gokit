# 一、基础数据 / 编码工具

纯函数库，无状态，直接调用。

## conv — 进制 / 大整数 / 切片

| 函数 | 说明 |
|---|---|
| `BigToHex(*big.Int) string` | 大整数转 hex 字符串 |
| `Uint64ToHex(uint64) string` | uint64 转 hex |
| `Bin2Hex([]byte) string` | 二进制转 hex |
| `Hex2Bin(string) []byte` | hex 转二进制 |
| `Hex2BinCoinBaseTags(string) []byte` | 币基 tag 专用解析 |
| `ParseBigInt(string) (*big.Int, error)` | 字符串转大整数 |
| `ParseUint(string) (uint64, error)` | 字符串转 uint64 |
| `IntsToStrings([]int) []string` | int 切片转 string 切片 |
| `RemoveDuplicateElement([]T) []T` | 切片去重 |

## convert — 字符串/数值/时间转换

**类型**：`Num`

| 函数 | 说明 |
|---|---|
| `ArrToString([]any, sep) string` | 切片转分隔字符串 |
| `StringToArr(s, sep) []string` | 分隔字符串转切片 |
| `NumToNum(src, dst)` | 跨数字类型转换 |
| `Unique([]T) []T` | 去重 |
| `StringToDate(string) time.Time` | 字符串转日期 |
| `WIthString2Int(string) int` | 字符串转 int |
| `WithStringToTime / WithStringToLocalTime` | 字符串 → time.Time / Local |
| `WithTime2String / WithTime2East8String / WithTime2LocalString` | time → 字符串 |

## cryptos — 摘要

`MD5` / `SHA256` / `HMACsha256` / `DoubleHashH` — 一次性摘要，返回 hex/base64。

## sign — MD5 签名

- `GenerateSign(params map[string]string, secret string) string` — 参数排序后 MD5 签名
- `VerifySign(params, sign, secret) bool`

## random — Snowflake ID

- `Init(nodeID int64)` — 初始化节点
- `GenerateID() string`
- `GenerateIDWithPrefix(prefix string) string`

## masking — 日志脱敏

- `LogMaskingConfig` — 全局脱敏规则
- `RegExpReplaceAllString(pattern, src, repl) string`

## types — 自定义 JSON

**类型**：`JsonStr` — 内嵌 JSON 字符串扁平化，实现 `MarshalJSON` / `UnmarshalJSON`。

## time — 东八区时间

- `ParseEast8(layout, value) time.Time`
- `ParseEast8UnixOrZero(string) int64`
- `ParseLocal(layout, value) time.Time`
- `ToLocalString(t) string`

## 测试

均为纯函数，直接 `go test -v ./conv/... ./convert/... ...`。

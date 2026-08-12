# gokit header / jwt / jwtAuth 使用说明

本文以 idx 项目（`/Users/zyl/work/idx/idx/`）为落地范例，说明 gokit 三个基础包
如何配合使用：

- `github.com/bitdlv/gokit/header` — HTTP header 常量 + 透传注册表
- `github.com/bitdlv/gokit/jwt` — JWT 签发 / 校验（HS/RS/PS/ES/EdDSA）
- `github.com/bitdlv/gokit/middleware` — `JwtAuthMiddle` 鉴权中间件、`Header2ContextHandle` header→ctx 透传

---

## 1. 设计约定

gokit **不再内置**任何业务 header 常量（历史遗留的 `token`/`userId`/`pcode`
等 camelCase 契约已从 `header` 包移除）。原因：

1. 业务 header 名是**项目契约**，不同项目不一样，硬编码到公共库会耦合。
2. camelCase HTTP header 违反 RFC 7230（不区分大小写、建议 Canonical 形式）。
   保留在公共库会持续误导新项目。

新项目应使用 `X-Foo-Bar` 短横线 canonical 形式；老项目（如 idx）作为**历史契约**
在自己项目内声明常量并注入到 gokit 注册表即可。

---

## 2. idx 项目落地示例

### 2.1 在 idx 项目内声明业务 header 常量

新建 `idx/internal/consts/headers.go`：

```go
package consts

// idx 历史契约（camelCase） — 仅本项目内部使用，请勿外传。
const (
    HeaderToken       = "token"       // JWT / Raw token
    HeaderUserId      = "userId"      // 用户主键
    HeaderUserName    = "username"    // 登录名（URL 转义后）
    HeaderPhone       = "userPhone"   // 手机号
    HeaderBpmUid      = "bpmUserId"   // BPM 用户 ID
    HeaderBpmUname    = "bpmUserName" // BPM 账号
    HeaderProjectCode = "pcode"       // 项目编码
)

// AllPassthroughKeys 是需要 grpc-gateway 透传到 gRPC 下游的 header 集合。
var AllPassthroughKeys = []string{
    HeaderToken, HeaderUserId, HeaderUserName, HeaderPhone,
    HeaderBpmUid, HeaderBpmUname, HeaderProjectCode,
}
```

### 2.2 启动时注册透传 header

在 `idx.go`（服务入口）里，构造 gateway 前调用一次：

```go
import (
    "github.com/bitdlv/gokit/header"
    "github.com/bitdlv/gokit/kit/grpcgw"
    "idx/internal/consts"
)

func main() {
    // 注入 idx 私有 header 契约到 gokit 透传注册表
    header.Register(consts.AllPassthroughKeys...)

    // 之后 grpcgw.NewHeaderMatcher(nil) 会自动读取注册表快照
    gw, _ := grpcgw.DefaultHandler(ctx, grpcEndpoint,
        []func(http.HandlerFunc) http.HandlerFunc{jwtAuthMw, header2CtxMw},
        grpcgw.Service(pb.RegisterSysServiceHandlerFromEndpoint),
    )
    // ...
}
```

### 2.3 配置 JwtAuthMiddle

`internal/middlewares/jwtauth.go` 里：

```go
import (
    "github.com/bitdlv/gokit/middleware"
    "idx/internal/consts"
    "idx/internal/svc"
)

func NewJwtAuth(svcCtx *svc.ServiceContext) *middleware.JwtAuthMiddle {
    cfg := middleware.JwtAuthConfig{
        Secret:       svcCtx.Config.JwtAuth.AccessSecret,
        JwtAuthOpen:  svcCtx.Config.JwtAuth.Open,
        SignAuthOpen: svcCtx.Config.SignAuth.Open,
        InnerAPIKey:  svcCtx.Config.InnerAPIKey,
        InnerSalt:    svcCtx.Config.InnerSalt,
        ServiceName:  "idx",

        // idx 契约 —— 全部在项目侧声明
        TokenHeader:        consts.HeaderToken,
        InnerSignHeader:    "sign",
        InnerAccountHeader: "userAccount",
        IdentityHeaders: middleware.IdentityHeaderMap{
            UserID:     consts.HeaderUserId,
            UserName:   consts.HeaderUserName,
            Phone:      consts.HeaderPhone,
            BmpID:      consts.HeaderBpmUid,
            BmpAccount: consts.HeaderBpmUname,
        },
    }
    return middleware.NewJwtAuth(cfg, newIdxUserLoader(svcCtx),
        middleware.WithSignVerifier(newIdxSignVerifier(svcCtx)),
    )
}
```

### 2.4 用户模型适配 AuthUser 接口

对 `model.SysUser` 加薄适配（或让 model 直接实现）：

```go
type idxAuthUser struct{ u *model.SysUser }

func (a idxAuthUser) GetID() string         { return strconv.FormatInt(a.u.ID, 10) }
func (a idxAuthUser) GetUserName() string   { return a.u.UserName }
func (a idxAuthUser) GetPhone() string      { return a.u.Phone }
func (a idxAuthUser) GetBmpID() string      { return a.u.BpmUserID }
func (a idxAuthUser) GetBmpAccount() string { return a.u.BpmUserName }
func (a idxAuthUser) IsEnabled() bool       { return a.u.Status == model.UserStatusEnabled }
```

`UserLoader` 三个方法（`LoadByID` / `LoadByPhone` / `LoadByAccount`）由项目按
DB + Redis 缓存自行实现。

### 2.5 下游 handler 读取用户身份

原 idx 代码里 `r.Header.Get(header.HeaderUserId)` 改为读本项目常量：

```go
// internal/logic/sysservice/accessTokenLogic.go
userId := r.Header.Get(consts.HeaderUserId)
```

或使用 `middleware.IsAuthenticated(ctx)` 判断是否已鉴权。

### 2.6 响应字段脱敏（可选）

原 `grpcgw.HeaderUserId` 用作 `ResponseHandler` 读取用户 ID 的默认 key，
现改为显式配置：

```go
// 启动阶段一次性设置
grpcgw.SetUserIDHeader(consts.HeaderUserId)
```

未调用 `SetUserIDHeader` 时 `ResponseHandler` 直接跳过脱敏。

---

## 3. JWT 快捷用法

```go
import "github.com/bitdlv/gokit/jwt"

// 签发（HS256）
token, err := jwt.GenerateHS256(secret, time.Now().Unix(), 7200,
    jwt.K("userId", "1001"),
    jwt.K("phone",  "13800000000"),
)

// 校验（返回 StandardClaims{UserID, Phone}）
claims, err := jwt.ValidateHS256(token, secret)
_ = claims.UserID
```

高级用法（RS/ES/EdDSA、自定义 claim 结构）见 `jwt/jwt.go` 的 `Sign` / `ParseInto`。

---

## 4. Header → Context 透传

`middleware.Header2ContextHandle()` 会把 `header.Keys()` 内的 key 从
`r.Header` 拷贝到 `ctx`，供下游 grpc handler 用 `metadata` 拿到：

```go
mw := middleware.Header2ContextHandle()  // 自动读注册表
gw, _ := grpcgw.DefaultHandler(ctx, endpoint,
    []func(http.HandlerFunc) http.HandlerFunc{jwtAuth, mw},
    ...
)
```

---

## 5. 迁移检查清单

从 `xxx.cn/business/common` → `github.com/bitdlv/gokit`
时按以下顺序改造：

1. 项目内新建 `consts/headers.go`（业务 header 常量集中）
2. 服务入口调用 `header.Register(consts.AllPassthroughKeys...)`
3. `JwtAuthConfig` 填 `TokenHeader` + `IdentityHeaders`
4. `SysUser` 实现 `middleware.AuthUser` 接口（薄适配）
5. `UserLoader` 三个方法在项目内实现（可复用旧 svc 层）
6. `grpcgw.SetUserIDHeader(consts.HeaderUserId)`（如启用响应脱敏）
7. 全项目 grep 替换：
   - `common.HeaderUserId` → `consts.HeaderUserId`
   - `common.AuthToken`    → `consts.HeaderToken`
   - 其余同名 header 类推

---

## 6. 相关文件

- `gokit/header/header.go` — 注册表 + 标准 header 常量
- `gokit/jwt/jwt.go` — JWT Sign/Parse
- `gokit/middleware/jwtauth.go` — `JwtAuthMiddle` + `IdentityHeaderMap`
- `gokit/middleware/header2ctx.go` — header→ctx 透传
- `gokit/kit/grpcgw/gateway.go` — `NewHeaderMatcher` / `SetUserIDHeader`

idx 项目参考实现：
- `idx.go`（入口注册）
- `internal/middlewares/{jwtauth,restheader2ctx}.go`（中间件挂载）
- `internal/logic/sysservice/{accessToken,callback*}Logic.go`（读 header）
- `internal/ws/handler_auth.go`（WS token 透传）

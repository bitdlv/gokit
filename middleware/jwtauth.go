// Package middleware 中的 JwtAuthMiddle 是从 idx 项目通用化后的 HTTP 鉴权中间件。
//
// 与 idx 版本相比，本实现将三处强耦合抽象为接口 / 配置项：
//
//  1. 用户模型 → AuthUser 接口 + UserLoader 接口（调用方注入 DB / 缓存实现）
//  2. 项目状态常量 → AuthUser.IsEnabled() 由调用方决定
//  3. 外部签名校验 → 可选注入 SignVerifier
//
// 支持三条鉴权分支（与 idx 一致）：
//   - Token 认证：Header[cfg.TokenHeader]，支持 JWT 与 Raw+手机号 两种格式
//   - 内部服务签名认证：Header[cfg.InnerSignHeader] + Header[cfg.InnerAccountHeader]
//   - 外部签名认证：SignVerifier（可选）
//
// 鉴权通过后，中间件按 cfg.IdentityHeaders 表把用户字段回写到请求 header，
// 供下游 handler 读取。IdentityHeaders 中任一值为空则跳过对应字段。
package middleware

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/bitdlv/gokit/jwt"
)

// ─────────────────────── 抽象 ────────────────────────────────────────────────

// AuthUser 表示中间件视角下的鉴权用户；调用方自行实现。
type AuthUser interface {
	// GetID 返回用户主键的字符串形式（写入 IdentityHeaders.UserID 指向的 header）。
	GetID() string
	// GetUserName 返回登录名（写入 IdentityHeaders.UserName；中间件做 URL 转义）。
	GetUserName() string
	// GetPhone 返回手机号。
	GetPhone() string
	// GetBmpID / GetBmpAccount 返回 BPM 侧标识。
	GetBmpID() string
	GetBmpAccount() string
	// IsEnabled 报告用户是否处于可用状态；false 直接返回 401。
	IsEnabled() bool
}

// UserLoader 抽象用户加载来源；调用方可在实现内部集成 gorm / redis 缓存等。
type UserLoader interface {
	LoadByID(ctx context.Context, id string) (AuthUser, error)
	LoadByPhone(ctx context.Context, phone string) (AuthUser, error)
	LoadByAccount(ctx context.Context, account string) (AuthUser, error)
}

// SignVerifier 校验外部服务签名；返回 false 表示签名无效，中间件将回复 401。
type SignVerifier func(r *http.Request) bool

// ─────────────────────── 配置 ────────────────────────────────────────────────

// IdentityHeaderMap 描述鉴权通过后写回请求 header 的字段目标。
// 任一值为空字符串则该字段不会被写入 header。
type IdentityHeaderMap struct {
	UserID     string // 用户主键
	UserName   string // 登录名（会做 url.QueryEscape）
	Phone      string // 手机号
	BmpID      string // BPM 用户 ID
	BmpAccount string // BPM 账号
}

// JwtAuthConfig 描述中间件运行所需的配置。
type JwtAuthConfig struct {
	Secret       string // JWT HS256 密钥
	JwtAuthOpen  bool   // 是否启用 JWT / 内部签名 强校验
	SignAuthOpen bool   // 是否启用外部签名（SignVerifier）
	InnerAPIKey  string // 内部服务间调用 API Key
	InnerSalt    string // 内部签名 salt
	ServiceName  string // 内部签名公式中的服务名前缀，默认 "svc"

	// TokenHeader 是携带 JWT / Raw token 的 header key；默认 "Authorization"。
	TokenHeader string
	// InnerSignHeader 是内部服务间签名 header key；默认 "sign"。
	InnerSignHeader string
	// InnerAccountHeader 是内部服务间账号 header key；默认 "userAccount"。
	InnerAccountHeader string
	// IdentityHeaders 决定鉴权成功后用户字段回写的 header 名。全部为空则不注入任何字段。
	IdentityHeaders IdentityHeaderMap

	UserStatusDisabledMsg string // 用户禁用提示语；默认 "user status disabled"
}

// JwtAuthOption 可选项。
type JwtAuthOption func(*JwtAuthMiddle)

// WithSignVerifier 注入外部签名校验器。未注入时外部签名分支跳过校验直接放行。
func WithSignVerifier(v SignVerifier) JwtAuthOption {
	return func(j *JwtAuthMiddle) { j.signVerifier = v }
}

// ─────────────────────── 内部签名 ────────────────────────────────────────────

// GenerateInnerSign 生成内部服务间调用签名：
// md5(serviceName + ":" + userAccount + ":" + salt + ":" + innerAPIKey)。
func GenerateInnerSign(serviceName, userAccount, salt, innerAPIKey string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s:%s", serviceName, userAccount, salt, innerAPIKey))))
}

// VerifyInnerSign 校验内部服务间调用签名。
func VerifyInnerSign(signVal, serviceName, userAccount, salt, innerAPIKey string) bool {
	if signVal == "" || userAccount == "" || innerAPIKey == "" {
		return false
	}
	expected := GenerateInnerSign(serviceName, userAccount, salt, innerAPIKey)
	return signVal == expected
}

// ─────────────────────── 中间件 ──────────────────────────────────────────────

// JwtAuthMiddle 是可复用的 HTTP 鉴权中间件。
type JwtAuthMiddle struct {
	cfg          JwtAuthConfig
	loader       UserLoader
	signVerifier SignVerifier
}

// NewJwtAuth 构造 JwtAuthMiddle。loader 不可为 nil；未提供 SignVerifier 时外部签名分支放行。
func NewJwtAuth(cfg JwtAuthConfig, loader UserLoader, opts ...JwtAuthOption) *JwtAuthMiddle {
	if cfg.ServiceName == "" {
		cfg.ServiceName = "svc"
	}
	if cfg.UserStatusDisabledMsg == "" {
		cfg.UserStatusDisabledMsg = "user status disabled"
	}
	if cfg.TokenHeader == "" {
		cfg.TokenHeader = "Authorization"
	}
	if cfg.InnerSignHeader == "" {
		cfg.InnerSignHeader = "sign"
	}
	if cfg.InnerAccountHeader == "" {
		cfg.InnerAccountHeader = "userAccount"
	}
	j := &JwtAuthMiddle{cfg: cfg, loader: loader}
	for _, o := range opts {
		o(j)
	}
	return j
}

// Handle 返回符合 grpc-gateway 中间件签名的处理函数。
func (j *JwtAuthMiddle) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get(j.cfg.TokenHeader)

		// ── 1. Token 认证（JWT / Raw）─────────────────────────────────────
		if token != "" {
			userInfo, err := j.resolveTokenUser(r)
			if err != nil {
				// token 认证失败，降级尝试内部签名认证
				if j.applyInnerSignAuth(r) {
					next(w, withAuthenticatedUser(r))
					return
				}
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			if !userInfo.IsEnabled() {
				http.Error(w, j.cfg.UserStatusDisabledMsg, http.StatusUnauthorized)
				return
			}
			j.setUserHeaders(r, userInfo)
			next(w, withAuthenticatedUser(r))
			return
		}

		// ── 2. 内部服务间签名认证 ─────────────────────────────────────────
		if innerSign := r.Header.Get(j.cfg.InnerSignHeader); innerSign != "" {
			userAccount := r.Header.Get(j.cfg.InnerAccountHeader)
			if j.cfg.JwtAuthOpen && !VerifyInnerSign(innerSign, j.cfg.ServiceName, userAccount, j.cfg.InnerSalt, j.cfg.InnerAPIKey) {
				http.Error(w, "inner sign is invalid", http.StatusUnauthorized)
				return
			}
			authenticated := j.applyInnerSignAuth(r)
			if j.cfg.JwtAuthOpen && !authenticated {
				http.Error(w, "inner sign user is invalid", http.StatusUnauthorized)
				return
			}
			if authenticated {
				next(w, withAuthenticatedUser(r))
				return
			}
			next(w, r)
			return
		}

		// ── 3. 外部签名认证 ─────────────────────────────────────────────
		if j.cfg.SignAuthOpen && j.signVerifier != nil {
			if !j.signVerifier(r) {
				http.Error(w, "sign is invalid", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

// resolveTokenUser 根据 JWT / Raw token 解析并返回用户信息。
func (j *JwtAuthMiddle) resolveTokenUser(r *http.Request) (AuthUser, error) {
	token := r.Header.Get(j.cfg.TokenHeader)
	if strings.HasPrefix(token, "Raw") {
		re := regexp.MustCompile(`(?i)\bRaw\s*(?:\+?86[-\s]?)?(1\d{10})\b`)
		m := re.FindStringSubmatch(token)
		if m == nil {
			return nil, fmt.Errorf("Authorization Failed: invalid Raw token format")
		}
		return j.loader.LoadByPhone(r.Context(), m[1])
	}
	claims, err := jwt.ValidateHS256(token, j.cfg.Secret)
	if err != nil || claims == nil {
		return nil, fmt.Errorf("token is invalid: %v", err)
	}
	u, err := j.loader.LoadByID(r.Context(), claims.UserID)
	if err != nil || u == nil {
		return nil, fmt.Errorf("the userinfo in token is invalid: %v", err)
	}
	return u, nil
}

// applyInnerSignAuth 校验内部签名，通过后查询用户并注入 header。
func (j *JwtAuthMiddle) applyInnerSignAuth(r *http.Request) bool {
	innerSign := r.Header.Get(j.cfg.InnerSignHeader)
	if innerSign == "" {
		return false
	}
	userAccount := r.Header.Get(j.cfg.InnerAccountHeader)
	if !VerifyInnerSign(innerSign, j.cfg.ServiceName, userAccount, j.cfg.InnerSalt, j.cfg.InnerAPIKey) {
		return false
	}
	u, err := j.loader.LoadByAccount(r.Context(), userAccount)
	if err != nil || u == nil {
		return false
	}
	j.setUserHeaders(r, u)
	return true
}

// setUserHeaders 按 IdentityHeaders 表把用户字段写入请求 header。
// 表内为空的字段跳过——从而允许调用方按需选择要下传的身份字段。
func (j *JwtAuthMiddle) setUserHeaders(r *http.Request, u AuthUser) {
	h := j.cfg.IdentityHeaders
	if h.UserID != "" {
		r.Header.Set(h.UserID, u.GetID())
	}
	if h.UserName != "" {
		r.Header.Set(h.UserName, url.QueryEscape(u.GetUserName())) // 中文名 URL 转义
	}
	if h.Phone != "" {
		r.Header.Set(h.Phone, u.GetPhone())
	}
	if h.BmpID != "" {
		r.Header.Set(h.BmpID, u.GetBmpID())
	}
	if h.BmpAccount != "" {
		r.Header.Set(h.BmpAccount, u.GetBmpAccount())
	}
}

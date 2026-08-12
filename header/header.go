// Package header 提供 HTTP header key 常量、命名规范以及并发安全的可扩展注册表。
//
// 命名规范：
//   - 标准 HTTP header 采用 http.CanonicalHeaderKey 规范（首字母大写 + 连字符），
//     便于与 net/http 直接互操作。
//   - 新增业务 header 建议使用 "X-" 前缀 + 短横线分隔的规范化命名。
//   - 本包**不再**内置任何项目专属的业务 header 常量（例如 "token" / "userId"
//     等历史 camelCase 契约）。调用方需在自身项目内声明常量，并通过 Register
//     注入到透传注册表；示例见 gokit/docs/usage-header-jwtauth.md。
//
// 该包位于依赖树最底层，不引用 gokit 内其他业务包，可被 grpcgw / middleware
// 等同时依赖而不产生循环。
package header

import (
	"strings"
	"sync"
)

// —— 标准 HTTP header（Canonical 形式）——
const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"
	HeaderAccept        = "Accept"
	HeaderUserAgent     = "User-Agent"
	HeaderRequestID     = "X-Request-Id"
	HeaderTraceID       = "X-Trace-Id"
	HeaderSpanID        = "X-Span-Id"
	HeaderXForwardedFor = "X-Forwarded-For"
	HeaderXRealIP       = "X-Real-Ip"
	HeaderTenantID      = "X-Tenant-Id"
	HeaderLang          = "Accept-Language"
)

var (
	mu sync.RWMutex
	// keys 保存需要透传到下游（grpc-gateway → grpc）的自定义 header key 集合。
	// 默认为空——所有业务 header 由调用方通过 Register 注入。
	keys []string
)

// Keys 返回当前注册的 header key 集合的副本（并发安全）。
func Keys() []string {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]string, len(keys))
	copy(out, keys)
	return out
}

// Register 追加自定义 header key 到全局注册表（去重，大小写不敏感）。并发安全。
func Register(customKeys ...string) {
	if len(customKeys) == 0 {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	exists := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		exists[strings.ToLower(k)] = struct{}{}
	}
	for _, k := range customKeys {
		if k == "" {
			continue
		}
		lk := strings.ToLower(k)
		if _, ok := exists[lk]; ok {
			continue
		}
		exists[lk] = struct{}{}
		keys = append(keys, k)
	}
}

// Set 全量替换 header key 注册表。传入 nil 或空切片将清空注册表。并发安全。
func Set(newKeys []string) {
	mu.Lock()
	defer mu.Unlock()
	next := make([]string, 0, len(newKeys))
	seen := make(map[string]struct{}, len(newKeys))
	for _, k := range newKeys {
		if k == "" {
			continue
		}
		lk := strings.ToLower(k)
		if _, ok := seen[lk]; ok {
			continue
		}
		seen[lk] = struct{}{}
		next = append(next, k)
	}
	keys = next
}

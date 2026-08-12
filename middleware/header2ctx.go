package middleware

import (
	"net/http"
	"strings"

	"github.com/bitdlv/gokit/header"
	"google.golang.org/grpc/metadata"
)

// Header2ContextHandle 返回一个 HTTP 中间件，将指定 header 注入到 gRPC metadata。
//
// keys 为空时使用 header.Keys() 的运行时快照（默认业务 header + 用户注册的自定义 header）；
// 传入自定义 keys 则以调用方给定集合为准。所有 key 在写入 metadata 前统一转小写以匹配 gRPC 规范。
//
// Usage:
//
//	// 使用全局注册表
//	handler := middleware.Header2ContextHandle()
//	// 覆盖为自定义 key 集合
//	handler := middleware.Header2ContextHandle("X-Tenant-Id", "X-Request-Id")
func Header2ContextHandle(keys ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			active := keys
			if len(active) == 0 {
				active = header.Keys()
			}
			md := metadata.MD{}
			for _, key := range active {
				val := r.Header.Get(key)
				if val == "" {
					continue
				}
				md[strings.ToLower(key)] = []string{val}
			}
			next(w, r.WithContext(metadata.NewIncomingContext(r.Context(), md)))
		}
	}
}

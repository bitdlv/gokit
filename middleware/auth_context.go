package middleware

import (
	"context"
	"net/http"
)

type authenticatedContextKey struct{}

// IsAuthenticated 报告请求是否通过了 JwtAuthMiddle 的其中一条鉴权分支。
// 只有身份 header 不足以判定——它们可能被不受信任的客户端伪造。
func IsAuthenticated(ctx context.Context) bool {
	authenticated, _ := ctx.Value(authenticatedContextKey{}).(bool)
	return authenticated
}

// withAuthenticatedUser 在请求上下文中标记"已通过鉴权"。
func withAuthenticatedUser(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authenticatedContextKey{}, true))
}

package middleware

import (
	"net/http"

	"github.com/google/uuid"
)

const (
	RequestIdKey    = "X-Request-Id"
	LogRequestIdKey = "request-id"
)

func RequestIdMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestId := r.Header.Get(RequestIdKey)
		if requestId == "" {
			requestId = uuid.New().String()
		}
		w.Header().Set(RequestIdKey, requestId)
		next(w, r.WithContext(r.Context()))
	}
}

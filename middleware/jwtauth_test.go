package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bitdlv/gokit/jwt"
	"github.com/bitdlv/gokit/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// —— 测试用桩 ——

type stubUser struct {
	id, uname, phone, bmpID, bmpAcc string
	enabled                         bool
}

func (u *stubUser) GetID() string         { return u.id }
func (u *stubUser) GetUserName() string   { return u.uname }
func (u *stubUser) GetPhone() string      { return u.phone }
func (u *stubUser) GetBmpID() string      { return u.bmpID }
func (u *stubUser) GetBmpAccount() string { return u.bmpAcc }
func (u *stubUser) IsEnabled() bool       { return u.enabled }

type stubLoader struct {
	byID      map[string]*stubUser
	byPhone   map[string]*stubUser
	byAccount map[string]*stubUser
}

func (l *stubLoader) LoadByID(_ context.Context, id string) (middleware.AuthUser, error) {
	if u, ok := l.byID[id]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (l *stubLoader) LoadByPhone(_ context.Context, p string) (middleware.AuthUser, error) {
	if u, ok := l.byPhone[p]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}
func (l *stubLoader) LoadByAccount(_ context.Context, a string) (middleware.AuthUser, error) {
	if u, ok := l.byAccount[a]; ok {
		return u, nil
	}
	return nil, errors.New("not found")
}

func newHandler(mw *middleware.JwtAuthMiddle, captured *http.Request) http.HandlerFunc {
	return mw.Handle(func(w http.ResponseWriter, r *http.Request) {
		*captured = *r
		w.WriteHeader(http.StatusOK)
	})
}

// idxCfg 模拟 idx 项目的 header 契约（camelCase 历史契约）。
func idxCfg(secret string) middleware.JwtAuthConfig {
	return middleware.JwtAuthConfig{
		Secret:      secret,
		TokenHeader: "token",
		IdentityHeaders: middleware.IdentityHeaderMap{
			UserID:     "userId",
			UserName:   "username",
			Phone:      "userPhone",
			BmpID:      "bpmUserId",
			BmpAccount: "bpmUserName",
		},
	}
}

// —— 用例 ——

func TestJwtAuth_JWTToken_Success(t *testing.T) {
	u := &stubUser{id: "1001", uname: "alice", phone: "13800000000", enabled: true}
	loader := &stubLoader{byID: map[string]*stubUser{"1001": u}}
	mw := middleware.NewJwtAuth(idxCfg("s3cr3t"), loader)

	token, err := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600, jwt.K("userId", "1001"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("token", token)
	w := httptest.NewRecorder()

	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1001", captured.Header.Get("userId"))
	assert.True(t, middleware.IsAuthenticated(captured.Context()))
}

func TestJwtAuth_JWTToken_UserDisabled(t *testing.T) {
	u := &stubUser{id: "1001", enabled: false}
	loader := &stubLoader{byID: map[string]*stubUser{"1001": u}}
	mw := middleware.NewJwtAuth(idxCfg("s3cr3t"), loader)

	token, _ := jwt.GenerateHS256("s3cr3t", time.Now().Unix(), 3600, jwt.K("userId", "1001"))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("token", token)
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJwtAuth_RawToken(t *testing.T) {
	u := &stubUser{id: "1001", phone: "13812345678", enabled: true}
	loader := &stubLoader{byPhone: map[string]*stubUser{"13812345678": u}}
	mw := middleware.NewJwtAuth(idxCfg("s"), loader)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("token", "Raw 13812345678")
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "13812345678", captured.Header.Get("userPhone"))
}

func TestJwtAuth_InnerSign(t *testing.T) {
	u := &stubUser{id: "1002", enabled: true}
	loader := &stubLoader{byAccount: map[string]*stubUser{"svc-a": u}}
	cfg := idxCfg("")
	cfg.JwtAuthOpen = true
	cfg.InnerAPIKey = "k"
	cfg.InnerSalt = "sa"
	cfg.ServiceName = "idx"
	mw := middleware.NewJwtAuth(cfg, loader)

	sig := middleware.GenerateInnerSign("idx", "svc-a", "sa", "k")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("sign", sig)
	req.Header.Set("userAccount", "svc-a")
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "1002", captured.Header.Get("userId"))
	assert.True(t, middleware.IsAuthenticated(captured.Context()))
}

func TestJwtAuth_InnerSign_Invalid(t *testing.T) {
	loader := &stubLoader{}
	cfg := idxCfg("")
	cfg.JwtAuthOpen = true
	cfg.InnerAPIKey = "k"
	cfg.InnerSalt = "sa"
	mw := middleware.NewJwtAuth(cfg, loader)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("sign", "bad")
	req.Header.Set("userAccount", "svc-a")
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJwtAuth_ExternalSign(t *testing.T) {
	loader := &stubLoader{}
	verified := false
	cfg := idxCfg("")
	cfg.SignAuthOpen = true
	mw := middleware.NewJwtAuth(
		cfg,
		loader,
		middleware.WithSignVerifier(func(r *http.Request) bool { verified = true; return true }),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, verified)
	assert.False(t, middleware.IsAuthenticated(captured.Context()))
}

func TestJwtAuth_NoAuth_Passthrough(t *testing.T) {
	loader := &stubLoader{}
	mw := middleware.NewJwtAuth(idxCfg(""), loader)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	var captured http.Request
	newHandler(mw, &captured)(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, middleware.IsAuthenticated(captured.Context()))
}

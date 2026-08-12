package grpcgw

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/bitdlv/gokit/result"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"gorm.io/gorm"
)

// ─────────────────────── Header Constants ───────────────────────────────────

const (
	AuthToken         = "token"
	HeaderUserId      = "userId"
	HeaderUserName    = "username"
	HeaderPhone       = "userPhone"
	HeaderBpmUid      = "bpmUserId"
	HeaderBpmUname    = "bpmUserName"
	HeaderProjectCode = "pcode" // Header 头中项目编码
)

// DefaultHeaderKeys is the default set of header keys to pass through the gateway.
var DefaultHeaderKeys = []string{
	HeaderBpmUid,
	HeaderBpmUname,
	HeaderUserId,
	HeaderPhone,
	HeaderUserName,
	HeaderProjectCode,
}

// ─────────────────────── Header Matcher ─────────────────────────────────────

// NewHeaderMatcher returns a grpc-gateway IncomingHeaderMatcher that passes
// through all keys listed in extraKeys (case-insensitive) and falls back to
// the default runtime matcher for anything else.
//
// Usage:
//
//	matcher := grpcgw.NewHeaderMatcher([]string{"userId", "username", "userPhone", "bpmUserId", "bpmUserName", "pcode"})
func NewHeaderMatcher(extraKeys []string) func(string) (string, bool) {
	lower := make(map[string]string, len(extraKeys))
	for _, k := range extraKeys {
		lower[strings.ToLower(k)] = strings.ToLower(k)
	}
	return func(key string) (string, bool) {
		if v, ok := lower[strings.ToLower(key)]; ok {
			return v, true
		}
		return runtime.DefaultHeaderMatcher(key)
	}
}

// ─────────────────────── Error Handler ───────────────────────────────────────

// ErrorHandlerFunc is the signature for a custom gRPC-gateway error handler.
type ErrorHandlerFunc func(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error)

// DefaultErrorHandler uses common/result.Response to write error responses.
func DefaultErrorHandler(ctx context.Context, mux *runtime.ServeMux, m runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}
	result.Response(r, w, nil, err)
}

// ─────────────────────── Custom Marshaler ────────────────────────────────────

// WrappedMarshaler wraps gRPC-gateway JSON responses with {"code":200,"msg":"OK","data":...}.
//
// DB is an optional database connection that can be used by a ResponseProcessor
// registered via WithResponseProcessor.
type WrappedMarshaler struct {
	runtime.JSONPb

	// DB is an optional database connection made available to ResponseProcessor.
	DB *gorm.DB
}

func (c *WrappedMarshaler) Marshal(v interface{}) ([]byte, error) {
	b, err := c.JSONPb.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	err = c.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return json.Marshal(map[string]any{"code": 200, "msg": "OK", "data": m})
}

// ─────────────────────── Response Processor ──────────────────────────────────

// ResponseProcessor is a hook invoked after every successful response.
// It receives the originating HTTP request (giving access to headers, URL, etc.),
// an optional *gorm.DB, and the decoded response data map.
// The returned map is re-encoded and sent to the caller.
//
// Typical use-cases: field masking / desensitization based on user role or request path.
//
// Example:
//
//	func myProcessor(r *http.Request, db *gorm.DB, data map[string]any) map[string]any {
//	    role := r.Header.Get("X-User-Role")
//	    if role != "admin" {
//	        if phone, ok := data["phone"].(string); ok {
//	            data["phone"] = MaskPhone(phone)
//	        }
//	    }
//	    return data
//	}
type ResponseProcessor func(r *http.Request, db *gorm.DB, data map[string]any) map[string]any

// ─────────────────────── Response Interceptor Middleware ─────────────────────

// bufferedResponseWriter buffers the response body so the processor can modify it
// before it is sent to the client.
type bufferedResponseWriter struct {
	http.ResponseWriter
	buf    bytes.Buffer
	status int
}

func (b *bufferedResponseWriter) WriteHeader(status int) {
	b.status = status
}

func (b *bufferedResponseWriter) Write(p []byte) (int, error) {
	return b.buf.Write(p)
}

// newResponseProcessorMiddleware returns an HTTP middleware that:
//  1. Buffers the downstream response body.
//  2. Parses the JSON envelope produced by WrappedMarshaler.
//  3. Calls processor on the "data" field (with full request access).
//  4. Re-encodes and writes the modified response.
func newResponseProcessorMiddleware(db *gorm.DB, processor ResponseProcessor) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			brw := &bufferedResponseWriter{ResponseWriter: w, status: http.StatusOK}
			next(brw, r)

			body := brw.buf.Bytes()

			// Only process JSON responses; pass others through as-is.
			var envelope map[string]any
			if err := json.Unmarshal(body, &envelope); err != nil {
				w.WriteHeader(brw.status)
				_, _ = w.Write(body)
				return
			}

			// Apply the processor only to the "data" field of the envelope.
			if data, ok := envelope["data"].(map[string]any); ok {
				envelope["data"] = processor(r, db, data)
			}

			modified, err := json.Marshal(envelope)
			if err != nil {
				modified = body
			}

			// Remove Content-Length because the body length may have changed.
			w.Header().Del("Content-Length")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(brw.status)
			_, _ = w.Write(modified)
		}
	}
}

// ─────────────────────── Options ────────────────────────────────────────────

// Option configures NewGatewayMux behavior.
type Option func(*gatewayConfig)

type gatewayConfig struct {
	headerMatcher     func(string) (string, bool)
	errorHandler      ErrorHandlerFunc
	marshaler         runtime.Marshaler
	db                *gorm.DB
	responseProcessor ResponseProcessor
}

// WithHeaderMatcher sets a custom IncomingHeaderMatcher.
func WithHeaderMatcher(fn func(string) (string, bool)) Option {
	return func(c *gatewayConfig) { c.headerMatcher = fn }
}

// WithErrorHandler sets a custom error handler.
func WithErrorHandler(fn ErrorHandlerFunc) Option {
	return func(c *gatewayConfig) { c.errorHandler = fn }
}

// WithMarshaler sets a custom marshaler.
func WithMarshaler(m runtime.Marshaler) Option {
	return func(c *gatewayConfig) { c.marshaler = m }
}

// WithDB injects a *gorm.DB that is passed to the ResponseProcessor.
// Has no effect when a fully custom marshaler is set via WithMarshaler.
func WithDB(db *gorm.DB) Option {
	return func(c *gatewayConfig) { c.db = db }
}

// WithResponseProcessor registers a hook called for every successful response.
// The hook receives the full *http.Request so headers, URL, query params, etc.
// are all accessible — enabling request-aware transformations such as role-based
// field masking.
//
// The middleware is automatically prepended to the middleware chain inside Handler
// and DefaultHandler.  When using NewGatewayMux directly, wrap the returned mux
// manually with newResponseProcessorMiddleware.
//
// Example:
//
//	grpcgw.WithResponseProcessor(func(r *http.Request, db *gorm.DB, data map[string]any) map[string]any {
//	    if r.Header.Get("X-User-Role") != "admin" {
//	        if phone, ok := data["phone"].(string); ok {
//	            data["phone"] = MaskPhone(phone)
//	        }
//	    }
//	    return data
//	})
func WithResponseProcessor(fn ResponseProcessor) Option {
	return func(c *gatewayConfig) { c.responseProcessor = fn }
}

func defaultConfig() *gatewayConfig {
	return &gatewayConfig{
		headerMatcher: NewHeaderMatcher(DefaultHeaderKeys),
		errorHandler:  DefaultErrorHandler,
		marshaler: &WrappedMarshaler{
			JSONPb: runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{
					EmitUnpopulated: true,
				},
				UnmarshalOptions: protojson.UnmarshalOptions{
					DiscardUnknown: true,
				},
			},
		},
	}
}

// ─────────────────────── Middleware Helper ───────────────────────────────────

// WithMiddlewares wraps an http.Handler with a chain of middlewares (applied in order).
func WithMiddlewares(handler http.Handler, middlewares ...func(http.HandlerFunc) http.HandlerFunc) http.Handler {
	if len(middlewares) == 0 {
		return handler
	}
	var hf http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}
	for _, m := range middlewares {
		hf = m(hf)
	}
	return hf
}

// ─────────────────────── Gateway Mux ────────────────────────────────────────

// NewGatewayMux builds a grpc-gateway runtime.ServeMux with all services registered.
// It returns the raw mux without any HTTP middleware wrapping.
// Note: a ResponseProcessor registered via WithResponseProcessor is NOT applied here;
// use Handler / DefaultHandler to get automatic processor middleware injection.
func NewGatewayMux(ctx context.Context, grpcEndpoint string, opts []Option, registerFuncs ...func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error) (*runtime.ServeMux, error) {
	cfg := defaultConfig()
	for _, o := range opts {
		o(cfg)
	}

	// Inject DB into the default WrappedMarshaler when no custom marshaler was provided.
	if wm, ok := cfg.marshaler.(*WrappedMarshaler); ok && cfg.db != nil {
		wm.DB = cfg.db
	}

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(cfg.headerMatcher),
		runtime.WithErrorHandler(runtime.ErrorHandlerFunc(cfg.errorHandler)),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, cfg.marshaler),
	)

	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, reg := range registerFuncs {
		if err := reg(ctx, mux, grpcEndpoint, dialOpts); err != nil {
			return nil, err
		}
	}
	return mux, nil
}

// ─────────────────────── Service Builder ────────────────────────────────────

// ServiceRegistration wraps a grpc-gateway register function with optional per-service middlewares.
type ServiceRegistration struct {
	registerFunc func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
	middlewares  []func(http.HandlerFunc) http.HandlerFunc
}

// Service creates a ServiceRegistration for use with Handler.
func Service(fn func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error) ServiceRegistration {
	return ServiceRegistration{registerFunc: fn}
}

// With attaches per-service middlewares to this ServiceRegistration.
func (s ServiceRegistration) With(mws ...func(http.HandlerFunc) http.HandlerFunc) ServiceRegistration {
	s.middlewares = append(s.middlewares, mws...)
	return s
}

// ─────────────────────── Gateway ────────────────────────────────────────────

// Gateway holds both the raw gateway mux and the middleware-wrapped handler.
//
//   - Mux:     raw grpc-gateway mux, use for routes that bypass the middleware chain
//   - Handler: gateway mux wrapped with the shared middleware chain
type Gateway struct {
	Mux     http.Handler // raw gateway mux without middleware
	Handler http.Handler // gateway mux wrapped with middleware chain
}

// Handler builds a Gateway from ServiceRegistration entries with a shared middleware chain.
// If a ResponseProcessor is configured via WithResponseProcessor, it is automatically
// prepended to the middleware chain so every response body can be transformed with
// full access to the originating *http.Request.
//
// Usage:
//
//	gw, err := grpcgw.Handler(ctx, endpoint,
//	    []grpcgw.Option{
//	        grpcgw.WithDB(db),
//	        grpcgw.WithResponseProcessor(myProcessor),
//	    },
//	    []func(http.HandlerFunc) http.HandlerFunc{jwtAuth, header2Ctx},
//	    grpcgw.Service(pb.RegisterSysServiceHandlerFromEndpoint),
//	)
func Handler(ctx context.Context, grpcEndpoint string, gwOpts []Option, middlewares []func(http.HandlerFunc) http.HandlerFunc, services ...ServiceRegistration) (*Gateway, error) {
	// Resolve config here so we can access responseProcessor for middleware injection.
	cfg := defaultConfig()
	for _, o := range gwOpts {
		o(cfg)
	}

	fns := make([]func(ctx context.Context, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error, len(services))
	for i, svc := range services {
		fns[i] = svc.registerFunc
	}

	// Inject DB into WrappedMarshaler.
	if wm, ok := cfg.marshaler.(*WrappedMarshaler); ok && cfg.db != nil {
		wm.DB = cfg.db
	}

	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(cfg.headerMatcher),
		runtime.WithErrorHandler(runtime.ErrorHandlerFunc(cfg.errorHandler)),
		runtime.WithMarshalerOption(runtime.MIMEWildcard, cfg.marshaler),
	)
	dialOpts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	for _, reg := range fns {
		if err := reg(ctx, mux, grpcEndpoint, dialOpts); err != nil {
			return nil, err
		}
	}

	// Prepend the response processor middleware so it wraps all other middlewares
	// and therefore has access to the unmodified request headers.
	allMiddlewares := middlewares
	if cfg.responseProcessor != nil {
		interceptor := newResponseProcessorMiddleware(cfg.db, cfg.responseProcessor)
		allMiddlewares = append([]func(http.HandlerFunc) http.HandlerFunc{interceptor}, middlewares...)
	}

	return &Gateway{
		Mux:     mux,
		Handler: WithMiddlewares(mux, allMiddlewares...),
	}, nil
}

// ─────────────────────── Default Handler (convenience) ──────────────────────

// DefaultHandler builds a Gateway using built-in defaults (DefaultHeaderKeys, DefaultErrorHandler, WrappedMarshaler).
// This is a convenience wrapper so callers don't need to pass []Option.
//
// Usage:
//
//	gw, err := grpcgw.DefaultHandler(ctx, endpoint,
//	    []func(http.HandlerFunc) http.HandlerFunc{jwtAuth, header2Ctx},
//	    grpcgw.Service(pb.RegisterSysServiceHandlerFromEndpoint),
//	    grpcgw.Service(pb.RegisterBOMServiceHandlerFromEndpoint),
//	)
func DefaultHandler(ctx context.Context, grpcEndpoint string, middlewares []func(http.HandlerFunc) http.HandlerFunc, services ...ServiceRegistration) (*Gateway, error) {
	return Handler(ctx, grpcEndpoint, nil, middlewares, services...)
}

func HandleNonAsciiHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("operatorNameAscii", url.QueryEscape(r.Header.Get("operatorName")))
		next(w, r)
	}
}

package es

import (
	"net"
	"net/http"
	"strings"
	"time"

	es7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/estransport"
	"github.com/zeromicro/go-zero/core/trace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type (
	Config struct {
		Addresses       []string
		Username        string
		Password        string
		MaxRetries      int
		MaxIdleConns    int           // 全局最大空闲连接数
		MaxConnsPerHost int           // 每主机最大连接数
		IdleConnTimeout time.Duration // 空闲连接超时时间
	}

	Es struct {
		*es7.Client
	}

	// esTransport is a transport for elasticsearch client
	esTransport struct {
		baseTransport *http.Transport
	}
)

func (t *esTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	var (
		ctx  = req.Context()
		span oteltrace.Span
		//startTime  = time.Now()
		propagator = otel.GetTextMapPropagator()
		indexName  = strings.Split(req.URL.RequestURI(), "/")[1]
		tracer     = trace.TracerFromContext(ctx)
	)

	ctx, span = tracer.Start(ctx,
		req.URL.Path,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(semconv.HTTPClientAttributesFromHTTPRequest(req)...),
	)
	defer func() {
		//metric
		//metricClientReqDur.Observe(time.Since(startTime).Milliseconds(), indexName)
		//metricClientReqErrTotal.Inc(indexName, strconv.FormatBool(err != nil))

		span.End()
	}()

	req = req.WithContext(ctx)
	propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))

	// 调用基础 Transport 执行请求
	resp, err = t.baseTransport.RoundTrip(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}

	span.SetAttributes(semconv.DBSQLTableKey.String(indexName))
	span.SetAttributes(semconv.HTTPAttributesFromHTTPStatusCode(resp.StatusCode)...)
	span.SetStatus(semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(resp.StatusCode, oteltrace.SpanKindClient))

	return
}

func NewEs(conf *Config) (*Es, error) {
	transport := &http.Transport{
		MaxIdleConns:        conf.MaxIdleConns,    // 全局最大空闲连接数
		MaxIdleConnsPerHost: conf.MaxConnsPerHost, // 每主机最大空闲连接数
		MaxConnsPerHost:     conf.MaxConnsPerHost, // 每主机最大连接数
		IdleConnTimeout:     conf.IdleConnTimeout, // 空闲连接超时时间
		DialContext: (&net.Dialer{
			Timeout:   3 * time.Second, // 建立连接超时时间
			KeepAlive: time.Hour,       // 保持活动连接的时间
		}).DialContext,
		TLSHandshakeTimeout: 10 * time.Second, // TLS 握手超时时间
	}

	// 自定义连接池函数
	// 作用
	// 1. 多节点请求分发
	// 2. 负载均衡
	// 3. 故障节点管理
	customConnectionPoolFunc := func(addrs []*estransport.Connection, selector estransport.Selector) estransport.ConnectionPool {
		// 使用 RoundRobinConnectionPool（轮询连接池）
		cp, err := estransport.NewConnectionPool(addrs, selector)
		if err != nil {
			panic(err)
		}

		return cp
	}

	c := es7.Config{
		Addresses:          conf.Addresses,
		Username:           conf.Username,
		Password:           conf.Password,
		MaxRetries:         conf.MaxRetries,
		Transport:          &esTransport{baseTransport: transport},
		ConnectionPoolFunc: customConnectionPoolFunc,
	}

	client, err := es7.NewClient(c)
	if err != nil {
		return nil, err
	}

	return &Es{
		Client: client,
	}, nil
}

func MustNewEs(conf *Config) *Es {
	es, err := NewEs(conf)
	if err != nil {
		panic(err)
	}

	return es
}

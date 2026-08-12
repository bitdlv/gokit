package httpx

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/go-playground/form/v4"
	"github.com/gogf/gf/v2/util/gconv"
)

type IClient interface {
	SetHeader(header http.Header) IClient
	AddHeader(key string, value string) IClient
	Debug() IClient
	SetTimeout(d time.Duration) IClient
	SetQuery(values url.Values) IClient
	AddQuery(key string, value string) IClient
	SetCookies(cookies []*http.Cookie) IClient
	AddCookie(cookie *http.Cookie) IClient
	Get(url string) iResponse
	Delete(url string) iResponse
	PutJson(url string, data any) iResponse
	PutXml(url string, data any) iResponse
	PostJson(url string, data any) iResponse
	PostXml(url string, data any) iResponse
	PostForm(u string, data any) iResponse
	Request(req *http.Request) (resp *http.Response, err error)
}

type iResponse interface {
	Response() (res *http.Response, err error)
	ScanJsonBody(data any) (err error)
	ScanXmlBody(data any) (err error)
}

type OptFunc func(*client)

func WithBaseUrl(baseUrl string) OptFunc {
	return func(client *client) {
		client.BaseUrl = baseUrl
	}
}

func WithTimeout(d time.Duration) OptFunc {
	return func(c *client) {
		c.timeout = d
	}
}

func NewClient(opts ...OptFunc) IClient {
	c := &client{
		header:  make(http.Header),
		query:   make(url.Values),
		cookies: make([]*http.Cookie, 0),
	}
	for _, opt := range opts {
		opt(c)
	}

	return c
}

type client struct {
	BaseUrl string
	debug   bool
	header  http.Header
	timeout time.Duration
	query   url.Values
	res     *http.Response
	Err     error
	cookies []*http.Cookie
}

func (c client) SetHeader(header http.Header) IClient {
	c.header = header
	return &c
}

func (c client) AddHeader(key string, value string) IClient {
	c.header.Add(key, value)
	return &c
}

func (c client) Debug() IClient {
	c.debug = true
	return &c
}

func (c client) SetTimeout(d time.Duration) IClient {
	c.timeout = d
	return &c
}

func (c client) SetQuery(values url.Values) IClient {
	c.query = values
	return &c
}

func (c client) AddQuery(key string, value string) IClient {
	c.query.Add(key, value)
	return &c
}

func (c client) SetCookies(cookies []*http.Cookie) IClient {
	c.cookies = cookies
	return &c
}

func (c client) AddCookie(cookie *http.Cookie) IClient {
	c.cookies = append(c.cookies, cookie)
	return &c
}

func (c client) Get(url string) iResponse {
	c.get(url)
	return &c
}

func (c *client) get(url string) {
	var req *http.Request
	req, c.Err = c.makeRequest(http.MethodGet, url, nil)
	if c.Err != nil {
		return
	}

	c.res, c.Err = c.Request(req)
}

func (c client) Delete(url string) iResponse {
	c.delete(url)
	return &c
}

func (c *client) delete(url string) {
	var req *http.Request
	req, c.Err = c.makeRequest(http.MethodDelete, url, nil)
	if c.Err != nil {
		return
	}

	c.res, c.Err = c.Request(req)
}

func (c client) PutJson(url string, data any) iResponse {
	var (
		content []byte
	)
	content, c.Err = json.Marshal(data)
	if c.Err != nil {
		return &c
	}
	c.header.Set("Content-Type", "application/json")
	c.put(url, bytes.NewReader(content))
	return &c
}

func (c client) PutXml(url string, data any) iResponse {
	var content []byte
	content, c.Err = xml.Marshal(data)
	if c.Err != nil {
		return &c
	}
	c.header.Set("Content-Type", "text/xml")
	c.put(url, bytes.NewReader(content))
	return &c
}

func (c *client) put(url string, body io.Reader) {
	var req *http.Request
	req, c.Err = c.makeRequest(http.MethodPut, url, body)
	if c.Err != nil {
		return
	}

	c.res, c.Err = c.Request(req)
}

func (c client) PostJson(url string, data any) iResponse {
	var content []byte
	content, c.Err = json.Marshal(data)
	if c.Err != nil {
		return &c
	}

	c.header.Set("Content-Type", "application/json")
	c.post(url, bytes.NewReader(content))
	return &c
}

func (c client) PostXml(url string, data any) iResponse {
	var content []byte
	content, c.Err = xml.Marshal(data)
	if c.Err != nil {
		return &c
	}
	c.header.Set("Content-Type", "text/xml")
	c.post(url, bytes.NewReader(content))
	return &c
}

func (c client) PostForm(u string, data any) iResponse {
	c.header.Set("Content-Type", "application/x-www-form-urlencoded")
	vData := reflect.ValueOf(data)
	if vData.Kind() == reflect.Pointer {
		vData = vData.Elem()
	}
	if vData.Kind() == reflect.String {
		c.post(u, strings.NewReader(vData.Interface().(string)))
		return &c
	}

	if vData.Kind() != reflect.Struct {
		c.Err = errors.New("unsupported type of data")
		return &c
	}

	tData := reflect.TypeOf(data)
	if tData.Kind() == reflect.Pointer {
		tData = tData.Elem()
	}

	values := url.Values{}
	for i := 0; i < vData.NumField(); i++ {
		fieldValue := vData.Field(i)
		if fieldValue.IsValid() {
			continue
		}
		fieldType := tData.Field(i)

		var (
			key   string
			value string
		)
		parts := strings.Split(fieldType.Tag.Get("form"), ",")
		if len(parts) > 0 {
			key = parts[0]
		} else {
			key = fieldType.Name
		}

		value = gconv.String(fieldValue.Interface())

		values.Add(key, value)
	}

	c.post(u, strings.NewReader(values.Encode()))
	return &c
}

func (c *client) post(url string, body io.Reader) {
	req, err := c.makeRequest(http.MethodPost, url, body)
	if err != nil {
		return
	}

	c.res, c.Err = c.Request(req)
}

func (c client) buildUrl(u string) string {
	if !strings.HasPrefix(u, "http") {
		u = fmt.Sprintf(
			"%s/%s",
			strings.TrimRight(c.BaseUrl, "/"),
			strings.TrimLeft(u, "/"),
		)
	}

	var parsedUrl *url.URL
	parsedUrl, c.Err = url.Parse(u)
	if c.Err != nil {
		return ""
	}

	for k, values := range c.query {
		for _, v := range values {
			q := parsedUrl.Query()
			q.Add(k, v)
			parsedUrl.RawQuery = q.Encode()
		}
	}

	return parsedUrl.String()
}

func (c *client) makeRequest(method string, url string, body io.Reader) (req *http.Request, err error) {
	req, err = http.NewRequest(method, c.buildUrl(url), body)
	if err != nil {
		return
	}

	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}

	for k, values := range c.header {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}

	return
}

func (c *client) Request(req *http.Request) (resp *http.Response, err error) {
	client := http.Client{
		Timeout: c.timeout,
	}

	var start time.Time
	var content []byte
	if c.debug {
		if req.Body != nil {
			content, err = io.ReadAll(req.Body)
			if err != nil {
				return
			}
			req.Body.Close()
			req.Body = io.NopCloser(bytes.NewReader(content))
		}
		start = time.Now()
	}

	resp, err = client.Do(req)
	if err != nil {
		return
	}

	if c.debug {
		var respContent []byte
		respContent, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return
		}
		resp.Body = io.NopCloser(bytes.NewReader(respContent))
		fmt.Printf(
			"[request: method=%s, url=%s, body=%s, header=%v]\n[request time: %fs]\n[response: body=%s]\n",
			req.Method,
			req.URL.String(),
			string(content),
			req.Header,
			time.Since(start).Seconds(),
			string(respContent),
		)
	}

	c.header = make(http.Header)

	return
}

func (c client) Response() (res *http.Response, err error) {
	return c.res, c.Err
}

func (c client) ScanJsonBody(data any) (err error) {
	if c.Err != nil {
		return c.Err
	}
	defer c.res.Body.Close()
	content, err := io.ReadAll(c.res.Body)
	if err != nil {
		return
	}
	err = json.Unmarshal(content, data)
	return
}

func (c client) ScanXmlBody(data any) (err error) {
	if c.Err != nil {
		return c.Err
	}
	defer c.res.Body.Close()
	content, err := io.ReadAll(c.res.Body)
	if err != nil {
		return
	}
	err = xml.Unmarshal(content, data)
	return
}

func ParseParams(r *http.Request) (map[string]interface{}, error) {
	params := make(map[string]interface{})
	// 1. GET 请求参数（QueryString）
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			params[k] = v[0]
		} else {
			params[k] = v
		}
	}

	// 2. 读取 body（备份后恢复）
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		// 恢复 r.Body，后续流程还能继续使用
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}

	// 3. 根据 Content-Type 处理
	contentType := r.Header.Get("Content-Type")

	switch {
	case strings.HasPrefix(contentType, "application/json"):
		if len(bodyBytes) > 0 {
			var jsonParams map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &jsonParams); err != nil {
				return nil, err
			}
			for k, v := range jsonParams {
				params[k] = v
			}
		}

	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
		if err := r.ParseForm(); err != nil {
			return nil, err
		}
		for k, v := range r.Form {
			if len(v) == 1 {
				params[k] = v[0]
			} else {
				params[k] = v
			}
		}

	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return nil, err
		}
		for k, v := range r.MultipartForm.Value {
			if len(v) == 1 {
				params[k] = v[0]
			} else {
				params[k] = v
			}
		}
		for k, v := range r.MultipartForm.File {
			params[k] = v
		}

	default:
		if len(bodyBytes) > 0 {
			params["raw"] = string(bodyBytes)
		}
	}

	return params, nil
}

var decoder = form.NewDecoder()

// Parse 自动解析 HTTP 请求参数并绑定到 resp 结构体中
func XParse(r *http.Request, resp any) error {
	var bodyBytes []byte
	if r.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err != nil {
			return err
		}
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	contentType := r.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(contentType, "application/json"):
		if len(bodyBytes) > 0 {
			if err := json.Unmarshal(bodyBytes, resp); err != nil {
				return err
			}
		}
	case strings.HasPrefix(contentType, "application/x-www-form-urlencoded"):
		if err := r.ParseForm(); err != nil {
			return err
		}
		if err := decoder.Decode(resp, r.Form); err != nil {
			return err
		}
	case strings.HasPrefix(contentType, "multipart/form-data"):
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB
			return err
		}
		formVals := url.Values{}
		for k, v := range r.MultipartForm.Value {
			for _, vv := range v {
				formVals.Add(k, vv)
			}
		}
		if err := decoder.Decode(resp, formVals); err != nil {
			return err
		}
	default:
		if len(r.URL.Query()) > 0 {
			if err := decoder.Decode(resp, r.URL.Query()); err != nil {
				return err
			}
		} else if len(bodyBytes) > 0 {
			// 非标准 Content-Type，也尝试 JSON
			_ = json.Unmarshal(bodyBytes, resp)
		}
	}
	return nil
}

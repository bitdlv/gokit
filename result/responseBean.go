package result

type ResponseSuccessBean struct {
	Code uint64      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
type NullJson struct{}

func Success(data interface{}) *ResponseSuccessBean {
	return &ResponseSuccessBean{200, "OK", data}
}

type ResponseErrorBean struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
}

func Error(errCode uint64, errMsg string) *ResponseErrorBean {
	return &ResponseErrorBean{errCode, errMsg}
}

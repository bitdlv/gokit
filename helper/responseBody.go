package helper

type LogicResponse struct {
	Code    uint64      `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func FormatResponseBody(data interface{}) *LogicResponse {
	return &LogicResponse{200, "OK", data}
}

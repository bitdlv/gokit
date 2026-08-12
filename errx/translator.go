package errx

const (
	UNKNOWN_ERROR_COMMON int = 100000 // 未知错误
)

var COMMON_MESSAGE_EN map[int]string = map[int]string{
	UNKNOWN_ERROR_COMMON: "common unknow error",
}

var COMMON_MESSAGE_ZH_CN map[int]string = map[int]string{
	UNKNOWN_ERROR_COMMON: "未知错误",
}

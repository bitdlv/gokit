package errx

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"strings"
)

func LimitCanNotLt0() error {
	return NewWithPos(INVALID_REQUEST, "limit 不能小于0", 1)
}

func LimitCanNotLtMinus1() error {
	return NewWithPos(INVALID_REQUEST, "limit 不能小于-1", 1)
}

func OffsetCanNotLt0() error {
	return NewWithPos(INVALID_REQUEST, "offset 不能小于0", 1)
}

func QueryTotalFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "查询记录总数失败", 1)
}

func QueryFailed(err error, msg ...string) error {
	message := "查询记录失败"
	if len(msg) > 0 {
		message = strings.Join(msg, ",")
	}
	return wrap(err, DB_QUERY_FAILED, message, 1)
}

func ThirdPartyServiceFailure(err error, message ...string) error {
	msg := strings.Join(message, "")
	if msg == "" {
		msg = MapErrMsg(THIRD_PARTY_SERVICE_FAILURE)
	}
	return wrap(err, THIRD_PARTY_SERVICE_FAILURE, msg, 1)
}

func FileCanNotGt(m int) error {
	return NewWithPos(INVALID_REQUEST, fmt.Sprintf("文件不能大于%dM", m), 1)
}

func GenFileFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "生成文件失败", 1)
}

func FileImportFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "文件导入失败", 1)
}

func SaveFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "记录保存失败", 1)
}

func JsonEncodeFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "json编码失败", 1)
}

func JsonDecodeFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "json解码失败", 1)
}

func DeleteFailed(err error) error {
	return wrap(err, INTERNAL_SERVER_ERROR, "删除记录失败", 1)
}

func Conflict(msg string) error {
	if msg == "" {
		msg = MapErrMsg(INVALID_REQUEST)
	}
	return NewWithPos(INVALID_REQUEST, msg, 1)
}

func Timeout(msg string) error {
	if msg == "" {
		msg = "超时"
	}
	return NewWithPos(codes.DeadlineExceeded, msg, 1)
}

func NotFound(msg string) error {
	if msg == "" {
		msg = "资源未找到"
	}
	return NewWithPos(RESOURCE_NOT_FOUND, msg, 1)
}

func InvalidRequest(msg string) error {
	return NewWithPos(INVALID_REQUEST, msg, 1)
}

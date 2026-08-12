package legacy

import (
	"fmt"
	"regexp"
	"runtime"
	"strconv"
)

const (
	skip = 2
)

type Error interface {
	error
	GetErrCode() uint64
	GetErrMsg() string
	Unwrap() error
}

var _ Error = (*CodeError)(nil)

// CodeError 常用通用固定错误
type CodeError struct {
	errCode uint64
	errMsg  string
	err     error
	file    string
	line    int
}

// GetErrCode 返回给前端的错误码
func (e CodeError) GetErrCode() uint64 {
	return e.errCode
}

// GetErrMsg 返回给前端显示端错误信息
func (e CodeError) GetErrMsg() string {
	return e.errMsg
}

func (e CodeError) Error() string {
	return fmt.Sprintf("ErrCode:%d，ErrMsg:%s", e.errCode, e.errMsg)
}

func (e CodeError) Unwrap() error {
	return e.err
}

func New(errCode uint64, errMsg string, skip int) Error {
	e := &CodeError{errCode: errCode, errMsg: errMsg}
	_, e.file, e.line, _ = runtime.Caller(skip)
	return e
}

func NewErrCodeMsg(errCode uint64, errMsg string) Error {
	return New(errCode, errMsg, skip)
}

func NewErrCode(errCode uint64) Error {
	return New(errCode, MapErrMsg(errCode), skip)
}

func NewErrMsg(errMsg string) Error {
	return New(SERVER_COMMON_ERROR, errMsg, skip)
}

func WrapWithCodeMsg(err error, errCode uint64, errMsg string) Error {
	e := New(errCode, errMsg, skip).(*CodeError)
	e.err = err
	return e
}

func WrapWithCode(err error, errCode uint64) Error {
	e := New(errCode, MapErrMsg(errCode), skip).(*CodeError)
	e.err = err
	return e
}

func WrapWithCodeOriginalMsg(err error, errCode uint64) Error {
	if err == nil {
		return nil
	}
	e := New(errCode, MapErrMsg(errCode)+": "+err.Error(), skip).(*CodeError)
	e.err = err
	return e
}

func WrapWithMsg(err error, errMsg string) Error {
	e := New(SERVER_COMMON_ERROR, errMsg, skip).(*CodeError)
	e.err = err
	return e
}

func C(errCode uint64) *CodeError {
	if errCode == 0 {
		errCode = UNKNOWN_ERROR
	}
	e := &CodeError{errCode: errCode, errMsg: MapErrMsg(errCode)}
	_, e.file, e.line, _ = runtime.Caller(skip)
	return e
}

// E should return Error instead of *CodeError,
// in order to return real nil error
func (e *CodeError) E(err error) Error {
	if err == nil {
		return nil
	}
	e.err = err
	if e.errMsg != "" {
		e.errMsg += ": "
	}
	e.errMsg += e.err.Error()
	return e
}

func (e *CodeError) F(format string, a ...any) Error {
	err := fmt.Errorf(format, a...)
	return e.E(err)
}

func Errorf(errCode uint64, format string, a ...any) Error {
	return WrapWithCodeOriginalMsg(fmt.Errorf(format, a...), errCode)
}

func ParseError(errString string) (code uint64, msg string, err error) {
	// 定义正则表达式模式
	pattern := `ErrCode:(\d+)，ErrMsg:(.+)` // 这里用了中文逗号

	// 编译正则表达式
	re := regexp.MustCompile(pattern)

	// 查找匹配项
	matches := re.FindStringSubmatch(errString)

	// 提取匹配项中的值
	if len(matches) >= 3 {
		codeStr := matches[1]
		msg = matches[2]
		codeInt, _ := strconv.Atoi(codeStr)
		code = uint64(codeInt)
	} else {
		err = fmt.Errorf("ParseError fail for msg [%s]", errString)
	}
	return
}

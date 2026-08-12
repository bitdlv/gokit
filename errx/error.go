package errx

import (
	"fmt"
	"runtime"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/bitdlv/gokit/helper"
	"github.com/bitdlv/gokit/pb"
)

// PrintErrorDetail 打印err的详细信息
func PrintErrorDetail(err error) {
	s, ok := status.FromError(err)
	if !ok {
		println("none status error")
		return
	}

	fmt.Printf("%#v\n", NewFromStatus(s))
}

func wrap(err error, code codes.Code, message string, skip int) error {
	e := NewWithPos(code, message, skip+1)
	tmp := e.(*Error)
	tmp.err = err
	if oriErr, ok := err.(*Error); ok && oriErr.Details["stack"] != "" {
		tmp.Details["stack"] = fmt.Sprintf(
			"%s%s",
			tmp.Details["stack"],
			oriErr.Details["stack"],
		)
	} else {
		tmp.Details["stack"] = fmt.Sprintf(
			"%s%s",
			tmp.Details["stack"],
			strings.Trim(err.Error(), "\n")+"\n",
		)
	}
	return e
}

func Wrap(err error, code codes.Code, message string) error {
	return wrap(err, code, message, 1)
}

func WrapCode(err error, code codes.Code) error {
	return wrap(err, code, MapErrMsg(code), 1)
}

func WrapMsg(err error, message string) error {
	return wrap(err, INTERNAL_SERVER_ERROR, message, 1)
}

// NewFromStatus new an error from status.Status
func NewFromStatus(s *status.Status) error {
	e := NewWithPos(s.Code(), s.Message(), 1)
	if d := s.Details(); len(d) > 0 {
		if details, ok := d[0].(*pb.Map); ok {
			var err error
			e.(*Error).Details, err = helper.PbMap2MapStrAny(details)
			if err != nil {
				panic(WrapMsg(err, "convert grpc status failed"))
			}
		}
	}
	return e
}

// New e error
func New(code codes.Code, message string) error {
	return NewWithPos(code, message, 1)
}

// Deprecated: use New instead.
func NewErrCodeMsg(errCode codes.Code, errMsg string) error {
	return NewWithPos(errCode, errMsg, 1)
}

// NewErrCode new an error with given code.
// if code has been predefined, it'll find error message auto.
// if not, it'll use the zero value of string.
func NewErrCode(errCode codes.Code) error {
	return NewWithPos(errCode, MapErrMsg(errCode), 1)
}

// NewErrMsg new an error with given error message.
// it'll fill field Code with UNKNOWN_ERROR.
func NewErrMsg(errMsg string) error {
	return NewWithPos(UNKNOWN_ERROR, errMsg, 1)
}

// NewWithPos new an error with caller info(file and line).
func NewWithPos(code codes.Code, message string, skip int) error {
	e := &Error{
		Code:    code,
		Msg:     message,
		Details: make(map[string]any),
	}
	_, file, line, _ := runtime.Caller(1 + skip)
	e.Details["file"] = file
	e.Details["line"] = line
	e.Details["stack"] = fmt.Sprintf(
		"%s:%d:%s\n",
		file,
		line,
		message,
	)

	return e
}

type ErrorDetails map[string]any

type Error struct {
	Code    codes.Code
	Msg     string
	Details ErrorDetails
	err     error
}

func (e *Error) WithDetails(details ErrorDetails) error {
	ori := e.Details
	e.Details = details
	e.Details["file"] = ori["file"]
	e.Details["line"] = ori["line"]
	e.Details["stack"] = ori["stack"]
	return e
}

func (e Error) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s", e.Msg, e.err.Error())
	}
	return e.Msg
}

func (e Error) Unwrap() error {
	return e.err
}

func (e Error) GRPCStatus() *status.Status {
	s := status.New(e.Code, e.Error())

	if len(e.Details) > 0 {
		var (
			err error
		)
		if err != nil { // 如果构建status报错，就直接使用报错信息构建status
			return WrapMsg(err, "build status failed").(*Error).GRPCStatus()
		}
		m := &pb.Map{}
		m.Fields, err = helper.Map2Pb(e.Details)
		if err != nil {
			return WrapMsg(err, "build status failed").(*Error).GRPCStatus()
		}
		s, err = s.WithDetails(m)
		if err != nil { // 如果构建status报错，就直接使用报错信息构建status
			return WrapMsg(err, "build status failed").(*Error).GRPCStatus()
		}
	}

	return s
}

// Format 实现Format接口来在打印error时展示更详细的信息,记录了调用栈
func (e Error) Format(s fmt.State, c rune) {
	switch c {
	case 'v':
		switch {
		case s.Flag('+'):
			_, _ = s.Write([]byte(fmt.Sprintf("%s:%d:%s\n", e.Details["file"], e.Details["line"], e.Msg)))
		case s.Flag('#'):
			_, _ = s.Write([]byte(e.Details["stack"].(string)))
		default:
			if e.err != nil {
				_, _ = s.Write([]byte(fmt.Sprintf("%s]<=[%v", e.Msg, e.err)))
			} else {
				_, _ = s.Write([]byte(e.Msg))
			}
		}
	}
}

func (e *Error) Wrap(err error) {
	e.err = err
}

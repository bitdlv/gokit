package errx

import (
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/status"
	"runtime"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	_, file, line, _ := runtime.Caller(0)

	err := New(INTERNAL_SERVER_ERROR, "test").(*Error)

	require.Equal(t, file, err.Details["file"])
	require.Equal(t, line+2, err.Details["line"])
}

func TestWrap(t *testing.T) {
	_, file, line, _ := runtime.Caller(0)

	err := NewErrMsg("test1").(*Error)
	err2 := WrapMsg(err, "test2").(*Error)

	require.Equal(t, file, err.Details["file"])
	require.Equal(t, line+2, err.Details["line"])

	require.Equal(t, file, err2.Details["file"])
	require.Equal(t, line+3, err2.Details["line"])
}

func TestFormat(t *testing.T) {
	_, file, line, _ := runtime.Caller(0)

	var err1 *Error
	errors.As(NewErrMsg("test1"), &err1)
	err2 := WrapMsg(err1, "test2").(*Error)
	err3 := WrapMsg(err2, "test3").(*Error)

	require.Equal(t, "test3]<=[test2]<=[test1", fmt.Sprintf("%v", err3))
	require.Equal(t, fmt.Sprintf("%s:%d:test3\n", file, line+4), fmt.Sprintf("%+v", err3))

	except := []string{
		fmt.Sprintf("%s:%d:%s\n", file, line+4, err3.Msg),
		fmt.Sprintf("%s:%d:%s\n", file, line+3, err2.Msg),
		fmt.Sprintf("%s:%d:%s\n", file, line+2, err1.Msg),
	}
	require.Equal(t, strings.Join(except, ""), fmt.Sprintf("%#v", err3))
	require.Equal(t, strings.Join(except, ""), err3.Details["stack"])
}

func TestStatus(t *testing.T) {
	_, file, line, _ := runtime.Caller(0)

	err := NewErrMsg("test") //新建一个error

	s, ok := status.FromError(err) //转status

	//判断各参数是否正常
	require.Equal(t, true, ok)
	require.Equal(t, UNKNOWN_ERROR, s.Code())
	require.Equal(t, "test", s.Message())

	//转回error，并且判断各参数是否正常
	err2 := NewFromStatus(s).(*Error)
	require.Equal(t, UNKNOWN_ERROR, err2.Code)
	require.Equal(t, "test", err2.Msg)
	require.Equal(t, int64(line+2), err2.Details["line"])
	require.Equal(t, file, err2.Details["file"])
}

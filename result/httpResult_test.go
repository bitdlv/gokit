package result

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bitdlv/gokit/errx"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"google.golang.org/grpc/status"
)

func TestResponse(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "http://www.baidu.com", strings.NewReader(""))
	w := httptest.NewRecorder()

	Response(r, w, nil, nil)
	require.Equal(t, http.StatusOK, w.Code)
	content, err := io.ReadAll(w.Body)
	require.Nil(t, err)
	require.Equal(t, int64(200), gjson.Get(string(content), "code").Int())
	require.Equal(t, "OK", gjson.Get(string(content), "msg").String())

	w = httptest.NewRecorder()
	err = errx.NewErrMsg("test")
	Response(r, w, nil, err)
	require.Equal(t, http.StatusOK, w.Code)
	content, err = io.ReadAll(w.Body)
	require.Nil(t, err)
	require.Equal(t, int64(100000), gjson.Get(string(content), "code").Int())
	require.Equal(t, "test", gjson.Get(string(content), "msg").String())

	// 调用栈展示
	w = httptest.NewRecorder()
	err = errx.NewErrMsg("test")
	err = errx.WrapMsg(err, "test2")
	s, _ := status.FromError(err)
	err = errx.NewFromStatus(s)
	Response(r, w, nil, err)
	require.Equal(t, http.StatusOK, w.Code)
	content, err = io.ReadAll(w.Body)
	require.Nil(t, err)
	require.Equal(t, int64(100000), gjson.Get(string(content), "code").Int())
	require.Equal(t, "test2", gjson.Get(string(content), "msg").String())

	// 普通错误
	w = httptest.NewRecorder()
	err = errors.New("test")
	Response(r, w, nil, err)
	require.Equal(t, http.StatusOK, w.Code)
	content, err = io.ReadAll(w.Body)
	require.Nil(t, err)
	require.Equal(t, int64(100000), gjson.Get(string(content), "code").Int())
	require.Equal(t, "test", gjson.Get(string(content), "msg").String())
}

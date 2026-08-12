package result

import (
	"encoding/json"
	"net/http"

	"github.com/bitdlv/gokit/errx"
	"github.com/bitdlv/gokit/errx/legacy"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/validation"
	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/status"
)

// ResponseMaskHook is an optional hook for response field desensitization.
// If set, it is called for every successful response before writing to the client.
// The hook receives the original *http.Request and the response data decoded as
// map[string]any, and must return the (possibly masked) map.
//
// Set this at application startup, for example:
//
//	h := grpcgw.NewResponseHandler4Api(rdb, db)
//	result.ResponseMaskHook = func(r *http.Request, data map[string]any) map[string]any {
//	    userIdStr := r.Header.Get(grpcgw.HeaderUserId)
//	    userId, _ := strconv.ParseInt(userIdStr, 10, 64)
//	    return h.ResponseHandle(r.Context(), userId, strings.ToUpper(r.Method), r.URL.Path, data)
//	}
var ResponseMaskHook func(r *http.Request, data any) any

// applyMask applies ResponseMaskHook to resp if the hook is configured and resp is
// non-nil. It round-trips through JSON to normalise resp into map[string]any so
// the hook can work generically across all response types.
func applyMask(r *http.Request, resp any) any {
	if ResponseMaskHook == nil || resp == nil {
		return resp
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return resp
	}
	var m any
	if err = json.Unmarshal(b, &m); err != nil {
		// resp might be a scalar / slice — return it unchanged
		return resp
	}
	return ResponseMaskHook(r, m)
}

func Response(r *http.Request, w http.ResponseWriter, resp any, err error) {
	if err == nil {
		httpx.OkJson(w, Success(applyMask(r, resp)))
		//httpx.WriteJson(w, http.StatusOK, Success(applyMask(r, resp)))
		return
	}
	logx.WithContext(r.Context()).Errorf("【API-ERR】 : %#v: %s ", err, err.Error())
	causeErr := errors.Cause(err)
	// 先看是不是status,  FromError中会使用errors.As找整个错误链
	if s, ok := status.FromError(causeErr); ok {
		e := errx.NewFromStatus(s).(*errx.Error)
		httpx.OkJson(w, Error(uint64(e.Code), e.Error()))
		//httpx.WriteJson(w, http.StatusOK, Error(uint64(e.Code), e.Error()))
		return
	}
	// 兜底
	httpx.OkJson(w, Error(uint64(errx.UNKNOWN_ERROR), err.Error()))
	//httpx.WriteJson(w, http.StatusOK, Error(uint64(errx.UNKNOWN_ERROR), err.Error()))
}

// HttpResult http返回
//
// Deprecated: use Response instead to avoid all http status error.
func HttpResult(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		// 成功返回
		r := Success(resp)
		httpx.WriteJson(w, http.StatusOK, r)
	} else {
		errcode := legacy.REUQEST_PARAM_ERROR
		errmsg := err.Error()
		// 错误返回 err 是 status.Status

		causeErr := errors.Cause(err) // err类型溯源 -> legacy.CodeError
		// 这里为啥捕获不到？？？大神解释下
		var parseErr error
		if e, ok := causeErr.(*legacy.CodeError); ok { // 自定义错误类型
			// 自定义CodeError
			errcode = e.GetErrCode()
			errmsg = e.GetErrMsg()
		} else if e, ok := causeErr.(validation.Validator); ok {
			errcode = legacy.REUQEST_PARAM_ERROR
			errmsg = e.Validate().Error()
		} else {
			goStatus, success := status.FromError(causeErr)
			if success { // grpc err错误
				// 退而求其次，正则解析出msg和error
				errcode, errmsg, parseErr = legacy.ParseError(goStatus.Message())
			}
			if parseErr != nil {
				errcode = legacy.UNKNOWN_ERROR
				errmsg = goStatus.Message()
			}
		}
		logx.WithContext(r.Context()).Errorf("【API-ERR】 : %+v ", err)

		httpx.WriteJson(w, http.StatusOK, Error(errcode, errmsg))
	}
}

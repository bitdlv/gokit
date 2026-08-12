# result包
> 对API返回结果做统一结构化, 若Resp出错，解析 xerr.ErrorCode

> API层返回统一的结构体 HttpResponse
```go
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
```
> 使用方法 在api的handler层，用result.HttpResult替代httpx.ErrorCtx和httpx.OkJsonCtx
```go

package item

import (
	"maintain/common/result"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"maintain/api/internal/logic/item"
	"maintain/api/internal/svc"
	"maintain/api/internal/types"
)

func AddHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Item
		if err := httpx.Parse(r, &req); err != nil {
			//httpx.ErrorCtx(r.Context(), w, err)
			//return
			result.Response(r, w, nil, err)
			
		}

		l := item.NewAddLogic(r.Context(), svcCtx)
		resp, err := l.Add(&req)
		//if err != nil {
		//	httpx.ErrorCtx(r.Context(), w, err)
		//} else {
		//	httpx.OkJsonCtx(r.Context(), w, resp)
		//}
		result.Response(r, w, resp, err)
	}
}
```
> result.Response(r, w, resp, err)中的err被RPC使用Status做了一层封装
> 我们希望在err中解析出 errCode和errMsg
> 详情见result包

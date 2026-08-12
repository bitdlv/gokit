package {{.PkgName}}

import (
    "git.zhiwei.dc-science.cn/business/common/result"
	"net/http"

	{{if .HasRequest}}"github.com/zeromicro/go-zero/rest/httpx"{{end}}
	{{.ImportPackages}}
)

func {{.HandlerName}}(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		{{if .HasRequest}}var req types.{{.RequestType}}
		if err := httpx.Parse(r, &req); err != nil {
            result.Response(r, w, nil, err)
			return
		}

		{{end}}l := {{.LogicName}}.New{{.LogicType}}(r.Context(), svcCtx)
		{{if .HasResp}}resp, {{end}}err := l.{{.Call}}({{if .HasRequest}}&req{{end}})
		if err != nil {
			result.Response(r, w, nil, err)
		} else {
			{{if .HasResp}}result.Response(r, w, resp, err){{else}}result.Response(r, w, nil, nil){{end}}
		}
	}
}

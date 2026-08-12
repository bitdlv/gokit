package helper

import (
	"context"
	"net/http"

	"google.golang.org/grpc/metadata"
)

func NewMetadataForRestToRpc(r *http.Request) *metadata.MD {
	areaGID := r.Header.Get("xjx-AreaGid")
	moduleGID := r.Header.Get("xjx-ModuleGid")
	operatorEmail := r.Header.Get("operatorEmail")

	md := metadata.New(map[string]string{
		"areagid":       areaGID,
		"modulegid":     moduleGID,
		"operatoremail": operatorEmail,
	})
	return &md
}

type headerKeyDef struct {
	AreaGid       string // 数据中心id
	ModuleGid     string // 数据中心-模组id
	OperatorEmail string // 当前账号邮箱
	Language      string // 系统语言
	Lang          string // 系统语言
}

var HeaderKey = headerKeyDef{
	AreaGid:       "areagid",
	ModuleGid:     "modulegid",
	OperatorEmail: "operatoremail",
	Language:      "language",
	Lang:          "lang",
}

func GetValueFromCtxByKey(ctx context.Context, key string) (value string, ok bool) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return
	}

	valueArr := md.Get(key)
	if len(valueArr) == 0 {
		return "", true
	}
	return valueArr[0], true
}

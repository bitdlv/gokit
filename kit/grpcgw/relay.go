package grpcgw

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"google.golang.org/grpc/metadata"
)

const (
	idKey       = "userid"
	emailKey    = "useremail"
	cnNameKey   = "usercnname"
	areaKey     = "areagid"
	moduleKey   = "modulegid"
	langKey     = "lang" //语言包
	defaultLang = "zh-cn"
)

func MetadataFromReq(r *http.Request) *metadata.MD {
	email := r.Header.Get("operator")
	areaGID := r.Header.Get("xjx-AreaGid")
	moduleGID := r.Header.Get("xjx-ModuleGid")
	lang := strings.ToLower(r.Header.Get("language"))
	cnName := url.QueryEscape(r.Header.Get("operatorName"))
	md := metadata.New(map[string]string{
		idKey:     r.Header.Get("operatorUid"),
		emailKey:  email,
		cnNameKey: cnName,
		areaKey:   areaGID,
		moduleKey: moduleGID,
		langKey:   lang,
	})
	return &md
}

func getFromMeta(md metadata.MD, keys ...string) map[string]string {
	resp := make(map[string]string)
	for _, k := range keys {
		r := md.Get(k)
		if len(r) == 0 {
			logx.Errorf("getFromMeta fail to get key[%s] from md[%+v]", k, md)
			continue
		}
		resp[k] = r[0]
	}
	return resp
}

func GetUser(ctx context.Context) (id int32, comboname, cnname string) {
	defer func() {
		if id == 0 {
			logx.Infof("cannot GetUser, use default admin[1] instead")
			id = 1
			comboname = "admin@example.com(管理员)"
			cnname = "管理员"
		}
	}()
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logx.Errorf("GetUser fail to get metadata")
		return
	}
	user := getFromMeta(md, idKey, emailKey, cnNameKey)
	if user == nil {
		logx.Errorf("GetUser fail to get user from md[%+v]", md)
		return
	}
	i, err := strconv.Atoi(user[idKey])
	if err != nil {
		logx.Errorf("GetUser fail to parse userid from[%s]", user[idKey])
		return
	}
	id = int32(i)
	cnname, err = url.QueryUnescape(user[cnNameKey])
	if err != nil {
		logx.Errorf("GetUser fail to decode cnname from[%s]", user[cnNameKey])
		return
	}
	comboname = user[emailKey] + "(" + cnname + ")"
	logx.Infof("GetUser: id[%d], comboname[%s], cnname[%s]", id, comboname, cnname)
	return
}

func GetLoc(ctx context.Context) (area, module string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logx.Errorf("GetLoc fail to get metadata")
		return
	}
	loc := getFromMeta(md, areaKey, moduleKey)
	if loc == nil {
		logx.Errorf("GetLoc fail to get loc from md[%+v]", md)
		return
	}
	area = loc[areaKey]
	module = loc[moduleKey]
	logx.Infof("GetLoc: area[%s], module[%s]", area, module)
	return
}

func GetLang(ctx context.Context) (lan string) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		logx.Errorf("GetLang fail to get metadata")
		return
	}
	llang := getFromMeta(md, langKey)
	if llang == nil {
		logx.Errorf("GetLang fail to get language from md[%+v]", md)
		return defaultLang
	}

	if lan, ok = llang[langKey]; ok {
		return lan
	}
	logx.Infof("GetLang: language [%s]", lan)
	return defaultLang
}

// HandleNonAsciiHeaders moved to gateway.go (identical implementation).

func Copy(ctx context.Context) context.Context {
	md, _ := metadata.FromIncomingContext(ctx)
	return metadata.NewOutgoingContext(ctx, md)
}

type MiddleWare func(next http.HandlerFunc) http.HandlerFunc

func MidWithWhiteList(mid MiddleWare, wl []string) MiddleWare {
	return func(next http.HandlerFunc) http.HandlerFunc {
		black := mid(next)
		return func(w http.ResponseWriter, r *http.Request) {
			logx.Infof("MidWithWhiteList: path[%s]", r.URL.Path)
			isIn := false
			for _, value := range wl {
				if value == r.URL.Path {
					isIn = true
					break
				}
			}
			if !isIn {
				logx.Infof("MidWithWhiteList: black")
				// do not modify `next` directly in handler, it will affect next call
				black(w, r)
				return
			}

			logx.Infof("MidWithWhiteList: white")
			next(w, r)
		}
	}
}
